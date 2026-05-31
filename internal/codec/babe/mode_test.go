package babe

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
)

func TestParseRenderMode(t *testing.T) {
	cases := map[string]RenderMode{
		"":           ModeQuadrant,
		"quadrant":   ModeQuadrant,
		"quad":       ModeQuadrant,
		"perceptual": ModePerceptual,
		"oklab":      ModePerceptual,
		"halfblock":  ModeHalfBlock,
		"half":       ModeHalfBlock,
	}
	for name, want := range cases {
		got, ok := ParseRenderMode(name)
		if !ok || got != want {
			t.Errorf("ParseRenderMode(%q) = %v,%v want %v,true", name, got, ok, want)
		}
	}
	if _, ok := ParseRenderMode("bogus"); ok {
		t.Error("unknown mode should return ok=false")
	}
}

func TestRenderModeString(t *testing.T) {
	if ModeQuadrant.String() != "quadrant" || ModePerceptual.String() != "perceptual" || ModeHalfBlock.String() != "halfblock" {
		t.Error("RenderMode.String mismatch")
	}
}

func TestImageToFrameModeDispatch(t *testing.T) {
	img := halfSplit(40, 40, color.White, color.Black)

	// Each mode returns a correctly-sized frame.
	for _, m := range []RenderMode{ModeQuadrant, ModePerceptual, ModeHalfBlock} {
		f := ImageToFrameMode(img, 20, 20, m)
		if f.Width != 20 || f.Height != 20 {
			t.Errorf("mode %v: frame %dx%d, want 20x20", m, f.Width, f.Height)
		}
	}

	// Half-block must use the ▀ glyph; the quadrant path generally does not for
	// a top/bottom split of a solid region.
	half := ImageToFrameMode(img, 20, 20, ModeHalfBlock)
	if half.Blocks[0][0].Glyph != '▀' {
		t.Errorf("halfblock mode glyph = %q, want ▀", half.Blocks[0][0].Glyph)
	}
}
