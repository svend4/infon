package group

import (
	"net"
	"strings"
	"sync"
	"testing"
)

// cellLabel returns the label drawn on the top row of the cell that starts at x.
func cellLabel(vg *VideoGrid, x, width int) string {
	out := vg.Render()
	var sb strings.Builder
	for dx := 0; dx < width && x+dx < out.Width; dx++ {
		sb.WriteRune(out.Blocks[0][x+dx].Glyph)
	}
	return strings.TrimSpace(sb.String())
}

// Each peer must keep its grid cell from one frame to the next; ranging over the
// frames map put the peers in a random order on every Render, so tiles swapped
// places at the display frame rate.
func TestVideoGridRenderKeepsPeerOrder(t *testing.T) {
	vg := NewVideoGrid(80, 30)
	vg.SetFrame("bob", nil)
	vg.SetFrame("alice", nil)

	for i := 0; i < 200; i++ {
		if left := cellLabel(vg, 0, 40); left != "alice" {
			t.Fatalf("render %d: left cell is %q, want %q (cells must be in a stable, sorted order)", i, left, "alice")
		}
		if right := cellLabel(vg, 40, 40); !strings.Contains(right, "bob") {
			t.Fatalf("render %d: right cell is %q, want %q", i, right, "bob")
		}
	}
}

// RenderStats runs on the display goroutine while the receive goroutine updates
// the same peers; it must read their counters and LastSeen under the peer lock.
// SetLayout likewise must not race with Render. Run with -race.
func TestVideoGridRenderStatsConcurrentWithPeerUpdates(t *testing.T) {
	vg := NewVideoGrid(80, 30)
	peer := NewPeerConnection("alice", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5000})
	peers := []*PeerConnection{peer}

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
			peer.UpdateLastSeen()
			peer.SetVideoFrame(nil)
			peer.SetAudioBuffer(nil)
			vg.SetLayout(Layout2x1)
		}
	}()

	for i := 0; i < 200; i++ {
		if s := vg.RenderStats(peers); !strings.Contains(s, "alice") {
			t.Fatalf("RenderStats output lacks the peer: %q", s)
		}
		_ = vg.Render()
	}
	close(stop)
	wg.Wait()

	frames, audio := peer.Counts()
	if frames == 0 || frames != audio {
		t.Fatalf("Counts() = %d frames, %d audio; want equal and non-zero", frames, audio)
	}
	if peer.LastSeenTime().IsZero() {
		t.Fatal("LastSeenTime() is zero")
	}
}
