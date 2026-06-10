package raytrace

import (
	"math"
	"testing"
)

// A downward ray meets the water plane at its level, with an up-ish, rippled normal
// and a reflective material.
func TestWaterIntersect(t *testing.T) {
	w := Water{Y: 1, Color: Vec3{X: 0.1, Y: 0.3, Z: 0.45}, Reflect: 0.5, Scale: 2, Amp: 0.3}
	r := Ray{Origin: Vec3{X: 0.7, Y: 4, Z: -0.3}, Dir: Vec3{X: 0, Y: -1, Z: 0}}
	h, ok := w.Intersect(r, geomEps, tFar)
	if !ok {
		t.Fatal("a downward ray should hit the water")
	}
	if math.Abs(h.P.Y-1) > 1e-9 {
		t.Errorf("hit should be at the water level, got y=%.3f", h.P.Y)
	}
	if h.N.Y < 0.7 {
		t.Errorf("water normal should point up-ish, got %v", h.N)
	}
	if math.Abs(h.N.Len()-1) > 1e-6 {
		t.Errorf("normal should be unit, got %.4f", h.N.Len())
	}
	if h.Mat.Reflect <= 0 {
		t.Error("water should be reflective")
	}
	if _, up := w.Intersect(Ray{Origin: Vec3{Y: 4}, Dir: Vec3{X: 1}}, geomEps, tFar); up {
		t.Error("a horizontal ray should not hit the water plane")
	}
}

// The waves move (normal changes over time) and vary across the surface.
func TestWaterRipples(t *testing.T) {
	a := Water{Y: 0, Scale: 2, Amp: 0.3, Time: 0}
	b := Water{Y: 0, Scale: 2, Amp: 0.3, Time: 1.5}
	p := Vec3{X: 1, Y: 0, Z: 1}
	if a.normal(p) == b.normal(p) {
		t.Error("water normal should change as the waves advance in time")
	}
	if a.normal(Vec3{X: 1}) == a.normal(Vec3{X: 5}) {
		t.Error("water normal should vary across the surface")
	}
}

// A cloud medium has FBM density: 0 in the thin gaps, positive (but capped) in the
// thick parts, with a bright forward-scattering albedo.
func TestCloudMedium(t *testing.T) {
	m := CloudMedium(Vec3{}, 20, 0.6)
	if m.Albedo.X < 0.5 || m.G <= 0 {
		t.Errorf("clouds should be bright and forward-scattering, got albedo %v g %.2f", m.Albedo, m.G)
	}
	var maxS, minS = 0.0, 1e9
	for i := 0; i < 400; i++ {
		s := m.sigma(Vec3{X: float64(i%20) - 10, Y: float64((i/20)%20) - 10, Z: float64(i % 7)})
		if s < 0 || s > 0.6+1e-9 {
			t.Fatalf("sigma out of [0,0.6]: %.3f", s)
		}
		maxS, minS = math.Max(maxS, s), math.Min(minS, s)
	}
	if maxS <= 0 {
		t.Error("clouds should be dense somewhere")
	}
	if minS != 0 {
		t.Error("clouds should be wispy (zero density in gaps)")
	}
}
