package arena_test

import (
	"testing"

	"github.com/svend4/infon/pkg/arena"
)

// A ranged unit (swan, range 2) hits a melee enemy (fox, range 1) from two
// tiles away while the melee enemy cannot strike back.
func TestRangedReachesFirst(t *testing.T) {
	a := &arena.Arena{W: 8, H: 1, Terrain: make([]uint8, 8)}
	a.Units = []arena.Unit{
		{Kind: 3, X: 0, Y: 0, HP: arena.Bestiary[3].HP, Faction: 0, Alive: true}, // swan, range 2
		{Kind: 0, X: 2, Y: 0, HP: arena.Bestiary[0].HP, Faction: 1, Alive: true}, // fox, range 1
	}
	rangedHP0 := a.Units[0].HP
	meleeHP0 := a.Units[1].HP
	a.Step(arena.RefCommander{}, arena.RefCommander{})
	if a.Units[1].Alive && a.Units[1].HP >= meleeHP0 {
		t.Errorf("ranged should hit at distance 2: melee HP %d -> %d", meleeHP0, a.Units[1].HP)
	}
	if a.Units[0].HP != rangedHP0 {
		t.Errorf("melee should not reach the ranged unit: HP %d -> %d", rangedHP0, a.Units[0].HP)
	}
}

// Fog of war: an enemy beyond Vision of any own unit is not in the Brief.
func TestFogHidesDistant(t *testing.T) {
	a := &arena.Arena{W: 8, H: 1, Terrain: make([]uint8, 8)}
	a.Units = []arena.Unit{
		{Kind: 0, X: 0, Y: 0, HP: 6, Faction: 0, Alive: true},
		{Kind: 0, X: 7, Y: 0, HP: 6, Faction: 1, Alive: true},
	}
	if got := len(a.Brief(0).Enemies); got != 0 {
		t.Errorf("distant enemy should be fogged, got %d enemies", got)
	}
	a.Units[1].X = 3 // within Vision (4)
	if got := len(a.Brief(0).Enemies); got != 1 {
		t.Errorf("near enemy should be visible, got %d enemies", got)
	}
}
