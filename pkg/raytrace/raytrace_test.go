package raytrace

import (
	"math"
	"testing"
)

func lum(r, g, b uint32) float64 { return 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b) }

func TestVecBasics(t *testing.T) {
	a := Vec3{1, 2, 3}
	b := Vec3{4, 5, 6}
	if got := a.Dot(b); got != 32 {
		t.Errorf("dot = %v, want 32", got)
	}
	if got := (Vec3{1, 0, 0}).Cross(Vec3{0, 1, 0}); got != (Vec3{0, 0, 1}) {
		t.Errorf("x cross y = %v, want (0,0,1)", got)
	}
	if l := (Vec3{3, 4, 0}).Len(); math.Abs(l-5) > 1e-12 {
		t.Errorf("len = %v, want 5", l)
	}
	if n := (Vec3{0, 5, 0}).Norm(); math.Abs(n.Len()-1) > 1e-12 {
		t.Errorf("norm len = %v, want 1", n.Len())
	}
	// reflecting straight down off a floor normal points straight up.
	if r := (Vec3{0, -1, 0}).Reflect(Vec3{0, 1, 0}); r != (Vec3{0, 1, 0}) {
		t.Errorf("reflect = %v, want (0,1,0)", r)
	}
}

func TestSphereIntersect(t *testing.T) {
	s := Sphere{Center: Vec3{0, 0, 5}, Radius: 1}
	r := Ray{Origin: Vec3{0, 0, 0}, Dir: Vec3{0, 0, 1}}
	h, ok := s.Intersect(r, 1e-4, 1e9)
	if !ok {
		t.Fatal("expected hit")
	}
	if math.Abs(h.T-4) > 1e-9 {
		t.Errorf("t = %v, want 4", h.T)
	}
	if h.N.Dot(r.Dir) >= 0 {
		t.Errorf("normal %v must face the ray", h.N)
	}
	if _, ok := (Sphere{Center: Vec3{10, 0, 5}, Radius: 1}).Intersect(r, 1e-4, 1e9); ok {
		t.Error("expected miss for off-axis sphere")
	}
}

func TestTriangleMollerTrumbore(t *testing.T) {
	tr := Triangle{A: Vec3{-1, -1, 5}, B: Vec3{1, -1, 5}, C: Vec3{0, 1, 5}}
	if _, ok := tr.Intersect(Ray{Origin: Vec3{0, 0, 0}, Dir: Vec3{0, 0, 1}}, 1e-4, 1e9); !ok {
		t.Error("ray through the triangle should hit")
	}
	if _, ok := tr.Intersect(Ray{Origin: Vec3{5, 5, 0}, Dir: Vec3{0, 0, 1}}, 1e-4, 1e9); ok {
		t.Error("ray outside the triangle should miss")
	}
	// parallel ray misses
	if _, ok := tr.Intersect(Ray{Origin: Vec3{0, 0, 0}, Dir: Vec3{0, 1, 0}}, 1e-4, 1e9); ok {
		t.Error("parallel ray should miss")
	}
}

func TestRenderGeometryAndShading(t *testing.T) {
	scene := &Scene{
		Objects:   []Object{Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{1, 0, 0}}}},
		Light:     Vec3{5, 5, 5},
		LightInt:  1.2,
		Ambient:   0.15,
		SkyTop:    Vec3{0.40, 0.60, 0.95},
		SkyBottom: Vec3{0.90, 0.95, 1.0},
	}
	cam := Camera{Pos: Vec3{0, 0, 5}, Yaw: math.Pi, FOV: math.Pi / 3}
	const W, H = 80, 80
	img := Render(scene, cam, W, H, Options{Samples: 1})

	px := func(x, y int) (rr, gg, bb uint32) { r, g, b, _ := img.At(x, y).RGBA(); return r, g, b }

	// Centre ray hits the red sphere: R dominates.
	cr, _, cb := px(W/2, H/2)
	if cr <= cb {
		t.Errorf("centre should be the red sphere: R=%d B=%d", cr>>8, cb>>8)
	}
	// A top-left corner ray escapes to the blue sky: B dominates.
	sr, _, sb := px(2, 2)
	if sb <= sr {
		t.Errorf("corner should be blue sky: R=%d B=%d", sr>>8, sb>>8)
	}
	// The side of the sphere toward the light is brighter than the far side.
	urR, urG, urB := px(W/2+W/10, H/2-H/10) // up-right, toward the light
	llR, llG, llB := px(W/2-W/10, H/2+H/10) // down-left, away from the light
	if lum(urR, urG, urB) <= lum(llR, llG, llB) {
		t.Errorf("lit side (%.0f) should be brighter than shadowed side (%.0f)",
			lum(urR, urG, urB), lum(llR, llG, llB))
	}
}

func TestHardShadow(t *testing.T) {
	// A uniform white floor, an occluder sphere above the origin, light overhead.
	scene := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{1, 1, 1}, C2: Vec3{1, 1, 1}},
			Sphere{Center: Vec3{0, 2, 0}, Radius: 1, Mat: Material{Color: Vec3{1, 1, 1}}},
		},
		Light:    Vec3{0, 10, 0},
		LightInt: 1,
		Ambient:  0.1,
	}
	// Floor point directly under the occluder (reached from the side, not through it).
	under := scene.Trace(Ray{Origin: Vec3{3, 3, 0}, Dir: Vec3{-1, -1, 0}.Norm()})
	// Floor point far away, in full light.
	far := scene.Trace(Ray{Origin: Vec3{6, 5, 0}, Dir: Vec3{0, -1, 0}})

	sum := func(v Vec3) float64 { return v.X + v.Y + v.Z }
	if sum(under) >= sum(far) {
		t.Errorf("shadowed floor (%.3f) should be darker than lit floor (%.3f)", sum(under), sum(far))
	}
	// Shadowed point should be ~ambient only.
	if sum(under)/3 > scene.Ambient+0.05 {
		t.Errorf("shadowed point too bright: %.3f (ambient %.2f)", sum(under)/3, scene.Ambient)
	}
}
