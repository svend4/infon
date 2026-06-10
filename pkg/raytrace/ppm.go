// ppm.go is a progressive photon mapping renderer (Hachisuka et al. 2008). Photon
// mapping density-estimates radiance from stored photons; its only error is the
// bias of a finite gather radius. PPM removes that bias progressively: it keeps a
// per-pixel "measurement point" (the first diffuse surface seen, after any mirror
// or glass) and runs repeated photon passes, shrinking each point's gather radius
// after every pass by the rule R' = R * sqrt((N + a*M)/(N + M)). The radius tends
// to zero while the photon count grows, so the estimate is consistent (converges
// to the exact solution) yet handles caustics and full global illumination in one
// pass structure — things a forward path tracer resolves only slowly. Clean-room.
package raytrace

import (
	"image"
	"math"
	"runtime"
	"sync"
)

// PPMOptions controls progressive photon mapping.
type PPMOptions struct {
	Passes         int     // photon passes (more = less noise and a smaller radius)
	PhotonsPerPass int     // photons emitted per pass
	Radius         float64 // initial gather radius (world units)
	Alpha          float64 // radius reduction in (0,1); 0.7 is typical
	MaxBounce      int     // bounce budget for camera and photon paths (default 8)
	Seed           uint64
}

// measPoint is a per-pixel measurement point plus its progressive statistics.
type measPoint struct {
	pos, normal, albedo Vec3
	beta                Vec3 // camera throughput (specular tint) to this point
	emit                Vec3 // emission/sky seen directly along the camera ray
	valid               bool // true if a diffuse measurement point was found

	radius  float64
	n       float64 // accumulated local photon count (the PPM N)
	tau     Vec3    // accumulated radius-corrected flux * BRDF
	mPass   float64 // photons gathered this pass (scratch)
	tauPass Vec3    // flux * BRDF gathered this pass (scratch)
}

// ppmRadiusUpdate is the PPM progressive update: it shrinks the radius and reports
// the flux rescale ratio R'^2/R^2 and the new accumulated count.
func ppmRadiusUpdate(r, n, m, alpha float64) (newR, newN, ratio float64) {
	newN = n + alpha*m
	ratio = newN / (n + m)
	return r * math.Sqrt(ratio), newN, ratio
}

// PPMRender renders the scene with progressive photon mapping.
func PPMRender(s *Scene, cam Camera, w, h int, opt PPMOptions) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if w <= 0 || h <= 0 {
		return img
	}
	passes := opt.Passes
	if passes < 1 {
		passes = 1
	}
	ppp := opt.PhotonsPerPass
	if ppp < 1 {
		ppp = 10000
	}
	r0 := opt.Radius
	if r0 <= 0 {
		r0 = 0.5
	}
	alpha := opt.Alpha
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.7
	}
	maxB := opt.MaxBounce
	if maxB < 1 {
		maxB = 8
	}
	s.gatherEmitters()

	mps := s.ppmCameraPass(cam, w, h, r0, maxB, opt.Seed)

	// Spatial hash over valid measurement points (cell = initial radius, which is
	// always >= every shrunk radius, so a photon's ball spans the 3x3x3 stencil).
	cell := r0
	cellOf := func(p Vec3) [3]int {
		return [3]int{int(math.Floor(p.X / cell)), int(math.Floor(p.Y / cell)), int(math.Floor(p.Z / cell))}
	}
	grid := map[[3]int][]int32{}
	for i := range mps {
		if mps[i].valid {
			c := cellOf(mps[i].pos)
			grid[c] = append(grid[c], int32(i))
		}
	}

	nE := len(s.emit)
	var emitted float64
	rg := newRNG(opt.Seed | 1)
	for pass := 0; pass < passes; pass++ {
		for i := range mps {
			mps[i].mPass, mps[i].tauPass = 0, Vec3{}
		}
		if nE > 0 {
			for k := 0; k < ppp; k++ {
				e := s.emit[int(rg.next()%uint64(nE))]
				nrm := rg.unit()
				origin := e.Center.Add(nrm.Scale(e.Radius + shadowEps))
				// Power = full emitted flux of the source (radiance * area * pi),
				// times nE because the source was picked uniformly; the final image
				// divides by the total photons emitted.
				power := e.Mat.Emit.Scale(4 * math.Pi * math.Pi * e.Radius * e.Radius * float64(nE))
				ray := Ray{Origin: origin, Dir: cosineSample(nrm, rg)}
				s.tracePhoton(ray, power, maxB, rg, grid, cellOf, mps)
			}
		}
		emitted += float64(ppp)
		for i := range mps {
			mp := &mps[i]
			if !mp.valid || mp.mPass <= 0 {
				continue
			}
			var ratio float64
			mp.radius, mp.n, ratio = ppmRadiusUpdate(mp.radius, mp.n, mp.mPass, alpha)
			mp.tau = mp.tau.Add(mp.tauPass).Scale(ratio)
		}
	}

	for i := range mps {
		mp := &mps[i]
		col := mp.emit
		if mp.valid && emitted > 0 {
			l := mp.tau.Scale(1 / (math.Pi * mp.radius * mp.radius * emitted))
			col = col.Add(mp.beta.Mul(l))
		}
		img.SetRGBA(i%w, i/w, toRGBA(col))
	}
	return img
}

// ppmCameraPass traces one primary ray per pixel to the first diffuse surface
// (following mirror/glass chains) and records a measurement point. Parallel.
func (s *Scene) ppmCameraPass(cam Camera, w, h int, r0 float64, maxB int, seed uint64) []measPoint {
	mps := make([]measPoint, w*h)
	b := cam.basis(w, h)
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
				for x := 0; x < w; x++ {
					rg := newRNG(seed*0x9e3779b97f4a7c15 + uint64(y)*0x100000001b3 + uint64(x)*0x85ebca6b + 1)
					u := float64(x) + rg.f()
					v := float64(y) + rg.f()
					mp := s.traceCamera(b.ray(u, v), maxB, rg)
					mp.radius = r0
					mps[y*w+x] = mp
				}
			}
		}()
	}
	wg.Wait()
	return mps
}

// traceCamera follows a camera ray through specular surfaces to the first diffuse
// hit (the measurement point), or records terminal emission / sky.
func (s *Scene) traceCamera(r Ray, maxB int, rg *rng) measPoint {
	beta := Vec3{X: 1, Y: 1, Z: 1}
	for d := 0; d < maxB; d++ {
		h, ok := s.closest(r, shadowEps, tFar)
		if !ok {
			return measPoint{emit: beta.Mul(s.sky(r.Dir))}
		}
		if h.Mat.Emit.LenSq() > 0 {
			return measPoint{emit: beta.Mul(h.Mat.Emit)}
		}
		switch {
		case h.Mat.Glass > 0:
			r = s.scatterGlass(r, h, rg)
		case h.Mat.Reflect > 0 || h.Mat.Metal > 0:
			if a := h.albedo(); a.LenSq() > 0 {
				beta = beta.Mul(a)
			}
			r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: r.Dir.Reflect(h.N).Norm()}
		default:
			return measPoint{pos: h.P, normal: h.N, albedo: h.albedo(), beta: beta, valid: true}
		}
	}
	return measPoint{}
}

// tracePhoton walks a photon from a light, depositing flux at diffuse surfaces
// into nearby measurement points and bouncing (Russian roulette on albedo) until
// it is absorbed or leaves the scene.
func (s *Scene) tracePhoton(r Ray, power Vec3, maxB int, rg *rng, grid map[[3]int][]int32, cellOf func(Vec3) [3]int, mps []measPoint) {
	for d := 0; d < maxB; d++ {
		h, ok := s.closest(r, shadowEps, tFar)
		if !ok {
			return
		}
		switch {
		case h.Mat.Glass > 0:
			r = s.scatterGlass(r, h, rg)
		case h.Mat.Reflect > 0 || h.Mat.Metal > 0:
			if a := h.albedo(); a.LenSq() > 0 {
				power = power.Mul(a)
			}
			r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: r.Dir.Reflect(h.N).Norm()}
		default:
			// Deposit into measurement points whose (shrinking) ball contains h.P.
			c := cellOf(h.P)
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					for dz := -1; dz <= 1; dz++ {
						for _, idx := range grid[[3]int{c[0] + dx, c[1] + dy, c[2] + dz}] {
							mp := &mps[idx]
							if mp.normal.Dot(h.N) < 0.5 {
								continue // different surface orientation: no flux leak
							}
							if mp.pos.Sub(h.P).LenSq() <= mp.radius*mp.radius {
								mp.tauPass = mp.tauPass.Add(mp.albedo.Scale(1 / math.Pi).Mul(power))
								mp.mPass++
							}
						}
					}
				}
			}
			a := h.albedo()
			p := math.Max(a.X, math.Max(a.Y, a.Z))
			if p <= 0 || rg.f() > p {
				return
			}
			power = power.Mul(a).Scale(1 / p)
			r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: cosineSample(h.N, rg)}
		}
	}
}
