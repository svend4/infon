package tangram

import (
	"sort"

	"github.com/svend4/infon/pkg/glyphset"
)

// PuzzleState is the wire form of a tangram puzzle and the payload a tvcp-ai
// brain receives as Request.State for game "tangram": assemble Target — a
// per-cell grid of glyph tokens ("" = empty) on an H×W grid — using Palette
// pieces. It turns "draw this shape from the fixed pieces" into a checkable,
// model-agnostic spatial-reasoning task.
type PuzzleState struct {
	Name    string     `json:"name,omitempty"`
	H       int        `json:"h"`
	W       int        `json:"w"`
	Palette []string   `json:"palette"`
	Target  [][]string `json:"target"`
	Bg      string     `json:"bg,omitempty"`
}

var defaultPalette = []string{
	"full", "tri-ul", "tri-ur", "tri-ll", "tri-lr", "top", "bottom", "left", "right",
}

// PuzzleFromFigure derives a puzzle whose answer is f: the per-cell target
// glyphs and the distinct piece vocabulary f uses (sorted, as the palette).
func PuzzleFromFigure(f Figure) PuzzleState {
	target := make([][]string, f.H)
	for y := range target {
		target[y] = make([]string, f.W)
	}
	seen := map[string]bool{}
	var palette []string
	for k, c := range f.cells() {
		if c.glyph == ' ' {
			continue
		}
		g := string(c.glyph)
		target[k[0]][k[1]] = g
		if !seen[g] {
			seen[g] = true
			palette = append(palette, g)
		}
	}
	sort.Strings(palette)
	return PuzzleState{Name: f.Name, H: f.H, W: f.W, Palette: palette, Target: target, Bg: f.Bg}
}

// maskAt rasterises a figure's pieces to an H*Sub × W*Sub boolean coverage mask
// (the union over pieces) and returns the count of covered sub-pixels.
func maskAt(f Figure, H, W int) ([][]bool, int) {
	S := glyphset.Sub
	m := make([][]bool, H*S)
	for i := range m {
		m[i] = make([]bool, W*S)
	}
	on := 0
	for _, p := range f.Pieces {
		if p.Y < 0 || p.Y >= H || p.X < 0 || p.X >= W {
			continue
		}
		for sy := 0; sy < S; sy++ {
			for sx := 0; sx < S; sx++ {
				if glyphset.Coverage(p.Glyph, sx, sy, S) {
					gy, gx := p.Y*S+sy, p.X*S+sx
					if !m[gy][gx] {
						on++
					}
					m[gy][gx] = true
				}
			}
		}
	}
	return m, on
}

// figure rebuilds the puzzle's target as a Figure (one piece per non-empty cell).
func (p PuzzleState) figure() Figure {
	var ps []Piece
	for y := 0; y < p.H && y < len(p.Target); y++ {
		for x := 0; x < p.W && x < len(p.Target[y]); x++ {
			if p.Target[y][x] != "" {
				ps = append(ps, Piece{Glyph: p.Target[y][x], Y: y, X: x, Color: p.Bg})
			}
		}
	}
	return Figure{Name: p.Name, H: p.H, W: p.W, Bg: p.Bg, Pieces: ps}
}

// bestMatch picks the palette glyph whose sub-cell mask best agrees with target.
func bestMatch(target string, palette []string) string {
	S := glyphset.Sub
	best, bestScore := palette[0], -1
	for _, g := range palette {
		score := 0
		for sy := 0; sy < S; sy++ {
			for sx := 0; sx < S; sx++ {
				if glyphset.Coverage(target, sx, sy, S) == glyphset.Coverage(g, sx, sy, S) {
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

// Solve is the reference baseline: for each non-empty target cell it places the
// palette piece that best matches that cell. When the palette contains the exact
// target glyph (as PuzzleFromFigure guarantees), the assembly is exact (IoU 1).
func Solve(p PuzzleState) Figure {
	pal := p.Palette
	if len(pal) == 0 {
		pal = defaultPalette
	}
	var ps []Piece
	for y := 0; y < p.H && y < len(p.Target); y++ {
		for x := 0; x < p.W && x < len(p.Target[y]); x++ {
			t := p.Target[y][x]
			if t == "" {
				continue
			}
			ps = append(ps, Piece{Glyph: bestMatch(t, pal), Y: y, X: x})
		}
	}
	return Figure{Name: p.Name + "-solved", H: p.H, W: p.W, Bg: p.Bg, Pieces: ps}
}

// NaiveSolve fills every non-empty target cell with a solid block — a weak
// baseline that covers the target but spills over every diagonal boundary.
func NaiveSolve(p PuzzleState) Figure {
	var ps []Piece
	for y := 0; y < p.H && y < len(p.Target); y++ {
		for x := 0; x < p.W && x < len(p.Target[y]); x++ {
			if p.Target[y][x] != "" {
				ps = append(ps, Piece{Glyph: "full", Y: y, X: x})
			}
		}
	}
	return Figure{Name: p.Name + "-naive", H: p.H, W: p.W, Bg: p.Bg, Pieces: ps}
}

// PuzzleScore measures how well a solution reproduces the target silhouette.
type PuzzleScore struct {
	IoU        float64 `json:"iou"`         // intersection-over-union of covered area
	Recall     float64 `json:"recall"`      // fraction of target area covered
	Overflow   float64 `json:"overflow"`    // fraction of solution area outside target
	WellFormed bool    `json:"well_formed"` // no piece collisions (tangram rule)
	TargetPx   int     `json:"target_px"`
	SolPx      int     `json:"sol_px"`
	InterPx    int     `json:"inter_px"`
}

// ScorePuzzle compares a solution figure against the puzzle's target silhouette.
func ScorePuzzle(p PuzzleState, sol Figure) PuzzleScore {
	tm, tn := maskAt(p.figure(), p.H, p.W)
	sm, sn := maskAt(sol, p.H, p.W)
	inter, union := 0, 0
	for y := range tm {
		for x := range tm[y] {
			a, b := tm[y][x], sm[y][x]
			if a && b {
				inter++
			}
			if a || b {
				union++
			}
		}
	}
	sc := PuzzleScore{TargetPx: tn, SolPx: sn, InterPx: inter, WellFormed: sol.Validate().WellFormed}
	if union > 0 {
		sc.IoU = float64(inter) / float64(union)
	}
	if tn > 0 {
		sc.Recall = float64(inter) / float64(tn)
	}
	if sn > 0 {
		sc.Overflow = float64(sn-inter) / float64(sn)
	}
	return sc
}
