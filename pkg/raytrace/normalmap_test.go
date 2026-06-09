package raytrace

import (
	"math"
	"testing"
)

// fixedNormal is a constant normal map (RGB-encoded) for tests.
type fixedNormal struct{ enc Vec3 }

func (f fixedNormal) At(_, _ float64, _ Vec3) Vec3 { return f.enc }

func encodeNormal(n Vec3) Vec3 {
	return Vec3{X: 0.5*n.X + 0.5, Y: 0.5*n.Y + 0.5, Z: 0.5*n.Z + 0.5}
}

// A flat normal map (0.5,0.5,1) must leave the shading normal unchanged.
func TestNormalMapFlatIdentity(t *testing.T) {
	s := &Scene{Objects: []Object{
		Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{1, 1, 1}, Bump: fixedNormal{Vec3{0.5, 0.5, 1}}}},
	}}
	h, ok := s.closest(Ray{Origin: Vec3{0, 0, -5}, Dir: Vec3{0, 0, 1}}, geomEps, tFar)
	if !ok {
		t.Fatal("ray should hit the sphere")
	}
	if h.N.Sub(Vec3{0, 0, -1}).Len() > 1e-9 {
		t.Errorf("flat normal map should not change the normal, got %v", h.N)
	}
}

// A tangent-space normal (a,b,c) tilts the shading normal so that its dot with the
// geometric normal equals c (the tangent and bitangent are perpendicular to it),
// independent of how the tangent frame is chosen.
func TestNormalMapTiltsByZComponent(t *testing.T) {
	tn := Vec3{X: 0.6, Y: 0, Z: 0.8} // unit tangent-space normal
	s := &Scene{Objects: []Object{
		Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{1, 1, 1}, Bump: fixedNormal{encodeNormal(tn)}}},
	}}
	h, ok := s.closest(Ray{Origin: Vec3{0, 0, -5}, Dir: Vec3{0, 0, 1}}, geomEps, tFar)
	if !ok {
		t.Fatal("ray should hit the sphere")
	}
	geom := Vec3{0, 0, -1}
	if d := h.N.Dot(geom); math.Abs(d-0.8) > 1e-9 {
		t.Errorf("perturbed normal . geometric = %.6f, want 0.8 (the tangent-space z)", d)
	}
	if l := h.N.Len(); math.Abs(l-1) > 1e-9 {
		t.Errorf("perturbed normal must be unit length, got %.6f", l)
	}
	if h.N.Sub(geom).Len() < 1e-3 {
		t.Error("perturbed normal should differ from the geometric normal")
	}
}

// A normal map must change the shading: the rendered floor with ripple bumps
// differs substantially from the flat floor (same scene, same seed).
func TestNormalMapAffectsShading(t *testing.T) {
	v := func(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }
	mk := func(bump Texture) *Scene {
		return &Scene{
			Objects: []Object{
				Plane{Y: 0, Size: 1, C1: v(0.7, 0.7, 0.7), C2: v(0.7, 0.7, 0.7), Mat: Material{Bump: bump}},
				Sphere{Center: v(0, 4, 1), Radius: 0.6, Mat: Material{Emit: v(25, 25, 25)}},
			},
			SkyTop: v(0.02, 0.02, 0.03), SkyBottom: v(0.02, 0.02, 0.03),
		}
	}
	cam := Camera{Pos: v(0, 1.6, -3), Pitch: -0.5, FOV: 1.0}
	opt := PathOptions{Samples: 24, MaxDepth: 2, Seed: 1, NEE: true}
	flat := PathRender(mk(nil), cam, 80, 60, opt)
	bumpy := PathRender(mk(WaveNormalTex{Freq: 6, Amp: 0.6}), cam, 80, 60, opt)
	if d := rmse(flat, bumpy); d < 4 {
		t.Errorf("normal map should change the shading, RMSE flat vs bumpy = %.2f", d)
	}
}
