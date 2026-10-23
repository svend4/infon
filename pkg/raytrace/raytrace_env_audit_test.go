package raytrace

import (
	"fmt"
	"image"
	"math"
	"testing"
)

// noPanic runs f and reports a panic as an error instead of crashing the test
// binary.
func noPanic(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	f()
	return nil
}

// An env image with no pixels has nothing to importance-sample: BuildEnvSampler
// must return nil (callers treat a nil Env as "no env sampling"), as
// BuildEnvSamplerFromSky already does for a degenerate size. Before the fix it
// built a sampler whose Sample divided by zero / indexed pix[-1] and whose Pdf
// indexed cum[-1], panicking inside a render worker goroutine.
func TestAuditEnvSamplerEmptyImage(t *testing.T) {
	for _, r := range []image.Rectangle{
		image.Rect(0, 0, 0, 0),
		image.Rect(0, 0, 5, 0),
		image.Rect(0, 0, 0, 5),
	} {
		var e *EnvSampler
		if err := noPanic(func() { e = BuildEnvSampler(image.NewRGBA(r)) }); err != nil {
			t.Fatalf("%v: BuildEnvSampler: %v", r, err)
		}
		if e != nil {
			err := noPanic(func() {
				_, _, pdf := e.Sample(newRNG(1))
				_ = e.Pdf(Vec3{X: 0, Y: 1, Z: 0}) + pdf
			})
			t.Errorf("%v: BuildEnvSampler returned a non-nil sampler for an empty image (Sample/Pdf: %v)", r, err)
		}
	}
}

// The zero EnvSampler (all a caller outside the package can build) must sample
// nothing rather than panic, and a path-traced scene using it must stay finite.
func TestAuditEnvSamplerZeroValue(t *testing.T) {
	var e EnvSampler
	if err := noPanic(func() {
		if _, _, pdf := e.Sample(newRNG(1)); pdf != 0 {
			t.Errorf("zero sampler Sample pdf = %v, want 0", pdf)
		}
		if pdf := e.Pdf(Vec3{X: 0, Y: 1, Z: 0}); pdf != 0 {
			t.Errorf("zero sampler Pdf = %v, want 0", pdf)
		}
	}); err != nil {
		t.Fatalf("zero sampler: %v", err)
	}
	if t.Failed() {
		return
	}
	gray := Vec3{X: 0.6, Y: 0.6, Z: 0.6}
	scene := &Scene{
		Objects:   []Object{Plane{Y: 0, Size: 1, C1: gray, C2: gray}},
		SkyTop:    Vec3{X: 0.5, Y: 0.6, Z: 0.9},
		SkyBottom: Vec3{X: 0.9, Y: 0.9, Z: 0.9},
		Env:       &e,
		Medium:    &Medium{Center: Vec3{X: 0, Y: 1, Z: 0}, Radius: 1, SigmaMax: 1, Albedo: Vec3{X: 0.8, Y: 0.8, Z: 0.8}},
	}
	cam := Camera{Pos: Vec3{X: 0, Y: 2.5, Z: 4}, Yaw: math.Pi, Pitch: -0.5, FOV: 0.9}
	buf := pathSum(scene, cam, 16, 12, PathOptions{Samples: 8, MaxDepth: 4, Seed: 1, NEE: true, MIS: true})
	for i, v := range buf {
		if math.IsNaN(v.X+v.Y+v.Z) || math.IsInf(v.X+v.Y+v.Z, 0) {
			t.Fatalf("pixel %d is not finite: %+v", i, v)
		}
	}
	if meanLinLum(buf) <= 0 {
		t.Error("sky light lost with a zero env sampler: the BSDF side must count it in full")
	}
}
