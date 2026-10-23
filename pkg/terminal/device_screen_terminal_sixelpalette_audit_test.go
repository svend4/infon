package terminal

import (
	"image"
	gocolor "image/color"
	"strings"
	"testing"
)

func rainbowImg(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, gocolor.RGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: uint8((x + y) * 4), A: 255})
		}
	}
	return img
}

// EncodeSixel's maxColors caps the palette. Uniform 2-bit bucketing yields up
// to 64 buckets whatever the cap, so small caps used to be ignored.
func TestEncodeSixelHonoursMaxColors(t *testing.T) {
	img := rainbowImg(64, 64)
	for _, maxColors := range []int{2, 8, 16, 63, 64, 256} {
		pal, idx := quantize(img, maxColors)
		if len(pal) == 0 || len(pal) > maxColors {
			t.Errorf("maxColors %d: palette has %d colours", maxColors, len(pal))
		}
		for i, p := range idx {
			if p < 0 || p >= len(pal) {
				t.Fatalf("maxColors %d: pixel %d has palette index %d of %d", maxColors, i, p, len(pal))
			}
		}

		out := EncodeSixel(img, maxColors)
		if n := strings.Count(out, ";2;"); n > maxColors {
			t.Errorf("EncodeSixel(maxColors=%d) declares %d colour registers", maxColors, n)
		}
	}
}

// Pixels whose bucket is dropped map to the nearest kept colour, so a
// two-colour cap on a black/white image with a grey pixel in each of two other
// buckets keeps black and white and sends each grey to the closer of the two.
func TestQuantizeFoldsDroppedBucketsIntoNearest(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 2))
	for x := 0; x < 10; x++ {
		img.SetRGBA(x, 0, gocolor.RGBA{A: 255})                         // black
		img.SetRGBA(x, 1, gocolor.RGBA{R: 255, G: 255, B: 255, A: 255}) // white
	}
	img.SetRGBA(0, 0, gocolor.RGBA{R: 80, G: 80, B: 80, A: 255})    // dark grey
	img.SetRGBA(0, 1, gocolor.RGBA{R: 180, G: 180, B: 180, A: 255}) // light grey

	pal, idx := quantize(img, 2)
	if len(pal) != 2 {
		t.Fatalf("palette = %v, want 2 colours", pal)
	}
	black, white := idx[1], idx[10+1]
	if pal[black] != [3]uint8{0, 0, 0} || pal[white] != [3]uint8{255, 255, 255} {
		t.Fatalf("palette = %v, want black and white", pal)
	}
	if idx[0] != black {
		t.Errorf("dark grey mapped to %v, want black", pal[idx[0]])
	}
	if idx[10] != white {
		t.Errorf("light grey mapped to %v, want white", pal[idx[10]])
	}
}
