package main

import (
	"net"
	"testing"

	"github.com/svend4/infon/pkg/raydir"
)

// The hub answers a region ack only for a joined peer, and with at most
// maxAckRegions of the missing regions (the earliest first), so a stray or
// spoofed ack can't make it send its whole world.
func TestAckReplyKnownPeerCapped(t *testing.T) {
	var regions []raydir.Region
	for i := 0; i < 100; i++ {
		regions = append(regions, raydir.Region{Index: i})
	}
	peer := &net.UDPAddr{IP: net.IPv4(10, 0, 0, 2), Port: 5001}
	stranger := &net.UDPAddr{IP: net.IPv4(203, 0, 113, 9), Port: 5001}
	peers := map[string]*net.UDPAddr{peer.String(): peer}

	if got := ackReply(peers, stranger, nil, regions); len(got) != 0 {
		t.Errorf("an ack from an unknown address got %d regions, want 0", len(got))
	}
	if got := ackReply(peers, nil, nil, regions); len(got) != 0 {
		t.Errorf("an ack with no source got %d regions, want 0", len(got))
	}
	got := ackReply(peers, peer, []int{0, 2}, regions)
	if len(got) != maxAckRegions || maxAckRegions <= 0 || maxAckRegions >= len(regions) {
		t.Fatalf("reply to a peer missing 98 regions = %d, want the cap %d", len(got), maxAckRegions)
	}
	want := []int{1, 3, 4, 5, 6, 7, 8, 9}
	for i, r := range got {
		if i < len(want) && r.Index != want[i] {
			t.Errorf("reply[%d] = region %d, want %d (earliest missing first)", i, r.Index, want[i])
		}
	}
	have := make([]int, len(regions))
	for i := range have {
		have[i] = i
	}
	if got := ackReply(peers, peer, have, regions); len(got) != 0 {
		t.Errorf("a peer with every region got %d, want 0", len(got))
	}
}
