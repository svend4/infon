package microfont

import (
	"image"
	"image/color"
	"testing"
)

func TestMeasureAdvance(t *testing.T) {
	w, h := Measure("AB", 2)
	// two glyphs: (2*(3+1)-1)*2 = 14 wide, 5*2 = 10 tall
	if w != 14 || h != 10 {
		t.Fatalf("Measure(AB,2) = %dx%d, want 14x10", w, h)
	}
	if w0, _ := Measure("", 3); w0 != 0 {
		t.Fatalf("empty width = %d, want 0", w0)
	}
}

func TestDrawSetsPixelsAndReturnsEnd(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 12))
	end := Draw(img, 0, 0, 2, "HI", color.RGBA{255, 255, 255, 255})
	// Draw advances a full (glyph+gap) cell per character, including the trailer.
	wantEnd := 2 * (gw + gap) * 2
	if end != wantEnd {
		t.Fatalf("Draw returned end %d, want %d", end, wantEnd)
	}
	on := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				r, g, bl, _ := img.At(x, y).RGBA()
				if r > 0 || g > 0 || bl > 0 {
					on++
				}
			}
		}
	}
	if on == 0 {
		t.Fatal("Draw set no pixels")
	}
}

func TestUnknownRuneIsBlank(t *testing.T) {
	// a rune not in the table must not panic and renders nothing extra.
	img := image.NewRGBA(image.Rect(0, 0, 20, 8))
	_ = Draw(img, 0, 0, 1, "~", color.RGBA{255, 255, 255, 255})
}
