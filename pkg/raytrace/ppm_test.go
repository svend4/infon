package raytrace

import (
	"image"
	"testing"
)

// The progressive radius must shrink (and rescale flux by < 1) whenever a pass
// gathered photons, which is what drives PPM's bias to zero.
func TestPPMRadiusShrinks(t *testing.T) {
	r, n := 0.5, 0.0
	prev := r
	for pass := 0; pass < 10; pass++ {
		var ratio float64
		r, n, ratio = ppmRadiusUpdate(r, n, 1000, 0.7)
		if r >= prev {
			t.Errorf("pass %d: radius did not shrink (%.4f >= %.4f)", pass, r, prev)
		}
		if ratio <= 0 || ratio >= 1 {
			t.Errorf("pass %d: flux rescale ratio out of (0,1): %.4f", pass, ratio)
		}
		prev = r
	}
	// With no photons this pass, nothing changes.
	r2, n2, _ := ppmRadiusUpdate(r, n, 0, 0.7)
	if r2 != r || n2 != n {
		t.Errorf("a pass with no photons must leave R and N unchanged")
	}
}

func ppmMeanLum(im image.Image) float64 {
	b := im.Bounds()
	var s float64
	cnt := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := im.At(x, y).RGBA()
			s += 0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(bl>>8)
			cnt++
		}
	}
	return s / float64(cnt)
}

// PPM and the path tracer solve the same rendering equation, so on a diffuse
// global-illumination scene their overall brightness must agree (the residual
// gap is PPM's finite-radius bias). This is the end-to-end correctness check of
// the photon flux normalization.
func TestPPMMatchesPathTracer(t *testing.T) {
	v := func(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }
	scene := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: v(0.7, 0.7, 0.7), C2: v(0.7, 0.7, 0.7)},
			Sphere{Center: v(0, 1, 0), Radius: 1, Mat: Material{Color: v(0.75, 0.4, 0.4)}},
			Sphere{Center: v(0, 5, 0), Radius: 0.6, Mat: Material{Emit: v(12, 12, 12)}},
		},
		SkyTop: v(0.02, 0.02, 0.03), SkyBottom: v(0.02, 0.02, 0.03),
	}
	cam := Camera{Pos: v(0, 2.2, -6), Pitch: -0.18, FOV: 1.0}
	const w, h = 80, 60
	pt := ppmMeanLum(PathRender(scene, cam, w, h, PathOptions{Samples: 250, MaxDepth: 5, Seed: 1, NEE: true, MIS: true}))
	pm := ppmMeanLum(PPMRender(scene, cam, w, h, PPMOptions{Passes: 16, PhotonsPerPass: 80000, Radius: 0.5, Alpha: 0.7, MaxBounce: 6, Seed: 1}))

	if pt < 5 || pm < 5 {
		t.Fatalf("both renders should be non-trivially lit: pt=%.2f ppm=%.2f", pt, pm)
	}
	if ratio := pm / pt; ratio < 0.75 || ratio > 1.3 {
		t.Errorf("PPM should match the path tracer's brightness, ratio PPM/PT=%.3f (pt=%.2f ppm=%.2f)", ratio, pt, pm)
	}
}

// A pixel whose camera ray escapes to the sky carries exactly the sky colour
// (no measurement point, no photons involved).
func TestPPMSkyPassthrough(t *testing.T) {
	scene := &Scene{SkyTop: Vec3{0.3, 0.5, 0.9}, SkyBottom: Vec3{0.3, 0.5, 0.9}}
	cam := Camera{Pos: Vec3{0, 0, 0}, FOV: 1}
	img := PPMRender(scene, cam, 8, 8, PPMOptions{Passes: 2, PhotonsPerPass: 100, Radius: 0.5})
	r, g, b, _ := img.At(4, 4).RGBA()
	// sky is constant; with no geometry every pixel is the sky colour (tone-mapped).
	if r>>8 == 0 && g>>8 == 0 && b>>8 == 0 {
		t.Error("empty scene should render the sky, not black")
	}
	if b>>8 <= r>>8 {
		t.Errorf("blue sky should have blue dominant, got r=%d b=%d", r>>8, b>>8)
	}
}
