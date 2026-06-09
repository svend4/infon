package raytrace

import "testing"

func inUnitC(v Vec3) bool {
	return v.X >= 0 && v.X <= 1 && v.Y >= 0 && v.Y <= 1 && v.Z >= 0 && v.Z <= 1
}

// The mandala texture is radially symmetric: it varies across the plane (not a flat
// fill) and stays in range.
func TestKaleidoTex(t *testing.T) {
	k := KaleidoTex{A: Vec3{X: 0.1, Y: 0.1, Z: 0.2}, B: Vec3{X: 0.9, Y: 0.8, Z: 0.4}, C: Vec3{X: 0.9, Y: 0.2, Z: 0.4}, Sym: 8, Scale: 1.2}
	c0 := k.At(0, 0, Vec3{X: 0.3, Y: 0, Z: 0.1})
	c1 := k.At(0, 0, Vec3{X: 1.7, Y: 0, Z: 0.9})
	if c0 == c1 {
		t.Error("mandala should vary across the plane")
	}
	for _, c := range []Vec3{c0, c1} {
		if !inUnitC(c) {
			t.Errorf("mandala colour out of range: %v", c)
		}
	}
	// rotational symmetry: a point and its wedge-rotated twin match (8-fold here)
	p := Vec3{X: 0.9, Y: 0, Z: 0.0}
	rot := Vec3{X: 0.9 * 0.70710678, Y: 0, Z: 0.9 * 0.70710678} // +45 deg = one wedge of 8
	if k.At(0, 0, p).Sub(k.At(0, 0, rot)).Len() > 0.05 {
		t.Errorf("8-fold symmetry broken: %v vs %v", k.At(0, 0, p), k.At(0, 0, rot))
	}
}

// The tessellation texture interlocks two colours (uses both, varies per cell).
func TestTileTex(t *testing.T) {
	tt := TileTex{A: Vec3{X: 1, Y: 1, Z: 1}, B: Vec3{X: 0, Y: 0, Z: 0.2}, Scale: 0.7}
	seenA, seenB := false, false
	for i := 0; i < 20; i++ {
		for j := 0; j < 20; j++ {
			c := tt.At(0, 0, Vec3{X: float64(i) * 0.31, Y: 0, Z: float64(j) * 0.27})
			if c == tt.A {
				seenA = true
			}
			if c == tt.B {
				seenB = true
			}
		}
	}
	if !seenA || !seenB {
		t.Error("tessellation should use both colours across the plane")
	}
}
