package group

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/pkg/terminal"
)

// PeerConnection represents a connection to a single peer in the group
type PeerConnection struct {
	ID          string          // Peer identifier (hostname or address)
	Address     *net.UDPAddr    // Peer's UDP address
	LastSeen    time.Time       // Last packet received time
	VideoFrame  *terminal.Frame // Latest video frame from peer
	AudioBuffer []int16         // Latest audio samples
	FrameCount  uint64          // Frames received
	AudioCount  uint64          // Audio packets received
	Stats       *PeerStats      // Connection statistics
	mutex       sync.RWMutex
}

// PeerStats tracks statistics for a peer connection
type PeerStats struct {
	BytesReceived uint64
	BytesSent     uint64
	PacketsLost   uint32
	Latency       time.Duration
	Jitter        time.Duration
}

// NewPeerConnection creates a new peer connection
func NewPeerConnection(id string, addr *net.UDPAddr) *PeerConnection {
	return &PeerConnection{
		ID:       id,
		Address:  addr,
		LastSeen: time.Now(),
		Stats:    &PeerStats{},
	}
}

// UpdateLastSeen updates the last seen timestamp
func (pc *PeerConnection) UpdateLastSeen() {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	pc.LastSeen = time.Now()
}

// LastSeenTime returns when a packet was last received from the peer. Read it
// through this (not the LastSeen field) while the receive goroutine may be
// calling UpdateLastSeen.
func (pc *PeerConnection) LastSeenTime() time.Time {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	return pc.LastSeen
}

// Counts returns how many video frames and audio packets were received from the
// peer, read under the same lock SetVideoFrame and SetAudioBuffer update them with.
func (pc *PeerConnection) Counts() (frames, audio uint64) {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	return pc.FrameCount, pc.AudioCount
}

// IsActive checks if peer is still active (received packet in last 10 seconds)
func (pc *PeerConnection) IsActive() bool {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	return time.Since(pc.LastSeen) < 10*time.Second
}

// SetVideoFrame updates the peer's video frame
func (pc *PeerConnection) SetVideoFrame(frame *terminal.Frame) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	pc.VideoFrame = frame
	pc.FrameCount++
}

// GetVideoFrame returns the peer's latest video frame
func (pc *PeerConnection) GetVideoFrame() *terminal.Frame {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	return pc.VideoFrame
}

// SetAudioBuffer updates the peer's audio buffer
func (pc *PeerConnection) SetAudioBuffer(samples []int16) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	pc.AudioBuffer = samples
	pc.AudioCount++
}

// GetAudioBuffer returns the peer's latest audio samples
func (pc *PeerConnection) GetAudioBuffer() []int16 {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	return pc.AudioBuffer
}

// GroupCall manages a multi-peer video call
type GroupCall struct {
	peers      map[string]*PeerConnection // Map of peer ID to connection
	transport  *network.Transport         // Network transport
	localPort  string                     // Local port
	peersMutex sync.RWMutex
	running    atomic.Bool
}

// NewGroupCall creates a new group call manager
func NewGroupCall(localPort string) (*GroupCall, error) {
	transport, err := network.NewTransport(localPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	return &GroupCall{
		peers:     make(map[string]*PeerConnection),
		transport: transport,
		localPort: localPort,
	}, nil
}

// AddPeer adds a peer to the group call
func (gc *GroupCall) AddPeer(id string, addr *net.UDPAddr) error {
	gc.peersMutex.Lock()
	defer gc.peersMutex.Unlock()

	if _, exists := gc.peers[id]; exists {
		return fmt.Errorf("peer %s already exists", id)
	}

	peer := NewPeerConnection(id, addr)
	gc.peers[id] = peer

	// Send join message to peer
	joinPacket := &network.Packet{
		Type:      network.PacketTypeControl,
		Sequence:  gc.transport.NextSequence(),
		Timestamp: uint64(time.Now().UnixMilli()),
		Payload:   []byte("JOIN"),
	}
	_ = gc.transport.SendPacket(joinPacket, addr)

	return nil
}

// RemovePeer removes a peer from the group call
func (gc *GroupCall) RemovePeer(id string) {
	gc.peersMutex.Lock()
	defer gc.peersMutex.Unlock()

	if peer, exists := gc.peers[id]; exists {
		// Send leave message
		leavePacket := &network.Packet{
			Type:      network.PacketTypeControl,
			Sequence:  gc.transport.NextSequence(),
			Timestamp: uint64(time.Now().UnixMilli()),
			Payload:   []byte("LEAVE"),
		}
		_ = gc.transport.SendPacket(leavePacket, peer.Address)

		delete(gc.peers, id)
	}
}

// GetPeer returns a peer by ID
func (gc *GroupCall) GetPeer(id string) (*PeerConnection, bool) {
	gc.peersMutex.RLock()
	defer gc.peersMutex.RUnlock()
	peer, exists := gc.peers[id]
	return peer, exists
}

// GetAllPeers returns all peers
func (gc *GroupCall) GetAllPeers() []*PeerConnection {
	gc.peersMutex.RLock()
	defer gc.peersMutex.RUnlock()

	peers := make([]*PeerConnection, 0, len(gc.peers))
	for _, peer := range gc.peers {
		peers = append(peers, peer)
	}
	return peers
}

// GetActivePeers returns only active peers
func (gc *GroupCall) GetActivePeers() []*PeerConnection {
	gc.peersMutex.RLock()
	defer gc.peersMutex.RUnlock()

	peers := make([]*PeerConnection, 0)
	for _, peer := range gc.peers {
		if peer.IsActive() {
			peers = append(peers, peer)
		}
	}
	return peers
}

// BroadcastPacket sends a packet to all peers
func (gc *GroupCall) BroadcastPacket(packet *network.Packet) {
	gc.peersMutex.RLock()
	defer gc.peersMutex.RUnlock()

	for _, peer := range gc.peers {
		if peer.IsActive() {
			_ = gc.transport.SendPacket(packet, peer.Address)
		}
	}
}

// SendToPeer sends a packet to a specific peer
func (gc *GroupCall) SendToPeer(peerID string, packet *network.Packet) error {
	gc.peersMutex.RLock()
	defer gc.peersMutex.RUnlock()

	peer, exists := gc.peers[peerID]
	if !exists {
		return fmt.Errorf("peer %s not found", peerID)
	}

	return gc.transport.SendPacket(packet, peer.Address)
}

// Start starts the group call
func (gc *GroupCall) Start() error {
	gc.running.Store(true)
	return nil
}

// Stop stops the group call. It is safe to call more than once. The goroutines
// started for the call poll IsRunning (not a channel), so Stop only has to flip the
// flag, send LEAVE to the peers and close the transport — the previous unbuffered
// stopChan send had no receiver and blocked Stop forever, hanging Ctrl+C.
func (gc *GroupCall) Stop() error {
	if !gc.running.Swap(false) {
		return nil // already stopped
	}

	// Send leave message to all peers
	for _, peer := range gc.GetAllPeers() {
		gc.RemovePeer(peer.ID)
	}

	if gc.transport != nil {
		_ = gc.transport.Close()
	}

	return nil
}

// IsRunning returns whether the group call is running
func (gc *GroupCall) IsRunning() bool {
	return gc.running.Load()
}

// GetTransport returns the network transport
func (gc *GroupCall) GetTransport() *network.Transport {
	return gc.transport
}

// PeerCount returns the number of peers
func (gc *GroupCall) PeerCount() int {
	gc.peersMutex.RLock()
	defer gc.peersMutex.RUnlock()
	return len(gc.peers)
}

// ActivePeerCount returns the number of active peers
func (gc *GroupCall) ActivePeerCount() int {
	return len(gc.GetActivePeers())
}

// PingInactive sends a keep-alive to every peer that has gone quiet. Media goes
// only to active peers (BroadcastPacket), so without this two peers that each
// stopped hearing the other — one started late, or the link dropped for more
// than the activity window — would never send to each other again. Any packet
// from a peer marks it active, so the call resumes as soon as either side hears
// the other. Returns how many peers were pinged.
func (gc *GroupCall) PingInactive() int {
	gc.peersMutex.RLock()
	defer gc.peersMutex.RUnlock()

	pinged := 0
	for _, peer := range gc.peers {
		if peer.IsActive() {
			continue
		}
		_ = gc.transport.SendPacket(&network.Packet{
			Type:      network.PacketTypeKeepAlive,
			Sequence:  gc.transport.NextSequence(),
			Timestamp: uint64(time.Now().UnixMilli()),
		}, peer.Address)
		pinged++
	}
	return pinged
}

// CleanupInactivePeers removes peers that haven't sent packets recently. A
// group whose membership is fixed up front (as `tvcp group` is) should not use
// it: a removed peer's later packets are no longer recognised, so it can never
// rejoin. Use PingInactive to keep such peers reachable instead.
func (gc *GroupCall) CleanupInactivePeers() {
	gc.peersMutex.Lock()
	defer gc.peersMutex.Unlock()

	toRemove := []string{}
	for id, peer := range gc.peers {
		if !peer.IsActive() {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		delete(gc.peers, id)
	}
}
