package babe

import (
	"image"
	gocolor "image/color"
	"testing"

	"github.com/svend4/infon/pkg/color"
)

func TestHalfBlockDimensions(t *testing.T) {
	img := fillSolid(80, 60, color.RGB{R: 10, G: 20, B: 30})
	f := ImageToFrameHalfBlock(img, 40, 30)
	if f.Width != 40 || f.Height != 30 {
		t.Fatalf("frame = %dx%d, want 40x30", f.Width, f.Height)
	}
}

func TestHalfBlockUsesUpperHalfGlyph(t *testing.T) {
	img := fillSolid(10, 10, color.White)
	f := ImageToFrameHalfBlock(img, 5, 5)
	if g := f.Blocks[0][0].Glyph; g != '▀' {
		t.Errorf("glyph = %q, want ▀", g)
	}
}

func TestHalfBlockEncodesTwoExactColors(t *testing.T) {
	// Top half red, bottom half blue. A cell straddling the boundary is rare;
	// instead check a cell fully in the top region (fg=red,bg=red) and one fully
	// in the bottom region (fg=blue,bg=blue), proving colors are exact (no
	// clustering/averaging like the quadrant path).
	red := color.RGB{R: 255, G: 0, B: 0}
	blue := color.RGB{R: 0, G: 0, B: 255}
	img := halfSplit(20, 20, red, blue) // top half red, bottom half blue

	f := ImageToFrameHalfBlock(img, 10, 10)

	// Row 0 is well inside the red region; row 9 well inside the blue region.
	topCell := f.Blocks[0][5]
	botCell := f.Blocks[9][5]
	if topCell.Fg != red || topCell.Bg != red {
		t.Errorf("top cell colors = fg %v bg %v, want red/red", topCell.Fg, topCell.Bg)
	}
	if botCell.Fg != blue || botCell.Bg != blue {
		t.Errorf("bottom cell colors = fg %v bg %v, want blue/blue", botCell.Fg, botCell.Bg)
	}
}

func TestHalfBlockVerticalResolution(t *testing.T) {
	// A 1-cell-tall frame samples TWO distinct vertical pixels: make the top
	// pixel green and the bottom pixel black and confirm fg != bg, which the
	// quadrant path (one cell = blended) could not represent as two exact colors.
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, gocolor.RGBA{R: 0, G: 255, B: 0, A: 255}) // top
	img.SetRGBA(0, 1, gocolor.RGBA{R: 0, G: 0, B: 0, A: 255})   // bottom
	img.SetRGBA(1, 0, gocolor.RGBA{R: 0, G: 255, B: 0, A: 255})
	img.SetRGBA(1, 1, gocolor.RGBA{R: 0, G: 0, B: 0, A: 255})

	f := ImageToFrameHalfBlock(img, 1, 1)
	cell := f.Blocks[0][0]
	if cell.Fg == cell.Bg {
		t.Errorf("expected distinct top/bottom colors, got fg==bg==%v", cell.Fg)
	}
	if cell.Fg.G < 200 {
		t.Errorf("top (fg) should be green, got %v", cell.Fg)
	}
	if cell.Bg.G > 50 {
		t.Errorf("bottom (bg) should be dark, got %v", cell.Bg)
	}
}

func TestHalfBlockEmptyImageSafe(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	f := ImageToFrameHalfBlock(img, 4, 4)
	if f.Width != 4 || f.Height != 4 {
		t.Fatalf("frame = %dx%d, want 4x4", f.Width, f.Height)
	}
}
