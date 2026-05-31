//go:build experimental

// Package snake is a small, dependency-free, fully testable Snake engine. The
// engine is pure logic (no I/O); rendering and input live in the command. It
// exists to demonstrate a real-time, interactive game loop driving the terminal
// graphics pipeline (diff renderer), as opposed to the turn-based AI games.
package snake

// Dir is a movement direction.
type Dir int

const (
	Up Dir = iota
	Down
	Left
	Right
)

// Point is a board cell.
type Point struct{ X, Y int }

// State is the result of advancing one tick.
type State int

const (
	Playing State = iota
	Won           // the snake fills the board
	Lost          // ran into a wall or itself
)

// Game is a Snake game on a Width x Height grid. Construct with New.
type Game struct {
	Width, Height int
	Body          []Point // Body[0] is the head
	dir           Dir
	pendingDir    Dir
	Food          Point
	State         State
	Score         int

	rng func(n int) int // injectable for deterministic tests
}

// New creates a game with a centered 1-cell snake moving Right. rng(n) must
// return a value in [0,n); pass nil for a default (used only for food spawns).
func New(width, height int, rng func(n int) int) *Game {
	if width < 3 {
		width = 3
	}
	if height < 3 {
		height = 3
	}
	g := &Game{
		Width:      width,
		Height:     height,
		dir:        Right,
		pendingDir: Right, // must match dir so the first Tick doesn't auto-turn
		rng:        rng,
	}
	g.Body = []Point{{X: width / 2, Y: height / 2}}
	g.spawnFood()
	return g
}

// Dir returns the current (committed) direction.
func (g *Game) Dir() Dir { return g.dir }

// Head returns the snake's head cell.
func (g *Game) Head() Point { return g.Body[0] }

// Turn requests a direction change for the next tick. A 180-degree reversal is
// ignored (you cannot turn back into your own neck).
func (g *Game) Turn(d Dir) {
	if isOpposite(d, g.dir) {
		return
	}
	g.pendingDir = d
}

// Tick advances the game by one step and returns the new state.
func (g *Game) Tick() State {
	if g.State != Playing {
		return g.State
	}
	// Commit a queued turn (guard reversal again in case dir changed since Turn).
	if !isOpposite(g.pendingDir, g.dir) {
		g.dir = g.pendingDir
	}

	head := g.Body[0]
	next := move(head, g.dir)

	// Wall collision.
	if next.X < 0 || next.X >= g.Width || next.Y < 0 || next.Y >= g.Height {
		g.State = Lost
		return g.State
	}

	eating := next == g.Food

	// Self collision: check against the body. The tail cell will move away
	// unless we're eating (growing), so exclude it when not eating.
	limit := len(g.Body)
	if !eating {
		limit-- // tail vacates this tick
	}
	for i := 0; i < limit; i++ {
		if g.Body[i] == next {
			g.State = Lost
			return g.State
		}
	}

	// Advance: prepend new head.
	g.Body = append([]Point{next}, g.Body...)
	if eating {
		g.Score++
		if len(g.Body) == g.Width*g.Height {
			g.State = Won
			return g.State
		}
		g.spawnFood()
	} else {
		g.Body = g.Body[:len(g.Body)-1] // drop tail
	}
	return g.State
}

// Occupied reports whether p is part of the snake body.
func (g *Game) Occupied(p Point) bool {
	for _, b := range g.Body {
		if b == p {
			return true
		}
	}
	return false
}

// spawnFood places food on a random free cell. If the board is full it leaves
// Food unchanged (Won is detected separately).
func (g *Game) spawnFood() {
	free := g.freeCells()
	if len(free) == 0 {
		return
	}
	idx := 0
	if g.rng != nil {
		idx = g.rng(len(free))
		if idx < 0 || idx >= len(free) {
			idx = 0
		}
	}
	g.Food = free[idx]
}

func (g *Game) freeCells() []Point {
	occ := make(map[Point]bool, len(g.Body))
	for _, b := range g.Body {
		occ[b] = true
	}
	var free []Point
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			p := Point{X: x, Y: y}
			if !occ[p] {
				free = append(free, p)
			}
		}
	}
	return free
}

func move(p Point, d Dir) Point {
	switch d {
	case Up:
		return Point{p.X, p.Y - 1}
	case Down:
		return Point{p.X, p.Y + 1}
	case Left:
		return Point{p.X - 1, p.Y}
	default: // Right
		return Point{p.X + 1, p.Y}
	}
}

func isOpposite(a, b Dir) bool {
	return (a == Up && b == Down) || (a == Down && b == Up) ||
		(a == Left && b == Right) || (a == Right && b == Left)
}
