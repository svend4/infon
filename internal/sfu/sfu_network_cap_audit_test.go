package sfu

import (
	"fmt"
	"testing"
)

// Join packets are unauthenticated and pick their own IDs, so the number of
// rooms and participants they can create must be bounded server-wide.
func TestJoinFloodIsBounded(t *testing.T) {
	s, _ := NewSFUServer("127.0.0.1:0")
	if s.maxRooms <= 0 || s.maxParticipants <= 0 {
		t.Fatalf("default caps maxRooms=%d maxParticipants=%d, want > 0", s.maxRooms, s.maxParticipants)
	}
	s.maxRooms = 3
	s.maxParticipants = 5

	for i := 0; i < 50; i++ {
		s.handleControlPacket(&Packet{Type: PacketTypeControl,
			Data: joinPacket(fmt.Sprintf("room%04d", i), fmt.Sprintf("part%04d", i), "")})
	}
	if st := s.GetStatistics(); st.TotalRooms > 3 || st.TotalParticipants > 5 {
		t.Errorf("after a join flood rooms=%d participants=%d, want <= 3, <= 5", st.TotalRooms, st.TotalParticipants)
	}

	// Many participants in one existing room are bounded by the global cap too.
	s2, _ := NewSFUServer("127.0.0.1:0")
	s2.maxParticipants = 5
	for i := 0; i < 50; i++ {
		s2.handleControlPacket(&Packet{Type: PacketTypeControl,
			Data: joinPacket("room0001", fmt.Sprintf("part%04d", i), "")})
	}
	if st := s2.GetStatistics(); st.TotalParticipants != 5 || st.TotalRooms != 1 {
		t.Errorf("one-room flood: rooms=%d participants=%d, want 1, 5", st.TotalRooms, st.TotalParticipants)
	}

	// A known participant re-joining is not a new participant and is accepted.
	s2.handleControlPacket(&Packet{Type: PacketTypeControl, Data: joinPacket("room0001", "part0000", "")})
	if st := s2.GetStatistics(); st.TotalParticipants != 5 {
		t.Errorf("re-join at the cap changed participants to %d, want 5", st.TotalParticipants)
	}
}
