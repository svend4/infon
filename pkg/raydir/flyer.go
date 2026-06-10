package raydir

import (
	"math"

	"github.com/svend4/infon/pkg/raytrace"
)

// flyer.go puts a predator in the world: the "летун" the dream hackers warn about
// (Castaneda's flyer) — a dark, flattened shadow that stalks the walker and, when
// it catches up, drains your "luminosity". It is slower than you can run, so it is
// a thing to keep ahead of, not an unbeatable threat. Local and deterministic, so
// it works offline and is tested.

// FlyerColor is the predator's near-black, slightly cool body — a shadow that
// silhouettes against the sky rather than catching its light.
var FlyerColor = raytrace.Vec3{X: 0.03, Y: 0.03, Z: 0.045}

// Flyer is a predator that pursues the walker.
type Flyer struct {
	Pos    raytrace.Vec3
	Yaw    float64
	speed  float64
	drainR float64
	t      float64 // its own clock, for a hovering wobble
}

// NewFlyer makes a flyer starting at `at`.
func NewFlyer(at raytrace.Vec3) *Flyer {
	return &Flyer{Pos: at, speed: 1.6, drainR: 3.0}
}

// SpawnFlyer gives the world a predator starting at `at`.
func (w *World) SpawnFlyer(at raytrace.Vec3) { w.flyer = NewFlyer(at) }

// StepFlyer advances the predator and returns true while it is draining you.
func (w *World) StepFlyer(dt float64, player raytrace.Vec3) bool {
	if w.flyer == nil {
		return false
	}
	return w.flyer.Step(dt, player)
}

// HasFlyer reports whether a predator stalks the world.
func (w *World) HasFlyer() bool { return w.flyer != nil }

// Pose is the flyer's pose, for rendering.
func (f *Flyer) Pose() Pose { return Pose{Pos: f.Pos, Yaw: f.Yaw} }

// Step pursues the walker by dt seconds (hovering, facing its motion) and returns
// true when it is close enough to drain your luminosity.
func (f *Flyer) Step(dt float64, player raytrace.Vec3) bool {
	if f.speed <= 0 {
		f.speed = 1.6
	}
	f.t += dt
	flat := raytrace.Vec3{X: player.X - f.Pos.X, Z: player.Z - f.Pos.Z}
	dist := flat.Len()
	if dist > 1e-4 {
		dir := flat.Scale(1 / dist)
		step := f.speed * dt
		if step > dist {
			step = dist
		}
		f.Pos = f.Pos.Add(dir.Scale(step))
		f.Yaw = math.Atan2(dir.X, dir.Z)
	}
	f.Pos.Y = 1.6 + 0.3*math.Sin(f.t*1.7) // hover with a slow bob
	return dist < f.drainR
}

// Objects renders the flyer: a dark, flattened shadow-disc with a faint red glow.
func (f *Flyer) Objects() []raytrace.Object {
	emit := raytrace.Vec3{X: 0.012, Y: 0.012, Z: 0.015} // a barely-there core, neutral so it reads as shadow
	mat := raytrace.Material{Color: FlyerColor, Emit: emit, Rough: 0.9}
	out := []raytrace.Object{raytrace.Sphere{Center: f.Pos, Radius: 0.5, Mat: mat}}
	for i := 0; i < 6; i++ { // a flattened ring of lobes -> a manta-like shadow
		a := f.Yaw + float64(i)/6*2*math.Pi
		out = append(out, raytrace.Sphere{
			Center: raytrace.Vec3{X: f.Pos.X + math.Cos(a)*1.1, Y: f.Pos.Y - 0.1, Z: f.Pos.Z + math.Sin(a)*1.1},
			Radius: 0.34, Mat: mat,
		})
	}
	return out
}
