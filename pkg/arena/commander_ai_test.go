package arena_test

import (
	"testing"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/brain"
)

func TestBrainCommanderFights(t *testing.T) {
	a := &arena.Arena{W: 8, H: 3, Terrain: make([]uint8, 8*3)}
	a.Units = []arena.Unit{
		{Kind: 0, X: 0, Y: 1, HP: arena.Bestiary[0].HP, Faction: 0, Alive: true},
		{Kind: 0, X: 7, Y: 1, HP: arena.Bestiary[0].HP, Faction: 1, Alive: true},
	}
	br := a.Brief(0)
	if len(br.Units) != 1 {
		t.Fatalf("brief: %d units", len(br.Units))
	}
	c := arena.BrainCommander{B: brain.Local{}}
	for i := 0; i < 24; i++ {
		a.Step(c, c)
	}
	if a.AliveCount(0)+a.AliveCount(1) == 2 {
		t.Error("brain-commanded units should fight to a casualty")
	}
}
