package raydir

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/svend4/infon/pkg/microfont"
)

// q6map.go draws the hexvis-style map of the whole Q6 hypercube (svend4/meta): an
// 8×8 grid of all 64 hexagrams (row = upper trigram, column = lower), each cell
// shaded by its number of yang lines, with the current hexagram outlined and its
// six one-line neighbours dotted — a navigator over the 64 worlds.

// yangCount is the number of yang (solid) lines in a hexagram number.
func yangCount(n int) int {
	c := 0
	for n > 0 {
		c += n & 1
		n >>= 1
	}
	return c
}

// Q6Map renders the 8×8 hexagram grid; `current` is outlined and its neighbours
// dotted.
func Q6Map(current Hexagram, w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 22, A: 255}}, image.Point{}, draw.Src)
	microfont.Draw(img, 6, 6, 1, "Q6: 64 hexagrams (yang lines dark->gold)", color.RGBA{R: 220, G: 220, B: 230, A: 255})

	top := 22
	cell := (w) / 8
	if c2 := (h - top) / 8; c2 < cell {
		cell = c2
	}
	ox := (w - cell*8) / 2
	oy := top
	cur := current.Number()
	neigh := map[int]bool{}
	for _, nb := range current.Neighbors() {
		neigh[nb.Number()] = true
	}
	low := [3]float64{0.14, 0.2, 0.42} // 0 yang lines
	high := [3]float64{1.0, 0.85, 0.4} // 6 yang lines
	for row := 0; row < 8; row++ {     // upper trigram
		for col := 0; col < 8; col++ { // lower trigram
			num := (row << 3) | col
			t := float64(yangCount(num)) / 6
			c := color.RGBA{
				R: uint8((low[0]*(1-t) + high[0]*t) * 255),
				G: uint8((low[1]*(1-t) + high[1]*t) * 255),
				B: uint8((low[2]*(1-t) + high[2]*t) * 255),
				A: 255,
			}
			x0, y0 := ox+col*cell, oy+row*cell
			fillRect(img, x0+1, y0+1, cell-2, cell-2, c)
			if neigh[num] {
				r := cell / 8
				if r < 2 {
					r = 2
				}
				fillCircle(img, x0+cell/2, y0+cell/2, r, color.RGBA{R: 240, G: 240, B: 245, A: 255})
			}
			if num == cur { // bright outline on the current hexagram
				white := color.RGBA{R: 250, G: 250, B: 255, A: 255}
				fillRect(img, x0, y0, cell, 2, white)
				fillRect(img, x0, y0+cell-2, cell, 2, white)
				fillRect(img, x0, y0, 2, cell, white)
				fillRect(img, x0+cell-2, y0, 2, cell, white)
			}
		}
	}
	return img
}
