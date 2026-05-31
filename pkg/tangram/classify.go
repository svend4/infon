package tangram

import (
	"sort"
	"strings"

	"github.com/svend4/infon/pkg/glyphset"
)

// classify.go turns the numbering scheme into "gate" primitives: read a figure
// as an ADDRESS (a sequence of glyph-symbols), and NAME an arbitrary build by
// finding the nearest known figure (pose-invariant). Together they let the AI
// act as a dialing computer: it knows where you are and what you are near.

func hamming(a, b [][]int) int {
	h := 0
	for y := range a {
		for x := range a[y] {
			if a[y][x] != b[y][x] {
				h++
			}
		}
	}
	return h
}

func sameDims(a, b [][]int) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	return len(a[0]) == len(b[0])
}

// Distance is the pose-invariant cell distance from a to b: the minimum Hamming
// distance between a's grid and any dihedral variant of b (4 rotations x mirror)
// that shares a's dimensions. -1 if no orientation of b fits a's dimensions.
func Distance(a, b Figure) int {
	ag := a.digitGrid()
	best := -1
	cur := b.digitGrid()
	for r := 0; r < 4; r++ {
		for _, v := range [][][]int{cur, mirrorGridX(cur)} {
			if sameDims(ag, v) {
				if h := hamming(ag, v); best < 0 || h < best {
					best = h
				}
			}
		}
		cur = rotateGridCW(cur)
	}
	return best
}

// Match is a classification result: the nearest known figure and how close.
type Match struct {
	Name     string  `json:"name"`
	Distance int     `json:"distance"` // differing cells, pose-aligned
	Score    float64 `json:"score"`    // 1 - distance/cells, in [0,1]
}

// Classify finds the catalog figure nearest to f (pose-invariant) with a
// similarity score. Ties break by name for determinism.
func Classify(f Figure) Match {
	cat := Catalog()
	names := make([]string, 0, len(cat))
	for n := range cat {
		names = append(names, n)
	}
	sort.Strings(names)

	best := Match{Distance: 1 << 30}
	for _, name := range names {
		d := Distance(f, cat[name])
		if d < 0 {
			continue
		}
		if d < best.Distance {
			best = Match{Name: name, Distance: d}
		}
	}
	if best.Name != "" && f.H*f.W > 0 {
		best.Score = 1 - float64(best.Distance)/float64(f.H*f.W)
	}
	return best
}

// Address is the figure's "dialing sequence": its non-blank cell glyphs in
// reading order, as a glyph string — a Stargate-style address over the piece
// ring. It is lossless (the figure can be rebuilt from address + dimensions).
func (f Figure) Address() string {
	var b strings.Builder
	for _, row := range f.digitGrid() {
		for _, d := range row {
			if d != 0 {
				b.WriteRune(glyphset.Glyph(alphabet[d]))
			}
		}
	}
	return b.String()
}

// GateCode is a short, fixed-length quick-dial code hashed from the canonical
// (pose-invariant) number: n visible glyph-symbols. It is NOT unique — like a
// short Stargate address it only resolves together with the catalog/AI.
func (f Figure) GateCode(n int) string {
	var acc uint64 = 1469598103934665603
	for _, bb := range f.CanonicalNumber().Bytes() {
		acc ^= uint64(bb)
		acc *= 1099511628211
	}
	core := []int{1, 2, 3, 4, 5, 6, 7, 8} // visible (non-blank) digits
	var s strings.Builder
	for i := 0; i < n; i++ {
		s.WriteRune(glyphset.Glyph(alphabet[core[int(acc%uint64(len(core)))]]))
		acc ^= acc << 13
		acc ^= acc >> 7
		acc ^= acc << 17
	}
	return s.String()
}

// GlyphRows renders the figure as one display string per row (blank = space),
// for drawing the board in the terminal.
func (f Figure) GlyphRows() []string {
	g := f.digitGrid()
	out := make([]string, len(g))
	for y, row := range g {
		var b strings.Builder
		for _, d := range row {
			if d == 0 {
				b.WriteRune(' ')
			} else {
				b.WriteRune(glyphset.Glyph(alphabet[d]))
			}
		}
		out[y] = b.String()
	}
	return out
}
