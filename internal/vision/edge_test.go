package vision

import (
	"image"
	gocolor "image/color"
	"testing"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// vsplit makes an image with a hard vertical edge: left half black, right half
// white — a strong gradient down the middle column.
func vsplit(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := gocolor.RGBA{A: 255}
			if x >= w/2 {
				c = gocolor.RGBA{R: 255, G: 255, B: 255, A: 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestSobelDetectsVerticalEdge(t *testing.T) {
	em := Sobel(vsplit(20, 20))
	if em.Width != 20 || em.Height != 20 {
		t.Fatalf("edge map = %dx%d, want 20x20", em.Width, em.Height)
	}
	// The middle column should have a strong edge; the flat interior should not.
	mid := em.At(10, 10)
	flatLeft := em.At(2, 10)
	flatRight := em.At(17, 10)
	if mid < 100 {
		t.Errorf("middle edge magnitude = %d, expected strong (>100)", mid)
	}
	if flatLeft > 20 || flatRight > 20 {
		t.Errorf("flat regions should be near zero: left=%d right=%d", flatLeft, flatRight)
	}
}

func TestSobelSolidImageHasNoEdges(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetRGBA(x, y, gocolor.RGBA{R: 80, G: 80, B: 80, A: 255})
		}
	}
	em := Sobel(img)
	for _, m := range em.Mag {
		if m != 0 {
			t.Fatalf("solid image should have zero edges, found %d", m)
			break
		}
	}
}

func TestSobelTinyImageSafe(t *testing.T) {
	em := Sobel(image.NewRGBA(image.Rect(0, 0, 2, 2)))
	if em.Width != 2 || em.Height != 2 {
		t.Fatalf("tiny edge map = %dx%d", em.Width, em.Height)
	}
	// All zero (too small for the operator).
	for _, m := range em.Mag {
		if m != 0 {
			t.Error("tiny image should yield no edges")
		}
	}
}

func TestEdgeOverlayDrawsBrailleOnEdges(t *testing.T) {
	em := Sobel(vsplit(40, 40))
	f := terminal.NewFrame(10, 10)
	// Fill base with a known glyph so we can detect where the overlay changed it.
	f.Fill('.', color.White, color.Black)

	EdgeOverlay(f, em, 80, color.NewRGB(0, 255, 0))

	// Cells over the middle column should now hold a Braille glyph (U+2800+).
	edgeCells := 0
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			g := f.Blocks[y][x].Glyph
			if g >= 0x2800 && g <= 0x28FF {
				edgeCells++
			}
		}
	}
	if edgeCells == 0 {
		t.Error("expected some Braille overlay cells along the edge")
	}
	// Far-left column cells should be untouched ('.').
	if f.Blocks[5][0].Glyph != '.' {
		t.Errorf("flat region cell was overwritten: %q", f.Blocks[5][0].Glyph)
	}
}

func TestEdgeOverlayEmptyMapNoop(t *testing.T) {
	f := terminal.NewFrame(4, 4)
	f.Fill('x', color.White, color.Black)
	EdgeOverlay(f, &EdgeMap{}, 50, color.White)
	if f.Blocks[0][0].Glyph != 'x' {
		t.Error("empty edge map should not modify the frame")
	}
}
