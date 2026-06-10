package raytrace

import (
	"math"
	"testing"
)

// Apply/ApplyDir map points and directions through an affine transform.
func TestTransformApply(t *testing.T) {
	xf := Translate(Vec3{X: 10, Y: 0, Z: 0})
	if got := xf.Apply(Vec3{X: 1, Y: 2, Z: 3}); !vecClose(got, Vec3{X: 11, Y: 2, Z: 3}, 1e-9) {
		t.Errorf("translate point: %+v", got)
	}
	if got := xf.ApplyDir(Vec3{X: 1, Y: 0, Z: 0}); !vecClose(got, Vec3{X: 1, Y: 0, Z: 0}, 1e-9) {
		t.Errorf("translate should not move a direction: %+v", got)
	}
	rot := RotateY(math.Pi / 2) // +Z -> +X
	if got := rot.ApplyDir(Vec3{X: 0, Y: 0, Z: 1}); !vecClose(got, Vec3{X: 1, Y: 0, Z: 0}, 1e-9) {
		t.Errorf("rotateY(90) should turn +Z into +X, got %+v", got)
	}
}

// The portal quad is hit inside its rectangle and missed outside it, and reports a
// teleport link.
func TestPortalIntersect(t *testing.T) {
	p := NewPortal(Vec3{X: 0, Y: 1, Z: 5}, Vec3{X: 3, Y: 0, Z: 0}, Vec3{X: 0, Y: 3, Z: 0}, Translate(Vec3{X: 100}))
	hit, ok := p.Intersect(Ray{Origin: Vec3{Y: 1}, Dir: Vec3{Z: 1}}, 1e-4, 1e9)
	if !ok {
		t.Fatal("a ray through the middle of the portal should hit it")
	}
	if hit.Mat.Link == nil {
		t.Error("a portal hit should carry a teleport link")
	}
	if _, ok := p.Intersect(Ray{Origin: Vec3{X: 10, Y: 1}, Dir: Vec3{Z: 1}}, 1e-4, 1e9); ok {
		t.Error("a ray outside the rectangle should miss the portal")
	}
	if _, ok := p.Intersect(Ray{Origin: Vec3{Y: 1}, Dir: Vec3{X: 1}}, 1e-4, 1e9); ok {
		t.Error("a ray parallel to the portal plane should miss")
	}
}

// A portal reveals a place the straight ray could never reach.
func TestPortalRevealsLinkedSpace(t *testing.T) {
	red := Sphere{Center: Vec3{X: 100, Y: 1, Z: 40}, Radius: 2, Mat: Material{Emit: Vec3{X: 5}}}
	sky := Vec3{X: 0, Y: 0, Z: 0.3}

	// Without a portal the forward ray misses the off-axis sphere and sees sky.
	plain := &Scene{Objects: []Object{red}, SkyTop: sky, SkyBottom: sky}
	plain.BuildBVH()
	got := plain.radiance(Ray{Origin: Vec3{Y: 1}, Dir: Vec3{Z: 1}}, 8, newRNG(1), 0)
	if got.X > got.Z {
		t.Fatalf("without a portal the ray should see sky, not the sphere: %+v", got)
	}

	// A portal linking +100 in X brings the sphere into view straight ahead.
	portal := NewPortal(Vec3{X: 0, Y: 1, Z: 5}, Vec3{X: 3, Y: 0, Z: 0}, Vec3{X: 0, Y: 3, Z: 0}, Translate(Vec3{X: 100}))
	withP := &Scene{Objects: []Object{red, portal}, SkyTop: sky, SkyBottom: sky}
	withP.BuildBVH()
	got = withP.radiance(Ray{Origin: Vec3{Y: 1}, Dir: Vec3{Z: 1}}, 8, newRNG(1), 0)
	if got.X < 1 || got.X <= got.Z {
		t.Errorf("through the portal the ray should reach the red sphere: %+v", got)
	}
}

// An identity portal is invisible: a scene renders the same with or without it.
func TestPortalSeamless(t *testing.T) {
	red := Sphere{Center: Vec3{Y: 1, Z: 30}, Radius: 3, Mat: Material{Emit: Vec3{X: 5}}}
	sky := Vec3{Z: 0.3}
	ray := Ray{Origin: Vec3{Y: 1}, Dir: Vec3{Z: 1}}

	plain := &Scene{Objects: []Object{red}, SkyTop: sky, SkyBottom: sky}
	plain.BuildBVH()
	a := plain.radiance(ray, 8, newRNG(7), 0)

	portal := NewPortal(Vec3{Y: 1, Z: 5}, Vec3{X: 4}, Vec3{Y: 4}, Translate(Vec3{}))
	withP := &Scene{Objects: []Object{red, portal}, SkyTop: sky, SkyBottom: sky}
	withP.BuildBVH()
	b := withP.radiance(ray, 8, newRNG(7), 0)

	if !vecClose(a, b, 1e-6) {
		t.Errorf("an identity portal should be invisible: %+v vs %+v", a, b)
	}
}
