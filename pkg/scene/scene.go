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

// firstRune returns the first rune of s (def if s is empty), with control
// characters from the model replaced by a space so they never reach the frame.
func firstRune(s string, def rune) rune {
	for _, r := range s {
		return terminal.SanitizeGlyph(r)
	}
	return def
}

// gradEnds fills in a missing gradient endpoint ("from"/"to" are optional in
// the JSON) with the other one, or black when both are missing.
func gradEnds(from, to *[3]int) (*[3]int, *[3]int) {
	if from == nil {
		from = to
	}
	if to == nil {
		to = from
	}
	if from == nil {
		from, to = &[3]int{}, &[3]int{}
	}
	return from, to
}

// satAdd returns a+b, saturating at the int range instead of wrapping.
func satAdd(a, b int) int {
	s := a + b
	if b > 0 && s < a {
		return math.MaxInt
	}
	if b < 0 && s > a {
		return math.MinInt
	}
	return s
}

// clip intersects the run [start, start+n) with [0, limit), so an op only
// visits cells on the canvas however large (or negative) its size is.
func clip(start, n, limit int) (lo, hi int) {
	return max(start, 0), min(satAdd(start, n), limit)
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
			x0, x1 := clip(op.X, op.W, f.Width)
			y0, y1 := clip(op.Y, op.H, f.Height)
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					f.SetBlock(xx, yy, g, fg, keepBg(f, xx, yy, op.Bg, black))
				}
			}
		case "vgradient":
			g := firstRune(op.Glyph, ' ')
			from, to := gradEnds(op.From, op.To)
			x0, x1 := clip(op.X, op.W, f.Width)
			y0, y1 := clip(op.Y, op.H, f.Height)
			for yy := y0; yy < y1; yy++ {
				i := yy - op.Y
				t := 0.0
				if op.H > 1 {
					t = float64(i) / float64(op.H-1)
				}
				c := lerp(from, to, t)
				for xx := x0; xx < x1; xx++ {
					f.SetBlock(xx, yy, g, c, c)
				}
			}
		case "hgradient":
			g := firstRune(op.Glyph, '█')
			from, to := gradEnds(op.From, op.To)
			x0, x1 := clip(op.X, op.W, f.Width)
			y0, y1 := clip(op.Y, op.H, f.Height)
			for xx := x0; xx < x1; xx++ {
				i := xx - op.X
				t := 0.0
				if op.W > 1 {
					t = float64(i) / float64(op.W-1)
				}
				c := lerp(from, to, t)
				for yy := y0; yy < y1; yy++ {
					f.SetBlock(xx, yy, g, c, keepBg(f, xx, yy, op.Bg, black))
				}
			}
		case "text":
			x := op.X
			for _, ch := range op.S {
				f.SetBlock(x, op.Y, terminal.SanitizeGlyph(ch), fg, keepBg(f, x, op.Y, op.Bg, black))
				x++
			}
		case "box":
			f.DrawBox(op.X, op.Y, op.W, op.H, fg, rgb(op.Bg, black))
		case "quad":
			f.DrawQuadrant(op.X, op.Y, op.TL, op.TR, op.BL, op.BR, fg, rgb(op.Bg, black))
		case "disc":
			if op.R < 0 {
				break
			}
			g := firstRune(op.Glyph, '█')
			// Bounding box |dx| <= 2R, |dy| <= R, clipped to the canvas.
			d := satAdd(op.R, op.R)
			x0, x1 := max(satAdd(op.CX, -d), 0), min(satAdd(op.CX, d), f.Width-1)
			y0, y1 := max(satAdd(op.CY, -op.R), 0), min(satAdd(op.CY, op.R), f.Height-1)
			r := float64(op.R)
			for yy := y0; yy <= y1; yy++ {
				for xx := x0; xx <= x1; xx++ {
					fx := (float64(xx) - float64(op.CX)) * 0.5
					fy := float64(yy) - float64(op.CY)
					if fx*fx+fy*fy <= r*r {
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
