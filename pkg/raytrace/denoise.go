// denoise.go is an edge-aware à-trous wavelet denoiser for noisy (low-sample)
// path-traced frames. Each iteration convolves a 5-tap B3-spline kernel at an
// increasing pixel step (1, 2, 4, …), weighting each neighbour by similarity in
// colour and, optionally, in albedo and normal (guide buffers). Guides let it
// smooth Monte-Carlo noise hard while preserving texture and geometry edges that
// a colour-only filter would blur. It is a pure image-space filter.
package raytrace

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
)

// GBuffer renders the primary-hit albedo and shading normal per pixel (row-major)
// — the guide buffers for DenoiseGuided. Misses get a zero albedo and the ray
// direction as "normal".
func GBuffer(s *Scene, cam Camera, pxW, pxH int) (albedo, normal []Vec3) {
	albedo = make([]Vec3, pxW*pxH)
	normal = make([]Vec3, pxW*pxH)
	b := cam.basis(pxW, pxH)
	rows := make(chan int, pxH)
	for y := 0; y < pxH; y++ {
		rows <- y
	}
	close(rows)
	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range rows {
				for x := 0; x < pxW; x++ {
					r := b.ray(float64(x)+0.5, float64(y)+0.5)
					if h, ok := s.closest(r, shadowEps, tFar); ok {
						albedo[y*pxW+x] = h.albedo()
						normal[y*pxW+x] = h.N
					} else {
						normal[y*pxW+x] = r.Dir
					}
				}
			}
		}()
	}
	wg.Wait()
	return albedo, normal
}

// Denoise runs `iterations` à-trous passes with colour edge-stopping only.
func Denoise(img image.Image, iterations int, sigma float64) image.Image {
	return DenoiseGuided(img, nil, nil, iterations, sigma, 0, 0)
}

// DenoiseGuided runs the à-trous filter with optional albedo/normal guides
// (pass nil and 0 to disable a guide). sigC/sigA/sigN are the edge-stopping
// strengths for colour, albedo and normal (smaller = sharper).
func DenoiseGuided(img image.Image, albedo, normal []Vec3, iterations int, sigC, sigA, sigN float64) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 || iterations < 1 {
		return img
	}
	if sigC <= 0 {
		sigC = 0.1
	}
	useA := albedo != nil && sigA > 0 && len(albedo) == w*h
	useN := normal != nil && sigN > 0 && len(normal) == w*h

	cur := make([]Vec3, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			cur[y*w+x] = Vec3{X: float64(r) / 65535, Y: float64(g) / 65535, Z: float64(bl) / 65535}
		}
	}

	k := [5]float64{1.0 / 16, 1.0 / 4, 3.0 / 8, 1.0 / 4, 1.0 / 16}
	invC := 1.0 / (2 * sigC * sigC)
	invA, invN := 0.0, 0.0
	if useA {
		invA = 1.0 / (2 * sigA * sigA)
	}
	if useN {
		invN = 1.0 / (2 * sigN * sigN)
	}
	for it := 0; it < iterations; it++ {
		step := 1 << it
		next := make([]Vec3, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				ci := y*w + x
				cp := cur[ci]
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
						qi := ny*w + nx
						wgt := k[dx+2] * k[dy+2] * math.Exp(-cur[qi].Sub(cp).LenSq()*invC)
						if useA {
							wgt *= math.Exp(-albedo[qi].Sub(albedo[ci]).LenSq() * invA)
						}
						if useN {
							wgt *= math.Exp(-normal[qi].Sub(normal[ci]).LenSq() * invN)
						}
						sum = sum.Add(cur[qi].Scale(wgt))
						wsum += wgt
					}
				}
				if wsum > 0 {
					next[ci] = sum.Scale(1 / wsum)
				} else {
					next[ci] = cp
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
