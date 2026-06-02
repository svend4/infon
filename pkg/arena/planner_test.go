package arena

import "testing"

func factionHP(a *Arena, f uint8) int {
	t := 0
	for _, u := range a.Units {
		if u.Alive && u.Faction == f {
			t += int(u.HP)
		}
	}
	return t
}

// The planner's chosen joint move is, by its own rollout, never worse than
// standing still — the optimizer only ever improves on doing nothing.
func TestPlannerNeverWorseThanStanding(t *testing.T) {
	a := &Arena{W: 9, H: 5, Terrain: make([]uint8, 45)}
	for y := 0; y < 5; y++ {
		a.Terrain[y*9+4] = 2
	}
	for n, k := range []uint8{4, 3, 5} {
		y := uint8(1 + n)
		a.Units = append(a.Units,
			Unit{Kind: k, X: 1, Y: y, HP: Bestiary[k].HP, Faction: 0, Alive: true},
			Unit{Kind: k, X: 7, Y: y, HP: Bestiary[k].HP, Faction: 1, Alive: true},
		)
	}
	plan := Planner{}.Plan(a, 0)
	if len(plan) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(plan))
	}
	var stay []Action
	for i := range a.Units {
		if a.Units[i].Alive && a.Units[i].Faction == 0 {
			stay = append(stay, Action{Unit: i, DX: 0, DY: 0, Attack: -1})
		}
	}
	if a.rolloutScore(0, plan)+1e-9 < a.rolloutScore(0, stay) {
		t.Fatalf("planner move scored %.2f, worse than standing still %.2f",
			a.rolloutScore(0, plan), a.rolloutScore(0, stay))
	}
}

// The lookahead planner beats the reactive reference. To be fair on a symmetric
// board we play BOTH assignments (planner as faction 0, then as faction 1) and
// compare the planner side's aggregate survivors+HP against the reference side's
// — controlling for any positional bias.
func TestPlannerBeatsReference(t *testing.T) {
	board := func() *Arena {
		const W, H = 9, 5
		a := &Arena{W: W, H: H, Terrain: make([]uint8, W*H)}
		for y := 0; y < H; y++ {
			a.Terrain[y*W+4] = 3 // a central ridge to hold and shoot from
		}
		kinds := []uint8{4, 3, 5} // owl(R3), swan(R2), crab(R2) — range rewards holding ground
		for n, k := range kinds {
			y := 1 + n
			a.Units = append(a.Units,
				Unit{Kind: k, X: 1, Y: uint8(y), HP: Bestiary[k].HP, Faction: 0, Alive: true},
				Unit{Kind: k, X: 7, Y: uint8(y), HP: Bestiary[k].HP, Faction: 1, Alive: true},
			)
		}
		return a
	}
	play := func(c0, c1 Commander) *Arena {
		a := board()
		for tick := 0; tick < 80; tick++ {
			if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
				break
			}
			a.Step(c0, c1)
		}
		return a
	}

	m1 := play(Planner{}, RefCommander{})  // planner = faction 0
	m2 := play(RefCommander{}, Planner{})  // planner = faction 1
	planner := m1.AliveCount(0) + m2.AliveCount(1)
	ref := m1.AliveCount(1) + m2.AliveCount(0)
	plannerHP := factionHP(m1, 0) + factionHP(m2, 1)
	refHP := factionHP(m1, 1) + factionHP(m2, 0)
	if planner < ref || (planner == ref && plannerHP <= refHP) {
		t.Fatalf("planner did not beat reference: survivors %d vs %d, hp %d vs %d",
			planner, ref, plannerHP, refHP)
	}
}
