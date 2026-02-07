package stun

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

// ICECandidateType represents the type of ICE candidate
type ICECandidateType string

const (
	CandidateTypeHost  ICECandidateType = "host"
	CandidateTypeSrflx ICECandidateType = "srflx" // Server reflexive
	CandidateTypeRelay ICECandidateType = "relay"
)

// ICECandidate represents an ICE candidate
type ICECandidate struct {
	Type       ICECandidateType
	Address    *net.UDPAddr
	Base       *net.UDPAddr // Base address for derived candidates
	Priority   uint32
	Foundation string
	Component  int // 1 for RTP, 2 for RTCP
}

// ICEAgent handles ICE candidate gathering and connectivity checks
type ICEAgent struct {
	mu sync.RWMutex

	localCandidates  []*ICECandidate
	remoteCandidates []*ICECandidate
	candidatePairs   []*CandidatePair

	stunClient *STUNClient
	turnClient *TURNClient

	state ICEState

	// Configuration
	stunServers []string
	turnServers []TURNServer

	// Statistics
	candidatesGathered uint64
	checksPerformed    uint64
	checksSucceeded    uint64
	checksFailed       uint64

	// Selected pair
	selectedPair *CandidatePair
}

// ICEState represents the ICE agent state
type ICEState string

const (
	ICEStateNew         ICEState = "new"
	ICEStateGathering   ICEState = "gathering"
	ICEStateComplete    ICEState = "complete"
	ICEStateChecking    ICEState = "checking"
	ICEStateConnected   ICEState = "connected"
	ICEStateFailed      ICEState = "failed"
	ICEStateClosed      ICEState = "closed"
)

// CandidatePair represents a local-remote candidate pair
type CandidatePair struct {
	Local    *ICECandidate
	Remote   *ICECandidate
	Priority uint64
	State    PairState
	RTT      time.Duration
}

// PairState represents the state of a candidate pair
type PairState string

const (
	PairStateWaiting    PairState = "waiting"
	PairStateInProgress PairState = "in_progress"
	PairStateSucceeded  PairState = "succeeded"
	PairStateFailed     PairState = "failed"
)

// TURNServer represents TURN server configuration
type TURNServer struct {
	Address  string
	Username string
	Password string
}

// NewICEAgent creates a new ICE agent
func NewICEAgent(stunServers []string, turnServers []TURNServer) *ICEAgent {
	return &ICEAgent{
		state:            ICEStateNew,
		stunServers:      stunServers,
		turnServers:      turnServers,
		localCandidates:  make([]*ICECandidate, 0),
		remoteCandidates: make([]*ICECandidate, 0),
		candidatePairs:   make([]*CandidatePair, 0),
	}
}

// GatherCandidates gathers all ICE candidates
func (ia *ICEAgent) GatherCandidates() error {
	ia.mu.Lock()
	ia.state = ICEStateGathering
	ia.mu.Unlock()

	// Gather host candidates
	hostCandidates, err := ia.gatherHostCandidates()
	if err != nil {
		return fmt.Errorf("failed to gather host candidates: %w", err)
	}

	ia.mu.Lock()
	ia.localCandidates = append(ia.localCandidates, hostCandidates...)
	ia.candidatesGathered += uint64(len(hostCandidates))
	ia.mu.Unlock()

	// Gather server reflexive candidates (STUN)
	for _, stunServer := range ia.stunServers {
		srflxCandidates, err := ia.gatherSrflxCandidates(stunServer)
		if err != nil {
			// Log error but continue
			continue
		}

		ia.mu.Lock()
		ia.localCandidates = append(ia.localCandidates, srflxCandidates...)
		ia.candidatesGathered += uint64(len(srflxCandidates))
		ia.mu.Unlock()
	}

	// Gather relay candidates (TURN)
	for _, turnServer := range ia.turnServers {
		relayCandidates, err := ia.gatherRelayCandidates(turnServer)
		if err != nil {
			// Log error but continue
			continue
		}

		ia.mu.Lock()
		ia.localCandidates = append(ia.localCandidates, relayCandidates...)
		ia.candidatesGathered += uint64(len(relayCandidates))
		ia.mu.Unlock()
	}

	ia.mu.Lock()
	ia.state = ICEStateComplete
	ia.mu.Unlock()

	return nil
}

// AddRemoteCandidate adds a remote ICE candidate
func (ia *ICEAgent) AddRemoteCandidate(candidate *ICECandidate) {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	ia.remoteCandidates = append(ia.remoteCandidates, candidate)

	// Form candidate pairs with all local candidates
	for _, local := range ia.localCandidates {
		pair := &CandidatePair{
			Local:    local,
			Remote:   candidate,
			Priority: ia.calculatePairPriority(local, candidate),
			State:    PairStateWaiting,
		}
		ia.candidatePairs = append(ia.candidatePairs, pair)
	}

	// Sort pairs by priority (highest first)
	sort.Slice(ia.candidatePairs, func(i, j int) bool {
		return ia.candidatePairs[i].Priority > ia.candidatePairs[j].Priority
	})
}

// StartConnectivityChecks begins ICE connectivity checks
func (ia *ICEAgent) StartConnectivityChecks() error {
	ia.mu.Lock()
	if len(ia.candidatePairs) == 0 {
		ia.mu.Unlock()
		return fmt.Errorf("no candidate pairs to check")
	}
	ia.state = ICEStateChecking
	ia.mu.Unlock()

	// Perform checks in priority order
	for _, pair := range ia.candidatePairs {
		ia.mu.RLock()
		if ia.state == ICEStateConnected {
			ia.mu.RUnlock()
			break
		}
		ia.mu.RUnlock()

		if err := ia.performConnectivityCheck(pair); err == nil {
			// Check succeeded
			ia.mu.Lock()
			pair.State = PairStateSucceeded
			ia.checksSucceeded++
			ia.mu.Unlock()

			// Use first successful pair
			if ia.selectedPair == nil {
				ia.mu.Lock()
				ia.selectedPair = pair
				ia.state = ICEStateConnected
				ia.mu.Unlock()
				break
			}
		} else {
			ia.mu.Lock()
			pair.State = PairStateFailed
			ia.checksFailed++
			ia.mu.Unlock()
		}
	}

	ia.mu.RLock()
	state := ia.state
	ia.mu.RUnlock()

	if state != ICEStateConnected {
		ia.mu.Lock()
		ia.state = ICEStateFailed
		ia.mu.Unlock()
		return fmt.Errorf("no valid candidate pairs found")
	}

	return nil
}

// GetSelectedPair returns the selected candidate pair
func (ia *ICEAgent) GetSelectedPair() *CandidatePair {
	ia.mu.RLock()
	defer ia.mu.RUnlock()

	return ia.selectedPair
}

// GetLocalCandidates returns all local candidates
func (ia *ICEAgent) GetLocalCandidates() []*ICECandidate {
	ia.mu.RLock()
	defer ia.mu.RUnlock()

	// Return a copy
	candidates := make([]*ICECandidate, len(ia.localCandidates))
	copy(candidates, ia.localCandidates)
	return candidates
}

// GetState returns the current ICE state
func (ia *ICEAgent) GetState() ICEState {
	ia.mu.RLock()
	defer ia.mu.RUnlock()

	return ia.state
}

// GetStatistics returns ICE agent statistics
func (ia *ICEAgent) GetStatistics() ICEStatistics {
	ia.mu.RLock()
	defer ia.mu.RUnlock()

	return ICEStatistics{
		State:              ia.state,
		CandidatesGathered: ia.candidatesGathered,
		ChecksPerformed:    ia.checksPerformed,
		ChecksSucceeded:    ia.checksSucceeded,
		ChecksFailed:       ia.checksFailed,
		LocalCandidates:    len(ia.localCandidates),
		RemoteCandidates:   len(ia.remoteCandidates),
		CandidatePairs:     len(ia.candidatePairs),
		SelectedPair:       ia.selectedPair,
	}
}

// ICEStatistics represents ICE agent statistics
type ICEStatistics struct {
	State              ICEState
	CandidatesGathered uint64
	ChecksPerformed    uint64
	ChecksSucceeded    uint64
	ChecksFailed       uint64
	LocalCandidates    int
	RemoteCandidates   int
	CandidatePairs     int
	SelectedPair       *CandidatePair
}

// Close closes the ICE agent
func (ia *ICEAgent) Close() error {
	ia.mu.Lock()
	defer ia.mu.Unlock()

	ia.state = ICEStateClosed

	if ia.stunClient != nil {
		ia.stunClient.Close()
	}

	if ia.turnClient != nil {
		ia.turnClient.Close()
	}

	return nil
}

// Private helper methods

func (ia *ICEAgent) gatherHostCandidates() ([]*ICECandidate, error) {
	candidates := make([]*ICECandidate, 0)

	// Get all local network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// Skip IPv6 for simplicity
			if ipNet.IP.To4() == nil {
				continue
			}

			// Create host candidate on ephemeral port
			udpAddr := &net.UDPAddr{
				IP:   ipNet.IP,
				Port: 0, // OS will assign port
			}

			candidate := &ICECandidate{
				Type:       CandidateTypeHost,
				Address:    udpAddr,
				Base:       udpAddr,
				Priority:   ia.calculateCandidatePriority(CandidateTypeHost, 0),
				Foundation: ia.generateFoundation(CandidateTypeHost, udpAddr.IP),
				Component:  1,
			}

			candidates = append(candidates, candidate)
		}
	}

	return candidates, nil
}

func (ia *ICEAgent) gatherSrflxCandidates(stunServer string) ([]*ICECandidate, error) {
	candidates := make([]*ICECandidate, 0)

	// Create STUN client
	stunClient := NewSTUNClient(stunServer)
	ia.stunClient = stunClient

	// Get mapped address
	mappedAddr, err := stunClient.GetMappedAddress()
	if err != nil {
		return nil, err
	}

	// Get local address
	localAddr := stunClient.GetLocalAddress()

	candidate := &ICECandidate{
		Type:       CandidateTypeSrflx,
		Address:    mappedAddr,
		Base:       localAddr,
		Priority:   ia.calculateCandidatePriority(CandidateTypeSrflx, 0),
		Foundation: ia.generateFoundation(CandidateTypeSrflx, mappedAddr.IP),
		Component:  1,
	}

	candidates = append(candidates, candidate)

	return candidates, nil
}

func (ia *ICEAgent) gatherRelayCandidates(turnServer TURNServer) ([]*ICECandidate, error) {
	candidates := make([]*ICECandidate, 0)

	// Create TURN client
	turnClient := NewTURNClient(turnServer.Address, turnServer.Username, turnServer.Password)
	ia.turnClient = turnClient

	// Allocate relay address
	relayAddr, err := turnClient.Allocate()
	if err != nil {
		return nil, err
	}

	candidate := &ICECandidate{
		Type:       CandidateTypeRelay,
		Address:    relayAddr,
		Base:       relayAddr,
		Priority:   ia.calculateCandidatePriority(CandidateTypeRelay, 0),
		Foundation: ia.generateFoundation(CandidateTypeRelay, relayAddr.IP),
		Component:  1,
	}

	candidates = append(candidates, candidate)

	return candidates, nil
}

func (ia *ICEAgent) performConnectivityCheck(pair *CandidatePair) error {
	ia.mu.Lock()
	pair.State = PairStateInProgress
	ia.checksPerformed++
	ia.mu.Unlock()

	// Create STUN binding request
	stunClient := NewSTUNClient(pair.Remote.Address.String())
	defer stunClient.Close()

	start := time.Now()

	// Try to connect
	_, err := stunClient.GetMappedAddress()
	if err != nil {
		return err
	}

	pair.RTT = time.Since(start)

	return nil
}

func (ia *ICEAgent) calculateCandidatePriority(candType ICECandidateType, localPref uint16) uint32 {
	// Priority = (2^24)*(type preference) + (2^8)*(local preference) + (2^0)*(256 - component ID)
	var typePref uint32

	switch candType {
	case CandidateTypeHost:
		typePref = 126
	case CandidateTypeSrflx:
		typePref = 100
	case CandidateTypeRelay:
		typePref = 0
	default:
		typePref = 0
	}

	priority := (1 << 24) * typePref
	priority += (1 << 8) * uint32(localPref)
	priority += (256 - 1) // Component 1

	return priority
}

func (ia *ICEAgent) calculatePairPriority(local, remote *ICECandidate) uint64 {
	// RFC 5245: pair priority = 2^32*MIN(G,D) + 2*MAX(G,D) + (G>D?1:0)
	G := uint64(local.Priority)
	D := uint64(remote.Priority)

	var minPrio, maxPrio uint64
	var tieBreaker uint64

	if G > D {
		minPrio = D
		maxPrio = G
		tieBreaker = 1
	} else {
		minPrio = G
		maxPrio = D
		tieBreaker = 0
	}

	priority := (1 << 32) * minPrio
	priority += 2 * maxPrio
	priority += tieBreaker

	return priority
}

func (ia *ICEAgent) generateFoundation(candType ICECandidateType, ip net.IP) string {
	// Simplified foundation: just use type + first octet of IP
	return fmt.Sprintf("%s-%d", candType, ip[0])
}

// FormatCandidate formats an ICE candidate as SDP attribute
func FormatCandidate(c *ICECandidate) string {
	return fmt.Sprintf("candidate:%s %d udp %d %s %d typ %s",
		c.Foundation,
		c.Component,
		c.Priority,
		c.Address.IP.String(),
		c.Address.Port,
		c.Type,
	)
}

// ParseCandidate parses an ICE candidate from SDP format
func ParseCandidate(sdp string) (*ICECandidate, error) {
	// Simplified parsing
	// Format: "candidate:foundation component transport priority ip port typ type"

	var foundation string
	var component int
	var transport string
	var priority uint32
	var ip string
	var port int
	var typ string
	var candType string

	_, err := fmt.Sscanf(sdp, "candidate:%s %d %s %d %s %d typ %s",
		&foundation, &component, &transport, &priority, &ip, &port, &typ, &candType)

	if err != nil {
		return nil, fmt.Errorf("failed to parse candidate: %w", err)
	}

	addr := &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	var cType ICECandidateType
	switch candType {
	case "host":
		cType = CandidateTypeHost
	case "srflx":
		cType = CandidateTypeSrflx
	case "relay":
		cType = CandidateTypeRelay
	default:
		return nil, fmt.Errorf("unknown candidate type: %s", candType)
	}

	return &ICECandidate{
		Type:       cType,
		Address:    addr,
		Base:       addr,
		Priority:   priority,
		Foundation: foundation,
		Component:  component,
	}, nil
}
