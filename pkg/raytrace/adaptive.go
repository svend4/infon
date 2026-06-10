// adaptive.go renders with per-pixel ADAPTIVE sampling. Each pixel tracks its
// running mean and variance of luminance (Welford's algorithm) and stops drawing
// samples once its standard error falls below a relative threshold (after a small
// minimum), so the budget flows to the noisy regions - edges, indirect light,
// glossy reflections - instead of pixels that already converged. The per-pixel
// seed decorrelates neighbouring pixels' noise (a cheap blue-noise-like spread).
package raytrace

import (
	"image"
	"math"
	"runtime"
	"sync"
)

// AdaptiveRender draws up to opt.Samples paths per pixel, stopping a pixel early
// once its relative standard error drops below tol (after a minimum of 8). It
// returns the image and the per-pixel sample count (row-major), so callers can
// see where the budget went.
func AdaptiveRender(s *Scene, cam Camera, pxW, pxH int, opt PathOptions, tol float64) (image.Image, []int) {
	img := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	counts := make([]int, pxW*pxH)
	if pxW <= 0 || pxH <= 0 {
		return img, counts
	}
	maxS := opt.Samples
	if maxS < 1 {
		maxS = 1
	}
	const minS = 8
	depth := opt.MaxDepth
	if depth < 1 {
		depth = 6
	}
	if tol <= 0 {
		tol = 0.05
	}
	b := cam.basis(pxW, pxH)
	s.gatherEmitters()
	mode := opt.mode()

	rows := make(chan int, pxH)
	for y := 0; y < pxH; y++ {
		rows <- y
	}
	close(rows)

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range rows {
				for x := 0; x < pxW; x++ {
					seed := opt.Seed*0x9e3779b97f4a7c15 + uint64(y)*0x100000001b3 + uint64(x)*0x85ebca6b + 1
					rg := newRNG(seed)
					var meanC Vec3
					var meanL, m2 float64
					nsamp := 0
					for nsamp < maxS {
						u := float64(x) + rg.f()
						v := float64(y) + rg.f()
						c := s.radiance(b.lensRay(u, v, cam, rg), depth, rg, mode)
						nsamp++
						meanC = meanC.Add(c)
						l := 0.2126*c.X + 0.7152*c.Y + 0.0722*c.Z
						d := l - meanL
						meanL += d / float64(nsamp)
						m2 += d * (l - meanL)
						if nsamp >= minS {
							variance := m2 / float64(nsamp-1)
							if math.Sqrt(variance/float64(nsamp)) < tol*(meanL+1e-3) {
								break
							}
						}
					}
					counts[y*pxW+x] = nsamp
					img.SetRGBA(x, y, toRGBA(meanC.Scale(1/float64(nsamp))))
				}
			}
		}()
	}
	wg.Wait()
	return img, counts
}
