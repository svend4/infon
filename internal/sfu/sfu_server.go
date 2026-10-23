package sfu

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// SFU (Selective Forwarding Unit) Server for scalable group calls
// Receives streams from clients and forwards to other participants
// Reduces client bandwidth: N clients = N upload + 1 download (vs mesh: N*(N-1))

type SFUServer struct {
	mu sync.RWMutex

	// Server configuration
	listenAddr *net.UDPAddr
	conn       *net.UDPConn

	// Participants
	participants map[string]*Participant // participantID -> participant
	rooms        map[string]*Room        // roomID -> room

	// Server-wide caps on what join packets may create
	maxRooms        int
	maxParticipants int

	// Control
	running  bool
	stopChan chan struct{}

	// Statistics
	packetsReceived  uint64
	packetsForwarded uint64
	bytesReceived    uint64
	bytesForwarded   uint64
	roomsCreated     uint64
}

// Participant represents a connected client
type Participant struct {
	ID       string
	RoomID   string
	Addr     *net.UDPAddr
	Username string

	// Streams
	audioStream *Stream
	videoStream *Stream

	// State
	joined    time.Time
	lastSeen  time.Time
	isMuted   bool
	isPaused  bool

	// Statistics
	bytesSent       uint64
}

// Room represents a conference room
type Room struct {
	ID   string
	Name string

	participants map[string]*Participant // participantID -> participant

	created time.Time

	// Room settings
	maxParticipants int

	// Statistics
	totalPackets uint64
	totalBytes   uint64
}

// Stream represents an audio or video stream
type Stream struct {
	Type       StreamType
	SSRC       uint32 // Synchronization Source identifier
}

type StreamType int

const (
	StreamTypeAudio StreamType = iota
	StreamTypeVideo
)

// Packet represents a media packet
type Packet struct {
	Type          PacketType
	ParticipantID string
	RoomID        string
	SSRC          uint32
	SequenceNum   uint16
	Timestamp     uint32
	Data          []byte
	Addr          *net.UDPAddr // Source address of the datagram
}

type PacketType byte

const (
	PacketTypeAudio PacketType = 0x01
	PacketTypeVideo PacketType = 0x02
	PacketTypeControl PacketType = 0x03
)

// Join packets are unauthenticated and choose their own room and participant
// IDs, so the number of rooms and participants they can create is bounded.
const (
	DefaultMaxRooms        = 1000
	DefaultMaxParticipants = 5000
)

// NewSFUServer creates a new SFU server
func NewSFUServer(addr string) (*SFUServer, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	return &SFUServer{
		listenAddr:      udpAddr,
		participants:    make(map[string]*Participant),
		rooms:           make(map[string]*Room),
		maxRooms:        DefaultMaxRooms,
		maxParticipants: DefaultMaxParticipants,
		stopChan:        make(chan struct{}),
	}, nil
}

// Start starts the SFU server
func (s *SFUServer) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	conn, err := net.ListenUDP("udp", s.listenAddr)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.conn = conn
	s.running = true
	// Stop closes stopChan, so each run needs a fresh one.
	stop := make(chan struct{})
	s.stopChan = stop
	s.mu.Unlock()

	fmt.Printf("SFU server listening on %s\n", s.listenAddr.String())

	// Start packet receiver
	go s.receiveLoop(conn, stop)

	// Start cleanup routine
	go s.cleanupLoop(stop)

	return nil
}

// Stop stops the SFU server
func (s *SFUServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	close(s.stopChan)

	if s.conn != nil {
		_ = s.conn.Close()
	}

	return nil
}

// receiveLoop receives and forwards packets
func (s *SFUServer) receiveLoop(conn *net.UDPConn, stop <-chan struct{}) {
	buffer := make([]byte, 65536)

	for {
		select {
		case <-stop:
			return
		default:
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				select {
				case <-stop:
					return
				default:
				}
				fmt.Printf("Read error: %v\n", err)
				continue
			}

			s.handlePacket(buffer[:n], addr)
		}
	}
}

// handlePacket processes received packet
func (s *SFUServer) handlePacket(data []byte, addr *net.UDPAddr) {
	if len(data) < 8 {
		return // Too short
	}

	// Parse packet header
	packet := s.parsePacket(data, addr)
	if packet == nil {
		return
	}

	s.mu.Lock()
	s.packetsReceived++
	s.bytesReceived += uint64(len(data))
	s.mu.Unlock()

	// Handle based on type
	switch packet.Type {
	case PacketTypeAudio:
		s.forwardAudioPacket(packet)
	case PacketTypeVideo:
		s.forwardVideoPacket(packet)
	case PacketTypeControl:
		s.handleControlPacket(packet)
	}
}

// parsePacket parses packet from raw data
func (s *SFUServer) parsePacket(data []byte, addr *net.UDPAddr) *Packet {
	packet := &Packet{
		Type: PacketType(data[0]),
		Data: data,
		Addr: addr,
	}

	// Find participant by address
	s.mu.Lock()
	participant := s.findParticipantByAddrLocked(addr)
	if participant != nil {
		packet.ParticipantID = participant.ID
		packet.RoomID = participant.RoomID

		// Update last seen
		participant.lastSeen = time.Now()
	}
	s.mu.Unlock()

	return packet
}

// forwardAudioPacket forwards audio to all participants in room
func (s *SFUServer) forwardAudioPacket(packet *Packet) {
	// Skip muted participants
	s.forwardPacket(packet, func(p *Participant) bool { return p.isMuted })
}

// forwardVideoPacket forwards video to all participants in room
func (s *SFUServer) forwardVideoPacket(packet *Packet) {
	// Skip participants with video paused
	s.forwardPacket(packet, func(p *Participant) bool { return p.isPaused })
}

// forwardTarget is a recipient snapshotted under s.mu
type forwardTarget struct {
	participant *Participant
	addr        *net.UDPAddr
}

// forwardPacket forwards packet to every participant in its room except the
// sender and those for which skip returns true. Recipients are snapshotted
// under the read lock so room membership can change (e.g. stale participant
// cleanup) while packets are being sent.
func (s *SFUServer) forwardPacket(packet *Packet, skip func(*Participant) bool) {
	s.mu.RLock()
	room, exists := s.rooms[packet.RoomID]
	if !exists {
		s.mu.RUnlock()
		return
	}

	targets := make([]forwardTarget, 0, len(room.participants))
	for participantID, participant := range room.participants {
		if participantID == packet.ParticipantID {
			continue // Don't send back to sender
		}

		if skip(participant) {
			continue
		}

		targets = append(targets, forwardTarget{participant: participant, addr: participant.Addr})
	}
	s.mu.RUnlock()

	for _, target := range targets {
		s.sendPacket(packet.Data, target.addr)

		// Update statistics
		s.mu.Lock()
		target.participant.bytesSent += uint64(len(packet.Data))
		s.packetsForwarded++
		s.bytesForwarded += uint64(len(packet.Data))
		s.mu.Unlock()
	}

	// Update room statistics
	s.mu.Lock()
	room.totalPackets++
	room.totalBytes += uint64(len(packet.Data))
	s.mu.Unlock()
}

// handleControlPacket handles control messages
func (s *SFUServer) handleControlPacket(packet *Packet) {
	// Parse control message type
	if len(packet.Data) < 2 {
		return
	}

	controlType := packet.Data[1]

	switch controlType {
	case 0x01: // Join room
		s.handleJoinRoom(packet)
	case 0x02: // Leave room
		s.handleLeaveRoom(packet)
	case 0x03: // Mute
		s.handleMute(packet)
	case 0x04: // Unmute
		s.handleUnmute(packet)
	case 0x05: // Pause video
		s.handlePauseVideo(packet)
	case 0x06: // Resume video
		s.handleResumeVideo(packet)
	}
}

// handleJoinRoom handles participant joining
func (s *SFUServer) handleJoinRoom(packet *Packet) {
	// Parse join message (format: 0x03 0x01 roomID participantID username)
	if len(packet.Data) < 18 {
		return
	}

	roomID := string(packet.Data[2:10])
	participantID := string(packet.Data[10:18])
	username := ""
	if len(packet.Data) > 18 {
		username = string(packet.Data[18:])
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	participant, known := s.participants[participantID]
	if known && participant.Addr != nil && packet.Addr != nil && !sameUDPAddr(participant.Addr, packet.Addr) {
		// Joins are unauthenticated: re-binding the ID to whoever sent this would
		// hand them the participant's media. A client whose address really changed
		// can join again once the old binding goes stale.
		return
	}
	if !known && packet.Addr != nil {
		// One participant per source address: a client that comes back under a new
		// ID (a restart) replaces its old participant instead of adding another.
		if prev := s.findParticipantByAddrLocked(packet.Addr); prev != nil {
			s.removeParticipantLocked(prev)
		}
	}
	if !known && len(s.participants) >= s.maxParticipants {
		return
	}

	// Get or create room
	room, exists := s.rooms[roomID]
	if !exists {
		if len(s.rooms) >= s.maxRooms {
			return
		}
		room = &Room{
			ID:              roomID,
			participants:    make(map[string]*Participant),
			created:         time.Now(),
			maxParticipants: 50, // Default max
		}
		s.rooms[roomID] = room
		s.roomsCreated++
	}

	// Check room capacity (a member re-joining does not take another slot)
	if _, member := room.participants[participantID]; !member && len(room.participants) >= room.maxParticipants {
		// Send error back to client
		return
	}

	// Use the existing participant or create a new one
	if !known {
		participant = &Participant{
			ID:          participantID,
			RoomID:      roomID,
			Addr:        packet.Addr, // Media is sent to the join's source address
			Username:    username,
			joined:      time.Now(),
			lastSeen:    time.Now(),
			audioStream: &Stream{Type: StreamTypeAudio},
			videoStream: &Stream{Type: StreamTypeVideo},
		}

		s.participants[participantID] = participant
	} else {
		if participant.Addr == nil {
			participant.Addr = packet.Addr
		}
		// Joining another room leaves the old one
		if participant.RoomID != roomID {
			if old := s.rooms[participant.RoomID]; old != nil {
				delete(old.participants, participantID)
				if len(old.participants) == 0 {
					delete(s.rooms, old.ID)
				}
				s.broadcastParticipantLeft(old, participant)
			}
			participant.RoomID = roomID
		}
	}

	// Add to room
	room.participants[participantID] = participant

	fmt.Printf("Participant %s (%s) joined room %s\n", participantID, username, roomID)

	// Notify other participants
	s.broadcastParticipantJoined(room, participant)
}

// handleLeaveRoom handles participant leaving
func (s *SFUServer) handleLeaveRoom(packet *Packet) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if participant := s.participants[packet.ParticipantID]; participant != nil {
		s.removeParticipantLocked(participant)
	}
}

// removeParticipantLocked removes a participant from its room and the server,
// deleting the room if it empties and notifying the members left. s.mu must be held.
func (s *SFUServer) removeParticipantLocked(participant *Participant) {
	delete(s.participants, participant.ID)

	room := s.rooms[participant.RoomID]
	if room == nil {
		return
	}
	delete(room.participants, participant.ID)

	fmt.Printf("Participant %s left room %s\n", participant.ID, room.ID)

	// Delete room if empty
	if len(room.participants) == 0 {
		delete(s.rooms, room.ID)
	}

	// Notify other participants
	s.broadcastParticipantLeft(room, participant)
}

// handleMute sets participant mute status
func (s *SFUServer) handleMute(packet *Packet) {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.participants[packet.ParticipantID]
	if participant != nil {
		participant.isMuted = true
	}
}

// handleUnmute sets participant unmute status
func (s *SFUServer) handleUnmute(packet *Packet) {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.participants[packet.ParticipantID]
	if participant != nil {
		participant.isMuted = false
	}
}

// handlePauseVideo sets participant video pause
func (s *SFUServer) handlePauseVideo(packet *Packet) {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.participants[packet.ParticipantID]
	if participant != nil {
		participant.isPaused = true
	}
}

// handleResumeVideo sets participant video resume
func (s *SFUServer) handleResumeVideo(packet *Packet) {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.participants[packet.ParticipantID]
	if participant != nil {
		participant.isPaused = false
	}
}

// sendPacket sends packet to address
func (s *SFUServer) sendPacket(data []byte, addr *net.UDPAddr) {
	if addr == nil {
		return // Participant has no known address yet
	}
	_, err := s.conn.WriteToUDP(data, addr)
	if err != nil {
		fmt.Printf("Send error: %v\n", err)
	}
}

// findParticipantByAddr finds participant by UDP address
func (s *SFUServer) findParticipantByAddr(addr *net.UDPAddr) *Participant {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.findParticipantByAddrLocked(addr)
}

func (s *SFUServer) findParticipantByAddrLocked(addr *net.UDPAddr) *Participant {
	if addr == nil {
		return nil
	}

	// Find participant matching this address
	for _, participant := range s.participants {
		if participant.Addr != nil && sameUDPAddr(participant.Addr, addr) {
			return participant
		}
	}

	return nil
}

func sameUDPAddr(a, b *net.UDPAddr) bool {
	return a.IP.Equal(b.IP) && a.Port == b.Port
}

// broadcastParticipantJoined notifies room of new participant
func (s *SFUServer) broadcastParticipantJoined(room *Room, newParticipant *Participant) {
	// Send notification packet to all participants
	message := []byte{byte(PacketTypeControl), 0x10} // 0x10 = participant joined
	message = append(message, []byte(newParticipant.ID)...)

	for _, participant := range room.participants {
		if participant.ID != newParticipant.ID {
			s.sendPacket(message, participant.Addr)
		}
	}
}

// broadcastParticipantLeft notifies room of participant leaving
func (s *SFUServer) broadcastParticipantLeft(room *Room, leftParticipant *Participant) {
	message := []byte{byte(PacketTypeControl), 0x11} // 0x11 = participant left
	message = append(message, []byte(leftParticipant.ID)...)

	for _, participant := range room.participants {
		s.sendPacket(message, participant.Addr)
	}
}

// cleanupLoop removes stale participants
func (s *SFUServer) cleanupLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.cleanupStaleParticipants()
		}
	}
}

// cleanupStaleParticipants removes participants that haven't been seen
func (s *SFUServer) cleanupStaleParticipants() {
	s.mu.Lock()
	defer s.mu.Unlock()

	timeout := 30 * time.Second
	now := time.Now()

	for participantID, participant := range s.participants {
		if now.Sub(participant.lastSeen) > timeout {
			// Remove stale participant
			room := s.rooms[participant.RoomID]
			if room != nil {
				delete(room.participants, participantID)

				// Delete empty rooms
				if len(room.participants) == 0 {
					delete(s.rooms, room.ID)
				}
			}

			delete(s.participants, participantID)

			fmt.Printf("Removed stale participant: %s\n", participantID)
		}
	}
}

// GetStatistics returns server statistics
func (s *SFUServer) GetStatistics() SFUStatistics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SFUStatistics{
		TotalParticipants: len(s.participants),
		TotalRooms:        len(s.rooms),
		PacketsReceived:   s.packetsReceived,
		PacketsForwarded:  s.packetsForwarded,
		BytesReceived:     s.bytesReceived,
		BytesForwarded:    s.bytesForwarded,
		RoomsCreated:      s.roomsCreated,
	}
}

// GetRooms returns list of active rooms
func (s *SFUServer) GetRooms() []RoomInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rooms := make([]RoomInfo, 0, len(s.rooms))
	for _, room := range s.rooms {
		rooms = append(rooms, RoomInfo{
			ID:               room.ID,
			Name:             room.Name,
			ParticipantCount: len(room.participants),
			Created:          room.created,
		})
	}

	return rooms
}

// SFUStatistics represents server statistics
type SFUStatistics struct {
	TotalParticipants int
	TotalRooms        int
	PacketsReceived   uint64
	PacketsForwarded  uint64
	BytesReceived     uint64
	BytesForwarded    uint64
	RoomsCreated      uint64
}

// RoomInfo represents room information
type RoomInfo struct {
	ID               string
	Name             string
	ParticipantCount int
	Created          time.Time
}

// SetParticipantAddress associates a UDP address with a participant
func (s *SFUServer) SetParticipantAddress(participantID string, addr *net.UDPAddr) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant, exists := s.participants[participantID]
	if !exists {
		return fmt.Errorf("participant not found: %s", participantID)
	}

	participant.Addr = addr
	return nil
}

// GetRoom returns information about a specific room
func (s *SFUServer) GetRoom(roomID string) (*RoomInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, exists := s.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	return &RoomInfo{
		ID:               room.ID,
		Name:             room.Name,
		ParticipantCount: len(room.participants),
		Created:          room.created,
	}, nil
}

// GetParticipantsInRoom returns list of participants in a room
func (s *SFUServer) GetParticipantsInRoom(roomID string) ([]*Participant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, exists := s.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	participants := make([]*Participant, 0, len(room.participants))
	for _, p := range room.participants {
		participants = append(participants, p)
	}

	return participants, nil
}

// CreateRoom creates a new room with the given settings
func (s *SFUServer) CreateRoom(roomID, name string, maxParticipants int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rooms[roomID]; exists {
		return fmt.Errorf("room already exists: %s", roomID)
	}

	room := &Room{
		ID:              roomID,
		Name:            name,
		participants:    make(map[string]*Participant),
		created:         time.Now(),
		maxParticipants: maxParticipants,
	}

	s.rooms[roomID] = room
	s.roomsCreated++

	return nil
}

// DeleteRoom removes a room and disconnects all participants
func (s *SFUServer) DeleteRoom(roomID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, exists := s.rooms[roomID]
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	// Remove all participants from the room
	for participantID := range room.participants {
		delete(s.participants, participantID)
	}

	delete(s.rooms, roomID)

	return nil
}
