package raytrace

import "testing"

func bdptScene() (*Scene, Camera, int, int) {
	v := func(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }
	s := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: v(0.7, 0.7, 0.7), C2: v(0.7, 0.7, 0.7)},
			Sphere{Center: v(0, 1, 0), Radius: 1, Mat: Material{Color: v(0.7, 0.5, 0.5)}},
			Sphere{Center: v(0, 6, 0), Radius: 0.7, Mat: Material{Emit: v(20, 20, 20)}},
		},
		SkyTop: v(0, 0, 0), SkyBottom: v(0, 0, 0),
	}
	return s, Camera{Pos: v(0, 2.5, -6), Pitch: -0.25, FOV: 1.0}, 64, 48
}

// The eye subpath starts at the camera; the light subpath starts on an emitter.
func TestBDPTSubpathRoots(t *testing.T) {
	s, cam, _, _ := bdptScene()
	s.gatherEmitters()
	b := cam.basis(64, 48)
	rg := newRNG(1)
	ray := b.ray(32, 24)
	eye := s.generateEyeSubpath(cam.Pos, ray, 1, 6, rg)
	if len(eye) < 2 || !eye[0].camera {
		t.Fatalf("eye subpath should start at the camera, got %d verts", len(eye))
	}
	lt := s.generateLightSubpath(5, rg)
	if len(lt) < 1 || !lt[0].light {
		t.Fatalf("light subpath should start on a light, got %d verts", len(lt))
	}
}

// BDPT and the path tracer solve the same equation, so their tone-mapped images
// agree in overall brightness (a wrong MIS weight would break the partition of
// unity and show up as a wrong mean).
func TestBDPTMatchesPathTracer(t *testing.T) {
	s, cam, w, h := bdptScene()
	pt := ppmMeanLum(PathRender(s, cam, w, h, PathOptions{Samples: 600, MaxDepth: 5, Seed: 9, NEE: true, MIS: true}))
	bd := ppmMeanLum(BDPTRender(s, cam, w, h, BDPTOptions{Samples: 64, MaxDepth: 5, Seed: 1}))
	if pt < 5 || bd < 5 {
		t.Fatalf("both should be lit: pt=%.2f bd=%.2f", pt, bd)
	}
	if ratio := bd / pt; ratio < 0.92 || ratio > 1.08 {
		t.Errorf("BDPT should match the path tracer, ratio BDPT/PT=%.3f (pt=%.2f bd=%.2f)", ratio, pt, bd)
	}
}

// Rigorous unbiasedness in linear space (no tone-map confound): the BDPT mean
// matches the path tracer mean radiance.
func TestBDPTUnbiasedLinear(t *testing.T) {
	s, cam, w, h := bdptScene()
	const spp = 600
	ref := meanLinLum(pathSum(s, cam, w, h, PathOptions{Samples: spp, MaxDepth: 5, Seed: 9, NEE: true, MIS: true})) / float64(spp)
	bd := meanLinLum(s.bdptLinear(cam, w, h, BDPTOptions{Samples: 96, MaxDepth: 5, Seed: 1}))
	if ratio := bd / ref; ratio < 0.94 || ratio > 1.06 {
		t.Errorf("BDPT linear mean should match the path tracer: ratio=%.3f (bd=%.4f ref=%.4f)", ratio, bd, ref)
	}
}
