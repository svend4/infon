package babe

import (
	"image"
	gocolor "image/color"
	"testing"

	"github.com/svend4/infon/pkg/color"
)

func fillSolid(w, h int, c color.RGB) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	rc := gocolor.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, rc)
		}
	}
	return img
}

// halfSplit returns an image whose top half is colorA and bottom half is colorB.
func halfSplit(w, h int, top, bottom color.RGB) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		c := top
		if y >= h/2 {
			c = bottom
		}
		rc := gocolor.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, rc)
		}
	}
	return img
}

func TestImageToFrameDimensions(t *testing.T) {
	img := fillSolid(80, 60, color.RGB{R: 100, G: 150, B: 200})
	f := ImageToFrame(img, 40, 30)
	if f.Width != 40 || f.Height != 30 {
		t.Fatalf("frame = %dx%d, want 40x30", f.Width, f.Height)
	}
}

func TestEncodeBlockEmpty(t *testing.T) {
	glyph, fg, bg := EncodeBlock([4]color.RGB{}, [4]bool{})
	if glyph != ' ' || fg != color.Black || bg != color.Black {
		t.Errorf("empty block = %q/%v/%v, want space/black/black", glyph, fg, bg)
	}
}

func TestEncodeBlockSolidIsFull(t *testing.T) {
	// A uniform block: all four pixels equal -> all bits on the bright side, so
	// the glyph is the full block and both colors equal the input.
	red := color.RGB{R: 220, G: 20, B: 20}
	px := [4]color.RGB{red, red, red, red}
	valid := [4]bool{true, true, true, true}
	glyph, fg, _ := EncodeBlock(px, valid)
	if glyph != '█' {
		t.Errorf("solid block glyph = %q, want █", glyph)
	}
	if fg != red {
		t.Errorf("solid block fg = %v, want %v", fg, red)
	}
}

func TestEncodeBlockTopBottomSplit(t *testing.T) {
	// Bright top row, dark bottom row -> upper half block ▀, fg bright, bg dark.
	bright := color.RGB{R: 240, G: 240, B: 240}
	dark := color.RGB{R: 10, G: 10, B: 10}
	px := [4]color.RGB{bright, bright, dark, dark} // TL,TR,BL,BR
	valid := [4]bool{true, true, true, true}
	glyph, fg, bg := EncodeBlock(px, valid)
	if glyph != '▀' {
		t.Errorf("split glyph = %q, want ▀ (top set)", glyph)
	}
	if fg.Luminance() <= bg.Luminance() {
		t.Errorf("fg should be brighter than bg: fg=%v bg=%v", fg, bg)
	}
}

func TestEncodeBlockPerceptualMatchesStructure(t *testing.T) {
	// The perceptual encoder must produce the same glyph topology for an obvious
	// bright/dark split, and keep fg brighter than bg.
	bright := color.RGB{R: 250, G: 250, B: 250}
	dark := color.RGB{R: 5, G: 5, B: 5}
	px := [4]color.RGB{bright, bright, dark, dark}
	valid := [4]bool{true, true, true, true}
	glyph, fg, bg := EncodeBlockPerceptual(px, valid)
	if glyph != '▀' {
		t.Errorf("perceptual split glyph = %q, want ▀", glyph)
	}
	if fg.ToOKLab().L <= bg.ToOKLab().L {
		t.Errorf("perceptual fg should be lighter: fg=%v bg=%v", fg, bg)
	}
}

func TestEncodeBlockPerceptualEmpty(t *testing.T) {
	glyph, fg, bg := EncodeBlockPerceptual([4]color.RGB{}, [4]bool{})
	if glyph != ' ' || fg != color.Black || bg != color.Black {
		t.Errorf("empty perceptual block = %q/%v/%v", glyph, fg, bg)
	}
}

func TestImageToFramePerceptualDimensions(t *testing.T) {
	img := halfSplit(80, 60, color.White, color.Black)
	f := ImageToFramePerceptual(img, 40, 30)
	if f.Width != 40 || f.Height != 30 {
		t.Fatalf("frame = %dx%d, want 40x30", f.Width, f.Height)
	}
	// Top rows should be brighter than bottom rows for a top/bottom split image.
	topLum := int(f.Blocks[0][20].Fg.Luminance())
	botLum := int(f.Blocks[29][20].Fg.Luminance())
	if topLum < botLum {
		t.Errorf("expected brighter top (%d) than bottom (%d)", topLum, botLum)
	}
}

func TestAverageColorOKLab(t *testing.T) {
	// Averaging a color with itself returns ~itself.
	c := color.RGB{R: 120, G: 80, B: 200}
	got := averageColorOKLab([]color.RGB{c, c, c})
	if absI(int(got.R)-int(c.R)) > 2 || absI(int(got.G)-int(c.G)) > 2 || absI(int(got.B)-int(c.B)) > 2 {
		t.Errorf("avg of identical colors = %v, want ~%v", got, c)
	}
	// Empty slice returns black.
	if averageColorOKLab(nil) != color.Black {
		t.Error("empty average should be black")
	}
}

func absI(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
