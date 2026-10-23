package sfu

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func joinPacket(roomID, participantID, username string) []byte {
	data := []byte{byte(PacketTypeControl), 0x01}
	data = append(data, roomID...)
	data = append(data, participantID...)
	return append(data, username...)
}

func listenLoopback(t *testing.T) *net.UDPConn {
	t.Helper()
	c, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// A client that joins over UDP must be reachable at the datagram's source
// address: its later media is attributed to it and forwarded to the room.
func TestJoinOverUDPRecordsSourceAddress(t *testing.T) {
	s, err := NewSFUServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewSFUServer: %v", err)
	}
	s.conn = listenLoopback(t)
	a := listenLoopback(t)
	b := listenLoopback(t)
	addrA := a.LocalAddr().(*net.UDPAddr)
	addrB := b.LocalAddr().(*net.UDPAddr)

	s.handlePacket(joinPacket("room0001", "partAAAA", "alice"), addrA)
	s.handlePacket(joinPacket("room0001", "partBBBB", "bob"), addrB)

	if p := s.findParticipantByAddr(addrA); p == nil || p.ID != "partAAAA" {
		t.Fatalf("findParticipantByAddr(A) = %+v, want participant partAAAA", p)
	}

	media := []byte{byte(PacketTypeAudio), 0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8}
	s.handlePacket(media, addrA)

	buf := make([]byte, 1500)
	_ = b.SetReadDeadline(time.Now().Add(time.Second))
	for {
		n, _, err := b.ReadFromUDP(buf)
		if err != nil {
			t.Fatalf("B did not receive A's audio: %v", err)
		}
		if bytes.Equal(buf[:n], media) {
			break
		}
	}
	if st := s.GetStatistics(); st.PacketsForwarded != 1 {
		t.Errorf("PacketsForwarded = %d, want 1", st.PacketsForwarded)
	}
}

// Joining a second room moves the participant: it must be a member of exactly
// one room and its RoomID must name that room.
func TestRejoinDifferentRoomMovesParticipant(t *testing.T) {
	s, _ := NewSFUServer("127.0.0.1:0")
	s.handleControlPacket(&Packet{Type: PacketTypeControl, Data: joinPacket("room0001", "part0001", "")})
	s.handleControlPacket(&Packet{Type: PacketTypeControl, Data: joinPacket("room0002", "part0001", "")})

	if _, err := s.GetRoom("room0001"); err == nil {
		ps, _ := s.GetParticipantsInRoom("room0001")
		t.Errorf("old room still exists with %d participants after the move", len(ps))
	}
	ps, err := s.GetParticipantsInRoom("room0002")
	if err != nil || len(ps) != 1 || ps[0].ID != "part0001" {
		t.Fatalf("room0002 participants = %v, %v; want [part0001]", ps, err)
	}
	s.mu.RLock()
	roomID := s.participants["part0001"].RoomID
	s.mu.RUnlock()
	if roomID != "room0002" {
		t.Errorf("participant RoomID = %q, want room0002", roomID)
	}
	if st := s.GetStatistics(); st.TotalRooms != 1 || st.TotalParticipants != 1 {
		t.Errorf("rooms=%d participants=%d, want 1, 1", st.TotalRooms, st.TotalParticipants)
	}
}
