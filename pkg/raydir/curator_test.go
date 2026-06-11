package raydir

import "testing"

func TestCuratorFindsGlobalBestWhenOpen(t *testing.T) {
	// On an open cube the curator must find the viewer's very favourite world: the
	// corner whose signs match the preference (the engagement peak).
	for seed := int64(0); seed < 12; seed++ {
		q := NewQuest(seed, 0) // no walls -> everything reachable
		pref := VectorFromHexagram(HexagramFromNumber(int(seed*7+3) % 64))
		pref[AxWarm] = 0.9 // nudge off the orthant centre so the peak is unambiguous
		vm := ViewerModel{Pref: pref, Sigma: 0.3}
		c := NewCurator(q, vm)
		best, _, visited := c.Curate(q.Start, 64)
		if best != pref.Hexagram() {
			t.Fatalf("seed %d: curator found %06b, viewer's favourite is %06b", seed, best.Number(), pref.Hexagram().Number())
		}
		if len(visited) == 0 {
			t.Fatal("curator should visit at least the start")
		}
	}
}

func TestCuratorNeverWorseThanStartAndAvoidsWalls(t *testing.T) {
	q := NewQuest(16, 0.6)
	vm := ViewerModel{Pref: SceneVector{0.2, 0.8, 0.7, 0.7, 0.4, 0.6}, Sigma: 0.3}
	c := NewCurator(q, vm)
	best, bestE, visited := c.Curate(q.Start, 64)
	if bestE < c.Engagement(q.Start) {
		t.Errorf("the curated world should be at least as loved as the start")
	}
	if q.IsWall(best) {
		t.Error("curator settled on a collapsed world")
	}
	seen := map[int]bool{}
	for _, h := range visited {
		if q.IsWall(h) {
			t.Errorf("curator visited a wall %06b", h.Number())
		}
		if seen[h.Number()] {
			t.Errorf("curator visited %06b twice", h.Number())
		}
		seen[h.Number()] = true
	}
}

func TestCuratorBudgetAndDeterminism(t *testing.T) {
	q := NewQuest(3, 0.3)
	vm := ViewerModel{Pref: SceneVector{0.1, 0.9, 0.8, 0.8, 0.5, 0.7}, Sigma: 0.3}
	c := NewCurator(q, vm)
	_, _, v1 := c.Curate(q.Start, 10)
	if len(v1) > 10 {
		t.Errorf("budget exceeded: %d visited", len(v1))
	}
	_, _, v2 := c.Curate(q.Start, 10)
	if len(v1) != len(v2) {
		t.Fatal("same search must replay identically")
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("search diverged at %d", i)
		}
	}
}
