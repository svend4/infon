package pseudo

import (
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/terminal"
)

// MarkArt is a picture composed from the sub-cell alphabet (pkg/glyphset): a grid
// of mark tokens (names like "tri-ll" or literal glyphs like ◣), with optional
// per-cell colors. It lets a text model draw with diagonals/triangles/quadrants —
// e.g. a mountain from ◣◢, a roof from ◤◥ — not just place sigils.
type MarkArt struct {
	Rows   [][]string `json:"rows"`             // grid of mark tokens
	Colors [][]string `json:"colors,omitempty"` // optional parallel grid of color tokens
	Bg     string     `json:"bg,omitempty"`
	Fg     string     `json:"fg,omitempty"`
	pal    Palette
}

func (a *MarkArt) frame(cols, rows int) *terminal.Frame {
	bg := resolve(a.pal, a.Bg, color.RGB{R: 12, G: 14, B: 22})
	deffg := resolve(a.pal, a.Fg, color.RGB{R: 235, G: 238, B: 245})
	f := terminal.NewFrame(cols, rows)
	f.Fill(' ', bg, bg)

	ah := len(a.Rows)
	aw := 0
	for _, r := range a.Rows {
		if len(r) > aw {
			aw = len(r)
		}
	}
	if ah == 0 || aw == 0 {
		return f
	}
	ox := (cols - aw) / 2
	oy := (rows - ah) / 2
	if ox < 0 {
		ox = 0
	}
	if oy < 0 {
		oy = 0
	}
	for y, row := range a.Rows {
		for x, tok := range row {
			g := glyphset.Glyph(tok)
			if g == ' ' {
				continue
			}
			fg := deffg
			if y < len(a.Colors) && x < len(a.Colors[y]) && a.Colors[y][x] != "" {
				fg = resolve(a.pal, a.Colors[y][x], deffg)
			}
			f.SetBlock(ox+x, oy+y, g, fg, bg)
		}
	}
	return f
}
