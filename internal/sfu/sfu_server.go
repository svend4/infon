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
	packetsReceived uint64
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
	requirePassword bool
	password        string

	// Statistics
	totalPackets uint64
	totalBytes   uint64
}

// Stream represents an audio or video stream
type Stream struct {
	Type       StreamType
	SSRC       uint32 // Synchronization Source identifier
	lastPacket time.Time
	packets    uint64
	bytes      uint64
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
}

type PacketType byte

const (
	PacketTypeAudio PacketType = 0x01
	PacketTypeVideo PacketType = 0x02
	PacketTypeControl PacketType = 0x03
)

// NewSFUServer creates a new SFU server
func NewSFUServer(addr string) (*SFUServer, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	return &SFUServer{
		listenAddr:   udpAddr,
		participants: make(map[string]*Participant),
		rooms:        make(map[string]*Room),
		stopChan:     make(chan struct{}),
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
	s.mu.Unlock()

	fmt.Printf("SFU server listening on %s\n", s.listenAddr.String())

	// Start packet receiver
	go s.receiveLoop()

	// Start cleanup routine
	go s.cleanupLoop()

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
		s.conn.Close()
	}

	return nil
}

// receiveLoop receives and forwards packets
func (s *SFUServer) receiveLoop() {
	buffer := make([]byte, 65536)

	for {
		select {
		case <-s.stopChan:
			return
		default:
			n, addr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				if s.running {
					fmt.Printf("Read error: %v\n", err)
				}
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
	}

	// Find participant by address
	participant := s.findParticipantByAddr(addr)
	if participant != nil {
		packet.ParticipantID = participant.ID
		packet.RoomID = participant.RoomID

		// Update last seen
		participant.lastSeen = time.Now()
	}

	return packet
}

// forwardAudioPacket forwards audio to all participants in room
func (s *SFUServer) forwardAudioPacket(packet *Packet) {
	s.mu.RLock()
	room, exists := s.rooms[packet.RoomID]
	s.mu.RUnlock()

	if !exists {
		return
	}

	// Forward to all participants except sender
	for participantID, participant := range room.participants {
		if participantID == packet.ParticipantID {
			continue // Don't send back to sender
		}

		if participant.isMuted {
			continue // Skip muted participants
		}

		s.sendPacket(packet.Data, participant.Addr)

		// Update statistics
		s.mu.Lock()
		participant.bytesSent += uint64(len(packet.Data))
		s.packetsForwarded++
		s.bytesForwarded += uint64(len(packet.Data))
		s.mu.Unlock()
	}

	// Update room statistics
	room.totalPackets++
	room.totalBytes += uint64(len(packet.Data))
}

// forwardVideoPacket forwards video to all participants in room
func (s *SFUServer) forwardVideoPacket(packet *Packet) {
	s.mu.RLock()
	room, exists := s.rooms[packet.RoomID]
	s.mu.RUnlock()

	if !exists {
		return
	}

	// Forward to all participants except sender
	for participantID, participant := range room.participants {
		if participantID == packet.ParticipantID {
			continue
		}

		if participant.isPaused {
			continue // Skip participants with video paused
		}

		s.sendPacket(packet.Data, participant.Addr)

		// Update statistics
		s.mu.Lock()
		participant.bytesSent += uint64(len(packet.Data))
		s.packetsForwarded++
		s.bytesForwarded += uint64(len(packet.Data))
		s.mu.Unlock()
	}

	room.totalPackets++
	room.totalBytes += uint64(len(packet.Data))
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
	if len(packet.Data) < 10 {
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

	// Get or create room
	room, exists := s.rooms[roomID]
	if !exists {
		room = &Room{
			ID:              roomID,
			participants:    make(map[string]*Participant),
			created:         time.Now(),
			maxParticipants: 50, // Default max
		}
		s.rooms[roomID] = room
		s.roomsCreated++
	}

	// Check room capacity
	if len(room.participants) >= room.maxParticipants {
		// Send error back to client
		return
	}

	// Find existing participant by ID or create new
	participant, exists := s.participants[participantID]
	if !exists {
		// Parse address from packet data
		// In a real implementation, the address would come from the UDP packet source
		// For now, we'll need to track it separately
		participant = &Participant{
			ID:          participantID,
			RoomID:      roomID,
			Username:    username,
			joined:      time.Now(),
			lastSeen:    time.Now(),
			audioStream: &Stream{Type: StreamTypeAudio},
			videoStream: &Stream{Type: StreamTypeVideo},
		}

		s.participants[participantID] = participant
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

	participant := s.participants[packet.ParticipantID]
	if participant == nil {
		return
	}

	room := s.rooms[participant.RoomID]
	if room == nil {
		return
	}

	// Remove from room
	delete(room.participants, participant.ID)

	// Remove participant
	delete(s.participants, participant.ID)

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
		if participant.Addr != nil &&
			participant.Addr.IP.Equal(addr.IP) &&
			participant.Addr.Port == addr.Port {
			return participant
		}
	}

	return nil
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
func (s *SFUServer) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
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
