package arena_test

import (
	"testing"

	"github.com/svend4/infon/pkg/arena"
)

func small() *arena.Arena {
	a := &arena.Arena{W: 8, H: 3, Terrain: make([]uint8, 8*3)}
	a.Units = []arena.Unit{
		{Kind: 0, X: 0, Y: 1, HP: arena.Bestiary[0].HP, Faction: 0, Alive: true},
		{Kind: 0, X: 7, Y: 1, HP: arena.Bestiary[0].HP, Faction: 1, Alive: true},
	}
	return a
}

func TestUnitsCloseAndFight(t *testing.T) {
	a := small()
	hp0 := a.TotalHP()
	dist := func() int {
		return int(a.Units[1].X) - int(a.Units[0].X)
	}
	d0 := dist()
	for i := 0; i < 4; i++ {
		a.Step(arena.RefCommander{}, arena.RefCommander{})
	}
	if dist() >= d0 {
		t.Errorf("units should approach: %d -> %d", d0, dist())
	}
	for i := 0; i < 20; i++ {
		a.Step(arena.RefCommander{}, arena.RefCommander{})
	}
	if a.TotalHP() >= hp0 {
		t.Errorf("combat should reduce total HP: %d -> %d", hp0, a.TotalHP())
	}
	if a.AliveCount(0)+a.AliveCount(1) == 2 {
		t.Error("expected at least one casualty after 24 ticks")
	}
}

func TestSnapshotWidthAndRender(t *testing.T) {
	a := small()
	if len(a.Snapshot()) != a.FrameWidth() {
		t.Fatalf("snapshot %d != width %d", len(a.Snapshot()), a.FrameWidth())
	}
	if len(arena.Faces(a)) == 0 {
		t.Fatal("no faces")
	}
}
