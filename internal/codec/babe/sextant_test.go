package babe

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
)

func TestSextantDimensions(t *testing.T) {
	img := fillSolid(60, 90, color.RGB{R: 30, G: 60, B: 90})
	f := ImageToFrameSextant(img, 20, 30)
	if f.Width != 20 || f.Height != 30 {
		t.Fatalf("frame = %dx%d, want 20x30", f.Width, f.Height)
	}
}

func TestSextantSolidIsFullGlyph(t *testing.T) {
	// A uniform image: all six sub-pixels equal -> all above mean -> full block,
	// and both colors equal the input.
	c := color.RGB{R: 100, G: 140, B: 180}
	img := fillSolid(20, 30, c)
	f := ImageToFrameSextant(img, 10, 10)
	cell := f.Blocks[5][5]
	if cell.Glyph != '█' {
		t.Errorf("solid sextant glyph = %q, want █", cell.Glyph)
	}
}

func TestSextantTopBottomSplit(t *testing.T) {
	// Bright top third, dark bottom: foreground (set bits) should be the bright
	// color and brighter than the background.
	bright := color.RGB{R: 250, G: 250, B: 250}
	dark := color.RGB{R: 5, G: 5, B: 5}
	img := halfSplit(20, 30, bright, dark)
	f := ImageToFrameSextant(img, 10, 10)

	top := f.Blocks[0][5]
	if top.Fg.ToOKLab().L <= top.Bg.ToOKLab().L {
		t.Errorf("top cell fg should be lighter than bg: fg=%v bg=%v", top.Fg, top.Bg)
	}
}

func TestSextantModeDispatch(t *testing.T) {
	img := halfSplit(40, 60, color.White, color.Black)
	f := ImageToFrameMode(img, 20, 20, ModeSextant)
	if f.Width != 20 || f.Height != 20 {
		t.Fatalf("frame = %dx%d, want 20x20", f.Width, f.Height)
	}
	if m, ok := ParseRenderMode("sextant"); !ok || m != ModeSextant {
		t.Errorf("ParseRenderMode(sextant) = %v,%v", m, ok)
	}
	if ModeSextant.String() != "sextant" {
		t.Errorf("ModeSextant.String() = %q", ModeSextant.String())
	}
}

func TestSextantEmptyImageSafe(t *testing.T) {
	f := ImageToFrameSextant(fillSolid(0, 0, color.Black), 4, 4)
	if f.Width != 4 || f.Height != 4 {
		t.Fatalf("frame = %dx%d, want 4x4", f.Width, f.Height)
	}
}
