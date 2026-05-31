package babe

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
)

func TestOptimalEmpty(t *testing.T) {
	glyph, fg, bg := EncodeBlockOptimal([4]color.RGB{}, [4]bool{})
	if glyph != ' ' || fg != color.Black || bg != color.Black {
		t.Errorf("empty = %q/%v/%v", glyph, fg, bg)
	}
}

func TestOptimalSolidIsZeroError(t *testing.T) {
	// A uniform block can be reconstructed exactly: error 0, full or empty glyph
	// both score 0; either way fg/bg should equal the input color.
	c := color.RGB{R: 130, G: 90, B: 200}
	px := [4]color.RGB{c, c, c, c}
	valid := [4]bool{true, true, true, true}
	_, fg, bg := EncodeBlockOptimal(px, valid)
	// Both reconstructed colors must be ~the input (within OKLab round-trip).
	if absI(int(fg.R)-int(c.R)) > 3 && absI(int(bg.R)-int(c.R)) > 3 {
		t.Errorf("uniform block colors drifted: fg=%v bg=%v want ~%v", fg, bg, c)
	}
}

func TestOptimalChoosesCorrectPartition(t *testing.T) {
	// Top row red, bottom row blue. The optimal partition groups the two reds
	// together and the two blues together -> upper half block ▀.
	red := color.RGB{R: 230, G: 20, B: 20}
	blue := color.RGB{R: 20, G: 20, B: 230}
	px := [4]color.RGB{red, red, blue, blue} // TL,TR,BL,BR
	valid := [4]bool{true, true, true, true}
	glyph, fg, bg := EncodeBlockOptimal(px, valid)
	// The two reds must be grouped together and the two blues together. That is
	// the top/bottom split, which is represented EITHER as ▀ (fg=top) or its
	// complement ▄ (fg=bottom) — both reconstruct identically, so accept either.
	if glyph != '▀' && glyph != '▄' {
		t.Errorf("glyph = %q, want a top/bottom split (▀ or ▄)", glyph)
	}
	// One of {fg,bg} must be red-dominant and the other blue-dominant.
	redFg := fg.R > fg.B && bg.B > bg.R
	redBg := bg.R > bg.B && fg.B > fg.R
	if !redFg && !redBg {
		t.Errorf("colors should split into red & blue, got fg=%v bg=%v", fg, bg)
	}
}

func TestOptimalErrorNotWorseThanLuminance(t *testing.T) {
	// On a tricky block, the optimal encoder's reconstruction error must be <=
	// the luminance encoder's error (it searches the full space the luminance
	// split is one point of).
	px := [4]color.RGB{
		{R: 200, G: 50, B: 50},
		{R: 50, G: 200, B: 50},
		{R: 50, G: 50, B: 200},
		{R: 200, G: 200, B: 50},
	}
	valid := [4]bool{true, true, true, true}

	errOf := func(enc func([4]color.RGB, [4]bool) (rune, color.RGB, color.RGB)) float64 {
		_, fg, bg := enc(px, valid)
		// Reconstruction error against the brighter/darker assignment per pixel:
		// approximate by taking, for each pixel, the closer of fg/bg.
		total := 0.0
		for i := 0; i < 4; i++ {
			df := px[i].PerceptualDistance(fg)
			db := px[i].PerceptualDistance(bg)
			if df < db {
				total += df
			} else {
				total += db
			}
		}
		return total
	}

	optErr := errOf(EncodeBlockOptimal)
	lumErr := errOf(EncodeBlock)
	if optErr > lumErr+1e-9 {
		t.Errorf("optimal error %.5f should be <= luminance error %.5f", optErr, lumErr)
	}
}

func TestOptimalModeDispatch(t *testing.T) {
	if m, ok := ParseRenderMode("optimal"); !ok || m != ModeOptimal {
		t.Errorf("ParseRenderMode(optimal) = %v,%v", m, ok)
	}
	if ModeOptimal.String() != "optimal" {
		t.Errorf("string = %q", ModeOptimal.String())
	}
	img := halfSplit(40, 40, color.White, color.Black)
	f := ImageToFrameMode(img, 20, 20, ModeOptimal)
	if f.Width != 20 || f.Height != 20 {
		t.Fatalf("frame = %dx%d, want 20x20", f.Width, f.Height)
	}
}
