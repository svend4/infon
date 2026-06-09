package raytrace

import (
	"math"
	"testing"
)

// Perlin noise stays in roughly [-1,1] and is deterministic.
func TestPerlinRangeAndDeterminism(t *testing.T) {
	rg := newRNG(1)
	for i := 0; i < 20000; i++ {
		p := Vec3{X: rg.f()*40 - 20, Y: rg.f()*40 - 20, Z: rg.f()*40 - 20}
		n := Perlin3(p)
		if n < -1.01 || n > 1.01 {
			t.Fatalf("Perlin3(%v) = %.4f out of range", p, n)
		}
		if Perlin3(p) != n {
			t.Fatal("Perlin3 should be deterministic")
		}
	}
}

// Improved Perlin noise is exactly zero at integer lattice points.
func TestPerlinZeroAtLattice(t *testing.T) {
	for x := -3; x <= 3; x++ {
		for y := -3; y <= 3; y++ {
			for z := -3; z <= 3; z++ {
				if n := Perlin3(Vec3{X: float64(x), Y: float64(y), Z: float64(z)}); math.Abs(n) > 1e-12 {
					t.Errorf("Perlin3 at lattice (%d,%d,%d) = %.2e, want 0", x, y, z, n)
				}
			}
		}
	}
}

// Perlin noise is continuous: nearby points have nearby values.
func TestPerlinContinuity(t *testing.T) {
	rg := newRNG(2)
	for i := 0; i < 5000; i++ {
		p := Vec3{X: rg.f() * 10, Y: rg.f() * 10, Z: rg.f() * 10}
		q := p.Add(Vec3{X: 1e-3, Y: 0, Z: 0})
		if math.Abs(Perlin3(p)-Perlin3(q)) > 0.05 {
			t.Fatalf("noise should be smooth, jump %.4f at %v", math.Abs(Perlin3(p)-Perlin3(q)), p)
		}
	}
}

func TestFBMRange(t *testing.T) {
	rg := newRNG(3)
	for i := 0; i < 20000; i++ {
		p := Vec3{X: rg.f()*30 - 15, Y: rg.f()*30 - 15, Z: rg.f()*30 - 15}
		if n := FBM(p, 5, 2, 0.5); n < -1.01 || n > 1.01 {
			t.Fatalf("FBM = %.4f out of range", n)
		}
	}
}

// Worley noise is non-negative, deterministic, and small (the nearest of the 27
// neighbouring cells' feature points is always within ~sqrt(3)).
func TestWorleyRange(t *testing.T) {
	rg := newRNG(4)
	for i := 0; i < 20000; i++ {
		p := Vec3{X: rg.f()*30 - 15, Y: rg.f()*30 - 15, Z: rg.f()*30 - 15}
		d := Worley3(p)
		if d < 0 || d > 1.8 {
			t.Fatalf("Worley3(%v) = %.4f out of expected range", p, d)
		}
		if Worley3(p) != d {
			t.Fatal("Worley3 should be deterministic")
		}
	}
}

// The texture wrappers blend strictly between their two colours.
func TestNoiseTexturesBlend(t *testing.T) {
	a, b := Vec3{0, 0, 0}, Vec3{1, 1, 1}
	check := func(name string, tex Texture) {
		rg := newRNG(5)
		for i := 0; i < 5000; i++ {
			p := Vec3{X: rg.f()*20 - 10, Y: rg.f()*20 - 10, Z: rg.f()*20 - 10}
			c := tex.At(0, 0, p)
			if c.X < -1e-9 || c.X > 1+1e-9 {
				t.Fatalf("%s produced %v outside [A,B]", name, c)
			}
		}
	}
	check("NoiseTex", NoiseTex{A: a, B: b, Scale: 2, Octaves: 4})
	check("MarbleTex", MarbleTex{A: a, B: b, Scale: 2, Turb: 1})
}
