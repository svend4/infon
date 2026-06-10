package raytrace

import (
	"image"
	"math"
	"testing"
)

// The first 16 samples of the Owen-scrambled (0,2)-sequence must fall exactly one
// per cell of the 4x4 dyadic grid — the defining stratification property, which
// Owen scrambling preserves.
func Test02SequenceStratified(t *testing.T) {
	rg := newRNG(1)
	rg.qmc = true
	rg.qseed = 0x0bADc0de
	var seen [16]bool
	for i := 0; i < 16; i++ {
		rg.qidx = uint32(i)
		u, v := rg.qmc2(0)
		cx, cy := int(u*4), int(v*4)
		cell := cy*4 + cx
		if seen[cell] {
			t.Fatalf("cell %d (%.3f,%.3f) sampled twice: not a (0,2)-sequence", cell, u, v)
		}
		seen[cell] = true
	}
	for c, ok := range seen {
		if !ok {
			t.Errorf("4x4 cell %d never sampled", c)
		}
	}
}

func rmse(a, b image.Image) float64 {
	bnd := a.Bounds()
	var sum float64
	n := 0
	for y := bnd.Min.Y; y < bnd.Max.Y; y++ {
		for x := bnd.Min.X; x < bnd.Max.X; x++ {
			ar, ag, ab, _ := a.At(x, y).RGBA()
			br, bg, bb, _ := b.At(x, y).RGBA()
			dr := float64(int(ar>>8) - int(br>>8))
			dg := float64(int(ag>>8) - int(bg>>8))
			db := float64(int(ab>>8) - int(bb>>8))
			sum += dr*dr + dg*dg + db*db
			n += 3
		}
	}
	return math.Sqrt(sum / float64(n))
}

// At equal (low) spp, Sobol-stratified anti-aliasing must be closer to the
// converged reference than pure random sampling: the silhouette of an emissive
// sphere on black is pure AA error, which stratification reduces.
func TestSobolReducesAANoise(t *testing.T) {
	scene := &Scene{Objects: []Object{
		Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Emit: Vec3{4, 4, 4}}},
	}}
	cam := Camera{Pos: Vec3{0, 0, -4}, FOV: 1}
	const w, h = 80, 80
	ref := PathRender(scene, cam, w, h, PathOptions{Samples: 400, MaxDepth: 1, Seed: 99})

	low := PathOptions{Samples: 4, MaxDepth: 1, Seed: 7}
	random := rmse(PathRender(scene, cam, w, h, low), ref)
	low.Sobol = true
	sobol := rmse(PathRender(scene, cam, w, h, low), ref)

	if sobol >= random {
		t.Errorf("Sobol AA should beat random at equal spp: sobol=%.3f random=%.3f", sobol, random)
	}
}

// Sobol is unbiased: at high spp it converges to the same image as random
// sampling (same scene), so their difference is small.
func TestSobolUnbiased(t *testing.T) {
	scene := &Scene{Objects: []Object{
		Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Emit: Vec3{4, 4, 4}}},
	}}
	cam := Camera{Pos: Vec3{0, 0, -4}, FOV: 1}
	const w, h = 64, 64
	random := PathRender(scene, cam, w, h, PathOptions{Samples: 300, MaxDepth: 1, Seed: 1})
	sobol := PathRender(scene, cam, w, h, PathOptions{Samples: 300, MaxDepth: 1, Seed: 1, Sobol: true})
	if d := rmse(random, sobol); d > 4 {
		t.Errorf("Sobol and random should converge to the same image, RMSE=%.3f", d)
	}
}
