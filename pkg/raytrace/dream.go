// dream.go is a "lens & film" post-process that pushes a frame toward the
// dreamlike: chromatic aberration (the colour channels drift apart toward the
// edges, like a cheap lens), barrel distortion (a curved, fish-eyed field), film
// grain, and a soft vignette. It is screen-space and deterministic (seeded grain),
// so it composes after any render or grade.
package raytrace

import (
	"image"
	"math"
)

// DreamOptions configures the dream post (Dream).
type DreamOptions struct {
	Chroma   float64 // radial RGB separation toward the edges (0 none .. ~0.02)
	Grain    float64 // film grain amount (0 none .. ~0.1)
	Vignette float64 // corner darkening (0..1)
	Distort  float64 // barrel (+) / pincushion (-) lens distortion (0 none)
	Seed     uint32  // grain seed (vary per frame for a shimmer)
}

// bilinear samples a display-space buffer at fractional (fx,fy), edge-clamped.
func bilinear(buf []Vec3, w, h int, fx, fy float64) Vec3 {
	if fx < 0 {
		fx = 0
	} else if fx > float64(w-1) {
		fx = float64(w - 1)
	}
	if fy < 0 {
		fy = 0
	} else if fy > float64(h-1) {
		fy = float64(h - 1)
	}
	x0, y0 := int(fx), int(fy)
	x1, y1 := x0+1, y0+1
	if x1 > w-1 {
		x1 = w - 1
	}
	if y1 > h-1 {
		y1 = h - 1
	}
	tx, ty := fx-float64(x0), fy-float64(y0)
	a := buf[y0*w+x0].Scale(1 - tx).Add(buf[y0*w+x1].Scale(tx))
	b := buf[y1*w+x0].Scale(1 - tx).Add(buf[y1*w+x1].Scale(tx))
	return a.Scale(1 - ty).Add(b.Scale(ty))
}

// dreamNoise is a deterministic 0..1 hash of a pixel and seed (for film grain).
func dreamNoise(x, y, seed uint32) float64 {
	n := x*374761393 + y*668265263 + seed*2246822519
	n = (n ^ (n >> 13)) * 1274126177
	n ^= n >> 16
	return float64(n) / 4294967295.0
}

// Dream applies the lens-and-film post to a finished frame.
func Dream(img image.Image, o DreamOptions) image.Image {
	src, w, h := imgToBuf(img)
	if w == 0 || h == 0 {
		return img
	}
	cx, cy := float64(w-1)/2, float64(h-1)/2
	norm := math.Hypot(cx, cy)
	if norm == 0 {
		norm = 1
	}
	out := make([]Vec3, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			r := math.Hypot(dx, dy) / norm // 0 at centre .. ~1 at corners
			base := 1 + o.Distort*r*r      // barrel/pincushion warp of the sample radius
			at := func(chScale float64) Vec3 {
				return bilinear(src, w, h, cx+dx*base*chScale, cy+dy*base*chScale)
			}
			var c Vec3
			if o.Chroma != 0 { // each channel from its own radius: edge colour fringing
				c = Vec3{X: at(1 + o.Chroma*r).X, Y: at(1).Y, Z: at(1 - o.Chroma*r).Z}
			} else {
				c = at(1)
			}
			if o.Grain > 0 {
				n := (dreamNoise(uint32(x), uint32(y), o.Seed) - 0.5) * 2 * o.Grain
				c = c.Add(Vec3{X: n, Y: n, Z: n})
			}
			if o.Vignette > 0 {
				f := 1 - o.Vignette*r*r
				if f < 0 {
					f = 0
				}
				c = c.Scale(f)
			}
			out[y*w+x] = c
		}
	}
	return bufToImg(out, w, h)
}
