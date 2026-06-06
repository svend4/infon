package automode

import (
	"testing"

	"github.com/svend4/infon/pkg/glyphset"
)

func maskOf(tok string) [][]bool {
	S := glyphset.Sub
	m := make([][]bool, S)
	for y := 0; y < S; y++ {
		m[y] = make([]bool, S)
		for x := 0; x < S; x++ {
			m[y][x] = glyphset.Coverage(tok, x, y, S)
		}
	}
	return m
}

// The chooser recovers a glyph when handed that glyph's own coverage.
func TestBestRecoversEachGlyph(t *testing.T) {
	for _, tok := range []string{"", "full", "left", "top", "q-tl", "tri-ul"} {
		if got := Best(maskOf(tok)); got != tok {
			t.Fatalf("Best(mask of %q) = %q", tok, got)
		}
	}
}

// On a diagonal (triangle) target, the full alphabet beats quadrant-only.
func TestAutoBeatsQuadOnDiagonal(t *testing.T) {
	m := maskOf("tri-ul")
	tAuto, auto := BestAmong(m, Candidates)
	_, quad := BestAmong(m, QuadCandidates)
	if auto < 0.999 {
		t.Fatalf("auto IoU %.3f (chose %q), want ~1.0 with a triangle available", auto, tAuto)
	}
	if !(auto > quad) {
		t.Fatalf("auto %.3f should beat quadrant-only %.3f on a diagonal", auto, quad)
	}
}
