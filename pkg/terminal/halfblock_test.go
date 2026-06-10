package terminal

import (
	"image"
	"image/color"
	"testing"

	pcolor "github.com/svend4/infon/pkg/color"
)

func TestHalfBlockStacksTwoRows(t *testing.T) {
	// 1-wide, 2-tall image: row 0 red, row 1 blue -> one cell, fg=red, bg=blue.
	img := image.NewRGBA(image.Rect(0, 0, 1, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(0, 1, color.RGBA{B: 255, A: 255})

	f := HalfBlock(img, 1, 1)
	if f.Width != 1 || f.Height != 1 {
		t.Fatalf("frame = %dx%d, want 1x1", f.Width, f.Height)
	}
	blk := f.Blocks[0][0]
	if blk.Glyph != '▀' {
		t.Errorf("glyph = %q, want ▀", blk.Glyph)
	}
	if blk.Fg != (pcolor.RGB{R: 255}) {
		t.Errorf("fg = %+v, want red", blk.Fg)
	}
	if blk.Bg != (pcolor.RGB{B: 255}) {
		t.Errorf("bg = %+v, want blue", blk.Bg)
	}
}

func TestHalfBlockHandlesEmpty(t *testing.T) {
	if f := HalfBlock(image.NewRGBA(image.Rect(0, 0, 0, 0)), 4, 4); f.Width != 4 {
		t.Errorf("empty image should still yield a 4x4 frame, got %dx%d", f.Width, f.Height)
	}
}
