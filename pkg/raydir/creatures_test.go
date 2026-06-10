package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// spread is the largest distance between any two boids — a measure of how
// dispersed the flock is.
func spread(f *Flock) float64 {
	mx := 0.0
	for i := range f.Boids {
		for j := i + 1; j < len(f.Boids); j++ {
			if d := f.Boids[i].Pos.Sub(f.Boids[j].Pos).Len(); d > mx {
				mx = d
			}
		}
	}
	return mx
}

func meanDist(f *Flock, p raytrace.Vec3) float64 {
	if len(f.Boids) == 0 {
		return 0
	}
	s := 0.0
	for _, b := range f.Boids {
		s += b.Pos.Sub(p).Len()
	}
	return s / float64(len(f.Boids))
}

// Separation: two near-coincident boids must push apart, not overlap forever.
func TestFlockSeparates(t *testing.T) {
	f := NewFlock(0, raytrace.Vec3{}, 1)
	f.Boids = []Boid{{Pos: raytrace.Vec3{Y: 4}}, {Pos: raytrace.Vec3{X: 0.1, Y: 4}}}
	far := raytrace.Vec3{Z: 1000} // walker far away: no fleeing
	before := f.Boids[0].Pos.Sub(f.Boids[1].Pos).Len()
	for i := 0; i < 200; i++ {
		f.Step(0.05, far, nil)
	}
	after := f.Boids[0].Pos.Sub(f.Boids[1].Pos).Len()
	if after <= before {
		t.Errorf("separation should push boids apart: before %.3f after %.3f", before, after)
	}
	if after < 0.8 {
		t.Errorf("boids stayed bunched up: distance %.3f", after)
	}
}

// Cohesion: a dispersed flock should pull itself together. Measured as the mean
// distance to the centroid (translation-invariant, so the flock's drift doesn't
// matter); we catch the tightest moment, since a lively flock breathes.
func TestFlockCohesion(t *testing.T) {
	f := NewFlock(0, raytrace.Vec3{}, 2)
	for x := -10.0; x <= 10; x += 4 { // a strung-out chain, each link within sight
		f.Boids = append(f.Boids, Boid{Pos: raytrace.Vec3{X: x, Y: 4}})
	}
	far := raytrace.Vec3{Z: 1000} // walker far away: no fleeing
	meanToCentroid := func() float64 {
		c := f.Centroid()
		s := 0.0
		for _, b := range f.Boids {
			s += b.Pos.Sub(c).Len()
		}
		return s / float64(len(f.Boids))
	}
	before := meanToCentroid()
	tightest := before
	for i := 0; i < 500; i++ {
		f.Step(0.05, far, nil)
		if v := meanToCentroid(); v < tightest {
			tightest = v
		}
	}
	if tightest >= before*0.6 {
		t.Errorf("cohesion should pull the flock together: start %.2f, tightest %.2f", before, tightest)
	}
}

// Flee: when the walker is among them, the flock scatters away.
func TestFlockFleesPlayer(t *testing.T) {
	f := NewFlock(12, raytrace.Vec3{Y: 4}, 3)
	player := raytrace.Vec3{Y: 4} // right in the middle of the spawn cluster
	before := meanDist(f, player)
	for i := 0; i < 100; i++ {
		f.Step(0.05, player, nil)
	}
	after := meanDist(f, player)
	if after <= before*1.5 {
		t.Errorf("flock should flee the walker: mean dist before %.2f after %.2f", before, after)
	}
}

// Bounded: with no walker and no landmarks, the flock stays finite and clustered
// — it must not blow up to infinity.
func TestFlockBounded(t *testing.T) {
	f := NewFlock(15, raytrace.Vec3{Y: 4}, 4)
	far := raytrace.Vec3{Z: 1e6}
	for i := 0; i < 500; i++ {
		f.Step(0.05, far, nil)
	}
	for _, b := range f.Boids {
		if math.IsNaN(b.Pos.X) || math.IsInf(b.Pos.X, 0) || math.IsNaN(b.Pos.Y) || math.IsNaN(b.Pos.Z) {
			t.Fatalf("boid position went non-finite: %+v", b.Pos)
		}
	}
	if s := spread(f); s > 60 {
		t.Errorf("flock dispersed without bound: spread %.1f", s)
	}
}

// Altitude band: boids should be steered to stay airborne.
func TestFlockStaysAirborne(t *testing.T) {
	f := NewFlock(10, raytrace.Vec3{Y: 4}, 5)
	for i := range f.Boids {
		f.Boids[i].Pos.Y = -2 // start them underground
	}
	far := raytrace.Vec3{Z: 1000}
	for i := 0; i < 200; i++ {
		f.Step(0.05, far, nil)
	}
	below := 0
	for _, b := range f.Boids {
		if b.Pos.Y < 1 {
			below++
		}
	}
	if below > 2 {
		t.Errorf("most boids should have climbed into the air, %d still underground", below)
	}
}

// Gather: a single distant landmark should draw the flock toward it.
func TestFlockGathers(t *testing.T) {
	f := NewFlock(10, raytrace.Vec3{Y: 4}, 6)
	marks := []Landmark{{Index: 0, At: raytrace.Vec3{X: 40, Y: 4, Z: 40}, Name: "Far Place"}}
	far := raytrace.Vec3{X: 0, Y: 4, Z: -1000} // walker not near the flock or the mark
	before := meanDist(f, marks[0].At)
	for i := 0; i < 200; i++ {
		f.Step(0.05, far, marks)
	}
	after := meanDist(f, marks[0].At)
	if after >= before {
		t.Errorf("flock should drift toward the landmark: before %.1f after %.1f", before, after)
	}
}

// Recenter pulls a flock that has fallen far behind back to the frontier.
func TestFlockRecenter(t *testing.T) {
	f := NewFlock(5, raytrace.Vec3{Y: 4, Z: -50}, 7)
	player := raytrace.Vec3{Y: 2}
	if !f.Recenter(player) {
		t.Fatal("a flock 50 behind the walker should be recentred")
	}
	if dz := f.Centroid().Z - player.Z; math.Abs(dz-12) > 6 {
		t.Errorf("recentred flock should sit ~12 ahead, got dz %.1f", dz)
	}
	if f.Recenter(player) { // now in view: nothing to do
		t.Error("a flock already near the walker should not be recentred")
	}
}

// Objects renders two spheres (body + heading marker) per boid.
func TestFlockObjects(t *testing.T) {
	f := NewFlock(8, raytrace.Vec3{Y: 4}, 8)
	if got := len(f.Objects()); got != 16 {
		t.Errorf("expected 16 objects for 8 boids, got %d", got)
	}
}
