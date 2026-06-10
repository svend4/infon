package raydir

import (
	"image"
	"math"

	"github.com/svend4/infon/pkg/brain"
)

// imagine.go runs the project's idea in reverse: it extracts MEANING from pixels.
// SceneFromImage reads a picture and composes a rayscene — sky and ground from the
// top and bottom bands, a sun from the brightest patch, and a few coloured forms
// from the dominant colours — so you can walk into a 3-D world derived from a
// photo. Offline and deterministic (heuristic); a live vision model can return a
// richer rayscene through the same SceneSpec.

type irgb struct{ r, g, b float64 }

func (c irgb) arr() [3]float64 { return [3]float64{c.r, c.g, c.b} }
func (c irgb) lum() float64    { return 0.2126*c.r + 0.7152*c.g + 0.0722*c.b }
func (c irgb) sat() float64 {
	mx := math.Max(c.r, math.Max(c.g, c.b))
	mn := math.Min(c.r, math.Min(c.g, c.b))
	return mx - mn
}
func (c irgb) dist(o irgb) float64 {
	return math.Abs(c.r-o.r) + math.Abs(c.g-o.g) + math.Abs(c.b-o.b)
}
func (c irgb) scale(s float64) irgb { return irgb{c.r * s, c.g * s, c.b * s} }

// SceneFromImage analyses img and returns a SceneSpec capturing its mood.
func SceneFromImage(img image.Image) brain.SceneSpec {
	b := img.Bounds()
	W, H := b.Dx(), b.Dy()
	if W < 2 || H < 2 {
		return fallbackSpec()
	}
	const gw, gh = 24, 16
	grid := make([][]irgb, gh)
	for gy := 0; gy < gh; gy++ {
		grid[gy] = make([]irgb, gw)
		for gx := 0; gx < gw; gx++ {
			x0, x1 := b.Min.X+gx*W/gw, b.Min.X+(gx+1)*W/gw
			y0, y1 := b.Min.Y+gy*H/gh, b.Min.Y+(gy+1)*H/gh
			var sr, sg, sb, n float64
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					r, g, bl, _ := img.At(x, y).RGBA()
					sr += float64(r) / 65535
					sg += float64(g) / 65535
					sb += float64(bl) / 65535
					n++
				}
			}
			if n > 0 {
				grid[gy][gx] = irgb{sr / n, sg / n, sb / n}
			}
		}
	}
	band := func(y0, y1 int) irgb {
		var s irgb
		var n float64
		for gy := y0; gy < y1; gy++ {
			for gx := 0; gx < gw; gx++ {
				s.r += grid[gy][gx].r
				s.g += grid[gy][gx].g
				s.b += grid[gy][gx].b
				n++
			}
		}
		if n == 0 {
			return irgb{}
		}
		return irgb{s.r / n, s.g / n, s.b / n}
	}

	spec := brain.SceneSpec{
		Light:  [3]float64{6, 9, -4},
		SkyTop: band(0, gh/6).arr(),
		SkyBot: band(gh/4, gh*2/5).arr(),
		Objects: []brain.ObjSpec{
			{Kind: "plane", Color: band(gh*4/5, gh).arr()},
		},
	}

	// sun: the brightest cell in the upper half becomes an emissive sphere there.
	bestL, bx, by := 0.0, -1, -1
	for gy := 0; gy < gh/2; gy++ {
		for gx := 0; gx < gw; gx++ {
			if l := grid[gy][gx].lum(); l > bestL {
				bestL, bx, by = l, gx, gy
			}
		}
	}
	if bestL > 0.6 {
		sx := (float64(bx)/float64(gw-1) - 0.5) * 14
		sy := 5 + (1-float64(by)/float64(gh/2))*4
		spec.Objects = append(spec.Objects, brain.ObjSpec{
			X: sx, Y: sy, Z: 3, R: 0.8, Emit: grid[by][bx].scale(20).arr(),
		})
	}

	// objects: the most saturated, mutually-distinct colours in the lower-middle
	// band become coloured spheres standing on the ground.
	var picks []irgb
	var px []float64
	for gy := gh / 2; gy < gh*4/5 && len(picks) < 4; gy++ {
		for gx := 0; gx < gw && len(picks) < 4; gx++ {
			c := grid[gy][gx]
			if c.sat() < 0.22 {
				continue
			}
			dup := false
			for _, p := range picks {
				if p.dist(c) < 0.3 {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			picks = append(picks, c)
			px = append(px, (float64(gx)/float64(gw-1)-0.5)*10)
		}
	}
	for i, c := range picks {
		spec.Objects = append(spec.Objects, brain.ObjSpec{
			X: px[i], Y: 1, Z: 4 + float64(i)*1.6, R: 1, Color: c.arr(), Rough: 0.4,
		})
	}
	if !hasRenderable(spec) { // always leave something to look at
		spec.Objects = append(spec.Objects, brain.ObjSpec{X: 0, Y: 1, Z: 5, R: 1, Color: [3]float64{0.7, 0.7, 0.72}})
	}
	spec.Name = "Imagined"
	return spec
}
