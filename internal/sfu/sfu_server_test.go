package sfu

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestNewSFUServer(t *testing.T) {
	server, err := NewSFUServer("127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Failed to create SFU server: %v", err)
	}

	if server == nil {
		t.Fatal("Server is nil")
	}

	if server.listenAddr == nil {
		t.Fatal("Listen address is nil")
	}

	if server.listenAddr.Port != 8080 {
		t.Errorf("Port = %d, expected 8080", server.listenAddr.Port)
	}
}

func TestNewSFUServerInvalidAddress(t *testing.T) {
	_, err := NewSFUServer("invalid:address:format")
	if err == nil {
		t.Error("Expected error for invalid address")
	}
}

func TestServerStartStop(t *testing.T) {
	server, err := NewSFUServer("127.0.0.1:0") // Port 0 = auto-assign
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	if !server.running {
		t.Error("Server should be running")
	}

	// Try to start again (should fail)
	err = server.Start()
	if err == nil {
		t.Error("Starting already running server should fail")
	}

	// Stop server
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	if server.running {
		t.Error("Server should not be running")
	}

	// Stop again (should be safe)
	err = server.Stop()
	if err != nil {
		t.Error("Stopping already stopped server should not error")
	}
}

func TestCreateRoom(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")

	// Create room
	err := server.CreateRoom("room1", "Test Room", 10)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	// Verify room exists
	room, err := server.GetRoom("room1")
	if err != nil {
		t.Fatalf("Failed to get room: %v", err)
	}

	if room.ID != "room1" {
		t.Errorf("Room ID = %s, expected room1", room.ID)
	}

	if room.Name != "Test Room" {
		t.Errorf("Room name = %s, expected Test Room", room.Name)
	}

	// Try to create duplicate room
	err = server.CreateRoom("room1", "Duplicate", 10)
	if err == nil {
		t.Error("Creating duplicate room should fail")
	}
}

func TestDeleteRoom(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")

	// Create and delete room
	server.CreateRoom("room1", "Test Room", 10)

	err := server.DeleteRoom("room1")
	if err != nil {
		t.Fatalf("Failed to delete room: %v", err)
	}

	// Verify room is gone
	_, err = server.GetRoom("room1")
	if err == nil {
		t.Error("Getting deleted room should fail")
	}

	// Try to delete non-existent room
	err = server.DeleteRoom("room1")
	if err == nil {
		t.Error("Deleting non-existent room should fail")
	}
}

func TestGetRooms(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")

	// Initially empty
	rooms := server.GetRooms()
	if len(rooms) != 0 {
		t.Errorf("Initial rooms count = %d, expected 0", len(rooms))
	}

	// Create some rooms
	server.CreateRoom("room1", "Room 1", 10)
	server.CreateRoom("room2", "Room 2", 20)
	server.CreateRoom("room3", "Room 3", 30)

	rooms = server.GetRooms()
	if len(rooms) != 3 {
		t.Errorf("Rooms count = %d, expected 3", len(rooms))
	}
}

func TestSetParticipantAddress(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")

	// Create participant manually
	server.mu.Lock()
	participant := &Participant{
		ID:       "p1",
		RoomID:   "room1",
		Username: "User1",
		joined:   time.Now(),
		lastSeen: time.Now(),
	}
	server.participants["p1"] = participant
	server.mu.Unlock()

	// Set address
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	err := server.SetParticipantAddress("p1", addr)
	if err != nil {
		t.Fatalf("Failed to set participant address: %v", err)
	}

	if participant.Addr == nil {
		t.Fatal("Participant address is nil")
	}

	if participant.Addr.Port != 12345 {
		t.Errorf("Port = %d, expected 12345", participant.Addr.Port)
	}

	// Try to set address for non-existent participant
	err = server.SetParticipantAddress("p999", addr)
	if err == nil {
		t.Error("Setting address for non-existent participant should fail")
	}
}

func TestFindParticipantByAddr(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")

	addr1, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	addr2, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12346")

	// Create participants
	server.mu.Lock()
	p1 := &Participant{
		ID:       "p1",
		RoomID:   "room1",
		Username: "User1",
		Addr:     addr1,
		joined:   time.Now(),
		lastSeen: time.Now(),
	}
	server.participants["p1"] = p1
	server.mu.Unlock()

	// Find by address
	found := server.findParticipantByAddr(addr1)
	if found == nil {
		t.Fatal("Participant not found")
	}

	if found.ID != "p1" {
		t.Errorf("Found participant ID = %s, expected p1", found.ID)
	}

	// Try to find with non-existent address
	found = server.findParticipantByAddr(addr2)
	if found != nil {
		t.Error("Should not find participant with different address")
	}

	// Try with nil address
	found = server.findParticipantByAddr(nil)
	if found != nil {
		t.Error("Should not find participant with nil address")
	}
}

func TestGetParticipantsInRoom(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")

	// Create room
	server.CreateRoom("room1", "Test Room", 10)

	// Add participants
	server.mu.Lock()
	room := server.rooms["room1"]
	p1 := &Participant{ID: "p1", RoomID: "room1", Username: "User1"}
	p2 := &Participant{ID: "p2", RoomID: "room1", Username: "User2"}
	server.participants["p1"] = p1
	server.participants["p2"] = p2
	room.participants["p1"] = p1
	room.participants["p2"] = p2
	server.mu.Unlock()

	// Get participants
	participants, err := server.GetParticipantsInRoom("room1")
	if err != nil {
		t.Fatalf("Failed to get participants: %v", err)
	}

	if len(participants) != 2 {
		t.Errorf("Participant count = %d, expected 2", len(participants))
	}

	// Try non-existent room
	_, err = server.GetParticipantsInRoom("room999")
	if err == nil {
		t.Error("Getting participants from non-existent room should fail")
	}
}

func TestStatistics(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")

	// Initial statistics
	stats := server.GetStatistics()
	if stats.TotalParticipants != 0 {
		t.Errorf("Initial participants = %d, expected 0", stats.TotalParticipants)
	}

	if stats.TotalRooms != 0 {
		t.Errorf("Initial rooms = %d, expected 0", stats.TotalRooms)
	}

	// Create room and add participants
	server.CreateRoom("room1", "Test Room", 10)

	server.mu.Lock()
	room := server.rooms["room1"]
	p1 := &Participant{ID: "p1", RoomID: "room1"}
	p2 := &Participant{ID: "p2", RoomID: "room1"}
	server.participants["p1"] = p1
	server.participants["p2"] = p2
	room.participants["p1"] = p1
	room.participants["p2"] = p2
	server.mu.Unlock()

	stats = server.GetStatistics()
	if stats.TotalParticipants != 2 {
		t.Errorf("Participants = %d, expected 2", stats.TotalParticipants)
	}

	if stats.TotalRooms != 1 {
		t.Errorf("Rooms = %d, expected 1", stats.TotalRooms)
	}

	if stats.RoomsCreated != 1 {
		t.Errorf("Rooms created = %d, expected 1", stats.RoomsCreated)
	}
}

func TestPacketTypes(t *testing.T) {
	if PacketTypeAudio != 0x01 {
		t.Errorf("PacketTypeAudio = %d, expected 0x01", PacketTypeAudio)
	}

	if PacketTypeVideo != 0x02 {
		t.Errorf("PacketTypeVideo = %d, expected 0x02", PacketTypeVideo)
	}

	if PacketTypeControl != 0x03 {
		t.Errorf("PacketTypeControl = %d, expected 0x03", PacketTypeControl)
	}
}

func TestStreamTypes(t *testing.T) {
	if StreamTypeAudio != 0 {
		t.Errorf("StreamTypeAudio = %d, expected 0", StreamTypeAudio)
	}

	if StreamTypeVideo != 1 {
		t.Errorf("StreamTypeVideo = %d, expected 1", StreamTypeVideo)
	}
}

func TestParsePacket(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")

	// Create participant with address
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	server.mu.Lock()
	p1 := &Participant{
		ID:       "p1",
		RoomID:   "room1",
		Addr:     addr,
		lastSeen: time.Now().Add(-5 * time.Second),
	}
	server.participants["p1"] = p1
	server.mu.Unlock()

	// Parse packet
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	packet := server.parsePacket(data, addr)

	if packet == nil {
		t.Fatal("Packet is nil")
	}

	if packet.Type != PacketTypeAudio {
		t.Errorf("Packet type = %d, expected %d", packet.Type, PacketTypeAudio)
	}

	if packet.ParticipantID != "p1" {
		t.Errorf("Participant ID = %s, expected p1", packet.ParticipantID)
	}

	if packet.RoomID != "room1" {
		t.Errorf("Room ID = %s, expected room1", packet.RoomID)
	}

	// Verify lastSeen was updated
	if time.Since(p1.lastSeen) > 1*time.Second {
		t.Error("lastSeen was not updated")
	}
}

func TestHandlePacketTooShort(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")

	// Packet too short (less than 8 bytes)
	data := []byte{0x01, 0x02, 0x03}

	// Should not panic
	server.handlePacket(data, addr)

	// Verify no packets were counted
	stats := server.GetStatistics()
	if stats.PacketsReceived != 0 {
		t.Errorf("Packets received = %d, expected 0", stats.PacketsReceived)
	}
}

func TestCleanupStaleParticipants(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")

	// Create room
	server.CreateRoom("room1", "Test Room", 10)

	// Add participants with different lastSeen times
	server.mu.Lock()
	room := server.rooms["room1"]

	// Fresh participant
	p1 := &Participant{
		ID:       "p1",
		RoomID:   "room1",
		lastSeen: time.Now(),
	}
	server.participants["p1"] = p1
	room.participants["p1"] = p1

	// Stale participant (not seen for 1 minute)
	p2 := &Participant{
		ID:       "p2",
		RoomID:   "room1",
		lastSeen: time.Now().Add(-1 * time.Minute),
	}
	server.participants["p2"] = p2
	room.participants["p2"] = p2
	server.mu.Unlock()

	// Run cleanup
	server.cleanupStaleParticipants()

	// Verify stale participant was removed
	server.mu.RLock()
	_, p1Exists := server.participants["p1"]
	_, p2Exists := server.participants["p2"]
	server.mu.RUnlock()

	if !p1Exists {
		t.Error("Fresh participant should not be removed")
	}

	if p2Exists {
		t.Error("Stale participant should be removed")
	}
}

func TestCleanupEmptyRoom(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")

	// Create room with one stale participant
	server.CreateRoom("room1", "Test Room", 10)

	server.mu.Lock()
	room := server.rooms["room1"]
	p1 := &Participant{
		ID:       "p1",
		RoomID:   "room1",
		lastSeen: time.Now().Add(-1 * time.Minute),
	}
	server.participants["p1"] = p1
	room.participants["p1"] = p1
	server.mu.Unlock()

	// Run cleanup
	server.cleanupStaleParticipants()

	// Verify room was deleted
	server.mu.RLock()
	_, roomExists := server.rooms["room1"]
	server.mu.RUnlock()

	if roomExists {
		t.Error("Empty room should be deleted")
	}
}

func TestForwardAudioPacket(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")
	server.Start()
	defer server.Stop()

	// Create room with participants
	server.CreateRoom("room1", "Test Room", 10)

	addr1, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	addr2, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12346")

	server.mu.Lock()
	room := server.rooms["room1"]

	p1 := &Participant{
		ID:       "p1",
		RoomID:   "room1",
		Addr:     addr1,
		isMuted:  false,
		lastSeen: time.Now(),
	}
	p2 := &Participant{
		ID:       "p2",
		RoomID:   "room1",
		Addr:     addr2,
		isMuted:  false,
		lastSeen: time.Now(),
	}

	server.participants["p1"] = p1
	server.participants["p2"] = p2
	room.participants["p1"] = p1
	room.participants["p2"] = p2
	server.mu.Unlock()

	// Forward audio packet from p1
	packet := &Packet{
		Type:          PacketTypeAudio,
		ParticipantID: "p1",
		RoomID:        "room1",
		Data:          []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	initialStats := server.GetStatistics()
	server.forwardAudioPacket(packet)

	// Check that packets were forwarded
	stats := server.GetStatistics()
	if stats.PacketsForwarded <= initialStats.PacketsForwarded {
		t.Error("Packets should have been forwarded")
	}
}

func TestForwardAudioPacketMuted(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")
	server.Start()
	defer server.Stop()

	// Create room with muted participant
	server.CreateRoom("room1", "Test Room", 10)

	addr1, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	addr2, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12346")

	server.mu.Lock()
	room := server.rooms["room1"]

	p1 := &Participant{
		ID:       "p1",
		RoomID:   "room1",
		Addr:     addr1,
		isMuted:  false,
		lastSeen: time.Now(),
	}
	p2 := &Participant{
		ID:       "p2",
		RoomID:   "room1",
		Addr:     addr2,
		isMuted:  true, // Muted
		lastSeen: time.Now(),
	}

	server.participants["p1"] = p1
	server.participants["p2"] = p2
	room.participants["p1"] = p1
	room.participants["p2"] = p2
	server.mu.Unlock()

	// Forward audio packet
	packet := &Packet{
		Type:          PacketTypeAudio,
		ParticipantID: "p1",
		RoomID:        "room1",
		Data:          []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	server.forwardAudioPacket(packet)

	// Muted participant should not receive any packets
	if p2.bytesSent > 0 {
		t.Error("Muted participant should not receive packets")
	}
}

func TestForwardVideoPacket(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")
	server.Start()
	defer server.Stop()

	// Create room with participants
	server.CreateRoom("room1", "Test Room", 10)

	addr1, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	addr2, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12346")

	server.mu.Lock()
	room := server.rooms["room1"]

	p1 := &Participant{
		ID:       "p1",
		RoomID:   "room1",
		Addr:     addr1,
		isPaused: false,
		lastSeen: time.Now(),
	}
	p2 := &Participant{
		ID:       "p2",
		RoomID:   "room1",
		Addr:     addr2,
		isPaused: false,
		lastSeen: time.Now(),
	}

	server.participants["p1"] = p1
	server.participants["p2"] = p2
	room.participants["p1"] = p1
	room.participants["p2"] = p2
	server.mu.Unlock()

	// Forward video packet
	packet := &Packet{
		Type:          PacketTypeVideo,
		ParticipantID: "p1",
		RoomID:        "room1",
		Data:          []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	initialStats := server.GetStatistics()
	server.forwardVideoPacket(packet)

	stats := server.GetStatistics()
	if stats.PacketsForwarded <= initialStats.PacketsForwarded {
		t.Error("Packets should have been forwarded")
	}
}

func TestForwardVideoPacketPaused(t *testing.T) {
	server, _ := NewSFUServer("127.0.0.1:0")
	server.Start()
	defer server.Stop()

	// Create room with paused participant
	server.CreateRoom("room1", "Test Room", 10)

	addr1, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	addr2, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12346")

	server.mu.Lock()
	room := server.rooms["room1"]

	p1 := &Participant{
		ID:       "p1",
		RoomID:   "room1",
		Addr:     addr1,
		isPaused: false,
		lastSeen: time.Now(),
	}
	p2 := &Participant{
		ID:       "p2",
		RoomID:   "room1",
		Addr:     addr2,
		isPaused: true, // Video paused
		lastSeen: time.Now(),
	}

	server.participants["p1"] = p1
	server.participants["p2"] = p2
	room.participants["p1"] = p1
	room.participants["p2"] = p2
	server.mu.Unlock()

	// Forward video packet
	packet := &Packet{
		Type:          PacketTypeVideo,
		ParticipantID: "p1",
		RoomID:        "room1",
		Data:          []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	server.forwardVideoPacket(packet)

	// Paused participant should not receive any packets
	if p2.bytesSent > 0 {
		t.Error("Paused participant should not receive video packets")
	}
}

// Benchmarks

func BenchmarkForwardAudioPacket(b *testing.B) {
	server, _ := NewSFUServer("127.0.0.1:0")
	server.Start()
	defer server.Stop()

	// Create room with 10 participants
	server.CreateRoom("room1", "Test Room", 100)

	server.mu.Lock()
	room := server.rooms["room1"]

	for i := 0; i < 10; i++ {
		addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
		p := &Participant{
			ID:       fmt.Sprintf("p%d", i),
			RoomID:   "room1",
			Addr:     addr,
			lastSeen: time.Now(),
		}
		server.participants[p.ID] = p
		room.participants[p.ID] = p
	}
	server.mu.Unlock()

	packet := &Packet{
		Type:          PacketTypeAudio,
		ParticipantID: "p0",
		RoomID:        "room1",
		Data:          make([]byte, 160), // Typical audio packet size
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.forwardAudioPacket(packet)
	}
}

func BenchmarkParsePacket(b *testing.B) {
	server, _ := NewSFUServer("127.0.0.1:0")

	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	server.mu.Lock()
	p := &Participant{
		ID:       "p1",
		RoomID:   "room1",
		Addr:     addr,
		lastSeen: time.Now(),
	}
	server.participants["p1"] = p
	server.mu.Unlock()

	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.parsePacket(data, addr)
	}
}
