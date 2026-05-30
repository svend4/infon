// Package sketch is a HIGH-LEVEL, weak-model-friendly drawing format: named
// colors plus a handful of shapes. It translates to the full draw-DSL (scene),
// so small models only emit easy JSON (no [r,g,b] math, few shape kinds).
package sketch

import "github.com/svend4/infon/pkg/scene"

type Shape struct {
	Kind  string `json:"kind"`  // sun moon star hill mountain building cloud band text
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	Color string `json:"color"` // a palette name
	S     string `json:"s"`     // for text
}

// Sketch: a sky/ground wash plus shapes.
type Sketch struct {
	Sky    string  `json:"sky"`
	Ground string  `json:"ground"`
	Shapes []Shape `json:"shapes"`
}

var palette = map[string][3]int{
	"black": {10, 10, 14}, "white": {240, 240, 245}, "gray": {130, 130, 140},
	"red": {230, 70, 70}, "orange": {240, 140, 60}, "orangered": {240, 95, 55},
	"yellow": {250, 220, 110}, "gold": {245, 200, 90}, "amber": {235, 170, 70},
	"green": {90, 200, 110}, "darkgreen": {32, 110, 62}, "teal": {70, 200, 180},
	"blue": {80, 150, 235}, "navy": {28, 38, 88}, "cyan": {120, 220, 235}, "skyblue": {120, 185, 235},
	"purple": {150, 110, 220}, "pink": {235, 130, 170}, "brown": {120, 80, 50}, "dusk": {120, 80, 140},
}

func colv(name string, def [3]int) [3]int {
	if c, ok := palette[name]; ok {
		return c
	}
	return def
}
func p(v [3]int) *[3]int { return &v }
func clip(x int) int {
	if x > 255 {
		return 255
	}
	if x < 0 {
		return 0
	}
	return x
}
func add(c [3]int, d int) [3]int { return [3]int{clip(c[0] + d), clip(c[1] + d), clip(c[2] + d)} }
func max1(x int) int {
	if x < 1 {
		return 1
	}
	return x
}

// ToScene translates a sketch into a draw-DSL scene at the given canvas size.
func ToScene(s Sketch, w, h int) scene.Scene {
	if w <= 0 {
		w = 64
	}
	if h <= 0 {
		h = 24
	}
	horizon := h * 2 / 3
	var ops []scene.Op
	sky := colv(s.Sky, [3]int{40, 50, 90})
	ops = append(ops, scene.Op{Op: "vgradient", X: 0, Y: 0, W: w, H: h, Glyph: " ", From: p(sky), To: p(add(sky, 70))})
	if s.Ground != "" {
		g := colv(s.Ground, [3]int{20, 30, 60})
		ops = append(ops, scene.Op{Op: "rect", X: 0, Y: horizon, W: w, H: h - horizon, Glyph: " ", Fg: p(g), Bg: p(g)})
	}
	for _, sh := range s.Shapes {
		c := colv(sh.Color, [3]int{240, 240, 245})
		switch sh.Kind {
		case "sun", "moon":
			x, y, r := sh.X, sh.Y, sh.W
			if x == 0 {
				x = w * 3 / 4
			}
			if y == 0 {
				y = h / 4
			}
			if r <= 0 {
				r = 3
			}
			ops = append(ops, scene.Op{Op: "disc", CX: x, CY: y, R: r, Glyph: "█", Fg: p(c)})
		case "hill", "mountain":
			cx := sh.X
			if cx == 0 {
				cx = w / 2
			}
			hh := sh.H
			if hh <= 0 {
				hh = h / 4
			}
			for i := 0; i < hh; i++ {
				half := hh - 1 - i // wide base at the horizon, narrow apex on top
				ops = append(ops, scene.Op{Op: "rect", X: cx - half, Y: horizon - 1 - i, W: 2*half + 1, H: 1, Glyph: "█", Fg: p(c), Bg: p(c)})
			}
		case "building":
			bw, bh := sh.W, sh.H
			if bw <= 0 {
				bw = 4
			}
			if bh <= 0 {
				bh = h / 4
			}
			ops = append(ops, scene.Op{Op: "rect", X: sh.X, Y: horizon - bh, W: bw, H: bh, Glyph: "█", Fg: p(c), Bg: p(c)})
		case "star":
			y := sh.Y
			if y == 0 {
				y = h / 6
			}
			ops = append(ops, scene.Op{Op: "text", X: sh.X, Y: y, S: "★", Fg: p(c)})
		case "cloud":
			y, cw := sh.Y, sh.W
			if y == 0 {
				y = h / 6
			}
			if cw <= 0 {
				cw = 8
			}
			ops = append(ops, scene.Op{Op: "rect", X: sh.X, Y: y, W: cw, H: 1, Glyph: "▀", Fg: p(c)})
		case "band":
			ops = append(ops, scene.Op{Op: "rect", X: 0, Y: sh.Y, W: w, H: max1(sh.H), Glyph: " ", Fg: p(c), Bg: p(c)})
		case "text":
			ops = append(ops, scene.Op{Op: "text", X: sh.X, Y: sh.Y, S: sh.S, Fg: p(c)})
		}
	}
	return scene.Scene{Width: w, Height: h, Ops: ops}
}
