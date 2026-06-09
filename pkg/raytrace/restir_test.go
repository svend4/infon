package raytrace

import (
	"math"
	"testing"
)

func meanLinLum(buf []Vec3) float64 {
	var s float64
	for _, v := range buf {
		s += lum3(v)
	}
	return s / float64(len(buf))
}

// Weighted reservoir sampling must select candidates in proportion to weight.
func TestReservoirDIStreamProportional(t *testing.T) {
	rg := newRNG(1)
	const n = 100000
	ones := 0
	for i := 0; i < n; i++ {
		var r reservoirDI
		r.stream(diSample{ei: 0}, 1.0, 1, rg)
		r.stream(diSample{ei: 1}, 3.0, 1, rg)
		if r.y.ei == 1 {
			ones++
		}
	}
	if frac := float64(ones) / float64(n); math.Abs(frac-0.75) > 0.01 {
		t.Errorf("weight-3 vs weight-1 chosen %.3f of the time, want 0.75", frac)
	}
}

func restirTestScene() (*Scene, Camera, int, int) {
	v := func(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }
	s := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: v(0.7, 0.7, 0.7), C2: v(0.7, 0.7, 0.7)},
			Sphere{Center: v(0, 1, 0), Radius: 1, Mat: Material{Color: v(0.8, 0.5, 0.5)}},
			Sphere{Center: v(-3, 3, 1), Radius: 0.3, Mat: Material{Emit: v(40, 40, 40)}},
			Sphere{Center: v(3, 4, -1), Radius: 0.3, Mat: Material{Emit: v(50, 50, 50)}},
			Sphere{Center: v(0, 5, 2), Radius: 0.3, Mat: Material{Emit: v(30, 30, 30)}},
		},
		SkyTop: v(0.02, 0.02, 0.03), SkyBottom: v(0.02, 0.02, 0.03),
	}
	return s, Camera{Pos: v(0, 2.5, -6), Pitch: -0.25, FOV: 1.0}, 64, 48
}

// ReSTIR (including spatial reuse, with the Z bias correction) must be unbiased:
// averaged over many frames in linear space it matches a brute-force direct-only
// reference. (Tone mapping would otherwise hide this behind noise.)
func TestReSTIRUnbiasedLinear(t *testing.T) {
	s, cam, w, h := restirTestScene()
	const refSpp = 250
	refSum := pathSum(s, cam, w, h, PathOptions{Samples: refSpp, MaxDepth: 1, Seed: 9, NEE: true})
	refMean := meanLinLum(refSum) / float64(refSpp)

	acc := make([]Vec3, w*h)
	const frames = 48
	for k := 0; k < frames; k++ {
		buf := s.restirLinear(cam, w, h, ReSTIROptions{Candidates: 8, SpatialPasses: 2, SpatialK: 5, Radius: 14, Seed: uint64(k) + 1})
		for i := range acc {
			acc[i] = acc[i].Add(buf[i])
		}
	}
	for i := range acc {
		acc[i] = acc[i].Scale(1 / float64(frames))
	}
	reMean := meanLinLum(acc)
	if ratio := reMean / refMean; ratio < 0.95 || ratio > 1.05 {
		t.Errorf("ReSTIR should be unbiased in linear space: ratio=%.3f (restir=%.3f ref=%.3f)", ratio, reMean, refMean)
	}
}

// Spatial reuse must lower error: at the same small candidate budget, reusing
// neighbours gets closer to the converged reference than initial RIS alone.
func TestReSTIRReuseReducesError(t *testing.T) {
	s, cam, w, h := restirTestScene()
	ref := PathRender(s, cam, w, h, PathOptions{Samples: 400, MaxDepth: 1, Seed: 9, NEE: true})
	noReuse := ReSTIRRender(s, cam, w, h, ReSTIROptions{Candidates: 4, SpatialPasses: 0, Seed: 1})
	reuse := ReSTIRRender(s, cam, w, h, ReSTIROptions{Candidates: 4, SpatialPasses: 2, SpatialK: 5, Radius: 16, Seed: 1})
	if rmse(reuse, ref) >= rmse(noReuse, ref) {
		t.Errorf("spatial reuse should reduce error: no-reuse=%.2f reuse=%.2f", rmse(noReuse, ref), rmse(reuse, ref))
	}
}
