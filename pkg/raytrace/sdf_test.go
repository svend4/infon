package raytrace

import (
	"math"
	"testing"
)

// A sphere-traced sphere field hits where the analytic sphere does, with the same
// outward normal — the sphere-tracing core is correct.
func TestMarchedMatchesAnalyticSphere(t *testing.T) {
	const R = 1.0
	m := Marched{
		DE:     func(p Vec3) float64 { return p.Len() - R },
		Center: Vec3{X: 0, Y: 0, Z: 5}, Radius: R * 1.6,
		Mat: Material{Color: Vec3{X: 1}},
	}
	ref := Sphere{Center: Vec3{X: 0, Y: 0, Z: 5}, Radius: R}
	r := Ray{Origin: Vec3{}, Dir: Vec3{X: 0, Y: 0, Z: 1}}
	hm, okm := m.Intersect(r, geomEps, tFar)
	hr, okr := ref.Intersect(r, geomEps, tFar)
	if !okm || !okr {
		t.Fatalf("both should hit (marched=%v analytic=%v)", okm, okr)
	}
	if math.Abs(hm.T-hr.T) > 2e-3 {
		t.Errorf("hit distance: marched %.4f vs analytic %.4f", hm.T, hr.T)
	}
	if hm.N.Sub(hr.N).Len() > 2e-2 {
		t.Errorf("normal: marched %v vs analytic %v", hm.N, hr.N)
	}
}

// The named fractals are hit by a ray aimed at them, with a unit normal, inside
// their bounding sphere.
func TestMarchedFractalsHit(t *testing.T) {
	for _, name := range []string{"mandelbulb", "menger", "mandala", "melt", "sierpinski", "lattice", "escher"} {
		m, ok := NewMarched(name, Vec3{X: 0, Y: 0, Z: 4}, 1.5, Material{Color: Vec3{X: 0.8}})
		if !ok {
			t.Fatalf("NewMarched(%q) should exist", name)
		}
		hit := false
		// fan a grid of rays through the bounding sphere — fractals (the Menger
		// sponge especially) are full of holes, so a single axis ray can miss.
		offs := []float64{-0.6, -0.35, -0.15, 0, 0.15, 0.35, 0.6}
		for _, dx := range offs {
			for _, dy := range offs {
				r := Ray{Origin: Vec3{X: dx, Y: dy, Z: 0}, Dir: Vec3{X: 0, Y: 0, Z: 1}}
				if h, ok := m.Intersect(r, geomEps, tFar); ok {
					if l := h.N.Len(); l < 0.9 || l > 1.1 {
						t.Errorf("%s: normal not unit (%.3f)", name, l)
					}
					if h.P.Sub(m.Center).Len() > m.Radius+1e-3 {
						t.Errorf("%s: hit outside bounding sphere", name)
					}
					hit = true
				}
			}
		}
		if !hit {
			t.Errorf("%s: no ray hit the fractal", name)
		}
	}
}

func TestMarchedUnknownName(t *testing.T) {
	if _, ok := NewMarched("not-a-fractal", Vec3{}, 1, Material{}); ok {
		t.Error("unknown fractal name should return ok=false")
	}
}

// A miss past the bounding sphere returns false cheaply.
func TestMarchedMissesBoundingSphere(t *testing.T) {
	m, _ := NewMarched("menger", Vec3{X: 0, Y: 0, Z: 5}, 1, Material{})
	r := Ray{Origin: Vec3{X: 10, Y: 0, Z: 0}, Dir: Vec3{X: 0, Y: 0, Z: 1}}
	if _, ok := m.Intersect(r, geomEps, tFar); ok {
		t.Error("a ray far from the field should miss")
	}
}

// smin is a true lower-rounded union: never above the hard min, never more than
// k/4 below it.
func TestSmoothMin(t *testing.T) {
	for _, tc := range []struct{ a, b, k float64 }{{1, 2, 0.5}, {-0.3, 0.4, 0.6}, {5, 5, 1}} {
		got := smin(tc.a, tc.b, tc.k)
		hard := math.Min(tc.a, tc.b)
		if got > hard+1e-9 || got < hard-tc.k/4-1e-9 {
			t.Errorf("smin(%v,%v,%v)=%v out of [%v-%v, %v]", tc.a, tc.b, tc.k, got, hard, tc.k/4, hard)
		}
	}
}

// A Marched object is BVH-boundable (so it accelerates, not just the linear rest).
func TestMarchedBounds(t *testing.T) {
	m, _ := NewMarched("menger", Vec3{X: 1, Y: 2, Z: 3}, 1.5, Material{})
	box, ok := objectBounds(m)
	if !ok {
		t.Fatal("Marched should be boundable")
	}
	if box.min.X > -0.5 || box.max.X < 2.5 {
		t.Errorf("bounds don't enclose the field: %+v", box)
	}
}

// A Marched object renders in a full scene through the path tracer without
// panicking, and the fractal is visible (not all black).
func TestMarchedInScene(t *testing.T) {
	frac, _ := NewMarched("menger", Vec3{X: 0, Y: 1, Z: 5}, 1.2, Material{Color: Vec3{X: 0.8, Y: 0.7, Z: 0.5}})
	frac.MaxSteps = 80
	s := &Scene{
		SkyTop: Vec3{X: 0.5, Y: 0.6, Z: 0.9}, SkyBottom: Vec3{X: 0.9, Y: 0.9, Z: 0.95},
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{X: 0.6, Y: 0.6, Z: 0.6}, C2: Vec3{X: 0.4, Y: 0.4, Z: 0.4}},
			Sphere{Center: Vec3{X: 0, Y: 7, Z: 4}, Radius: 1.2, Mat: Material{Emit: Vec3{X: 20, Y: 20, Z: 20}}},
			frac,
		},
	}
	img := PathRender(s, Camera{Pos: Vec3{X: 0, Y: 1.5, Z: 0}, Pitch: -0.05, FOV: 1.0}, 32, 24,
		PathOptions{Samples: 4, MaxDepth: 3, Seed: 1, NEE: true, MIS: true})
	var sum float64
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			sum += float64(r+g+bl) / 65535
		}
	}
	if sum <= 0 {
		t.Error("scene with a fractal rendered all black")
	}
}
