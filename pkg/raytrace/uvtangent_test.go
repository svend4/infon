package raytrace

import (
	"math"
	"testing"
)

// A triangle whose U axis runs along +X must yield a tangent along +X with
// right-handed handedness.
func TestTriTangentAlignsWithU(t *testing.T) {
	tr := Triangle{
		A: Vec3{0, 0, 0}, B: Vec3{1, 0, 0}, C: Vec3{0, 1, 0},
		UVa: [2]float64{0, 0}, UVb: [2]float64{1, 0}, UVc: [2]float64{0, 1},
	}
	tan, w, ok := triTangent(tr)
	if !ok {
		t.Fatal("triangle with valid UVs should have a tangent")
	}
	if tan.Sub(Vec3{1, 0, 0}).Len() > 1e-9 {
		t.Errorf("tangent = %v, want +X (the U direction)", tan)
	}
	if w != 1 {
		t.Errorf("handedness = %.0f, want +1", w)
	}
}

// Mirrored UVs flip the tangent-space handedness.
func TestTriTangentMirroredHandedness(t *testing.T) {
	tr := Triangle{
		A: Vec3{0, 0, 0}, B: Vec3{1, 0, 0}, C: Vec3{0, 1, 0},
		UVa: [2]float64{0, 0}, UVb: [2]float64{1, 0}, UVc: [2]float64{0, -1}, // V mirrored
	}
	_, w, ok := triTangent(tr)
	if !ok || w != -1 {
		t.Errorf("mirrored UVs should give handedness -1, got ok=%v w=%.0f", ok, w)
	}
}

// A normal map tilting toward +U must tilt the shading normal along the triangle's
// world U direction (+X here) - i.e. the frame is UV-aligned, not arbitrary.
func TestNormalMapUVAligned(t *testing.T) {
	tn := Vec3{X: 0.6, Y: 0, Z: 0.8} // tangent-space normal leaning toward +U
	tr := Triangle{
		A: Vec3{0, 0, 0}, B: Vec3{1, 0, 0}, C: Vec3{0, 1, 0},
		UVa: [2]float64{0, 0}, UVb: [2]float64{1, 0}, UVc: [2]float64{0, 1},
		Mat: Material{Color: Vec3{1, 1, 1}, Bump: fixedNormal{encodeNormal(tn)}},
	}
	s := &Scene{Objects: []Object{tr}}
	h, ok := s.closest(Ray{Origin: Vec3{0.25, 0.25, 2}, Dir: Vec3{0, 0, -1}}, geomEps, tFar)
	if !ok {
		t.Fatal("ray should hit the triangle")
	}
	// expect the shading normal = 0.6*tangent(+X) + 0.8*N(+Z) = (0.6, 0, 0.8).
	if math.Abs(h.N.X-0.6) > 1e-6 || math.Abs(h.N.Z-0.8) > 1e-6 || math.Abs(h.N.Y) > 1e-6 {
		t.Errorf("perturbed normal = %v, want (0.6,0,0.8) aligned to the U axis", h.N)
	}
}

// Without a normal map, no tangent is computed (hot path stays cheap).
func TestNoTangentWithoutBump(t *testing.T) {
	tr := Triangle{
		A: Vec3{0, 0, 0}, B: Vec3{1, 0, 0}, C: Vec3{0, 1, 0},
		UVa: [2]float64{0, 0}, UVb: [2]float64{1, 0}, UVc: [2]float64{0, 1},
	}
	h, _ := tr.Intersect(Ray{Origin: Vec3{0.25, 0.25, 2}, Dir: Vec3{0, 0, -1}}, geomEps, tFar)
	if h.Tan.LenSq() != 0 {
		t.Errorf("no normal map: tangent should be unset, got %v", h.Tan)
	}
}
