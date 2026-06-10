// restir.go adds ReSTIR direct lighting (Bitterli et al. 2020): the reuse layer on
// top of the RIS reservoirs in reservoir.go. Each pixel first builds a reservoir
// from a few candidate lights (initial RIS), then *reuses* its spatial neighbours'
// reservoirs: a neighbour's chosen light is re-evaluated against this pixel's
// surface and merged in, so every pixel ends up effectively sampling from the
// union of all the lights its neighbourhood found - hundreds of effective samples
// for the cost of a few. Reuse is kept unbiased by re-targeting each contribution
// to the receiving pixel and dividing by the bias-correction count Z (how many
// contributing pixels could have produced the chosen sample). One shadow ray per
// pixel shades the survivor. Clean-room implementation.
package raytrace

import (
	"image"
	"math"
	"runtime"
	"sync"
)

// ReSTIROptions controls ReSTIR direct lighting.
type ReSTIROptions struct {
	Candidates    int     // initial RIS candidate lights per pixel (M)
	SpatialPasses int     // spatial reuse passes
	SpatialK      int     // neighbours sampled per pass
	Radius        float64 // neighbour radius in pixels
	Seed          uint64
}

// diSample is a chosen light sample: a point on emitter ei.
type diSample struct {
	q  Vec3
	ei int
}

// reservoirDI is a screen-space DI reservoir.
type reservoirDI struct {
	y    diSample
	wsum float64
	W    float64 // unbiased contribution weight
	M    float64 // confidence (sample count)
	got  bool
}

// stream folds a candidate (sample s, resampling weight w, confidence m) into the
// reservoir, keeping the survivor with probability w/wsum.
func (r *reservoirDI) stream(smp diSample, w, m float64, rg *rng) {
	r.M += m
	if w <= 0 {
		return
	}
	r.wsum += w
	if rg.f()*r.wsum < w {
		r.y = smp
		r.got = true
	}
}

// pHatDI is the target function: the scalar (luminance) of the unshadowed
// contribution of light sample smp at surface point x with normal n and albedo a,
// returning also the full vector contribution, the light direction and distance.
func (s *Scene) pHatDI(x, n, a Vec3, smp diSample) (float64, Vec3, Vec3, float64) {
	e := s.emit[smp.ei]
	d := smp.q.Sub(x)
	dist := d.Len()
	if dist < geomEps {
		return 0, Vec3{}, Vec3{}, 0
	}
	wL := d.Scale(1 / dist)
	cosS := n.Dot(wL)
	if cosS <= 0 {
		return 0, Vec3{}, Vec3{}, 0
	}
	cosL := smp.q.Sub(e.Center).Norm().Dot(wL.Neg())
	if cosL <= 0 {
		return 0, Vec3{}, Vec3{}, 0
	}
	fArea := a.Scale(1 / math.Pi).Mul(e.Mat.Emit).Scale(cosS * cosL / (dist * dist))
	return lum3(fArea), fArea, wL, dist
}

// initialRIS builds a pixel's reservoir from m candidate light samples.
func (s *Scene) initialRIS(g measPoint, m int, rg *rng) reservoirDI {
	nE := len(s.emit)
	var r reservoirDI
	for i := 0; i < m; i++ {
		ei := int(rg.next() % uint64(nE))
		e := s.emit[ei]
		q := e.Center.Add(rg.unit().Scale(e.Radius))
		ph, _, _, _ := s.pHatDI(g.pos, g.normal, g.albedo, diSample{q, ei})
		pArea := 1 / (float64(nE) * 4 * math.Pi * e.Radius * e.Radius)
		r.stream(diSample{q, ei}, ph/pArea, 1, rg)
	}
	if r.got {
		if ph, _, _, _ := s.pHatDI(g.pos, g.normal, g.albedo, r.y); ph > 0 {
			r.W = r.wsum / (r.M * ph)
		}
	}
	return r
}

// spatialPass reuses neighbour reservoirs, returning the combined reservoirs.
func (s *Scene) spatialPass(g []measPoint, res []reservoirDI, w, h int, radius float64, k int, seed uint64) []reservoirDI {
	out := make([]reservoirDI, len(res))
	parallelRows(h, func(y int) {
		rg := newRNG(seed*0x9e3779b97f4a7c15 + uint64(y)*0x100000001b3 + 1)
		conts := make([]int, 0, k+1)
		for x := 0; x < w; x++ {
			i := y*w + x
			if !g[i].valid || !res[i].got {
				out[i] = res[i]
				continue
			}
			gi := g[i]
			conts = conts[:0]
			conts = append(conts, i) // self
			for t := 0; t < k; t++ {
				ang := 2 * math.Pi * rg.f()
				rr := radius * math.Sqrt(rg.f())
				nx := x + int(rr*math.Cos(ang))
				ny := y + int(rr*math.Sin(ang))
				if nx < 0 || nx >= w || ny < 0 || ny >= h {
					continue
				}
				j := ny*w + nx
				if j == i || !g[j].valid || !res[j].got || g[j].normal.Dot(gi.normal) < 0.9 {
					continue // skip across geometry edges
				}
				conts = append(conts, j)
			}
			var c reservoirDI
			for _, j := range conts {
				ph, _, _, _ := s.pHatDI(gi.pos, gi.normal, gi.albedo, res[j].y) // re-target to this pixel
				c.stream(res[j].y, ph*res[j].W*res[j].M, res[j].M, rg)
			}
			if c.got {
				phq, _, _, _ := s.pHatDI(gi.pos, gi.normal, gi.albedo, c.y)
				var z float64
				for _, j := range conts {
					if ph, _, _, _ := s.pHatDI(g[j].pos, g[j].normal, g[j].albedo, c.y); ph > 0 {
						z += res[j].M
					}
				}
				if z > 0 && phq > 0 {
					c.W = c.wsum / (z * phq)
				}
			}
			out[i] = c
		}
	})
	return out
}

// ReSTIRRender renders direct lighting with ReSTIR (initial RIS + spatial reuse).
// It captures direct illumination only; pair it with a path tracer for indirect.
func ReSTIRRender(s *Scene, cam Camera, w, h int, opt ReSTIROptions) image.Image {
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}
	return tonemapSum(s.restirLinear(cam, w, h, opt), w, h, 1)
}

// restirLinear returns the per-pixel linear radiance (pre-tone-map), so the
// estimator can be checked for unbiasedness without the nonlinear tone curve.
func (s *Scene) restirLinear(cam Camera, w, h int, opt ReSTIROptions) []Vec3 {
	buf := make([]Vec3, w*h)
	m := opt.Candidates
	if m < 1 {
		m = 8
	}
	s.gatherEmitters()
	g := s.ppmCameraPass(cam, w, h, 0, 8, opt.Seed)

	res := make([]reservoirDI, w*h)
	if len(s.emit) > 0 {
		parallelRows(h, func(y int) {
			for x := 0; x < w; x++ {
				i := y*w + x
				if g[i].valid {
					rg := newRNG(opt.Seed*0x100000001b3 + uint64(i)*0x9e3779b9 + 1)
					res[i] = s.initialRIS(g[i], m, rg)
				}
			}
		})
		radius := opt.Radius
		if radius <= 0 {
			radius = 16
		}
		k := opt.SpatialK
		if k < 1 {
			k = 5
		}
		for p := 0; p < opt.SpatialPasses; p++ {
			res = s.spatialPass(g, res, w, h, radius, k, opt.Seed+uint64(p)+1)
		}
	}

	parallelRows(h, func(y int) {
		for x := 0; x < w; x++ {
			i := y*w + x
			col := g[i].emit
			if g[i].valid && res[i].got && res[i].W > 0 {
				ph, fArea, wL, dist := s.pHatDI(g[i].pos, g[i].normal, g[i].albedo, res[i].y)
				if ph > 0 && !s.anyHit(Ray{Origin: g[i].pos.Add(g[i].normal.Scale(shadowEps)), Dir: wL}, shadowEps, dist-2*shadowEps) {
					col = col.Add(g[i].beta.Mul(fArea.Scale(res[i].W)))
				}
			}
			buf[i] = col
		}
	})
	return buf
}

// parallelRows runs fn over rows 0..h-1 across the CPU pool.
func parallelRows(h int, fn func(y int)) {
	rows := make(chan int, h)
	for y := 0; y < h; y++ {
		rows <- y
	}
	close(rows)
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	var wg sync.WaitGroup
	for wk := 0; wk < workers; wk++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range rows {
				fn(y)
			}
		}()
	}
	wg.Wait()
}
