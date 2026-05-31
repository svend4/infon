package babe

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
)

func TestBrailleDimensions(t *testing.T) {
	img := fillSolid(40, 80, color.RGB{R: 50, G: 100, B: 150})
	f := ImageToFrameBraille(img, 20, 20)
	if f.Width != 20 || f.Height != 20 {
		t.Fatalf("frame = %dx%d, want 20x20", f.Width, f.Height)
	}
}

func TestBrailleGlyphsAreInBrailleBlock(t *testing.T) {
	// Every produced glyph must be a Braille codepoint (U+2800..U+28FF).
	img := halfSplit(40, 80, color.White, color.Black)
	f := ImageToFrameBraille(img, 20, 20)
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			g := f.Blocks[y][x].Glyph
			if g < 0x2800 || g > 0x28FF {
				t.Fatalf("cell (%d,%d) glyph U+%04X is not Braille", x, y, g)
			}
		}
	}
}

func TestBrailleModeDispatch(t *testing.T) {
	if m, ok := ParseRenderMode("braille"); !ok || m != ModeBraille {
		t.Errorf("ParseRenderMode(braille) = %v,%v", m, ok)
	}
	if ModeBraille.String() != "braille" {
		t.Errorf("ModeBraille.String() = %q", ModeBraille.String())
	}
	img := fillSolid(20, 40, color.White)
	f := ImageToFrameMode(img, 10, 10, ModeBraille)
	if f.Width != 10 || f.Height != 10 {
		t.Fatalf("frame = %dx%d, want 10x10", f.Width, f.Height)
	}
}

func TestBrailleEmptyImageSafe(t *testing.T) {
	f := ImageToFrameBraille(fillSolid(0, 0, color.Black), 3, 3)
	if f.Width != 3 || f.Height != 3 {
		t.Fatalf("frame = %dx%d, want 3x3", f.Width, f.Height)
	}
}
