package raydir

import "testing"

func TestQuestSolvableAndStepsAreFlips(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		q := NewQuest(seed, 0.45)
		path, ok := q.Solve()
		if !ok {
			t.Fatalf("seed %d: generator must leave the goal reachable", seed)
		}
		if path[0] != q.Start || path[len(path)-1] != q.Goal {
			t.Fatalf("seed %d: path must run Start..Goal", seed)
		}
		for i := 1; i < len(path); i++ {
			if path[i-1].Hamming(path[i]) != 1 {
				t.Fatalf("seed %d: step %d is not a single line flip", seed, i)
			}
			if q.IsWall(path[i]) {
				t.Fatalf("seed %d: path crosses a wall at step %d", seed, i)
			}
		}
		// the maze can only lengthen the route beyond the straight Hamming line
		if len(path)-1 < q.Start.Hamming(q.Goal) {
			t.Fatalf("seed %d: route shorter than the Hamming distance", seed)
		}
	}
}

func TestQuestWallsForceDetours(t *testing.T) {
	// over many seeds at least one dense maze must force a detour longer than the
	// six-flip straight line — otherwise the walls do nothing.
	detours := 0
	for seed := int64(0); seed < 40; seed++ {
		q := NewQuest(seed, 0.6)
		if path, ok := q.Solve(); ok && len(path)-1 > q.Start.Hamming(q.Goal) {
			detours++
		}
	}
	if detours == 0 {
		t.Error("dense mazes never forced a detour — walls have no effect")
	}
}

func TestQuestReachableBounds(t *testing.T) {
	open := NewQuest(1, 0) // no walls: the whole cube is reachable
	if open.WallCount() != 0 {
		t.Fatalf("wallFrac 0 should place no walls, got %d", open.WallCount())
	}
	if r := open.Reachable(); r != 64 {
		t.Errorf("an open cube should reach all 64 rooms, got %d", r)
	}
	walled := NewQuest(2, 0.5)
	if r := walled.Reachable(); r < 2 || r > 64 {
		t.Errorf("reachable out of range: %d", r)
	}
}

func TestMove(t *testing.T) {
	from := HexagramFromNumber(0b000000)
	to := from.Flip(3) // line 3 -> yang ("sun")
	m := Move(from, to)
	if m.Line != 3 || !m.ToYang || m.Trait != "sun" {
		t.Errorf("Move = %+v, want line 3, yang, sun", m)
	}
	if Move(from, from.Antipode()).Line != -1 {
		t.Error("non-adjacent rooms should report no single changing line")
	}
}

func TestGrandTourIsHamiltonian(t *testing.T) {
	q := NewQuest(5, 0.3)
	tour := q.GrandTour()
	if len(tour) != 64 {
		t.Fatalf("grand tour should visit 64 rooms, got %d", len(tour))
	}
	seen := map[int]bool{}
	for i, h := range tour {
		if seen[h.Number()] {
			t.Fatalf("room %06b visited twice", h.Number())
		}
		seen[h.Number()] = true
		if i > 0 && tour[i-1].Hamming(h) != 1 {
			t.Fatalf("tour step %d is not a single line flip", i)
		}
	}
	if len(seen) != 64 {
		t.Errorf("tour covered %d distinct rooms, want 64", len(seen))
	}
	if tour[0] != q.Start {
		t.Errorf("tour should start at the quest start")
	}
}
