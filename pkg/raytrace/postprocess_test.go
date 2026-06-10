package raytrace

import (
	"image"
	"image/color"
	"testing"
)

func meanLuma(im image.Image) float64 {
	b := im.Bounds()
	var s float64
	n := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := im.At(x, y).RGBA()
			s += lum(r, g, bl)
			n++
		}
	}
	return s / float64(n)
}

func TestAutoExposureBrightensDimAndDarkensBright(t *testing.T) {
	dim := image.NewRGBA(image.Rect(0, 0, 8, 8))
	bright := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			dim.SetRGBA(x, y, color.RGBA{R: 20, G: 20, B: 20, A: 255})
			bright.SetRGBA(x, y, color.RGBA{R: 250, G: 250, B: 250, A: 255})
		}
	}
	if meanLuma(PostProcess(dim, 0, 0, 0)) <= meanLuma(dim) {
		t.Error("auto-exposure should brighten a dim image")
	}
	if meanLuma(PostProcess(bright, 0, 0, 0)) >= meanLuma(bright) {
		t.Error("auto-exposure should darken an over-bright image")
	}
}

func TestBloomGlowsAroundHighlights(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetRGBA(x, y, color.RGBA{A: 255}) // black
		}
	}
	img.SetRGBA(8, 8, color.RGBA{R: 255, G: 255, B: 255, A: 255}) // one hot pixel
	out := PostProcess(img, 1, 0.3, 1.5)                          // fixed exposure, bloom on
	// a neighbour of the hot pixel was black; bloom must give it some glow.
	r, _, _, _ := out.At(10, 8).RGBA()
	if r>>8 == 0 {
		t.Error("bloom should spread glow to neighbouring pixels")
	}
}
