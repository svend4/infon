package raytrace

import (
	"image"
	"image/color"
	"testing"
)

func solidGray(w, h int, v float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	g := u8(v)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: g, G: g, B: g, A: 255})
		}
	}
	return img
}

// Grain adds variance to a flat image; with grain off it stays flat.
func TestDreamGrain(t *testing.T) {
	flat := solidGray(48, 48, 0.5)
	none := Dream(flat, DreamOptions{Seed: 1})
	if v := lumVariance(none); v > 1e-4 {
		t.Errorf("with no grain a flat image should stay flat, variance %.5f", v)
	}
	grainy := Dream(flat, DreamOptions{Grain: 0.06, Seed: 1})
	if v := lumVariance(grainy); v < 1e-4 {
		t.Errorf("grain should add variance, got %.6f", v)
	}
}

// The vignette darkens the corners relative to the centre.
func TestDreamVignette(t *testing.T) {
	flat := solidGray(64, 64, 0.9)
	out := Dream(flat, DreamOptions{Vignette: 0.6})
	centre := lumPx(out, 32, 32)
	corner := lumPx(out, 1, 1)
	if corner >= centre {
		t.Errorf("vignette should darken corners: centre %.3f corner %.3f", centre, corner)
	}
}

// Chromatic aberration separates the colour channels — a grey edge gains colour
// fringing that wasn't there before.
func TestDreamChroma(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			v := u8(0.0)
			if x >= 52 && x <= 60 { // a grey stripe (R==G==B) off-centre, where chroma bites
				v = u8(1.0)
			}
			img.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	out := Dream(img, DreamOptions{Chroma: 0.06})
	maxSplit := 0.0
	b := out.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, _, bl, _ := out.At(x, y).RGBA()
			if s := abs(float64(r)-float64(bl)) / 65535; s > maxSplit {
				maxSplit = s
			}
		}
	}
	if maxSplit < 0.1 {
		t.Errorf("chroma should split R and B at edges, max split %.3f", maxSplit)
	}
}

// Grain is deterministic in the seed: same seed identical, different seed differs.
func TestDreamDeterministic(t *testing.T) {
	flat := solidGray(32, 32, 0.5)
	a := Dream(flat, DreamOptions{Grain: 0.08, Seed: 5})
	b := Dream(flat, DreamOptions{Grain: 0.08, Seed: 5})
	if !sameImage(a, b) {
		t.Error("same seed should give identical grain")
	}
	c := Dream(flat, DreamOptions{Grain: 0.08, Seed: 6})
	if sameImage(a, c) {
		t.Error("a different seed should give different grain")
	}
}

func TestDreamDims(t *testing.T) {
	out := Dream(solidGray(40, 24, 0.5), DreamOptions{Chroma: 0.01, Grain: 0.03, Vignette: 0.2, Distort: 0.1})
	if out.Bounds().Dx() != 40 || out.Bounds().Dy() != 24 {
		t.Errorf("Dream changed dimensions to %v", out.Bounds())
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func sameImage(a, b image.Image) bool {
	ab, bb := a.Bounds(), b.Bounds()
	if ab != bb {
		return false
	}
	for y := ab.Min.Y; y < ab.Max.Y; y++ {
		for x := ab.Min.X; x < ab.Max.X; x++ {
			r1, g1, b1, _ := a.At(x, y).RGBA()
			r2, g2, b2, _ := b.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 {
				return false
			}
		}
	}
	return true
}
