package raydir

import (
	"math"
	"math/rand"

	"github.com/svend4/infon/pkg/raytrace"
)

// creatures.go gives the world inhabitants of its own: a flock of small fliers
// that lives by simple local rules (Craig Reynolds' boids — separation,
// alignment, cohesion) and reacts to you. They drift between the world's named
// places, gather there, and scatter when you walk into them. Like the guide, the
// behaviour is local and deterministic (seeded), so it runs offline and is
// testable; unlike props, they move every frame from the shared clock.

// FlockColor is the default plumage: dark birds that read as silhouettes against
// the sky, with a touch of self-illumination so they stay visible over ground.
var FlockColor = raytrace.Vec3{X: 0.16, Y: 0.17, Z: 0.22}

// Boid is one creature: where it is and where it's going.
type Boid struct {
	Pos raytrace.Vec3
	Vel raytrace.Vec3
}

// Flock is a self-organising group of boids that wander the world, gather at
// landmarks, and flee the walker.
type Flock struct {
	Boids []Boid
	Color raytrace.Vec3
	rng   *rand.Rand

	// behaviour weights and radii (tuned for a calm, readable flock).
	perceptR float64 // neighbours within this radius influence a boid
	sepR     float64 // closer than this, push apart
	fleeR    float64 // within this of the walker, scatter
	maxSpeed float64
	wSep     float64
	wAli     float64
	wCoh     float64
	wFlee    float64
	wGather  float64 // pull toward the nearest landmark
	wBand    float64 // keep within the flight-altitude band
	bandLo   float64
	bandHi   float64
}

// NewFlock spawns n boids in a loose cluster around centre, with a fixed seed so
// the same world animates identically everywhere.
func NewFlock(n int, centre raytrace.Vec3, seed int64) *Flock {
	f := &Flock{
		Color:    FlockColor,
		rng:      rand.New(rand.NewSource(seed)),
		perceptR: 6.0,
		sepR:     1.6,
		fleeR:    7.0,
		maxSpeed: 4.0,
		wSep:     2.2,
		wAli:     1.0,
		wCoh:     0.9,
		wFlee:    9.0,
		wGather:  0.6,
		wBand:    2.0,
		bandLo:   2.5,
		bandHi:   7.0,
	}
	for i := 0; i < n; i++ {
		f.Boids = append(f.Boids, Boid{
			Pos: centre.Add(raytrace.Vec3{
				X: (f.rng.Float64()*2 - 1) * 4,
				Y: f.bandLo + f.rng.Float64()*(f.bandHi-f.bandLo),
				Z: (f.rng.Float64()*2 - 1) * 4,
			}),
			Vel: raytrace.Vec3{X: f.rng.Float64()*2 - 1, Y: 0, Z: f.rng.Float64()*2 - 1},
		})
	}
	return f
}

// nearestMark returns the position of the nearest landmark to p (at flight
// altitude), and whether there was one.
func nearestMark(p raytrace.Vec3, marks []Landmark) (raytrace.Vec3, bool) {
	best, bestD, ok := raytrace.Vec3{}, math.Inf(1), false
	for _, m := range marks {
		if d := m.At.Sub(p).LenSq(); d < bestD {
			best, bestD, ok = m.At, d, true
		}
	}
	return best, ok
}

// Step advances every boid by dt seconds under the boid rules plus fleeing the
// walker and gathering at the nearest landmark. Deterministic given the seed.
func (f *Flock) Step(dt float64, player raytrace.Vec3, marks []Landmark) {
	if dt <= 0 {
		return
	}
	gather, hasMark := nearestMark(player, marks)
	next := make([]raytrace.Vec3, len(f.Boids)) // new velocities (computed from old state)
	for i := range f.Boids {
		b := f.Boids[i]
		var sep, ali, coh raytrace.Vec3
		var nNeigh, nClose int
		for j := range f.Boids {
			if j == i {
				continue
			}
			o := f.Boids[j]
			off := b.Pos.Sub(o.Pos)
			d := off.Len()
			if d > 0 && d < f.sepR { // separation: steer away, stronger when closer
				sep = sep.Add(off.Scale(1 / d))
				nClose++
			}
			if d < f.perceptR { // alignment + cohesion over perceived neighbours
				ali = ali.Add(o.Vel)
				coh = coh.Add(o.Pos)
				nNeigh++
			}
		}
		acc := raytrace.Vec3{}
		if nClose > 0 {
			acc = acc.Add(sep.Norm().Scale(f.wSep))
		}
		if nNeigh > 0 {
			ali = ali.Scale(1 / float64(nNeigh))
			acc = acc.Add(ali.Sub(b.Vel).Norm().Scale(f.wAli))
			coh = coh.Scale(1 / float64(nNeigh))
			acc = acc.Add(coh.Sub(b.Pos).Norm().Scale(f.wCoh))
		}
		if hasMark { // drift toward the nearest place (so flocks haunt landmarks)
			acc = acc.Add(gather.Sub(b.Pos).Norm().Scale(f.wGather))
		}
		// flee the walker: the closer they are, the harder the boid bolts.
		if fl := b.Pos.Sub(player); fl.Len() < f.fleeR {
			d := fl.Len()
			if d < 1e-4 {
				fl, d = raytrace.Vec3{X: 1}, 1
			}
			acc = acc.Add(fl.Scale(1 / d).Scale(f.wFlee * (f.fleeR - d) / f.fleeR))
		}
		// keep them in the air, within a comfortable altitude band.
		switch {
		case b.Pos.Y < f.bandLo:
			acc = acc.Add(raytrace.Vec3{Y: f.wBand * (f.bandLo - b.Pos.Y)})
		case b.Pos.Y > f.bandHi:
			acc = acc.Add(raytrace.Vec3{Y: -f.wBand * (b.Pos.Y - f.bandHi)})
		}
		// a little wander so a settled flock never freezes (deterministic).
		acc = acc.Add(raytrace.Vec3{X: f.rng.Float64()*2 - 1, Y: (f.rng.Float64()*2 - 1) * 0.3, Z: f.rng.Float64()*2 - 1}.Scale(0.4))

		v := b.Vel.Add(acc.Scale(dt))
		if s := v.Len(); s > f.maxSpeed {
			v = v.Scale(f.maxSpeed / s)
		}
		next[i] = v
	}
	for i := range f.Boids { // integrate positions from the new velocities
		f.Boids[i].Vel = next[i]
		f.Boids[i].Pos = f.Boids[i].Pos.Add(next[i].Scale(dt))
	}
}

// Centroid is the flock's average position (its centre of mass).
func (f *Flock) Centroid() raytrace.Vec3 {
	if len(f.Boids) == 0 {
		return raytrace.Vec3{}
	}
	var c raytrace.Vec3
	for _, b := range f.Boids {
		c = c.Add(b.Pos)
	}
	return c.Scale(1 / float64(len(f.Boids)))
}

// Recenter keeps the flock near the walker's frontier: if it has fallen far
// behind (or raced too far ahead), nudge the whole flock back into view. Returns
// true if it moved them.
func (f *Flock) Recenter(player raytrace.Vec3) bool {
	c := f.Centroid()
	if dz := c.Z - player.Z; dz < -30 || dz > 40 {
		shift := raytrace.Vec3{Z: player.Z + 12 - c.Z}
		for i := range f.Boids {
			f.Boids[i].Pos = f.Boids[i].Pos.Add(shift)
		}
		return true
	}
	return false
}

// Objects renders the flock: each boid a small dark body with a tiny marker ahead
// showing its heading, so a moving flock reads as living silhouettes.
func (f *Flock) Objects() []raytrace.Object {
	out := make([]raytrace.Object, 0, len(f.Boids)*2)
	emit := f.Color.Scale(0.5)
	for _, b := range f.Boids {
		fwd := b.Vel.Norm()
		if fwd.LenSq() == 0 {
			fwd = raytrace.Vec3{Z: 1}
		}
		out = append(out,
			raytrace.Sphere{Center: b.Pos, Radius: 0.16, Mat: raytrace.Material{Color: f.Color, Emit: emit}},
			raytrace.Sphere{Center: b.Pos.Add(fwd.Scale(0.26)), Radius: 0.07, Mat: raytrace.Material{Color: f.Color, Emit: emit}},
		)
	}
	return out
}
