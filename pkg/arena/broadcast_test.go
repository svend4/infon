package arena_test

import (
	"bytes"
	"testing"

	"github.com/svend4/infon/pkg/arena"
)

func match(n int) [][]byte {
	a := &arena.Arena{W: 8, H: 3, Terrain: make([]uint8, 24)}
	a.Units = []arena.Unit{
		{Kind: 0, X: 0, Y: 1, HP: 6, Faction: 0, Alive: true},
		{Kind: 0, X: 7, Y: 1, HP: 6, Faction: 1, Alive: true},
	}
	var fr [][]byte
	for i := 0; i < n; i++ {
		fr = append(fr, a.Snapshot())
		a.Step(arena.RefCommander{}, arena.RefCommander{})
	}
	return fr
}

func eq(a, b []byte) bool { return bytes.Equal(a, b) }

func TestBroadcastLossless(t *testing.T) {
	fr := match(12)
	wires := arena.EncodeMatch(fr, 4, 10)
	got, _ := arena.Spectate(wires, 10, len(fr[0]))
	for i := range fr {
		if !eq(fr[i], got[i]) {
			t.Fatalf("tick %d differs without loss", i)
		}
	}
}

func TestBroadcastSurvivesLoss(t *testing.T) {
	fr := match(12)
	wires := arena.EncodeMatch(fr, 4, 10)
	wires[5] = nil // drop a delta packet (tick 5, GOP keyframe at 4)
	got, live := arena.Spectate(wires, 10, len(fr[0]))
	if live[5] {
		t.Error("tick 5 should be frozen (packet dropped)")
	}
	if !eq(fr[6], got[6]) {
		t.Error("tick 6 should be exact again (delta vs keyframe)")
	}
}

func TestSpectatorStreaming(t *testing.T) {
	fr := match(12)
	wires := arena.EncodeMatch(fr, 4, 10)
	sp := arena.NewSpectator(10, len(fr[0]))
	for i, w := range wires {
		if i == 5 {
			sp.Miss() // lose tick 5
			continue
		}
		got, live := sp.Recv(w)
		if !live || !eq(got, fr[i]) {
			t.Errorf("tick %d streaming mismatch", i)
		}
	}
}
