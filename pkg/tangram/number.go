package tangram

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/svend4/infon/pkg/glyphset"
)

// A figure is, at heart, a grid of cells each holding one piece-glyph. Reading
// that grid as digits in a fixed radix gives every figure a UNIQUE INTEGER — the
// same construction the I Ching uses (6 cells x 2 states -> 0..63), just with a
// bigger alphabet and grid. Number is that catalog id; FromNumber inverts it.
//
// The alphabet is ordered so the 8 CORE area glyphs occupy digits 0..7: a figure
// built only from {blank, full, the four triangles, the two halves} is
// "trigram-clean" — each cell is an exact 3-bit I Ching trigram. The full
// 16-glyph alphabet makes each cell a 4-bit nibble.

// AlphabetSize is the encoding radix (one digit per cell).
const AlphabetSize = 16

var alphabet = []string{
	"",        // 0  blank
	"full",    // 1
	"tri-ul",  // 2  (digits 0..7 = the I Ching trigram core)
	"tri-ur",  // 3
	"tri-ll",  // 4
	"tri-lr",  // 5
	"top",     // 6
	"bottom",  // 7
	"left",    // 8  (digits 8..15 extend to a full nibble)
	"right",   // 9
	"q-tl",    // 10
	"q-tr",    // 11
	"q-bl",    // 12
	"q-br",    // 13
	"diag-/",  // 14
	"diag-\\", // 15
}

var digitOf = func() map[string]int {
	m := make(map[string]int, len(alphabet))
	for i, n := range alphabet {
		m[n] = i
	}
	return m
}()

// GlyphForDigit / DigitForGlyph expose the alphabet mapping.
func GlyphForDigit(d int) string {
	if d >= 0 && d < len(alphabet) {
		return alphabet[d]
	}
	return ""
}
func DigitForGlyph(name string) int { return digitOf[name] }

var rotCWName = map[string]string{
	"": "", "full": "full",
	"tri-ul": "tri-ur", "tri-ur": "tri-lr", "tri-lr": "tri-ll", "tri-ll": "tri-ul",
	"top": "right", "right": "bottom", "bottom": "left", "left": "top",
	"q-tl": "q-tr", "q-tr": "q-br", "q-br": "q-bl", "q-bl": "q-tl",
	"diag-/": "diag-\\", "diag-\\": "diag-/",
}

var mirrorXName = map[string]string{
	"": "", "full": "full",
	"tri-ul": "tri-ur", "tri-ur": "tri-ul", "tri-ll": "tri-lr", "tri-lr": "tri-ll",
	"top": "top", "bottom": "bottom", "left": "right", "right": "left",
	"q-tl": "q-tr", "q-tr": "q-tl", "q-bl": "q-br", "q-br": "q-bl",
	"diag-/": "diag-\\", "diag-\\": "diag-/",
}

func rotDigit(d int) int    { return digitOf[rotCWName[alphabet[d]]] }
func mirrorDigit(d int) int { return digitOf[mirrorXName[alphabet[d]]] }

// digitGrid is the figure's H x W grid of alphabet digits (one per cell).
func (f Figure) digitGrid() [][]int {
	g := make([][]int, f.H)
	for y := range g {
		g[y] = make([]int, f.W)
	}
	for k, c := range f.cells() {
		if c.glyph != ' ' {
			g[k[0]][k[1]] = digitOf[glyphset.NameOf(c.glyph)]
		}
	}
	return g
}

func gridToFigure(name string, g [][]int, bg string) Figure {
	h := len(g)
	w := 0
	if h > 0 {
		w = len(g[0])
	}
	f := Figure{Name: name, H: h, W: w, Bg: bg}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if g[y][x] != 0 {
				f.Pieces = append(f.Pieces, Piece{Glyph: alphabet[g[y][x]], Y: y, X: x})
			}
		}
	}
	return f
}

func rotateGridCW(g [][]int) [][]int {
	h := len(g)
	if h == 0 {
		return nil
	}
	w := len(g[0])
	out := make([][]int, w)
	for i := 0; i < w; i++ {
		out[i] = make([]int, h)
		for j := 0; j < h; j++ {
			out[i][j] = rotDigit(g[h-1-j][i])
		}
	}
	return out
}

func mirrorGridX(g [][]int) [][]int {
	h := len(g)
	if h == 0 {
		return nil
	}
	w := len(g[0])
	out := make([][]int, h)
	for y := 0; y < h; y++ {
		out[y] = make([]int, w)
		for x := 0; x < w; x++ {
			out[y][x] = mirrorDigit(g[y][w-1-x])
		}
	}
	return out
}

// RotateCW returns the figure rotated 90 degrees clockwise (glyphs rotated too).
func (f Figure) RotateCW() Figure {
	return gridToFigure(f.Name, rotateGridCW(f.digitGrid()), f.Bg)
}

// MirrorX returns the figure mirrored left-to-right (glyphs mirrored too).
func (f Figure) MirrorX() Figure {
	return gridToFigure(f.Name, mirrorGridX(f.digitGrid()), f.Bg)
}

// Number is the figure's catalog id: its glyph grid read row-major as a base-16
// (AlphabetSize) integer. Distinct grids give distinct numbers.
func (f Figure) Number() *big.Int {
	n := new(big.Int)
	base := big.NewInt(int64(AlphabetSize))
	d := new(big.Int)
	for _, row := range f.digitGrid() {
		for _, dig := range row {
			n.Mul(n, base)
			d.SetInt64(int64(dig))
			n.Add(n, d)
		}
	}
	return n
}

// FromNumber reconstructs the H x W figure whose Number is n (the inverse of
// Number). Shape only — colors are not encoded.
func FromNumber(n *big.Int, h, w int) Figure {
	base := big.NewInt(int64(AlphabetSize))
	q := new(big.Int).Set(n)
	r := new(big.Int)
	total := h * w
	digs := make([]int, total)
	for i := total - 1; i >= 0; i-- {
		q.QuoRem(q, base, r)
		digs[i] = int(r.Int64())
	}
	g := make([][]int, h)
	for y := 0; y < h; y++ {
		g[y] = digs[y*w : (y+1)*w]
	}
	return gridToFigure("from-number", g, "navy")
}

// serial keys a figure for canonical comparison: dims first, then its digits.
func serial(g [][]int) string {
	h := len(g)
	w := 0
	if h > 0 {
		w = len(g[0])
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%03d:%03d:", h, w)
	for _, row := range g {
		for _, d := range row {
			b.WriteByte(byte('a' + d))
		}
	}
	return b.String()
}

// Canonical returns the orientation-normalized figure: the smallest of the 8
// dihedral variants (4 rotations x optional mirror). A silhouette and any of its
// rotations/reflections share one Canonical, so it is a pose-invariant id.
func (f Figure) Canonical() Figure {
	cur := f.digitGrid()
	bestGrid := cur
	bestKey := serial(cur)
	for r := 0; r < 4; r++ {
		for _, v := range [][][]int{cur, mirrorGridX(cur)} {
			if k := serial(v); k < bestKey {
				bestKey, bestGrid = k, v
			}
		}
		cur = rotateGridCW(cur)
	}
	out := gridToFigure(f.Name+"-canon", bestGrid, f.Bg)
	return out
}

// CanonicalNumber is Number of the Canonical figure: an id invariant to rotation
// and reflection.
func (f Figure) CanonicalNumber() *big.Int { return f.Canonical().Number() }

// TrigramClean reports whether every cell uses one of the 8 core glyphs (digits
// 0..7), so the figure reads as a grid of exact 3-bit I Ching trigrams.
func (f Figure) TrigramClean() bool {
	for _, row := range f.digitGrid() {
		for _, d := range row {
			if d >= 8 {
				return false
			}
		}
	}
	return true
}
