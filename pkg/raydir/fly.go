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
	clouds            bool             // volumetric cloud medium on
	sndWater          bool             // soundscape features present in the world
	sndForest         int
	sndBirds          bool
	sndHum            bool
	trace             *Trace               // worn paths (the world's memory of being walked)
	env               *raytrace.EnvSampler // cached sky importance sampler (env-NEE)
	envAt             float64              // time the env sampler was built for
	applied           []Region             // applied regions (kept so far-behind ones can be pruned)
	flock             *Flock               // optional living inhabitants (boids)
	weather           *Weather             // optional precipitation/fog
}

// SetWeather gives the world weather of the given kind ("rain", "snow", "fog");
// an empty kind clears it.
func (w *World) SetWeather(kind string) {
	if kind == "" {
		w.weather = nil
		return
	}
	w.weather = NewWeather(kind, 7)
}

// StepWeather advances the weather by dt seconds, keeping the band around the
// walker (centre).
func (w *World) StepWeather(dt float64, centre raytrace.Vec3) {
	if w.weather != nil {
		w.weather.Step(dt, centre)
	}
}

// HasWeather reports whether the world currently has weather.
func (w *World) HasWeather() bool { return w.weather != nil }

// SpawnFlock gives the world a flock of n inhabitants near `at` (seeded, so it
// animates identically everywhere).
func (w *World) SpawnFlock(n int, at raytrace.Vec3, seed int64) { w.flock = NewFlock(n, at, seed) }

// StepCreatures advances the world's inhabitants by dt seconds — they flee the
// walker and gather at landmarks — and keeps them near the frontier.
func (w *World) StepCreatures(dt float64, player raytrace.Vec3) {
	if w.flock == nil {
		return
	}
	w.flock.Step(dt, player, w.landmarks)
	w.flock.Recenter(player)
}

// HasCreatures reports whether the world has living inhabitants.
func (w *World) HasCreatures() bool { return w.flock != nil && len(w.flock.Boids) > 0 }

// Tread records a walker stepping at p, so paths wear into the ground over time.
func (w *World) Tread(p raytrace.Vec3) {
	if w.trace == nil {
		w.trace = NewTrace(1.5)
	}
	w.trace.Tread(p)
}

// Trace returns the world's path-wear (may be nil if nothing has been walked).
func (w *World) Trace() *Trace { return w.trace }

// SetTrace restores previously-saved path wear.
func (w *World) SetTrace(t *Trace) { w.trace = t }

// SetClouds toggles a volumetric cloud bank (a participating medium; path tracer
// only, and costly — opt in).
func (w *World) SetClouds(on bool) { w.clouds = on }

// animObj is one moving object: its spec, its region offset, and a stable seed so
// siblings desynchronise.
type animObj struct {
	spec brain.ObjSpec
	at   raytrace.Vec3
	seed int
	born float64 // animation-clock time when it appeared (for "grow")
}

// SetAnimTime sets the shared animation clock (seconds) used to place moving
// objects; the app advances it from wall time so peers animate roughly in step.
func (w *World) SetAnimTime(sec float64) { w.animTime = sec }

// HasAnimated reports whether the world contains any moving objects (so a viewer
// knows the frame isn't static and progressive refinement should restart).
func (w *World) HasAnimated() bool {
	return len(w.animated) > 0 || w.HasCreatures() || (w.weather != nil && len(w.weather.parts) > 0)
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
			// importance-sample the sky (env-NEE) for cleaner skylight; rebuilt only on
			// coarse time steps so it costs almost nothing per frame.
			if w.env == nil || math.Abs(w.Time-w.envAt) > 0.02 {
				w.env = raytrace.BuildEnvSamplerFromSky(s.Sky.At, 64, 32)
				w.envAt = w.Time
			}
			s.Env = w.env
			ahead := raytrace.Vec3{X: 0, Y: 0, Z: w.lastAt.Z} // keep the sun near the frontier
			sun = []raytrace.Object{raytrace.Sphere{Center: sunDir.Scale(90).Add(ahead), Radius: 6, Mat: raytrace.Material{Emit: sunColor}}}
		}
	}
	s.Objects = append(s.Objects, w.floor)
	s.Objects = append(s.Objects, w.props...)
	s.Objects = append(s.Objects, sun...)
	// moving (and growing) objects: re-place each from the shared animation clock.
	for _, a := range w.animated {
		switch {
		case a.spec.Kind == "water":
			s.Objects = append(s.Objects, waterObject(a.spec, a.at, w.animTime))
		case a.spec.Anim == "grow":
			s.Objects = append(s.Objects, objectsFromSpec(growSpec(a.spec, w.animTime-a.born), a.at, false)...)
		default:
			s.Objects = append(s.Objects, objectsFromSpec(animateSpec(a.spec, w.animTime, a.seed), a.at, false)...)
		}
	}
	if w.clouds { // volumetric cloud bank that drifts with the world frontier
		s.Medium = raytrace.CloudMedium(raytrace.Vec3{X: 0, Y: 11, Z: w.lastAt.Z}, 26, 0.5)
	}
	if w.trace != nil { // worn paths the world remembers
		s.Objects = append(s.Objects, w.trace.Objects()...)
	}
	if w.flock != nil { // living inhabitants, re-placed each frame
		s.Objects = append(s.Objects, w.flock.Objects()...)
	}
	if w.weather != nil { // rain/snow particles, and fog that hazes the distance
		s.Objects = append(s.Objects, w.weather.Objects()...)
		if d, c, on := w.weather.Fog(); on {
			s.FogDensity, s.FogColor = d, c
			s.SkyTop = s.SkyTop.Scale(0.45).Add(c.Scale(0.55)) // haze the sky to match
			s.SkyBottom = s.SkyBottom.Scale(0.35).Add(c.Scale(0.65))
		}
	}
	s.Objects = append(s.Objects, extra...)
	s.BuildBVH()
	return s
}
