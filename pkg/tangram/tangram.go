// Package tangram builds pictures the way the Chinese game does: from a small,
// fixed set of sub-cell pieces (the glyphset alphabet — triangles ◤◥◣◢, halves
// ▀▄▌▐, the diagonals ▞▚ and the full block █) placed on a cell grid. The same
// handful of pieces compose into a cat, a swan, a boat, a tree — a two-level
// compositional script where shapes, not pixels, are the tokens.
//
// A Figure compiles two ways:
//
//   - Frame()/Image() render it richly: when two COMPLEMENTARY pieces share a
//     cell (◤+◢, ▀+▄, …) they are composited into one terminal glyph whose
//     foreground is the first piece and whose background is the second, so a
//     two-colour diagonal boundary (a gold nose inside an amber face) shows both
//     pieces in a single character cell.
//   - Marks() lowers the figure to a pseudo.Spec in the "marks" format, so it
//     rides the existing render pipeline AND composes with pkg/semframe deltas —
//     that is how one figure MORPHS into another (cat → swan) as a tiny P-frame.
//
// The piece vocabulary's defining property is COHERENCE BY CONSTRUCTION: pieces
// meet edge-to-edge, so a well-formed figure never has two pieces fighting over
// the same area. Validate() checks exactly that (see validate.go).
package tangram

import (
	"image"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/terminal"
)

// Piece is one tangram tile: a sub-cell glyph (by mark token — "tri-ul"/"◤",
// "full"/"█", "diag-/"/"▞", "top"/"▀", …) placed at cell (Y,X) in a colour. The
// glyph's coverage mask IS the piece's shape, so two complementary triangles
// tile a single cell exactly along their shared hypotenuse.
type Piece struct {
	Glyph string `json:"glyph"`
	Y     int    `json:"y"`
	X     int    `json:"x"`
	Color string `json:"color,omitempty"`
}

// Figure is a named assembly of pieces on an H×W cell grid — a tangram picture.
type Figure struct {
	Name   string  `json:"name"`
	H      int     `json:"h"`
	W      int     `json:"w"`
	Bg     string  `json:"bg,omitempty"`
	Pieces []Piece `json:"pieces"`
}

// complementOf pairs each glyph with the piece that tiles the rest of its cell.
// Only clean pairs are listed: the two triangle pairs and the three half pairs.
var complementOf = map[rune]rune{
	'◤': '◢', '◢': '◤', '◥': '◣', '◣': '◥',
	'▀': '▄', '▄': '▀', '▌': '▐', '▐': '▌',
	'▞': '▚', '▚': '▞',
}

var (
	defaultBg = color.RGB{R: 18, G: 22, B: 40}
	defaultFg = color.RGB{R: 235, G: 238, B: 245}
)

// cellResolved is the single drawable glyph a cell's pieces collapse to.
type cellResolved struct {
	glyph   rune
	fg      string
	bg      string
	bgSet   bool
	overlap bool // two+ non-complementary pieces shared this cell
}

// group buckets pieces by cell, dropping any that fall outside the grid.
func (f Figure) group() map[[2]int][]Piece {
	g := make(map[[2]int][]Piece)
	for _, p := range f.Pieces {
		if p.Y < 0 || p.Y >= f.H || p.X < 0 || p.X >= f.W {
			continue
		}
		k := [2]int{p.Y, p.X}
		g[k] = append(g[k], p)
	}
	return g
}

// resolveCell collapses the pieces in one cell to a single drawable glyph.
func resolveCell(ps []Piece) cellResolved {
	switch len(ps) {
	case 0:
		return cellResolved{glyph: ' '}
	case 1:
		return cellResolved{glyph: glyphset.Glyph(ps[0].Glyph), fg: ps[0].Color}
	case 2:
		g0, g1 := glyphset.Glyph(ps[0].Glyph), glyphset.Glyph(ps[1].Glyph)
		if complementOf[g0] == g1 {
			// Composite: piece 0 in the foreground, piece 1 shows through the bg.
			return cellResolved{glyph: g0, fg: ps[0].Color, bg: ps[1].Color, bgSet: true}
		}
		return cellResolved{glyph: g1, fg: ps[1].Color, overlap: true}
	default:
		last := ps[len(ps)-1]
		return cellResolved{glyph: glyphset.Glyph(last.Glyph), fg: last.Color, overlap: true}
	}
}

// cells resolves every occupied cell to its drawable glyph. The result is
// deterministic: each cell is written once under a unique key.
func (f Figure) cells() map[[2]int]cellResolved {
	out := make(map[[2]int]cellResolved)
	for k, ps := range f.group() {
		out[k] = resolveCell(ps)
	}
	return out
}

// Frame renders the figure to a terminal.Frame, compositing complementary pairs.
// Colours resolve against the BASE palette (pseudo.Color); for preset moods
// (neon/dusk/…) render through Marks().Image instead.
func (f Figure) Frame() *terminal.Frame {
	bg := pseudo.Color(f.Bg, defaultBg)
	fr := terminal.NewFrame(f.W, f.H)
	fr.Fill(' ', bg, bg)
	for k, c := range f.cells() {
		y, x := k[0], k[1]
		fg := pseudo.Color(c.fg, defaultFg)
		cbg := bg
		if c.bgSet {
			cbg = pseudo.Color(c.bg, bg)
		}
		fr.SetBlock(x, y, c.glyph, fg, cbg)
	}
	return fr
}

// Image rasterises the composited frame to an RGBA of the given pixel size.
func (f Figure) Image(pxW, pxH int) image.Image {
	return glyphset.RasterizeSize(f.Frame(), pxW, pxH)
}

// Marks lowers the figure to a pseudo.Spec (marks format), one glyph per cell,
// so it renders through the pipeline and diffs/morphs via pkg/semframe. Preset
// palettes are honoured here. Complementary pairs collapse to their foreground
// glyph (the partner colour folds into the figure background).
func (f Figure) Marks(palette string) pseudo.Spec {
	rows := make([][]string, f.H)
	cols := make([][]string, f.H)
	for y := 0; y < f.H; y++ {
		rows[y] = make([]string, f.W)
		cols[y] = make([]string, f.W)
	}
	for k, c := range f.cells() {
		if c.glyph == ' ' {
			continue
		}
		rows[k[0]][k[1]] = string(c.glyph)
		cols[k[0]][k[1]] = c.fg
	}
	return pseudo.Spec{
		Format:  pseudo.FormatMarks,
		Cols:    f.W,
		Rows:    f.H,
		Title:   f.Name,
		Palette: palette,
		Marks:   &pseudo.MarkArt{Rows: rows, Colors: cols, Bg: f.Bg},
	}
}
