package raytrace

import "testing"

// Primary samples are always valid uniforms in [0,1), large step or small.
func TestPSSStateRange(t *testing.T) {
	p := &pssState{large: true, sigma: 0.01, src: 1}
	for i := 0; i < 2000; i++ {
		if v := p.next(); v < 0 || v >= 1 {
			t.Fatalf("large-step primary sample out of [0,1): %v", v)
		}
	}
	base := append([]float64(nil), p.u...)
	q := &pssState{base: base, sigma: 0.01, src: 2}
	for i := 0; i < len(base); i++ {
		if v := q.next(); v < 0 || v >= 1 {
			t.Fatalf("small-step primary sample out of [0,1): %v", v)
		}
	}
}

func mltScene() (*Scene, Camera, int, int) {
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

// MLT's stationary distribution reproduces the path tracer: their tone-mapped
// images agree in overall brightness (a wrong bootstrap b or splat would not).
func TestMLTMatchesPathTracer(t *testing.T) {
	s, cam, w, h := mltScene()
	pt := ppmMeanLum(PathRender(s, cam, w, h, PathOptions{Samples: 400, MaxDepth: 5, Seed: 9, NEE: true, MIS: true}))
	ml := ppmMeanLum(MLTRender(s, cam, w, h, MLTOptions{Mutations: 1000000, Bootstrap: 8000, MaxDepth: 5, NEE: true, MIS: true, Seed: 1}))
	if pt < 5 || ml < 5 {
		t.Fatalf("both should be lit: pt=%.2f mlt=%.2f", pt, ml)
	}
	if ratio := ml / pt; ratio < 0.9 || ratio > 1.1 {
		t.Errorf("MLT should match the path tracer, ratio MLT/PT=%.3f (pt=%.2f mlt=%.2f)", ratio, pt, ml)
	}
}

// Rigorous linear-space check (no tone-map confound) of MLT's normalisation.
func TestMLTUnbiasedLinear(t *testing.T) {
	s, cam, w, h := mltScene()
	const spp = 400
	ref := meanLinLum(pathSum(s, cam, w, h, PathOptions{Samples: spp, MaxDepth: 5, Seed: 9, NEE: true, MIS: true})) / float64(spp)
	ml := meanLinLum(s.mltLinear(cam, w, h, MLTOptions{Mutations: 1200000, Bootstrap: 8000, MaxDepth: 5, NEE: true, MIS: true, Seed: 1}))
	if ratio := ml / ref; ratio < 0.9 || ratio > 1.1 {
		t.Errorf("MLT linear mean should match the path tracer: ratio=%.3f (mlt=%.4f ref=%.4f)", ratio, ml, ref)
	}
}
