package chronicle

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/arena"
)

func board() *arena.Arena {
	const W, H = 9, 5
	a := &arena.Arena{W: W, H: H, Terrain: make([]uint8, W*H)}
	for y := 0; y < H; y++ {
		a.Terrain[y*W+4] = 3
	}
	for n, k := range []uint8{4, 3, 5} {
		y := uint8(1 + n)
		a.Units = append(a.Units,
			arena.Unit{Kind: k, X: 1, Y: y, HP: arena.Bestiary[k].HP, Faction: 0, Alive: true},
			arena.Unit{Kind: k, X: 7, Y: y, HP: arena.Bestiary[k].HP, Faction: 1, Alive: true},
		)
	}
	return a
}

func TestRecordDeterministic(t *testing.T) {
	l1 := Record(board(), arena.Planner{}, arena.RefCommander{}, 60)
	l2 := Record(board(), arena.Planner{}, arena.RefCommander{}, 60)
	if len(l1.Events) != len(l2.Events) {
		t.Fatalf("event count %d vs %d", len(l1.Events), len(l2.Events))
	}
	for i := range l1.Events {
		if l1.Events[i] != l2.Events[i] {
			t.Fatalf("event %d differs: %+v vs %+v", i, l1.Events[i], l2.Events[i])
		}
	}
}

func TestFallsMatchCasualties(t *testing.T) {
	a := board()
	start := a.AliveCount(0) + a.AliveCount(1)
	l := Record(a, arena.Planner{}, arena.RefCommander{}, 60)
	survivors := a.AliveCount(0) + a.AliveCount(1)
	falls := 0
	for _, e := range l.Events {
		if e.Kind == Fall {
			falls++
		}
	}
	if falls != start-survivors {
		t.Fatalf("recorded %d falls but %d casualties", falls, start-survivors)
	}
}

func TestNarrateAndBytes(t *testing.T) {
	l := Record(board(), arena.Planner{}, arena.RefCommander{}, 60)
	if s := l.Narrate(); !strings.Contains(s, "field") || !strings.Contains(s, "held") {
		t.Fatalf("narration missing key beats:\n%s", s)
	}
	if len(l.Bytes()) != 6*len(l.Events) {
		t.Fatalf("byte log %d != 6*%d events", len(l.Bytes()), len(l.Events))
	}
}
