// Package fold lifts a flat tangram7 figure into a pseudo-3D relief: each piece
// is extruded to its own height and the solid is drawn in isometric projection.
// Folding is an animation parameter t in [0,1] - at t=0 the figure is the
// original 2D tangram, at t=1 every piece has risen into a 3D block. The whole
// animation is described by a handful of numbers (FoldSpec), so a pseudo-3D
// scene ships in the same few hundred bytes as a semantic-video frame.
package fold

import (
	"image"
	stdcolor "image/color"
	"math"
	"sort"
	"strings"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/pseudo"
	t7 "github.com/svend4/infon/pkg/tangram7"
)

// Pt3 is a 3-D point (z up).
type Pt3 struct{ X, Y, Z float64 }

// Face is a flat polygon in 3-D with a colour and a light multiplier.
type Face struct {
	Pts   []Pt3
	Color string
	Shade float64
}

// iso projects a 3-D point to 2-D screen coords (2:1 isometric, z up).
func iso(p Pt3) (float64, float64) {
	const c = 0.8660254038 // cos 30
	const s = 0.5          // sin 30
	return (p.X - p.Y) * c, (p.X+p.Y)*s - p.Z
}

// FoldSpec is the whole animation as data: one target relief height per piece.
// Pixels are derived; the scene itself is just these numbers.
type FoldSpec struct {
	Names   []string  `json:"names"`
	Heights []float64 `json:"heights"`
}

// SpecFor returns the target heights for a figure (varied by piece kind so the
// folded figure reads as a relief, not a uniform slab).
func SpecFor(pieces []t7.Piece) FoldSpec {
	fs := FoldSpec{}
	for _, p := range pieces {
		fs.Names = append(fs.Names, p.Name)
		fs.Heights = append(fs.Heights, heightFor(p.Name))
	}
	return fs
}

func heightFor(name string) float64 {
	switch {
	case strings.HasPrefix(name, "large"):
		return 1.6
	case name == "medium":
		return 1.2
	case name == "square":
		return 1.0
	case name == "parallelogram":
		return 0.8
	default: // small1, small2, anything else
		return 0.6
	}
}

// Frame builds the folded solid for animation parameter t in [0,1]: every piece
// rises from the flat plane (t=0) to its full relief height (t=1).
func Frame(pieces []t7.Piece, t float64) []Face {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	var faces []Face
	for _, p := range pieces {
		h := heightFor(p.Name) * t
		n := len(p.Shape)
		if n < 3 {
			continue
		}
		top := make([]Pt3, n)
		var cx, cy float64
		for i, q := range p.Shape {
			top[i] = Pt3{q.X, q.Y, h}
			cx += q.X
			cy += q.Y
		}
		cx /= float64(n)
		cy /= float64(n)
		faces = append(faces, Face{Pts: top, Color: p.Color, Shade: 1.0})
		for i := 0; i < n; i++ {
			a, b := p.Shape[i], p.Shape[(i+1)%n]
			quad := []Pt3{{a.X, a.Y, 0}, {b.X, b.Y, 0}, {b.X, b.Y, h}, {a.X, a.Y, h}}
			ex, ey := b.X-a.X, b.Y-a.Y
			nx, ny := ey, -ex
			mx, my := (a.X+b.X)/2-cx, (a.Y+b.Y)/2-cy
			if nx*mx+ny*my < 0 {
				nx, ny = -nx, -ny
			}
			ln := math.Hypot(nx, ny)
			if ln == 0 {
				ln = 1
			}
			d := (nx*(-1) + ny*(-0.4)) / ln // light from upper-left
			faces = append(faces, Face{Pts: quad, Color: p.Color, Shade: 0.55 + 0.30*((d+1)/2)})
		}
	}
	return faces
}

// Render draws the faces in fixed isometric projection.
func Render(faces []Face, pxW, pxH int) image.Image { return RenderCam(faces, pxW, pxH, 0) }

// RenderCam is Render with a camera yaw: the whole scene is rotated about its
// vertical (z) axis by yaw radians before the isometric projection, turning the
// pseudo-3D world into a turntable. The camera is just one more number - a byte
// in the frame record - so rotation costs nothing in the codec.
func RenderCam(faces []Face, pxW, pxH int, yaw float64) image.Image {
	faces = rotZ(faces, yaw)
	img := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	bg := stdcolor.RGBA{245, 245, 248, 255}
	for y := 0; y < pxH; y++ {
		for x := 0; x < pxW; x++ {
			img.SetRGBA(x, y, bg)
		}
	}
	if len(faces) == 0 {
		return img
	}
	type pf struct {
		pts [][2]float64
		col stdcolor.RGBA
		key float64
	}
	var pfs []pf
	minx, miny, maxx, maxy := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, f := range faces {
		pts := make([][2]float64, len(f.Pts))
		var kx, ky, kz float64
		for i, p := range f.Pts {
			sx, sy := iso(p)
			pts[i] = [2]float64{sx, sy}
			kx += p.X
			ky += p.Y
			kz += p.Z
			minx, miny = math.Min(minx, sx), math.Min(miny, sy)
			maxx, maxy = math.Max(maxx, sx), math.Max(maxy, sy)
		}
		nn := float64(len(f.Pts))
		base := pseudo.Color(f.Color, color.RGB{R: 140, G: 140, B: 150})
		col := stdcolor.RGBA{
			uint8(math.Min(255, float64(base.R)*f.Shade)),
			uint8(math.Min(255, float64(base.G)*f.Shade)),
			uint8(math.Min(255, float64(base.B)*f.Shade)), 255,
		}
		pfs = append(pfs, pf{pts: pts, col: col, key: (kx + ky + kz) / nn})
	}
	sort.SliceStable(pfs, func(i, j int) bool { return pfs[i].key < pfs[j].key })

	const m = 26.0
	sc := math.Min((float64(pxW)-2*m)/(maxx-minx), (float64(pxH)-2*m)/(maxy-miny))
	ox := m + (float64(pxW)-2*m-sc*(maxx-minx))/2
	oy := m + (float64(pxH)-2*m-sc*(maxy-miny))/2
	ink := stdcolor.RGBA{20, 24, 36, 255}
	for _, f := range pfs {
		sp := make([][2]float64, len(f.pts))
		mnx, mny, mxx, mxy := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for i, p := range f.pts {
			x, y := ox+(p[0]-minx)*sc, oy+(p[1]-miny)*sc
			sp[i] = [2]float64{x, y}
			mnx, mny = math.Min(mnx, x), math.Min(mny, y)
			mxx, mxy = math.Max(mxx, x), math.Max(mxy, y)
		}
		for y := int(mny); y <= int(mxy); y++ {
			for x := int(mnx); x <= int(mxx); x++ {
				if inPoly(sp, float64(x)+0.5, float64(y)+0.5) && x >= 0 && x < pxW && y >= 0 && y < pxH {
					img.SetRGBA(x, y, f.col)
				}
			}
		}
		for i := range sp {
			line(img, sp[i], sp[(i+1)%len(sp)], ink)
		}
	}
	return img
}

func inPoly(p [][2]float64, x, y float64) bool {
	in, n := false, len(p)
	j := n - 1
	for i := 0; i < n; i++ {
		if (p[i][1] > y) != (p[j][1] > y) &&
			x < (p[j][0]-p[i][0])*(y-p[i][1])/(p[j][1]-p[i][1])+p[i][0] {
			in = !in
		}
		j = i
	}
	return in
}

func line(img *image.RGBA, a, b [2]float64, c stdcolor.RGBA) {
	x0, y0, x1, y1 := int(a[0]), int(a[1]), int(b[0]), int(b[1])
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	err := dx + dy
	b2 := img.Bounds()
	for {
		if x0 >= b2.Min.X && x0 < b2.Max.X && y0 >= b2.Min.Y && y0 < b2.Max.Y {
			img.SetRGBA(x0, y0, c)
		}
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

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
func sign(a int) int {
	if a > 0 {
		return 1
	}
	if a < 0 {
		return -1
	}
	return 0
}

// rotZ rotates every face about the scene's XY centroid by yaw (radians), the
// camera-yaw transform applied before projection.
func rotZ(faces []Face, yaw float64) []Face {
	if yaw == 0 {
		return faces
	}
	var cx, cy float64
	n := 0
	for _, f := range faces {
		for _, p := range f.Pts {
			cx += p.X
			cy += p.Y
			n++
		}
	}
	if n == 0 {
		return faces
	}
	cx /= float64(n)
	cy /= float64(n)
	cs, sn := math.Cos(yaw), math.Sin(yaw)
	out := make([]Face, len(faces))
	for i, f := range faces {
		pts := make([]Pt3, len(f.Pts))
		for j, p := range f.Pts {
			dx, dy := p.X-cx, p.Y-cy
			pts[j] = Pt3{X: cx + dx*cs - dy*sn, Y: cy + dx*sn + dy*cs, Z: p.Z}
		}
		out[i] = Face{Pts: pts, Color: f.Color, Shade: f.Shade}
	}
	return out
}
