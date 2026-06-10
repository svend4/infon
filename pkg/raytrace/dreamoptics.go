// dreamoptics.go reproduces the "unusual visual effects in lucid dreams" the
// hackers catalogue: image doubling (раздваивание), tunnel vision (туннельное
// видение), and the "complete violation of the laws of perspective" — a skew to
// one side. They are screen-space warps over a finished frame, deterministic and
// composable, a counterpart to the lens-and-film Dream pass.
package raytrace

import (
	"image"
	"math"
)

// PerspectiveSkew shears the frame horizontally by row — rows above centre lean
// one way, rows below the other — so the image leans, breaking perspective. amount
// is the horizontal shift in pixels per row away from the centre.
func PerspectiveSkew(img image.Image, amount float64) image.Image {
	src, w, h := imgToBuf(img)
	cy := float64(h-1) / 2
	out := make([]Vec3, w*h)
	for y := 0; y < h; y++ {
		shift := amount * (float64(y) - cy)
		for x := 0; x < w; x++ {
			out[y*w+x] = bilinear(src, w, h, float64(x)+shift, float64(y))
		}
	}
	return bufToImg(out, w, h)
}

// TunnelVision closes the view into a central circle, fading the surround toward
// black — the dream's tunnel vision. amount 0 none .. 1 the edges go dark.
func TunnelVision(img image.Image, amount float64) image.Image {
	src, w, h := imgToBuf(img)
	cx, cy := float64(w-1)/2, float64(h-1)/2
	norm := math.Hypot(cx, cy)
	if norm == 0 {
		norm = 1
	}
	out := make([]Vec3, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := math.Hypot(float64(x)-cx, float64(y)-cy) / norm
			f := 1 - amount*math.Pow(clamp01((r-0.3)/0.7), 1.5)
			out[y*w+x] = src[y*w+x].Scale(f)
		}
	}
	return bufToImg(out, w, h)
}

// DoubleVision overlays a ghost copy shifted by (dx,dy), the way a dream image
// splits in two.
func DoubleVision(img image.Image, dx, dy int) image.Image {
	src, w, h := imgToBuf(img)
	out := make([]Vec3, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src[y*w+x].Scale(0.65)
			gx, gy := x-dx, y-dy // the ghost samples a shifted source
			if gx >= 0 && gx < w && gy >= 0 && gy < h {
				c = c.Add(src[gy*w+gx].Scale(0.5))
			}
			out[y*w+x] = c
		}
	}
	return bufToImg(out, w, h)
}
