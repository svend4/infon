package sfu

import (
	"net"
	"testing"
)

// Joins are unauthenticated, so a join naming an existing participant ID from a
// different address must not re-bind that participant: the sender would receive
// the participant's media and could send as them.
func TestJoinFromAnotherAddressDoesNotHijack(t *testing.T) {
	s, err := NewSFUServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewSFUServer: %v", err)
	}
	s.conn = listenLoopback(t)
	owner := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 40001}
	other := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 40002}

	s.handlePacket(joinPacket("room0001", "partAAAA", "alice"), owner)
	s.handlePacket(joinPacket("room0002", "partAAAA", "mallory"), other)

	p := s.participants["partAAAA"]
	if p == nil || !sameUDPAddr(p.Addr, owner) || p.RoomID != "room0001" {
		t.Fatalf("participant after a join from another address = %+v, want bound to %v in room0001", p, owner)
	}
	if got := s.findParticipantByAddr(other); got != nil {
		t.Errorf("the other address is now attributed to %s", got.ID)
	}

	// The owner itself can still move rooms.
	s.handlePacket(joinPacket("room0002", "partAAAA", "alice"), owner)
	if p.RoomID != "room0002" {
		t.Errorf("owner's own re-join to room0002 left it in %s", p.RoomID)
	}
}

// One source address binds at most one participant: a client that comes back
// under a new ID replaces its old participant, so an address cannot fill rooms
// (or the server's participant cap) with IDs it never leaves.
func TestOneParticipantPerAddress(t *testing.T) {
	s, err := NewSFUServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewSFUServer: %v", err)
	}
	s.conn = listenLoopback(t)
	addr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 40003}

	for _, id := range []string{"part0001", "part0002", "part0003"} {
		s.handlePacket(joinPacket("room0001", id, "x"), addr)
	}
	if n := len(s.participants); n != 1 {
		t.Fatalf("one address holds %d participants, want 1", n)
	}
	if p := s.findParticipantByAddr(addr); p == nil || p.ID != "part0003" {
		t.Fatalf("address is bound to %+v, want the latest join part0003", p)
	}
	if room := s.rooms["room0001"]; room == nil || len(room.participants) != 1 {
		t.Fatalf("room0001 = %+v, want just part0003", room)
	}
}
