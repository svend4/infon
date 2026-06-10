package raytrace

import (
	"image"
	"image/color"
	"testing"
)

func gradImg(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: 100, A: 255})
		}
	}
	return img
}

// Pixelate makes each block uniform and preserves dimensions.
func TestPixelate(t *testing.T) {
	img := gradImg(64, 64)
	out := Pixelate(img, 8)
	if out.Bounds().Dx() != 64 || out.Bounds().Dy() != 64 {
		t.Fatalf("dims changed: %v", out.Bounds())
	}
	// two pixels in the same 8x8 block should be identical
	if lumPx(out, 1, 1) != lumPx(out, 6, 6) {
		t.Error("pixels within a block should be the same after pixelation")
	}
	// the block is the average, so it differs from the sharp original in general
	if distinctColors(out) >= distinctColors(img) {
		t.Error("pixelation should reduce the number of distinct colours")
	}
}

// Materialize is coarser early and sharp at the end.
func TestMaterialize(t *testing.T) {
	img := gradImg(64, 64)
	// t=1 -> sharp (block <= 1 -> returned unchanged)
	if Materialize(img, 1) != image.Image(img) {
		t.Error("at t=1 the frame should be sharp (unchanged)")
	}
	// t=0 is coarser than t=0.6: a big block early means more uniform neighbours.
	early := Materialize(img, 0.0)
	late := Materialize(img, 0.6)
	uniformRun := func(im image.Image) bool { // are the first ~16 px of a row equal?
		c0 := lumPx(im, 0, 20)
		for x := 1; x < 16; x++ {
			if lumPx(im, x, 20) != c0 {
				return false
			}
		}
		return true
	}
	if !uniformRun(early) {
		t.Error("at t=0 a wide run of pixels should be one block (coarse)")
	}
	if uniformRun(late) {
		t.Error("at t=0.6 the blocks should be smaller than 16 wide")
	}
}
