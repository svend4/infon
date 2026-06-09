// fly.go assembles the pieces into one experience: walking through a world the
// brain authors and extends on the fly. A FlyCam is a free first-person camera;
// a World is a shared floor plus clusters of props the brain composes on demand
// (game:rayscene). Each Grow ships only a compact scene description - meaning, not
// pixels - which is rendered locally, so the world unfolds as you explore it for a
// few bytes per chunk. This is the heart of "the AI directs a world, you walk it,
// in the terminal". Clean-room.
package raydir

import (
	"math"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// FlyCam is a free-fly first-person camera for walking through a world.
type FlyCam struct {
	Pos        raytrace.Vec3
	Yaw, Pitch float64
	FOV        float64
}

// groundForward is the heading projected onto the ground plane (for walking).
func (c FlyCam) groundForward() raytrace.Vec3 {
	return raytrace.Vec3{X: math.Sin(c.Yaw), Y: 0, Z: math.Cos(c.Yaw)}
}

// groundRight is the sideways direction on the ground plane (for strafing).
func (c FlyCam) groundRight() raytrace.Vec3 {
	return raytrace.Vec3{X: math.Cos(c.Yaw), Y: 0, Z: -math.Sin(c.Yaw)}
}

// Walk moves along the ground: fwd along the heading, strafe to the side.
func (c *FlyCam) Walk(fwd, strafe float64) {
	c.Pos = c.Pos.Add(c.groundForward().Scale(fwd)).Add(c.groundRight().Scale(strafe))
}

// Turn yaws and pitches the camera (pitch clamped to avoid flipping over).
func (c *FlyCam) Turn(dyaw, dpitch float64) {
	c.Yaw += dyaw
	c.Pitch = clampf(c.Pitch+dpitch, -1.2, 1.2)
}

// Camera converts the fly state into a raytrace.Camera.
func (c FlyCam) Camera() raytrace.Camera {
	fov := c.FOV
	if fov <= 0 {
		fov = math.Pi / 3
	}
	return raytrace.Camera{Pos: c.Pos, Yaw: c.Yaw, Pitch: c.Pitch, FOV: fov}
}

// World is a growing, brain-authored scene the camera can walk through: a shared
// floor plus clusters of props the brain authors on demand.
type World struct {
	SkyTop, SkyBottom raytrace.Vec3
	floor             raytrace.Object
	props             []raytrace.Object
	chunks            int
	seen              map[int]bool // applied region indices (idempotent growth)
}

// NewWorld returns a world with a checkerboard floor and a soft sky, no props yet.
func NewWorld() *World {
	return &World{
		SkyTop:    raytrace.Vec3{X: 0.45, Y: 0.6, Z: 0.85},
		SkyBottom: raytrace.Vec3{X: 0.85, Y: 0.88, Z: 0.95},
		floor:     raytrace.Plane{Y: 0, Size: 1.2, C1: raytrace.Vec3{X: 0.55, Y: 0.55, Z: 0.58}, C2: raytrace.Vec3{X: 0.3, Y: 0.3, Z: 0.33}},
	}
}

// Chunks reports how many times the world has been grown.
func (w *World) Chunks() int { return w.chunks }

// Props reports how many props the world currently holds.
func (w *World) Props() int { return len(w.props) }

// Grow asks the brain to author a scene chunk (game:rayscene) and places its props
// centred at `at`, so the world extends by meaning. Returns the props added.
func (w *World) Grow(b brain.Brain, prompt string, at raytrace.Vec3) (int, error) {
	_, spec, err := AuthorScene(b, prompt)
	if err != nil {
		return 0, err
	}
	return w.applyRegion(w.chunks, at, spec), nil
}

// Scene builds a renderable, BVH-accelerated scene from the floor and all props.
func (w *World) Scene() *raytrace.Scene { return w.SceneWith(nil) }

// SceneWith is Scene plus extra transient objects (e.g. a remote player's avatar)
// that change every frame and so can't live in the persistent prop list.
func (w *World) SceneWith(extra []raytrace.Object) *raytrace.Scene {
	s := &raytrace.Scene{SkyTop: w.SkyTop, SkyBottom: w.SkyBottom}
	s.Objects = append(s.Objects, w.floor)
	s.Objects = append(s.Objects, w.props...)
	s.Objects = append(s.Objects, extra...)
	s.BuildBVH()
	return s
}
