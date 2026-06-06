package pseudo

import (
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// GlyphArt is colored ASCII / Unicode art: lines of characters plus an optional
// per-character color map. This is the most natural format for a text model —
// it literally "types the picture" and tags which character is which color.
type GlyphArt struct {
	Rows    []string          `json:"rows"`              // lines of characters
	Palette map[string]string `json:"palette,omitempty"` // single-char key -> color token
	Bg      string            `json:"bg,omitempty"`      // background color token
	Fg      string            `json:"fg,omitempty"`      // default foreground token
	pal     Palette
}

func (a *GlyphArt) frame(cols, rows int) *terminal.Frame {
	bg := resolve(a.pal, a.Bg, color.RGB{R: 12, G: 12, B: 18})
	deffg := resolve(a.pal, a.Fg, color.RGB{R: 230, G: 230, B: 235})
	f := terminal.NewFrame(cols, rows)
	f.Fill(' ', bg, bg)

	ah := len(a.Rows)
	aw := 0
	for _, r := range a.Rows {
		if n := len([]rune(r)); n > aw {
			aw = n
		}
	}
	if ah == 0 || aw == 0 {
		return f
	}

	// char -> color
	cmap := make(map[rune]color.RGB, len(a.Palette))
	for k, v := range a.Palette {
		for _, kr := range k {
			cmap[kr] = resolve(a.pal, v, deffg)
			break
		}
	}

	// center the art on the canvas
	ox := (cols - aw) / 2
	oy := (rows - ah) / 2
	if ox < 0 {
		ox = 0
	}
	if oy < 0 {
		oy = 0
	}

	for y, line := range a.Rows {
		x := 0
		for _, ch := range line {
			if ch != ' ' {
				fg := deffg
				if c, ok := cmap[ch]; ok {
					fg = c
				}
				f.SetBlock(ox+x, oy+y, ch, fg, bg)
			}
			x++
		}
	}
	return f
}
