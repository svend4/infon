// Package automode picks, per cell, the sub-cell glyph whose coverage best
// matches a target mask — dynamically switching between triangles, diagonals,
// halves and quadrants instead of using quadrants alone. It is the "auto
// triangle/quadrant" idea from the GLYPH_applications note: diagonal content
// (faces, horizons, sails, hills) renders sharper at the same bandwidth.
package automode

import "github.com/svend4/infon/pkg/glyphset"

// Candidates is the full alphabet the chooser may use (blank first, then solid,
// halves, quadrants, triangles, diagonals).
var Candidates = []string{
	"", "full",
	"top", "bottom", "left", "right",
	"q-tl", "q-tr", "q-bl", "q-br",
	"tri-ul", "tri-ur", "tri-ll", "tri-lr",
	"diag-/", "diag-\\",
}

// QuadCandidates is the conservative set (no triangles/diagonals) — the baseline
// a quadrant-only renderer would be limited to.
var QuadCandidates = []string{
	"", "full",
	"top", "bottom", "left", "right",
	"q-tl", "q-tr", "q-bl", "q-br",
}

func iou(mask [][]bool, tok string, S int) float64 {
	inter, uni := 0, 0
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			a := y < len(mask) && x < len(mask[y]) && mask[y][x]
			b := tok != "" && glyphset.Coverage(tok, x, y, S)
			if a && b {
				inter++
			}
			if a || b {
				uni++
			}
		}
	}
	if uni == 0 {
		return 1 // empty target perfectly matched by the blank glyph
	}
	return float64(inter) / float64(uni)
}

// BestAmong returns the candidate token whose coverage maximizes IoU with the
// SxS target mask, and that IoU. Ties resolve to the earlier candidate.
func BestAmong(mask [][]bool, cands []string) (string, float64) {
	best, bestIoU := "", -1.0
	S := glyphset.Sub
	for _, tok := range cands {
		if v := iou(mask, tok, S); v > bestIoU {
			bestIoU, best = v, tok
		}
	}
	return best, bestIoU
}

// Best chooses over the full alphabet (auto triangle/quadrant).
func Best(mask [][]bool) string {
	t, _ := BestAmong(mask, Candidates)
	return t
}

// Score is the IoU the chosen token achieves for the mask (1.0 = exact).
func Score(mask [][]bool) float64 {
	_, v := BestAmong(mask, Candidates)
	return v
}
