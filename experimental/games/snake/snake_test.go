//go:build experimental

package snake

import "testing"

func TestNewCentersSnakeMovingRight(t *testing.T) {
	g := New(10, 8, nil)
	if g.Dir() != Right {
		t.Errorf("initial dir = %v, want Right", g.Dir())
	}
	if got := g.Head(); got != (Point{5, 4}) {
		t.Errorf("head = %v, want {5,4}", got)
	}
	if len(g.Body) != 1 {
		t.Errorf("body len = %d, want 1", len(g.Body))
	}
}

func TestFirstTickMovesRightNotUp(t *testing.T) {
	// Regression: pendingDir must start equal to dir, else the snake auto-turns.
	g := New(10, 8, nil)
	g.Tick()
	if got := g.Head(); got != (Point{6, 4}) {
		t.Errorf("after one tick head = %v, want {6,4} (moved Right)", got)
	}
}

func TestWallCollisionLoses(t *testing.T) {
	g := New(3, 3, nil) // head at {1,1}, moving Right
	g.Tick()            // -> {2,1}
	if g.State != Playing {
		t.Fatalf("state = %v, want Playing", g.State)
	}
	if s := g.Tick(); s != Lost { // -> x=3 out of bounds
		t.Fatalf("state = %v, want Lost (hit wall)", s)
	}
}

func TestReversalIsIgnored(t *testing.T) {
	g := New(10, 8, nil) // moving Right
	g.Turn(Left)         // 180° reversal -> must be ignored
	g.Tick()
	if g.Dir() != Right {
		t.Errorf("dir = %v, want Right (reversal ignored)", g.Dir())
	}
}

func TestTurnTakesEffectNextTick(t *testing.T) {
	g := New(10, 8, nil)
	g.Turn(Down)
	g.Tick()
	if g.Dir() != Down {
		t.Fatalf("dir = %v, want Down", g.Dir())
	}
	if got := g.Head(); got != (Point{5, 5}) {
		t.Errorf("head = %v, want {5,5} (moved Down)", got)
	}
}

// place returns an rng that puts food on a specific cell by locating it in the
// current free-cell ordering.
func place(g *Game, target Point) func(n int) int {
	free := g.freeCells()
	idx := 0
	for i, p := range free {
		if p == target {
			idx = i
			break
		}
	}
	return func(n int) int { return idx }
}

func TestEatingGrowsAndScores(t *testing.T) {
	g := New(10, 8, nil) // head {5,4}, moving Right
	// Put food directly ahead at {6,4}.
	g.rng = place(g, Point{6, 4})
	g.spawnFood()
	if g.Food != (Point{6, 4}) {
		t.Fatalf("food = %v, want {6,4}", g.Food)
	}

	g.Tick() // eat
	if g.Score != 1 {
		t.Errorf("score = %d, want 1", g.Score)
	}
	if len(g.Body) != 2 {
		t.Errorf("body len = %d, want 2 (grew)", len(g.Body))
	}
	if g.Head() != (Point{6, 4}) {
		t.Errorf("head = %v, want {6,4}", g.Head())
	}
}

func TestNonEatingKeepsLength(t *testing.T) {
	g := New(10, 8, nil)
	g.rng = place(g, Point{0, 0}) // food far away
	g.spawnFood()
	start := len(g.Body)
	g.Tick()
	if len(g.Body) != start {
		t.Errorf("body len = %d, want %d (no growth)", len(g.Body), start)
	}
}

func TestSelfCollisionLoses(t *testing.T) {
	// Build a snake long enough to bite itself with a tight loop.
	g := New(10, 8, nil)
	// Grow to length 5 by feeding straight to the right.
	for i := 0; i < 4; i++ {
		g.rng = place(g, move(g.Head(), Right))
		g.spawnFood()
		if g.Tick() != Playing {
			t.Fatalf("unexpected state while growing at step %d: %v", i, g.State)
		}
	}
	if len(g.Body) != 5 {
		t.Fatalf("body len = %d, want 5", len(g.Body))
	}
	// Now loop: Down, Left, Up -> head walks back into the body.
	g.Turn(Down)
	g.Tick()
	g.Turn(Left)
	g.Tick()
	g.Turn(Up)
	s := g.Tick()
	if s != Lost {
		t.Errorf("state = %v, want Lost (bit itself)", s)
	}
}

func TestWinConditionTriggers(t *testing.T) {
	// When the snake eats the final free cell, the state becomes Won. Lay an
	// 8-cell snake on a 3x3 board with head at {1,2}, the single free cell at
	// {0,2} directly to the head's left, and food there.
	g := New(3, 3, nil)
	g.Body = []Point{
		{1, 2}, {1, 1}, {0, 1}, {0, 0}, {1, 0}, {2, 0}, {2, 1}, {2, 2},
	}
	g.dir = Left
	g.pendingDir = Left
	g.Food = Point{0, 2}
	if s := g.Tick(); s != Won {
		t.Fatalf("state = %v, want Won (ate last free cell). head=%v len=%d", s, g.Head(), len(g.Body))
	}
}

func TestOccupied(t *testing.T) {
	g := New(5, 5, nil)
	if !g.Occupied(g.Head()) {
		t.Error("head should be occupied")
	}
	if g.Occupied(Point{-1, -1}) {
		t.Error("off-board cell should not be occupied")
	}
}
