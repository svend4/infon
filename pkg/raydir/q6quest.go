package raydir

import "math/rand"

// q6quest.go turns the 64 hexagrams into a roguelike dungeon. The dungeon IS the Q6
// hypercube: every hexagram is a room, and the only move is to change one line —
// flip a single trait of the world — stepping to an adjacent room. Some rooms have
// collapsed (walls), so reaching the goal is a real maze through six-dimensional
// space, not just the Hamming bee-line. A generator scatters walls but guarantees
// the goal stays reachable; the solver is a breadth-first search over the line-flip
// graph, returning the shortest route and the exact changing lines along it. The
// "grand tour" — visiting every room once — is the Gray-code Hamiltonian path.

// Quest is a maze on the Q6 hypercube: start, goal and a set of collapsed rooms.
type Quest struct {
	Start, Goal Hexagram
	walls       [64]bool
}

// NewQuest builds a solvable maze: the goal is the start's antipode (the far corner,
// six flips away) and walls are scattered over a wallFrac (0..1) of the other rooms,
// greedily, never blocking the last route — so a path always exists, but rarely the
// straight one.
func NewQuest(seed int64, wallFrac float64) *Quest {
	rng := rand.New(rand.NewSource(seed))
	q := &Quest{}
	q.Start = HexagramFromNumber(rng.Intn(64))
	q.Goal = q.Start.Antipode()
	if wallFrac < 0 {
		wallFrac = 0
	}
	if wallFrac > 0.9 {
		wallFrac = 0.9
	}
	target := int(wallFrac * 64)
	added := 0
	for _, n := range rng.Perm(64) {
		if added >= target {
			break
		}
		if n == q.Start.Number() || n == q.Goal.Number() {
			continue
		}
		q.walls[n] = true
		if _, ok := q.Solve(); !ok {
			q.walls[n] = false // this wall would seal the goal off — leave it open
		} else {
			added++
		}
	}
	return q
}

// IsWall reports whether a room has collapsed.
func (q *Quest) IsWall(h Hexagram) bool { return q.walls[h.Number()] }

// WallCount is the number of collapsed rooms.
func (q *Quest) WallCount() int {
	n := 0
	for _, w := range q.walls {
		if w {
			n++
		}
	}
	return n
}

// Solve is a breadth-first search for the shortest route from Start to Goal that
// steps one line at a time and never enters a wall. It returns the path (inclusive of
// both ends) and whether one exists.
func (q *Quest) Solve() ([]Hexagram, bool) {
	start, goal := q.Start.Number(), q.Goal.Number()
	if q.walls[start] || q.walls[goal] {
		return nil, false
	}
	prev := [64]int{}
	for i := range prev {
		prev[i] = -2 // unvisited
	}
	prev[start] = -1
	queue := []int{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == goal {
			break
		}
		h := HexagramFromNumber(cur)
		for line := 0; line < 6; line++ {
			nb := h.Flip(line).Number()
			if q.walls[nb] || prev[nb] != -2 {
				continue
			}
			prev[nb] = cur
			queue = append(queue, nb)
		}
	}
	if prev[goal] == -2 {
		return nil, false
	}
	var path []Hexagram
	for c := goal; c != -1; c = prev[c] {
		path = append(path, HexagramFromNumber(c))
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path, true
}

// Reachable counts the rooms reachable from Start without crossing a wall.
func (q *Quest) Reachable() int {
	start := q.Start.Number()
	if q.walls[start] {
		return 0
	}
	seen := [64]bool{}
	seen[start] = true
	queue := []int{start}
	n := 0
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		n++
		h := HexagramFromNumber(cur)
		for line := 0; line < 6; line++ {
			nb := h.Flip(line).Number()
			if q.walls[nb] || seen[nb] {
				continue
			}
			seen[nb] = true
			queue = append(queue, nb)
		}
	}
	return n
}

// ChangingLine is the line (0 = bottom .. 5 = top) that differs between two adjacent
// hexagrams, with the direction of the change — the I-Ching "moving line" of a step.
// Line is -1 if the two are not exactly one flip apart.
type ChangingLine struct {
	Line   int    // 0..5, or -1
	ToYang bool   // the line became yang (true) or yin (false)
	Trait  string // the world trait this line governs (fog/warmth/...)
}

// traitNames labels each line with the SceneVector axis it bends, tying a move to a
// visible change in the world.
var traitNames = [6]string{"fog", "warmth", "density", "sun", "scale", "glow"}

// Move describes the single changing line between from and to (adjacent rooms).
func Move(from, to Hexagram) ChangingLine {
	if from.Hamming(to) != 1 {
		return ChangingLine{Line: -1}
	}
	for i := 0; i < 6; i++ {
		if from.Lines[i] != to.Lines[i] {
			return ChangingLine{Line: i, ToYang: to.Lines[i], Trait: traitNames[i]}
		}
	}
	return ChangingLine{Line: -1}
}

// GrandTour is the Gray-code Hamiltonian path from Start: visit every one of the 64
// rooms exactly once, changing a single line at each step (ignores walls — it is the
// puzzle of touring the whole hypercube).
func (q *Quest) GrandTour() []Hexagram { return q.Start.GrayWalk() }
