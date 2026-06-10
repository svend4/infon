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
	seen              map[int]bool     // applied region indices (idempotent growth)
	lastSpec          *brain.SceneSpec // last authored region (continuity context)
	lastAt            raytrace.Vec3    // where it sat (to derive the heading)
	Time              float64          // time of day in [0,1) when timeSet
	timeSet           bool             // sky and sun follow Time (day/night cycle)
	animated          []animObj        // moving objects, re-placed each frame
	animTime          float64          // shared animation clock (seconds)
	landmarks         []Landmark       // named region positions (for the map)
}

// animObj is one moving object: its spec, its region offset, and a stable seed so
// siblings desynchronise.
type animObj struct {
	spec brain.ObjSpec
	at   raytrace.Vec3
	seed int
}

// SetAnimTime sets the shared animation clock (seconds) used to place moving
// objects; the app advances it from wall time so peers animate roughly in step.
func (w *World) SetAnimTime(sec float64) { w.animTime = sec }

// HasAnimated reports whether the world contains any moving objects (so a viewer
// knows the frame isn't static and progressive refinement should restart).
func (w *World) HasAnimated() bool { return len(w.animated) > 0 }

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
	_, spec, err := AuthorSceneCtx(b, prompt, w.context(w.chunks, at))
	if err != nil {
		return 0, err
	}
	w.lastSpec, w.lastAt = &spec, at
	return w.applyRegion(w.chunks, at, spec), nil
}

// context builds the continuity hints for authoring the next region: its index and
// the heading from the previous region to this one (zero on the first region).
func (w *World) context(index int, at raytrace.Vec3) SceneContext {
	ctx := SceneContext{Index: index, Prev: w.lastSpec}
	if w.lastSpec != nil {
		ctx.Heading = at.Sub(w.lastAt)
	}
	return ctx
}

// Scene builds a renderable, BVH-accelerated scene from the floor and all props.
func (w *World) Scene() *raytrace.Scene { return w.SceneWith(nil) }

// SceneWith is Scene plus extra transient objects (e.g. a remote player's avatar)
// that change every frame and so can't live in the persistent prop list.
func (w *World) SceneWith(extra []raytrace.Object) *raytrace.Scene {
	s := &raytrace.Scene{SkyTop: w.SkyTop, SkyBottom: w.SkyBottom}
	var sun []raytrace.Object
	if w.timeSet { // day/night: sky and a sun (a distant emitter) follow the time
		top, bottom, sunDir, sunColor, up := SkyForTime(w.Time)
		s.SkyTop, s.SkyBottom = top, bottom
		s.Light = sunDir.Scale(100) // raster directional light
		if up {
			// a physical (Preetham) sky while the sun is up — blue overhead, warming
			// and reddening toward the horizon/sun automatically as it sinks.
			s.Sky = raytrace.NewPreethamSky(sunDir, 2.6)
			ahead := raytrace.Vec3{X: 0, Y: 0, Z: w.lastAt.Z} // keep the sun near the frontier
			sun = []raytrace.Object{raytrace.Sphere{Center: sunDir.Scale(90).Add(ahead), Radius: 6, Mat: raytrace.Material{Emit: sunColor}}}
		}
	}
	s.Objects = append(s.Objects, w.floor)
	s.Objects = append(s.Objects, w.props...)
	s.Objects = append(s.Objects, sun...)
	// moving objects: re-place each from the shared animation clock.
	for _, a := range w.animated {
		s.Objects = append(s.Objects, objectsFromSpec(animateSpec(a.spec, w.animTime, a.seed), a.at, false)...)
	}
	s.Objects = append(s.Objects, extra...)
	s.BuildBVH()
	return s
}
