package raytrace

import (
	"image"
	"image/color"
	"math/rand"
	"testing"
)

func TestDenoiseReducesNoiseAndKeepsEdge(t *testing.T) {
	const w, h = 40, 40
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	rng := rand.New(rand.NewSource(3))
	base := func(x int) float64 {
		if x < w/2 {
			return 0.2
		}
		return 0.8
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := clamp01(base(x) + (rng.Float64()-0.5)*0.3) // noisy step edge
			c := uint8(v*255 + 0.5)
			img.SetRGBA(x, y, color.RGBA{R: c, G: c, B: c, A: 255})
		}
	}
	den := Denoise(img, 3, 0.2)

	variance := func(im image.Image) float64 {
		var s, s2 float64
		n := 0
		for y := 5; y < h-5; y++ {
			for x := w/2 + 6; x < w-3; x++ { // flat bright region
				r, _, _, _ := im.At(x, y).RGBA()
				v := float64(r >> 8)
				s += v
				s2 += v * v
				n++
			}
		}
		m := s / float64(n)
		return s2/float64(n) - m*m
	}
	if variance(den) >= variance(img) {
		t.Errorf("denoise should reduce variance: before=%.1f after=%.1f", variance(img), variance(den))
	}

	mean := func(im image.Image, x0, x1 int) float64 {
		var s float64
		n := 0
		for y := 5; y < h-5; y++ {
			for x := x0; x < x1; x++ {
				r, _, _, _ := im.At(x, y).RGBA()
				s += float64(r >> 8)
				n++
			}
		}
		return s / float64(n)
	}
	if c := mean(den, w-6, w) - mean(den, 0, 6); c < 100 { // ideal ~153; à-trous keeps most
		t.Errorf("edge contrast not preserved: %.1f", c)
	}
}
