// Package tangram7 is the AUTHENTIC seven-piece geometric tangram: two large
// triangles, one medium, two small, the square and the parallelogram — as
// continuous polygons with free 45-degree rotation and placement, rendered as
// filled silhouettes. The canonical Square() assembly is verified to tile a
// 4x4 square exactly (area 16, no overlap), so the engine is grounded in the
// real dissection, not a grid approximation.
package tangram7

import (
	"image"
	stdcolor "image/color"
	"math"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/pseudo"
)

// Pt is a 2-D point; Poly is a simple polygon (CCW or CW).
type Pt struct{ X, Y float64 }
type Poly []Pt

// Piece is one named, coloured tangram polygon.
type Piece struct {
	Name  string
	Color string
	Shape Poly
}

// Square returns the seven pieces arranged as the classic 4x4 square (the
// canonical solution), each in a distinct colour.
func Square() []Piece {
	return []Piece{
		{"large1", "blue", Poly{{4, 4}, {4, 0}, {2, 2}}},
		{"large2", "teal", Poly{{0, 4}, {4, 4}, {2, 2}}},
		{"medium", "gold", Poly{{0, 0}, {2, 0}, {0, 2}}},
		{"square", "red", Poly{{2, 0}, {3, 1}, {2, 2}, {1, 1}}},
		{"parallelogram", "orange", Poly{{0, 2}, {0, 4}, {1, 3}, {1, 1}}},
		{"small1", "green", Poly{{4, 0}, {2, 0}, {3, 1}}},
		{"small2", "pink", Poly{{1, 1}, {2, 2}, {1, 3}}},
	}
}

// Transform rotates the polygon by rot*45 degrees (optionally mirrored in x
// first) and translates it. The tangram only ever uses 45-degree steps.
func (poly Poly) Transform(rot int, flip bool, tx, ty float64) Poly {
	a := float64(rot) * math.Pi / 4
	cs, sn := math.Cos(a), math.Sin(a)
	out := make(Poly, len(poly))
	for i, p := range poly {
		x := p.X
		if flip {
			x = -x
		}
		y := p.Y
		out[i] = Pt{x*cs - y*sn + tx, x*sn + y*cs + ty}
	}
	return out
}

// Place applies a transform to a whole figure (all pieces).
func Place(pieces []Piece, rot int, flip bool, tx, ty float64) []Piece {
	out := make([]Piece, len(pieces))
	for i, p := range pieces {
		out[i] = Piece{p.Name, p.Color, p.Shape.Transform(rot, flip, tx, ty)}
	}
	return out
}

// Tray lays the seven pieces out separately (an exploded view) so each distinct
// shape is visible.
func Tray() []Piece {
	sq := Square()
	pos := []Pt{{0, 0}, {5, 0}, {10, 0}, {0, 5}, {4, 5}, {8, 5}, {11, 5}}
	out := make([]Piece, len(sq))
	for i, p := range sq {
		// move each piece's centroid to a slot
		cx, cy := centroid(p.Shape)
		out[i] = Piece{p.Name, p.Color, p.Shape.Transform(0, false, pos[i].X-cx+1, pos[i].Y-cy+1)}
	}
	return out
}

func centroid(p Poly) (float64, float64) {
	var sx, sy float64
	for _, q := range p {
		sx += q.X
		sy += q.Y
	}
	return sx / float64(len(p)), sy / float64(len(p))
}

func polyArea(p Poly) float64 {
	a := 0.0
	n := len(p)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		a += p[i].X*p[j].Y - p[j].X*p[i].Y
	}
	return math.Abs(a) / 2
}

func inPoly(p Poly, x, y float64) bool {
	in := false
	n := len(p)
	j := n - 1
	for i := 0; i < n; i++ {
		if (p[i].Y > y) != (p[j].Y > y) &&
			x < (p[j].X-p[i].X)*(y-p[i].Y)/(p[j].Y-p[i].Y)+p[i].X {
			in = !in
		}
		j = i
	}
	return in
}

func bbox(pieces []Piece) (minx, miny, maxx, maxy float64) {
	minx, miny = math.Inf(1), math.Inf(1)
	maxx, maxy = math.Inf(-1), math.Inf(-1)
	for _, pc := range pieces {
		for _, q := range pc.Shape {
			minx, miny = math.Min(minx, q.X), math.Min(miny, q.Y)
			maxx, maxy = math.Max(maxx, q.X), math.Max(maxy, q.Y)
		}
	}
	return
}

// Report is the outcome of Verify.
type Report struct {
	Pieces  int
	Area    float64
	Overlap float64 // fraction of sampled covered points hit by 2+ pieces
}

// Verify reports total polygon area and the overlap fraction (sampled). A valid
// assembly has ~0 overlap.
func Verify(pieces []Piece, samples int) Report {
	r := Report{Pieces: len(pieces)}
	for _, p := range pieces {
		r.Area += polyArea(p.Shape)
	}
	minx, miny, maxx, maxy := bbox(pieces)
	covered, over := 0, 0
	for iy := 0; iy < samples; iy++ {
		for ix := 0; ix < samples; ix++ {
			x := minx + (float64(ix)+0.5)/float64(samples)*(maxx-minx)
			y := miny + (float64(iy)+0.5)/float64(samples)*(maxy-miny)
			c := 0
			for _, p := range pieces {
				if inPoly(p.Shape, x, y) {
					c++
				}
			}
			if c >= 1 {
				covered++
			}
			if c >= 2 {
				over++
			}
		}
	}
	if covered > 0 {
		r.Overlap = float64(over) / float64(covered)
	}
	return r
}

// Render rasterises the pieces to a pxW x pxH image, fitting their bounding box,
// each piece filled in its colour with a dark edge.
func Render(pieces []Piece, pxW, pxH int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	bg := stdcolor.RGBA{245, 245, 248, 255}
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	draw := func(x, y int, c stdcolor.RGBA) {
		if x >= 0 && x < pxW && y >= 0 && y < pxH {
			img.SetRGBA(x, y, c)
		}
	}
	for y := 0; y < pxH; y++ {
		for x := 0; x < pxW; x++ {
			img.SetRGBA(x, y, bg)
		}
	}
	minx, miny, maxx, maxy := bbox(pieces)
	const m = 18.0
	sc := math.Min((float64(pxW)-2*m)/(maxx-minx), (float64(pxH)-2*m)/(maxy-miny))
	ox := m + (float64(pxW)-2*m-sc*(maxx-minx))/2
	oy := m + (float64(pxH)-2*m-sc*(maxy-miny))/2
	toPix := func(p Pt) (float64, float64) { return ox + (p.X-minx)*sc, oy + (p.Y-miny)*sc }

	for _, pc := range pieces {
		col := color.RGB{}
		col = pseudo.Color(pc.Color, color.RGB{R: 120, G: 120, B: 130})
		px := make([]Pt, len(pc.Shape))
		var mnx, mny, mxx, mxy = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for i, q := range pc.Shape {
			x, y := toPix(q)
			px[i] = Pt{x, y}
			mnx, mny = math.Min(mnx, x), math.Min(mny, y)
			mxx, mxy = math.Max(mxx, x), math.Max(mxy, y)
		}
		for y := int(mny); y <= int(mxy); y++ {
			for x := int(mnx); x <= int(mxx); x++ {
				if inPoly(Poly(px), float64(x)+0.5, float64(y)+0.5) {
					draw(x, y, stdcolor.RGBA{col.R, col.G, col.B, 255})
				}
			}
		}
		ink := stdcolor.RGBA{20, 24, 36, 255}
		for i := range px {
			line(draw, px[i], px[(i+1)%len(px)], ink)
		}
	}
	return img
}

func line(draw func(int, int, stdcolor.RGBA), a, b Pt, c stdcolor.RGBA) {
	x0, y0, x1, y1 := int(a.X), int(a.Y), int(b.X), int(b.Y)
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	err := dx + dy
	for {
		draw(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func sign(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}
