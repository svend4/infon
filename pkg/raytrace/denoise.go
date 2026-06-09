// denoise.go is an edge-aware à-trous wavelet denoiser for noisy (low-sample)
// path-traced frames. Each iteration convolves a 5-tap B3-spline kernel at an
// increasing pixel step (1, 2, 4, …), weighting each neighbour by colour
// similarity so flat regions smooth out while edges are preserved. It is a pure
// image-space filter, so it works on any image (and is cheap enough to run per
// frame in the terminal).
package raytrace

import (
	"image"
	"image/color"
	"math"
)

// Denoise applies `iterations` à-trous passes with colour edge-stopping strength
// sigma (smaller = sharper / less smoothing). Returns a new image.
func Denoise(img image.Image, iterations int, sigma float64) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 || iterations < 1 {
		return img
	}
	if sigma <= 0 {
		sigma = 0.1
	}
	cur := make([]Vec3, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			cur[y*w+x] = Vec3{X: float64(r) / 65535, Y: float64(g) / 65535, Z: float64(bl) / 65535}
		}
	}

	k := [5]float64{1.0 / 16, 1.0 / 4, 3.0 / 8, 1.0 / 4, 1.0 / 16}
	inv2s2 := 1.0 / (2 * sigma * sigma)
	for it := 0; it < iterations; it++ {
		step := 1 << it
		next := make([]Vec3, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				cp := cur[y*w+x]
				var sum Vec3
				var wsum float64
				for dy := -2; dy <= 2; dy++ {
					ny := y + dy*step
					if ny < 0 || ny >= h {
						continue
					}
					for dx := -2; dx <= 2; dx++ {
						nx := x + dx*step
						if nx < 0 || nx >= w {
							continue
						}
						cq := cur[ny*w+nx]
						cw := math.Exp(-cp.Sub(cq).LenSq() * inv2s2)
						wgt := k[dx+2] * k[dy+2] * cw
						sum = sum.Add(cq.Scale(wgt))
						wsum += wgt
					}
				}
				if wsum > 0 {
					next[y*w+x] = sum.Scale(1 / wsum)
				} else {
					next[y*w+x] = cp
				}
			}
		}
		cur = next
	}

	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cur[y*w+x]
			out.SetRGBA(x, y, color.RGBA{
				R: uint8(clamp01(c.X)*255 + 0.5),
				G: uint8(clamp01(c.Y)*255 + 0.5),
				B: uint8(clamp01(c.Z)*255 + 0.5),
				A: 255,
			})
		}
	}
	return out
}
