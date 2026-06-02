// Package relief is path 2 of the pseudo-3D program: 2.5D shading of a marks
// grid. A tangram Figure is a glyph+colour per cell; relief adds one height byte
// per cell and extrudes each glyph into an isometric block, so the diagonal
// glyphs (tri-ul/ur/ll/lr) become slanted facets that catch the light - an
// isometric "voxel / Q*bert" view of the same grid. It reuses the path-1
// isometric renderer (fold.Face / fold.Render), the first of the cross-path
// overlaps.
package relief

import (
	"image"
	"math"

	"github.com/svend4/infon/pkg/fold"
	"github.com/svend4/infon/pkg/tangram"
)

// ZScale maps a height byte 0..255 onto world z 0..ZScale.
const ZScale = 2.6

// glyphPoly returns the unit-cell outline (0..1 coords, y down) of a glyph.
func glyphPoly(name string) [][2]float64 {
	switch name {
	case "tri-ul": // upper-left, right angle at (0,0)
		return [][2]float64{{0, 0}, {1, 0}, {0, 1}}
	case "tri-ur": // upper-right, right angle at (1,0)
		return [][2]float64{{0, 0}, {1, 0}, {1, 1}}
	case "tri-ll": // lower-left, right angle at (0,1)
		return [][2]float64{{0, 0}, {0, 1}, {1, 1}}
	case "tri-lr": // lower-right, right angle at (1,1)
		return [][2]float64{{1, 0}, {1, 1}, {0, 1}}
	case "top":
		return [][2]float64{{0, 0}, {1, 0}, {1, 0.5}, {0, 0.5}}
	case "bottom":
		return [][2]float64{{0, 0.5}, {1, 0.5}, {1, 1}, {0, 1}}
	case "left":
		return [][2]float64{{0, 0}, {0.5, 0}, {0.5, 1}, {0, 1}}
	case "right":
		return [][2]float64{{0.5, 0}, {1, 0}, {1, 1}, {0.5, 1}}
	default: // full, diag-*, anything else -> the whole cell
		return [][2]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	}
}

// Faces builds isometric extrusion faces for a figure given a per-cell height
// map (height[y][x] is a byte; 0 means no block). Each cell becomes a top face
// in its glyph shape plus shaded side walls.
func Faces(f tangram.Figure, height [][]uint8) []fold.Face {
	var faces []fold.Face
	for _, p := range f.Pieces {
		y, x := p.Y, p.X
		var b uint8 = 110
		if y >= 0 && y < len(height) && x >= 0 && x < len(height[y]) {
			b = height[y][x]
		}
		if b == 0 {
			continue
		}
		h := ZScale * float64(b) / 255
		poly := glyphPoly(p.Glyph)
		n := len(poly)
		var cx, cy float64
		top := make([]fold.Pt3, n)
		for i, pt := range poly {
			top[i] = fold.Pt3{X: float64(x) + pt[0], Y: float64(y) + pt[1], Z: h}
			cx += pt[0]
			cy += pt[1]
		}
		cx /= float64(n)
		cy /= float64(n)
		faces = append(faces, fold.Face{Pts: top, Color: p.Color, Shade: 1.0})
		for i := 0; i < n; i++ {
			a, bb := poly[i], poly[(i+1)%n]
			ax, ay := float64(x)+a[0], float64(y)+a[1]
			bx, by := float64(x)+bb[0], float64(y)+bb[1]
			quad := []fold.Pt3{{X: ax, Y: ay, Z: 0}, {X: bx, Y: by, Z: 0}, {X: bx, Y: by, Z: h}, {X: ax, Y: ay, Z: h}}
			ex, ey := bb[0]-a[0], bb[1]-a[1]
			nx, ny := ey, -ex
			mx, my := (a[0]+bb[0])/2-cx, (a[1]+bb[1])/2-cy
			if nx*mx+ny*my < 0 {
				nx, ny = -nx, -ny
			}
			ln := math.Hypot(nx, ny)
			if ln == 0 {
				ln = 1
			}
			d := (nx*(-1) + ny*(-0.4)) / ln
			faces = append(faces, fold.Face{Pts: quad, Color: p.Color, Shade: 0.5 + 0.32*((d+1)/2)})
		}
	}
	return faces
}

// Render rasterises the figure as a 2.5D isometric relief.
func Render(f tangram.Figure, height [][]uint8, pxW, pxH int) image.Image {
	return fold.Render(Faces(f, height), pxW, pxH)
}

// Uniform is an H x W height map set to b at every occupied cell.
func Uniform(f tangram.Figure, b uint8) [][]uint8 {
	h := make([][]uint8, f.H)
	for y := range h {
		h[y] = make([]uint8, f.W)
	}
	for _, p := range f.Pieces {
		if p.Y >= 0 && p.Y < f.H && p.X >= 0 && p.X < f.W {
			h[p.Y][p.X] = b
		}
	}
	return h
}

// Gradient ramps the height from lo (top row) to hi (bottom row) at occupied
// cells - a cheap way to give the relief a tilt.
func Gradient(f tangram.Figure, lo, hi uint8) [][]uint8 {
	h := make([][]uint8, f.H)
	for y := range h {
		h[y] = make([]uint8, f.W)
	}
	for _, p := range f.Pieces {
		if p.Y < 0 || p.Y >= f.H || p.X < 0 || p.X >= f.W {
			continue
		}
		t := 0.0
		if f.H > 1 {
			t = float64(p.Y) / float64(f.H-1)
		}
		h[p.Y][p.X] = uint8(float64(lo) + t*(float64(hi)-float64(lo)))
	}
	return h
}

// SpecBytes is the size of the height map (one byte per grid cell) - the extra
// data 2.5D costs over the flat marks grid.
func SpecBytes(f tangram.Figure) int { return f.H * f.W }
