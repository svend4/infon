package sfu

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// The forward loops run on the receive goroutine while cleanupLoop deletes
// participants from the same room map on its own goroutine. The forward path
// must not iterate room.participants without holding s.mu.
func TestForwardConcurrentWithCleanup(t *testing.T) {
	s, err := NewSFUServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewSFUServer: %v", err)
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	defer conn.Close()
	sink, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("ListenUDP sink: %v", err)
	}
	defer sink.Close()
	s.conn = conn
	sinkAddr := sink.LocalAddr().(*net.UDPAddr)

	const roomID = "room0001"
	if err := s.CreateRoom(roomID, "r", 1000); err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}

	// addStale (re)adds participants that cleanupStaleParticipants will remove,
	// writing the maps under s.mu exactly as the server's own code does.
	addStale := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		room := s.rooms[roomID]
		if room == nil {
			return
		}
		for i := 0; i < 8; i++ {
			id := fmt.Sprintf("stale%03d", i)
			p := &Participant{ID: id, RoomID: roomID, Addr: sinkAddr, lastSeen: time.Now().Add(-time.Hour)}
			s.participants[id] = p
			room.participants[id] = p
		}
	}
	// The sender stays fresh so the room is never deleted.
	s.mu.Lock()
	sender := &Participant{ID: "sender01", RoomID: roomID, Addr: sinkAddr, lastSeen: time.Now().Add(time.Hour)}
	s.participants[sender.ID] = sender
	s.rooms[roomID].participants[sender.ID] = sender
	s.mu.Unlock()
	addStale()

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			addStale()
			s.cleanupStaleParticipants()
		}
	}()

	pkt := &Packet{ParticipantID: sender.ID, RoomID: roomID, Data: []byte{byte(PacketTypeAudio), 0, 0, 0, 0, 0, 0, 0}}
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		pkt.Type = PacketTypeAudio
		s.forwardAudioPacket(pkt)
		pkt.Type = PacketTypeVideo
		s.forwardVideoPacket(pkt)
	}
	close(stop)
	wg.Wait()

	st := s.GetStatistics()
	if st.PacketsForwarded == 0 {
		t.Errorf("PacketsForwarded = 0, want > 0")
	}
}
