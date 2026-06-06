package glyphset

import (
	"image"
	"image/color"
	"testing"
)

// A cell split by the anti-diagonal (bright upper-left, dark lower-right) must be
// rendered as the upper-left triangle ◤ — the whole point of the diagonal codec.
func TestRenderTrianglesPicksDiagonal(t *testing.T) {
	const n = 16
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if x+y <= n-1 {
				img.Set(x, y, color.RGBA{240, 240, 240, 255})
			} else {
				img.Set(x, y, color.RGBA{10, 10, 20, 255})
			}
		}
	}
	f := RenderTriangles(img, 1, 1)
	if g := f.Blocks[0][0].Glyph; g != '◤' {
		t.Fatalf("glyph = %q, want ◤ (upper-left triangle)", g)
	}
}

func TestCoverageGeometry(t *testing.T) {
	if !coverage('◤', 0, 0, sub) || coverage('◤', sub-1, sub-1, sub) {
		t.Fatal("◤ should cover top-left, not bottom-right")
	}
	if !coverage('█', 3, 3, sub) || coverage(' ', 3, 3, sub) {
		t.Fatal("full/blank coverage wrong")
	}
}

func TestRasterizeAndChart(t *testing.T) {
	f := Chart(8)
	if f == nil || f.Width == 0 {
		t.Fatal("empty chart")
	}
	img := Rasterize(f, 8)
	if b := img.Bounds(); b.Dx() != f.Width*8 {
		t.Fatalf("raster dims %v", b)
	}
}
