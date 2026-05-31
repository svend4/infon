package babe

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
)

func TestOctantDimensions(t *testing.T) {
	img := fillSolid(80, 160, color.RGB{R: 40, G: 80, B: 120})
	f := ImageToFrameOctant(img, 20, 20)
	if f.Width != 20 || f.Height != 20 {
		t.Fatalf("frame = %dx%d, want 20x20", f.Width, f.Height)
	}
}

func TestOctantSolidIsFull(t *testing.T) {
	c := color.RGB{R: 120, G: 130, B: 140}
	img := fillSolid(40, 80, c)
	f := ImageToFrameOctant(img, 10, 10)
	if f.Blocks[5][5].Glyph != '█' {
		t.Errorf("solid octant glyph = %q, want █", f.Blocks[5][5].Glyph)
	}
}

func TestOctantTopBottomSplit(t *testing.T) {
	bright := color.RGB{R: 250, G: 250, B: 250}
	dark := color.RGB{R: 5, G: 5, B: 5}
	img := halfSplit(40, 80, bright, dark)
	f := ImageToFrameOctant(img, 10, 10)
	top := f.Blocks[0][5]
	if top.Fg.ToOKLab().L <= top.Bg.ToOKLab().L {
		t.Errorf("top cell fg should be lighter: fg=%v bg=%v", top.Fg, top.Bg)
	}
}

func TestOctantModeDispatch(t *testing.T) {
	if m, ok := ParseRenderMode("octant"); !ok || m != ModeOctant {
		t.Errorf("ParseRenderMode(octant) = %v,%v", m, ok)
	}
	if ModeOctant.String() != "octant" {
		t.Errorf("string = %q", ModeOctant.String())
	}
	img := halfSplit(40, 80, color.White, color.Black)
	f := ImageToFrameMode(img, 20, 20, ModeOctant)
	if f.Width != 20 || f.Height != 20 {
		t.Fatalf("frame = %dx%d, want 20x20", f.Width, f.Height)
	}
}

func TestOctantEmptyImageSafe(t *testing.T) {
	f := ImageToFrameOctant(fillSolid(0, 0, color.Black), 4, 4)
	if f.Width != 4 || f.Height != 4 {
		t.Fatalf("frame = %dx%d, want 4x4", f.Width, f.Height)
	}
}
