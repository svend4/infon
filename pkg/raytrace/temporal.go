// temporal.go reuses the previous frame across small camera motions (temporal
// reprojection): each pixel's world hit point is projected back into the previous
// camera; if it lands on a pixel whose stored hit point matches (no occlusion
// change), that pixel's history is blended with a cheap fresh sample. Static or
// slowly-moving views then converge like an accumulator while a moving camera
// still gets most of the noise reduction - at a fraction of the samples.
package raytrace

import (
	"image"
	"runtime"
	"sync"
)

// Project maps a world point to fractional pixel coordinates for cam (the inverse
// of primary ray generation). ok is false if the point is behind the camera or
// off-screen.
func Project(cam Camera, p Vec3, pxW, pxH int) (px, py float64, ok bool) {
	b := cam.basis(pxW, pxH)
	v := p.Sub(b.origin)
	z := v.Dot(b.forward)
	if z <= geomEps {
		return 0, 0, false
	}
	ndcX := v.Dot(b.right) / z / b.halfW
	ndcY := v.Dot(b.up) / z / b.halfH
	px = (ndcX + 1) / 2 * b.pxW
	py = (1 - ndcY) / 2 * b.pxH
	ok = px >= 0 && px < b.pxW && py >= 0 && py < b.pxH
	return px, py, ok
}

// primaryHits returns the first-hit world point per pixel (camera-ray centres)
// and a validity mask (false where the ray escaped to the sky).
func primaryHits(s *Scene, cam Camera, pxW, pxH int) ([]Vec3, []bool) {
	pos := make([]Vec3, pxW*pxH)
	valid := make([]bool, pxW*pxH)
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
					if h, ok := s.closest(b.ray(float64(x)+0.5, float64(y)+0.5), shadowEps, tFar); ok {
						pos[y*pxW+x] = h.P
						valid[y*pxW+x] = true
					}
				}
			}
		}()
	}
	wg.Wait()
	return pos, valid
}

// TemporalReprojector accumulates a path-traced sequence across camera motion.
type TemporalReprojector struct {
	w, h      int
	alpha     float64 // blend toward the fresh frame (0..1); lower = more history
	n         int     // frame counter (decorrelates each fresh frame's samples)
	prevCol   []Vec3
	prevPos   []Vec3
	prevValid []bool
	prevCam   Camera
	has       bool
}

// NewTemporalReprojector makes one for a w x h image; alpha in (0,1] (default 0.3).
func NewTemporalReprojector(w, h int, alpha float64) *TemporalReprojector {
	if alpha <= 0 || alpha > 1 {
		alpha = 0.3
	}
	return &TemporalReprojector{w: w, h: h, alpha: alpha}
}

// Frame renders one frame, reusing matching history from the previous frame, and
// returns the tone-mapped result.
func (tr *TemporalReprojector) Frame(s *Scene, cam Camera, opt PathOptions) image.Image {
	spp := opt.Samples
	if spp < 1 {
		spp = 1
	}
	o := opt
	o.Seed = opt.Seed + uint64(tr.n) + 1 // fresh, decorrelated samples each frame
	tr.n++
	sum := pathSum(s, cam, tr.w, tr.h, o)
	inv := 1.0 / float64(spp)
	lin := make([]Vec3, len(sum))
	for i := range sum {
		lin[i] = sum[i].Scale(inv)
	}
	pos, valid := primaryHits(s, cam, tr.w, tr.h)

	out := make([]Vec3, tr.w*tr.h)
	copy(out, lin)
	if tr.has {
		for y := 0; y < tr.h; y++ {
			for x := 0; x < tr.w; x++ {
				i := y*tr.w + x
				if !valid[i] {
					continue // sky: nothing to reproject
				}
				px, py, ok := Project(tr.prevCam, pos[i], tr.w, tr.h)
				if !ok {
					continue
				}
				j := int(py)*tr.w + int(px)
				if !tr.prevValid[j] {
					continue
				}
				// disocclusion guard: the history pixel must be the same surface.
				thresh := 0.02 * pos[i].Sub(cam.Pos).Len()
				if tr.prevPos[j].Sub(pos[i]).Len() > thresh+geomEps {
					continue
				}
				out[i] = tr.prevCol[j].Scale(1 - tr.alpha).Add(lin[i].Scale(tr.alpha))
			}
		}
	}
	tr.prevCol, tr.prevPos, tr.prevValid, tr.prevCam, tr.has = out, pos, valid, cam, true

	img := image.NewRGBA(image.Rect(0, 0, tr.w, tr.h))
	for y := 0; y < tr.h; y++ {
		for x := 0; x < tr.w; x++ {
			img.SetRGBA(x, y, toRGBA(out[y*tr.w+x]))
		}
	}
	return img
}
