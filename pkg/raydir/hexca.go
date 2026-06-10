package raydir

import (
	"image"
	"image/color"
	"math/rand"
)

// hexca.go is a cellular automaton on the hexagrams, after svend4/meta's hexca:
// every cell holds one of the 64 hexagram states (a 6-bit Q6 vertex) and evolves
// by a per-line neighbour vote — for each of the six lines, a cell takes the
// majority of its four neighbours' lines, keeping its own on a tie. The six lines
// evolve as six coupled majority automata, so yang and yin organise into drifting
// regions: a living Q6 pattern. Seeded and deterministic.

// HexCA is a toroidal grid of hexagram-state cells.
type HexCA struct {
	W, H  int
	cells []uint8
}

// NewHexCA seeds a w×h grid with random hexagram states (0..63).
func NewHexCA(w, h int, seed int64) *HexCA {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	rng := rand.New(rand.NewSource(seed))
	c := &HexCA{W: w, H: h, cells: make([]uint8, w*h)}
	for i := range c.cells {
		c.cells[i] = uint8(rng.Intn(64))
	}
	return c
}

// At reads the state at (x,y) with toroidal wrap.
func (c *HexCA) At(x, y int) uint8 {
	x = ((x % c.W) + c.W) % c.W
	y = ((y % c.H) + c.H) % c.H
	return c.cells[y*c.W+x]
}

// Grid returns the raw cell states (row-major), for inspection.
func (c *HexCA) Grid() []uint8 { return c.cells }

// Step advances the automaton once: each line is the majority of the four von
// Neumann neighbours' lines (ties keep the cell's own line).
func (c *HexCA) Step() {
	next := make([]uint8, len(c.cells))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			self := c.At(x, y)
			nb := [4]uint8{c.At(x, y-1), c.At(x, y+1), c.At(x-1, y), c.At(x+1, y)}
			var ns uint8
			for bit := 0; bit < 6; bit++ {
				cnt := 0
				for _, n := range nb {
					if n&(1<<bit) != 0 {
						cnt++
					}
				}
				set := self&(1<<bit) != 0 // tie (2-2): keep own
				if cnt > 2 {
					set = true
				} else if cnt < 2 {
					set = false
				}
				if set {
					ns |= 1 << bit
				}
			}
			next[y*c.W+x] = ns
		}
	}
	c.cells = next
}

// Render draws the grid with cells `px` pixels wide, shaded by yang-line count
// (dark blue 0 .. gold 6) — the same scale as the Q6 map.
func (c *HexCA) Render(px int) image.Image {
	if px < 1 {
		px = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, c.W*px, c.H*px))
	low := [3]float64{0.12, 0.16, 0.4}
	high := [3]float64{1.0, 0.85, 0.35}
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			t := float64(yangCount(int(c.cells[y*c.W+x]))) / 6
			col := color.RGBA{
				R: uint8((low[0]*(1-t) + high[0]*t) * 255),
				G: uint8((low[1]*(1-t) + high[1]*t) * 255),
				B: uint8((low[2]*(1-t) + high[2]*t) * 255),
				A: 255,
			}
			fillRect(img, x*px, y*px, px, px, col)
		}
	}
	return img
}
