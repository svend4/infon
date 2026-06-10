package babe

import (
	"image"
	gocolor "image/color"
	"testing"

	"github.com/svend4/infon/pkg/color"
)

func rampIndexOf(r rune) int {
	for i, c := range asciiRamp {
		if c == r {
			return i
		}
	}
	return -1
}

func TestASCIIRampBlackAndWhite(t *testing.T) {
	black := ImageToFrameASCII(fillSolid(40, 40, color.Black), 20, 10)
	if black.Width != 20 || black.Height != 10 {
		t.Fatalf("frame = %dx%d, want 20x10", black.Width, black.Height)
	}
	if g := black.Blocks[0][0].Glyph; g != asciiRamp[0] {
		t.Errorf("black cell glyph = %q, want the lightest ramp glyph %q", g, asciiRamp[0])
	}
	white := ImageToFrameASCII(fillSolid(40, 40, color.White), 20, 10)
	if g := white.Blocks[5][9].Glyph; g != asciiRamp[len(asciiRamp)-1] {
		t.Errorf("white cell glyph = %q, want the densest ramp glyph %q", g, asciiRamp[len(asciiRamp)-1])
	}
}

// A left-to-right brightness gradient must map to a non-decreasing ramp index:
// brighter cells get denser glyphs. This exercises the area-averaging too.
func TestASCIIRampFollowsBrightness(t *testing.T) {
	w := 64
	img := image.NewRGBA(image.Rect(0, 0, w, 8))
	for x := 0; x < w; x++ {
		v := uint8(x * 255 / (w - 1))
		for y := 0; y < 8; y++ {
			img.SetRGBA(x, y, gocolor.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	f := ImageToFrameASCII(img, 32, 1)
	prev := -1
	for x := 0; x < f.Width; x++ {
		idx := rampIndexOf(f.Blocks[0][x].Glyph)
		if idx < 0 {
			t.Fatalf("cell %d uses a glyph not in the ramp: %q", x, f.Blocks[0][x].Glyph)
		}
		if idx < prev {
			t.Errorf("ramp index dropped at x=%d (%d < %d): not monotonic with brightness", x, idx, prev)
		}
		prev = idx
	}
	if first := rampIndexOf(f.Blocks[0][0].Glyph); first >= rampIndexOf(f.Blocks[0][f.Width-1].Glyph) {
		t.Error("the dark end should map to a lighter glyph than the bright end")
	}
}

func TestASCIIModeWired(t *testing.T) {
	for _, name := range []string{"ascii", "ramp", "text", "art"} {
		if m, ok := ParseRenderMode(name); !ok || m != ModeASCII {
			t.Errorf("ParseRenderMode(%q) = %v,%v want ModeASCII,true", name, m, ok)
		}
	}
	if ModeASCII.String() != "ascii" {
		t.Errorf("ModeASCII.String() = %q, want ascii", ModeASCII.String())
	}
	f := ImageToFrameMode(fillSolid(20, 20, color.White), 10, 10, ModeASCII)
	if f.Width != 10 || f.Height != 10 {
		t.Errorf("dispatch frame = %dx%d, want 10x10", f.Width, f.Height)
	}
}
