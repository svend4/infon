package raydir

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raytrace"
)

// cartograph.go draws the dream world from above, the way the hackers' maps look:
// the ground you've walked is filled in, the rest is "white spots" (белые пятна,
// terra incognita) waiting to be explored, a boundary rings the known world, and
// the named places are marked — with the archetypal "reserve" and "prison" called
// out specially. Exploration is a coarse fog-of-war grid revealed as you walk.

// Cartograph is a fog-of-war record of where the walker has been.
type Cartograph struct {
	cell float64
	seen map[[2]int]bool
}

// NewCartograph makes a cartograph with the given fog cell size (world units).
func NewCartograph(cell float64) *Cartograph {
	if cell <= 0 {
		cell = 4
	}
	return &Cartograph{cell: cell, seen: map[[2]int]bool{}}
}

func (c *Cartograph) key(x, z float64) [2]int {
	return [2]int{int(math.Floor(x / c.cell)), int(math.Floor(z / c.cell))}
}

// Reveal marks every cell within `radius` of p as explored.
func (c *Cartograph) Reveal(p raytrace.Vec3, radius float64) {
	if radius < c.cell {
		radius = c.cell
	}
	steps := int(radius/c.cell) + 1
	base := c.key(p.X, p.Z)
	for dz := -steps; dz <= steps; dz++ {
		for dx := -steps; dx <= steps; dx++ {
			cx := (float64(base[0]+dx) + 0.5) * c.cell
			cz := (float64(base[1]+dz) + 0.5) * c.cell
			if math.Hypot(cx-p.X, cz-p.Z) <= radius {
				c.seen[[2]int{base[0] + dx, base[1] + dz}] = true
			}
		}
	}
}

// Seen reports whether the cell containing p has been explored.
func (c *Cartograph) Seen(p raytrace.Vec3) bool { return c.seen[c.key(p.X, p.Z)] }

// Count is the number of explored cells.
func (c *Cartograph) Count() int { return len(c.seen) }

// isSpecialPlace recognises the archetypal landmarks the hackers single out.
func isSpecialPlace(name string) (color.RGBA, bool) {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "reserve") || strings.Contains(n, "заповед") || strings.Contains(n, "sanctuary"):
		return color.RGBA{R: 240, G: 200, B: 80, A: 255}, true // where things accumulate — gold
	case strings.Contains(n, "prison") || strings.Contains(n, "тюрьм") || strings.Contains(n, "vault"):
		return color.RGBA{R: 220, G: 80, B: 80, A: 255}, true // where things are lost — red
	}
	return color.RGBA{}, false
}

// Render draws the bird's-eye map: explored ground filled, unexplored as white
// spots, a boundary frame, the named places (specials called out), and you.
func (c *Cartograph) Render(marks []Landmark, self raytrace.Vec3, w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 18, G: 18, B: 20, A: 255}}, image.Point{}, draw.Src)

	// world bounds over everything shown, padded.
	minX, maxX := self.X, self.X
	minZ, maxZ := self.Z, self.Z
	acc := func(x, z float64) {
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minZ, maxZ = math.Min(minZ, z), math.Max(maxZ, z)
	}
	for k := range c.seen {
		acc(float64(k[0])*c.cell, float64(k[1])*c.cell)
	}
	for _, m := range marks {
		acc(m.At.X, m.At.Z)
	}
	padW := math.Max(c.cell*2, (maxX-minX)*0.08)
	padH := math.Max(c.cell*2, (maxZ-minZ)*0.08)
	minX, maxX, minZ, maxZ = minX-padW, maxX+padW, minZ-padH, maxZ+padH
	const margin = 18
	sx := func(x float64) int {
		if maxX == minX {
			return w / 2
		}
		return margin + int((x-minX)/(maxX-minX)*float64(w-2*margin))
	}
	sy := func(z float64) int {
		if maxZ == minZ {
			return h / 2
		}
		return margin + int((z-minZ)/(maxZ-minZ)*float64(h-2*margin))
	}

	// fill the map area: explored cells -> ground; the rest -> white spots.
	ground := color.RGBA{R: 42, G: 66, B: 46, A: 255}
	whiteSpot := color.RGBA{R: 206, G: 202, B: 188, A: 255}
	for py := margin; py < h-margin; py++ {
		for px := margin; px < w-margin; px++ {
			wx := minX + (float64(px-margin)+0.5)/float64(w-2*margin)*(maxX-minX)
			wz := minZ + (float64(py-margin)+0.5)/float64(h-2*margin)*(maxZ-minZ)
			if c.seen[c.key(wx, wz)] {
				img.SetRGBA(px, py, ground)
			} else {
				img.SetRGBA(px, py, whiteSpot)
			}
		}
	}
	// boundary frame (the known world's mountain/sea edge).
	border := color.RGBA{R: 96, G: 74, B: 46, A: 255}
	for t := 0; t < 3; t++ {
		for px := margin - t; px < w-margin+t; px++ {
			img.SetRGBA(px, margin-t, border)
			img.SetRGBA(px, h-margin+t-1, border)
		}
		for py := margin - t; py < h-margin+t; py++ {
			img.SetRGBA(margin-t, py, border)
			img.SetRGBA(w-margin+t-1, py, border)
		}
	}
	// landmarks
	for _, m := range marks {
		px, py := sx(m.At.X), sy(m.At.Z)
		col := color.RGBA{R: 90, G: 150, B: 230, A: 255}
		rad := 3
		if sc, ok := isSpecialPlace(m.Name); ok {
			col, rad = sc, 5
		}
		fillCircle(img, px, py, rad, col)
		microfont.Draw(img, px+rad+2, py-3, 1, m.Name, color.RGBA{R: 30, G: 30, B: 34, A: 255})
	}
	// you, and a compass (N up).
	fillCircle(img, sx(self.X), sy(self.Z), 4, color.RGBA{R: 90, G: 220, B: 240, A: 255})
	microfont.Draw(img, w/2-3, margin+2, 1, "N", border)
	return img
}
