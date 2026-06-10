package raydir

import (
	"math"

	"github.com/svend4/infon/pkg/raytrace"
)

// guide.go puts an AI *inside* the world: a companion that walks with you, leads you
// from one named place to the next, and comments on what it shows you. It is a
// participant — rendered as an avatar, talking in the chat — not an off-stage
// director. Its behaviour is local and deterministic (a tour of the landmarks),
// so it works offline and is testable; a live brain can enrich what it says.

// GuideColor is the guide avatar's distinct (warm) colour.
var GuideColor = raytrace.Vec3{X: 1, Y: 0.82, Z: 0.4}

// Guide is a brain-spirited character that tours the world's landmarks.
type Guide struct {
	Pos     raytrace.Vec3
	Yaw     float64
	Name    string
	speed   float64
	curMark int // landmark index being headed for; -1 = accompanying the player
	visited map[int]bool
	nextSay float64
	arrived bool // just reached the current landmark (for a remark)
}

// NewGuide makes a guide starting at `at`.
func NewGuide(name string, at raytrace.Vec3) *Guide {
	return &Guide{Pos: at, Name: name, speed: 1.8, curMark: -1, visited: map[int]bool{}}
}

// Pose is the guide's pose, for rendering an avatar.
func (g *Guide) Pose() Pose { return Pose{Pos: g.Pos, Yaw: g.Yaw} }

// markByIndex finds a landmark by its Index.
func markByIndex(marks []Landmark, idx int) (Landmark, bool) {
	for _, m := range marks {
		if m.Index == idx {
			return m, true
		}
	}
	return Landmark{}, false
}

// pickNext chooses the nearest not-yet-visited landmark, or -1 to accompany the
// player when the tour is done.
func (g *Guide) pickNext(marks []Landmark) int {
	best, bestD := -1, math.Inf(1)
	for _, m := range marks {
		if g.visited[m.Index] {
			continue
		}
		if d := m.At.Sub(g.Pos).LenSq(); d < bestD {
			best, bestD = m.Index, d
		}
	}
	return best
}

// Step advances the guide by dt seconds: it heads for the next landmark (or stays
// near the player when the tour is done), facing its motion. Returns true if it
// moved.
func (g *Guide) Step(dt float64, player raytrace.Vec3, marks []Landmark) bool {
	if g.visited == nil {
		g.visited = map[int]bool{}
	}
	if g.speed <= 0 {
		g.speed = 1.8
	}
	g.arrived = false

	// not currently touring? start (or resume) the tour if any place is unvisited.
	if g.curMark < 0 {
		g.curMark = g.pickNext(marks)
	}
	// where are we headed?
	var target raytrace.Vec3
	if g.curMark >= 0 {
		if m, ok := markByIndex(marks, g.curMark); ok {
			target = m.At
		} else {
			g.curMark = -1
		}
	}
	if g.curMark < 0 {
		target = player.Add(raytrace.Vec3{X: 2, Z: -2}) // tour done: accompany, a step aside
	}

	flat := target.Sub(g.Pos)
	flat.Y = 0
	dist := flat.Len()

	// reached the target?
	if dist < 1.2 {
		if g.curMark >= 0 {
			g.visited[g.curMark] = true
			g.arrived = true
			g.curMark = -1 // re-pick the next place on the following step
		}
		return g.arrived
	}
	dir := flat.Scale(1 / dist)
	step := g.speed * dt
	if step > dist {
		step = dist
	}
	g.Pos = g.Pos.Add(dir.Scale(step))
	g.Yaw = math.Atan2(dir.X, dir.Z)
	return true
}

// Remark returns something for the guide to say this tick (throttled), commenting
// on where it is leading you, or false if it has nothing to add.
func (g *Guide) Remark(t float64, marks []Landmark) (string, bool) {
	if t < g.nextSay {
		return "", false
	}
	g.nextSay = t + 6
	if g.curMark < 0 {
		return "A fine place to rest. Lead on whenever you like.", true
	}
	if m, ok := markByIndex(marks, g.curMark); ok {
		return "This way — let me show you the " + m.Name + ".", true
	}
	return "Come, there's more to see.", true
}
