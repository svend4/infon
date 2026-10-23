package sfu

import "testing"

// A join message carries an 8-byte room ID and an 8-byte participant ID after
// the 2-byte header, so anything shorter than 18 bytes must be ignored rather
// than panicking or reading bytes past the end of the packet.
func TestHandleJoinRoomShortPacket(t *testing.T) {
	for n := 2; n < 18; n++ {
		s, _ := NewSFUServer("127.0.0.1:0")

		// Backing array is larger than the packet and holds stale bytes, like
		// the receive loop's reused 64 KiB buffer.
		buf := make([]byte, 64)
		for i := range buf {
			buf[i] = 'X'
		}
		copy(buf, []byte{byte(PacketTypeControl), 0x01, 'r', 'o', 'o', 'm', '0', '0', '0', '1', 'p', 'a', 'r', 't', '0', '0', '0', '1'})

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("len %d: handleJoinRoom panicked: %v", n, r)
				}
			}()
			s.handleControlPacket(&Packet{Type: PacketTypeControl, Data: buf[:n]})
			// Exact-capacity slice too.
			exact := append([]byte(nil), buf[:n]...)
			s.handleControlPacket(&Packet{Type: PacketTypeControl, Data: exact})
		}()

		if st := s.GetStatistics(); st.TotalParticipants != 0 || st.TotalRooms != 0 {
			t.Errorf("len %d: short join created %d participants, %d rooms; want 0, 0",
				n, st.TotalParticipants, st.TotalRooms)
		}
	}
}

func TestHandleJoinRoomExact18Bytes(t *testing.T) {
	s, _ := NewSFUServer("127.0.0.1:0")
	data := []byte{byte(PacketTypeControl), 0x01, 'r', 'o', 'o', 'm', '0', '0', '0', '1', 'p', 'a', 'r', 't', '0', '0', '0', '1'}
	s.handleControlPacket(&Packet{Type: PacketTypeControl, Data: data})

	ps, err := s.GetParticipantsInRoom("room0001")
	if err != nil {
		t.Fatalf("GetParticipantsInRoom: %v", err)
	}
	if len(ps) != 1 || ps[0].ID != "part0001" || ps[0].Username != "" {
		t.Errorf("participants = %+v, want one with ID part0001 and empty username", ps)
	}
}
