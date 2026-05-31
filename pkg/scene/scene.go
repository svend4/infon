// Package scene renders the tvcp-ai draw-DSL (a small JSON drawing language)
// into the repo's terminal.Frame. Shared by aidraw, the brain server, etc.
package scene

import (
	"encoding/json"
	"math"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// Op is one drawing instruction.
type Op struct {
	Op    string  `json:"op"`
	X     int     `json:"x"`
	Y     int     `json:"y"`
	W     int     `json:"w"`
	H     int     `json:"h"`
	CX    int     `json:"cx"`
	CY    int     `json:"cy"`
	R     int     `json:"r"`
	Glyph string  `json:"glyph"`
	S     string  `json:"s"`
	TL    bool    `json:"tl"`
	TR    bool    `json:"tr"`
	BL    bool    `json:"bl"`
	BR    bool    `json:"br"`
	Fg    *[3]int `json:"fg,omitempty"`
	Bg    *[3]int `json:"bg,omitempty"`
	From  *[3]int `json:"from,omitempty"`
	To    *[3]int `json:"to,omitempty"`
}

// Scene is a canvas plus a list of ops.
type Scene struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Bg     *[3]int `json:"bg,omitempty"`
	Ops    []Op    `json:"ops"`
}

func rgb(p *[3]int, def color.RGB) color.RGB {
	if p == nil {
		return def
	}
	return color.NewRGB(uint8(p[0]), uint8(p[1]), uint8(p[2]))
}

func firstRune(s string, def rune) rune {
	for _, r := range s {
		return r
	}
	return def
}

func lerp(a, b *[3]int, t float64) color.RGB {
	ch := func(i int) uint8 {
		return uint8(math.Round(float64(a[i]) + (float64(b[i])-float64(a[i]))*t))
	}
	return color.NewRGB(ch(0), ch(1), ch(2))
}

func keepBg(f *terminal.Frame, x, y int, bg *[3]int, def color.RGB) color.RGB {
	if bg != nil {
		return rgb(bg, def)
	}
	if x >= 0 && x < f.Width && y >= 0 && y < f.Height {
		return f.Blocks[y][x].Bg
	}
	return def
}

// Apply draws every op of sc onto f.
func Apply(f *terminal.Frame, sc *Scene) {
	black := color.NewRGB(0, 0, 0)
	white := color.NewRGB(255, 255, 255)
	for _, op := range sc.Ops {
		fg := rgb(op.Fg, white)
		switch op.Op {
		case "fill":
			f.Fill(firstRune(op.Glyph, ' '), fg, rgb(op.Bg, black))
		case "rect":
			g := firstRune(op.Glyph, ' ')
			for yy := op.Y; yy < op.Y+op.H; yy++ {
				for xx := op.X; xx < op.X+op.W; xx++ {
					f.SetBlock(xx, yy, g, fg, keepBg(f, xx, yy, op.Bg, black))
				}
			}
		case "vgradient":
			g := firstRune(op.Glyph, ' ')
			for i := 0; i < op.H; i++ {
				t := 0.0
				if op.H > 1 {
					t = float64(i) / float64(op.H-1)
				}
				c := lerp(op.From, op.To, t)
				for xx := op.X; xx < op.X+op.W; xx++ {
					f.SetBlock(xx, op.Y+i, g, c, c)
				}
			}
		case "hgradient":
			g := firstRune(op.Glyph, '█')
			for i := 0; i < op.W; i++ {
				t := 0.0
				if op.W > 1 {
					t = float64(i) / float64(op.W-1)
				}
				c := lerp(op.From, op.To, t)
				for yy := op.Y; yy < op.Y+op.H; yy++ {
					f.SetBlock(op.X+i, yy, g, c, keepBg(f, op.X+i, yy, op.Bg, black))
				}
			}
		case "text":
			x := op.X
			for _, ch := range op.S {
				f.SetBlock(x, op.Y, ch, fg, keepBg(f, x, op.Y, op.Bg, black))
				x++
			}
		case "box":
			f.DrawBox(op.X, op.Y, op.W, op.H, fg, rgb(op.Bg, black))
		case "quad":
			f.DrawQuadrant(op.X, op.Y, op.TL, op.TR, op.BL, op.BR, fg, rgb(op.Bg, black))
		case "disc":
			g := firstRune(op.Glyph, '█')
			for dy := -op.R; dy <= op.R; dy++ {
				for dx := -2 * op.R; dx <= 2*op.R; dx++ {
					fx := float64(dx) * 0.5
					if fx*fx+float64(dy*dy) <= float64(op.R*op.R) {
						xx, yy := op.CX+dx, op.CY+dy
						f.SetBlock(xx, yy, g, fg, keepBg(f, xx, yy, op.Bg, black))
					}
				}
			}
		}
	}
}

// Render parses a draw-DSL document and renders it to a Frame.
// RenderScene renders a parsed Scene, clamping the canvas to sane bounds so a
// misbehaving model cannot blow up the frame.
func RenderScene(sc *Scene) *terminal.Frame {
	if sc.Width <= 0 {
		sc.Width = 64
	}
	if sc.Height <= 0 {
		sc.Height = 28
	}
	if sc.Width > 400 {
		sc.Width = 400
	}
	if sc.Height > 200 {
		sc.Height = 200
	}
	black := color.NewRGB(0, 0, 0)
	f := terminal.NewFrame(sc.Width, sc.Height)
	if sc.Bg != nil {
		f.Fill(' ', rgb(sc.Bg, black), rgb(sc.Bg, black))
	}
	Apply(f, sc)
	return f
}

// Render parses a draw-DSL document and renders it.
func Render(data []byte) (*terminal.Frame, error) {
	var sc Scene
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, err
	}
	return RenderScene(&sc), nil
}
