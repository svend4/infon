package tangram

import (
	"strings"

	"github.com/svend4/infon/pkg/glyphset"
)

// SubMask returns the figure's silhouette as an H*Sub x W*Sub coverage grid —
// the union of all piece coverages, with no per-cell glyph identity.
func (f Figure) SubMask() [][]bool { m, _ := maskAt(f, f.H, f.W); return m }

// SilhouetteString renders the sub-pixel silhouette as ASCII ('#'/space),
// sampling every step-th pixel. It is what a "hard" solver (model) reads.
func (f Figure) SilhouetteString(step int) string {
	if step < 1 {
		step = 1
	}
	m := f.SubMask()
	var b strings.Builder
	for y := 0; y < len(m); y += step {
		for x := 0; x < len(m[y]); x += step {
			if m[y][x] {
				b.WriteByte('#')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// HardPuzzle is the silhouette-only task: reproduce Mask using Palette, with NO
// per-cell glyph hints (the hard counterpart of PuzzleState, whose Target names
// the glyph in every cell). The information needed to place a triangle vs. a
// solid lives entirely in the diagonal edges of the silhouette.
type HardPuzzle struct {
	Name    string   `json:"name,omitempty"`
	H       int      `json:"h"`
	W       int      `json:"w"`
	Palette []string `json:"palette"`
	Mask    [][]bool `json:"-"`
	Bg      string   `json:"bg,omitempty"`
}

// HardFromFigure builds the silhouette-only puzzle for f (same piece vocabulary
// as the easy puzzle, but only the mask is revealed).
func HardFromFigure(f Figure) HardPuzzle {
	ps := PuzzleFromFigure(f)
	m, _ := maskAt(f, f.H, f.W)
	return HardPuzzle{Name: f.Name, H: f.H, W: f.W, Palette: ps.Palette, Mask: m, Bg: f.Bg}
}

func cellOn(mask [][]bool, y, x, S int) int {
	on := 0
	for sy := 0; sy < S; sy++ {
		for sx := 0; sx < S; sx++ {
			if mask[y*S+sy][x*S+sx] {
				on++
			}
		}
	}
	return on
}

// cellGlyphFromMask picks the palette glyph (or "" for empty) whose coverage
// best agrees with the silhouette inside cell (y,x).
func cellGlyphFromMask(mask [][]bool, y, x int, palette []string) string {
	S := glyphset.Sub
	cands := append([]string{""}, palette...)
	best, bestScore := "", -1
	for _, g := range cands {
		score := 0
		for sy := 0; sy < S; sy++ {
			for sx := 0; sx < S; sx++ {
				on := mask[y*S+sy][x*S+sx]
				cov := g != "" && glyphset.Coverage(g, sx, sy, S)
				if on == cov {
					score++
				}
			}
		}
		if score > bestScore {
			bestScore, best = score, g
		}
	}
	return best
}

// SolveHard reconstructs a figure from the silhouette by quantising each cell to
// the nearest palette glyph. Because the diagonal edges disambiguate the
// triangles, this recovers the exact tiling — the silhouette is a sufficient
// statistic for the figure.
func SolveHard(hp HardPuzzle) Figure {
	pal := hp.Palette
	if len(pal) == 0 {
		pal = defaultPalette
	}
	var ps []Piece
	for y := 0; y < hp.H; y++ {
		for x := 0; x < hp.W; x++ {
			if g := cellGlyphFromMask(hp.Mask, y, x, pal); g != "" {
				ps = append(ps, Piece{Glyph: g, Y: y, X: x})
			}
		}
	}
	return Figure{Name: hp.Name + "-hard", H: hp.H, W: hp.W, Bg: hp.Bg, Pieces: ps}
}

// SolveCoarse reads only cell-level occupancy (any coverage -> a solid block),
// ignoring the diagonals. It is the weak baseline: it always covers the target
// but spills over every diagonal boundary.
func SolveCoarse(hp HardPuzzle) Figure {
	S := glyphset.Sub
	var ps []Piece
	for y := 0; y < hp.H; y++ {
		for x := 0; x < hp.W; x++ {
			if cellOn(hp.Mask, y, x, S) > 0 {
				ps = append(ps, Piece{Glyph: "full", Y: y, X: x})
			}
		}
	}
	return Figure{Name: hp.Name + "-coarse", H: hp.H, W: hp.W, Bg: hp.Bg, Pieces: ps}
}
