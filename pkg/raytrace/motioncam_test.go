package raytrace

import "testing"

func TestLerpCamera(t *testing.T) {
	a := Camera{Pos: Vec3{0, 0, 0}, Yaw: 0, Pitch: 0, FOV: 1, Aperture: 0, Focus: 0}
	b := Camera{Pos: Vec3{2, 4, 6}, Yaw: 2, Pitch: 1, FOV: 3, Aperture: 0.5, Focus: 10}
	if got := lerpCamera(a, b, 0); got != a {
		t.Errorf("t=0 -> %+v, want %+v", got, a)
	}
	if got := lerpCamera(a, b, 1); got != b {
		t.Errorf("t=1 -> %+v, want %+v", got, b)
	}
	m := lerpCamera(a, b, 0.5)
	if m.Pos.Sub(Vec3{1, 2, 3}).Len() > 1e-9 || m.Yaw != 1 || m.Pitch != 0.5 || m.FOV != 2 || m.Aperture != 0.25 || m.Focus != 5 {
		t.Errorf("midpoint = %+v", m)
	}
}

// A camera trucking sideways over the shutter must smear a static object across a
// wider band than the same (midpoint) camera held still.
func TestCameraMotionBlurWidensCoverage(t *testing.T) {
	scene := &Scene{Objects: []Object{
		Sphere{Center: Vec3{0, 0, 4}, Radius: 0.5, Mat: Material{Emit: Vec3{5, 5, 5}}},
	}}
	opt := PathOptions{Samples: 120, MaxDepth: 2, Seed: 1}
	mid := Camera{Pos: Vec3{0, 0, -2}}
	open := Camera{Pos: Vec3{-2.5, 0, -2}}
	clos := Camera{Pos: Vec3{2.5, 0, -2}}
	st := litSpan(PathRender(scene, mid, 64, 32, opt))
	mv := litSpan(PathRenderMotion(scene, open, clos, 64, 32, opt))
	if st == 0 {
		t.Fatal("static camera should see the emissive sphere")
	}
	if mv <= 2*st {
		t.Errorf("camera motion blur should smear coverage wider: moving=%d static=%d", mv, st)
	}
}

// open == close must reproduce the static render (same seed -> same pixels).
func TestCameraMotionBlurNoMotionMatchesStatic(t *testing.T) {
	scene := &Scene{Objects: []Object{
		Sphere{Center: Vec3{0, 0, 4}, Radius: 1, Mat: Material{Emit: Vec3{4, 4, 4}}},
	}}
	cam := Camera{Pos: Vec3{0, 0, -2}}
	opt := PathOptions{Samples: 16, MaxDepth: 2, Seed: 7}
	a := litSpan(PathRender(scene, cam, 48, 32, opt))
	b := litSpan(PathRenderMotion(scene, cam, cam, 48, 32, opt))
	if a != b {
		t.Errorf("open==close should match the static render: static span=%d motion span=%d", a, b)
	}
}
