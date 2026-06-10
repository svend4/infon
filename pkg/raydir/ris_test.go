package raydir

import (
	"image"
	"math"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// meanLumLin is the mean (roughly linear) luminance of an image.
func meanLumLin(img image.Image) float64 {
	b := img.Bounds()
	var s float64
	n := 0.0
	dec := func(c uint32) float64 { v := float64(c) / 65535; return v * v }
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			s += 0.2126*dec(r) + 0.7152*dec(g) + 0.0722*dec(bl)
			n++
		}
	}
	return s / n
}

// SetRIS propagates to the built scene.
func TestWorldRISPropagates(t *testing.T) {
	w := NewWorld()
	if w.SceneWith(nil).RISCandidates != 0 {
		t.Error("a fresh world should have no RIS")
	}
	w.SetRIS(12)
	if got := w.SceneWith(nil).RISCandidates; got != 12 {
		t.Errorf("SetRIS(12) should set the scene's RISCandidates, got %d", got)
	}
}

// manyLights builds a world lit by several small emitters above the floor.
func manyLights(ris int) *World {
	w := NewWorld()
	w.SetRIS(ris)
	for i := 0; i < 8; i++ {
		a := float64(i) / 8 * 2 * math.Pi
		w.AddDecor(raytrace.Sphere{
			Center: raytrace.Vec3{X: math.Cos(a) * 4, Y: 3, Z: math.Sin(a)*4 + 4},
			Radius: 0.28,
			Mat:    raytrace.Material{Emit: raytrace.Vec3{X: 7, Y: 7, Z: 7}},
		})
	}
	return w
}

// ReSTIR/RIS is unbiased: at equal samples its mean image matches plain NEE's.
func TestWorldRISMatchesNEE(t *testing.T) {
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 5, Z: -6}, Yaw: 0, Pitch: -0.5, FOV: 1}
	opt := raytrace.PathOptions{Samples: 80, MaxDepth: 4, Seed: 1, NEE: true, MIS: false, Sobol: true}

	nee := raytrace.PathRender(manyLights(0).SceneWith(nil), cam, 64, 64, opt)
	rs := raytrace.PathRender(manyLights(16).SceneWith(nil), cam, 64, 64, opt)

	a, b := meanLumLin(nee), meanLumLin(rs)
	if a < 1e-4 {
		t.Fatalf("the test scene should be lit, plain NEE mean %.5f", a)
	}
	if rel := math.Abs(a-b) / a; rel > 0.06 {
		t.Errorf("RIS should match plain NEE in the mean (unbiased): NEE %.4f RIS %.4f rel %.3f", a, b, rel)
	}
}
