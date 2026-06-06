package pseudo

import (
	"strings"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// Label is a short text caption placed at a normalized (0..1) position.
type Label struct {
	Text  string  `json:"text"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Color string  `json:"color,omitempty"`
}

// Mixed layers crisp symbols and captions over a soft pseudo-diffusion grid: the
// best of both worlds — a painterly background a text model describes as a color
// grid, with sharp sigils/labels it places on top.
type Mixed struct {
	Grid   *Grid   `json:"grid,omitempty"`
	Sigils []Sigil `json:"sigils,omitempty"`
	Labels []Label `json:"labels,omitempty"`
	pal    Palette
}

func (m *Mixed) frame(cols, rows int) *terminal.Frame {
	var f *terminal.Frame
	if m.Grid != nil {
		m.Grid.pal = m.pal
		f = m.Grid.frameSmooth(cols, rows)
	} else {
		f = terminal.NewFrame(cols, rows)
		bg := resolve(m.pal, "navy", color.RGB{R: 20, G: 24, B: 48})
		f.Fill(' ', bg, bg)
	}
	for _, it := range m.Sigils {
		r, ok := sigilRunes[strings.ToLower(strings.TrimSpace(it.Name))]
		if !ok {
			r = '◆'
		}
		x := clampi(int(it.X*float64(cols-1)+0.5), 0, cols-1)
		y := clampi(int(it.Y*float64(rows-1)+0.5), 0, rows-1)
		fg := resolve(m.pal, it.Color, color.RGB{R: 240, G: 230, B: 160})
		f.SetBlock(x, y, r, fg, f.Blocks[y][x].Bg)
	}
	for _, lb := range m.Labels {
		x := clampi(int(lb.X*float64(cols-1)+0.5), 0, cols-1)
		y := clampi(int(lb.Y*float64(rows-1)+0.5), 0, rows-1)
		fg := resolve(m.pal, lb.Color, color.RGB{R: 245, G: 245, B: 250})
		drawTextKeepBg(f, x, y, lb.Text, fg)
	}
	return f
}

// drawTextKeepBg writes text at (x,y) keeping each cell's existing background, so
// captions sit on top of a gradient without punching holes in it.
func drawTextKeepBg(f *terminal.Frame, x, y int, text string, fg color.RGB) {
	for i, ch := range text {
		xx := x + i
		if xx >= 0 && xx < f.Width && y >= 0 && y < f.Height {
			f.SetBlock(xx, y, ch, fg, f.Blocks[y][xx].Bg)
		}
	}
}
