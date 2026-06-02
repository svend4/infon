package arena_test

import (
	"testing"

	"github.com/svend4/infon/pkg/arena"
)

// A ranged unit with an adjacent enemy kites: it steps AWAY to keep distance.
func TestRangedKites(t *testing.T) {
	a := &arena.Arena{W: 8, H: 1, Terrain: make([]uint8, 8)}
	a.Units = []arena.Unit{
		{Kind: 4, X: 3, Y: 0, HP: arena.Bestiary[4].HP, Faction: 0, Alive: true}, // owl, range 3
		{Kind: 0, X: 2, Y: 0, HP: arena.Bestiary[0].HP, Faction: 1, Alive: true}, // fox, adjacent
	}
	acts := arena.RefCommander{}.Plan(a, 0)
	if len(acts) != 1 || acts[0].DX <= 0 {
		t.Fatalf("ranged unit should kite away (dx>0), got %+v", acts)
	}
}

// A low-HP unit retreats from the nearest enemy.
func TestLowHPRetreats(t *testing.T) {
	a := &arena.Arena{W: 8, H: 1, Terrain: make([]uint8, 8)}
	a.Units = []arena.Unit{
		{Kind: 0, X: 3, Y: 0, HP: 2, Faction: 0, Alive: true}, // fox max 6, hp 2 -> retreat
		{Kind: 0, X: 2, Y: 0, HP: 6, Faction: 1, Alive: true},
	}
	acts := arena.RefCommander{}.Plan(a, 0)
	if len(acts) != 1 || acts[0].DX <= 0 {
		t.Fatalf("low-hp unit should retreat (dx>0), got %+v", acts)
	}
}
