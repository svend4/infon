package pseudo

import (
	"strings"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// Sigil is a single named pictograph placed at a normalized (0..1) position.
type Sigil struct {
	Name  string  `json:"name"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Color string  `json:"color,omitempty"`
}

// SigilScene composes named pictographs over an optional sky/ground wash. This
// is the "hieroglyph / icon" route: a model picks symbols and where they go.
type SigilScene struct {
	Bg     string  `json:"bg,omitempty"`
	Sky    string  `json:"sky,omitempty"`
	Ground string  `json:"ground,omitempty"`
	Items  []Sigil `json:"items"`
	Labels []Label `json:"labels,omitempty"`
	pal    Palette
}

// sigilRunes maps friendly names to BMP glyphs (kept in the Basic Multilingual
// Plane so they render as single cells in most terminals).
var sigilRunes = map[string]rune{
	"sun": '☀', "moon": '☾', "star": '★', "sparkle": '❖', "heart": '♥',
	"club": '♣', "spade": '♠', "diamond": '♦', "house": '⌂', "home": '⌂',
	"mountain": '▲', "peak": '▲', "hill": '⌒', "tree": '♣', "flower": '✿',
	"leaf": '❧', "cloud": '☁', "rain": '☂', "umbrella": '☂', "snow": '❄',
	"bolt": '⚡', "lightning": '⚡', "fire": '▲', "wave": '∿', "water": '≈',
	"fish": '❭', "music": '♪', "note": '♫', "check": '✓', "cross": '✗',
	"circle": '●', "ring": '◯', "square": '■', "dot": '•', "arrow": '→',
	"up": '↑', "down": '↓', "left": '←', "right": '→', "anchor": '⚓',
	"flag": '⚑', "skull": '☠', "yinyang": '☯', "phone": '☎', "pencil": '✎',
	"scissors": '✂', "plane": '✈', "crown": '♛', "king": '♚', "pawn": '♟',
	"boat": '⛵', "comet": '☄', "atom": '⚛', "peace": '☮',
}

func (s *SigilScene) frame(cols, rows int) *terminal.Frame {
	f := terminal.NewFrame(cols, rows)

	// background wash
	if s.Sky != "" {
		top := resolve(s.pal, s.Sky, color.RGB{R: 30, G: 40, B: 80})
		bot := top.Blend(color.RGB{R: 255, G: 255, B: 255}, 0.25)
		for y := 0; y < rows; y++ {
			t := 0.0
			if rows > 1 {
				t = float64(y) / float64(rows-1)
			}
			c := top.Blend(bot, t)
			for x := 0; x < cols; x++ {
				f.SetBlock(x, y, ' ', c, c)
			}
		}
	} else {
		bg := resolve(s.pal, s.Bg, color.RGB{R: 12, G: 14, B: 22})
		f.Fill(' ', bg, bg)
	}
	if s.Ground != "" {
		g := resolve(s.pal, s.Ground, color.RGB{R: 30, G: 60, B: 40})
		for y := rows * 2 / 3; y < rows; y++ {
			for x := 0; x < cols; x++ {
				f.SetBlock(x, y, ' ', g, g)
			}
		}
	}

	for _, it := range s.Items {
		r, ok := sigilRunes[strings.ToLower(strings.TrimSpace(it.Name))]
		if !ok {
			r = '◆'
		}
		x := clampi(int(it.X*float64(cols-1)+0.5), 0, cols-1)
		y := clampi(int(it.Y*float64(rows-1)+0.5), 0, rows-1)
		fg := resolve(s.pal, it.Color, color.RGB{R: 240, G: 230, B: 160})
		bg := f.Blocks[y][x].Bg
		f.SetBlock(x, y, r, fg, bg)
	}
	for _, lb := range s.Labels {
		x := clampi(int(lb.X*float64(cols-1)+0.5), 0, cols-1)
		y := clampi(int(lb.Y*float64(rows-1)+0.5), 0, rows-1)
		fg := resolve(s.pal, lb.Color, color.RGB{R: 245, G: 245, B: 250})
		drawTextKeepBg(f, x, y, lb.Text, fg)
	}
	return f
}
