// pathtrace.go is a Monte-Carlo path tracer that shares the same scene and
// objects as the direct-lighting Render, but computes *global* illumination:
// light bounces diffusely, picks up colour from the surfaces it touches (colour
// bleeding) and is gathered from emissive materials and the sky (both act as
// area lights). It supports diffuse, glossy/metal (Material.Rough), mirror and
// glass (stochastic Fresnel) interactions, Russian-roulette path termination, a
// thin-lens camera for depth of field, and progressive averaging of N samples.
//
// Everything is deterministic: each pixel owns a seeded xorshift PRNG keyed by
// its coordinates and the base seed, so the parallel render is race-free and
// reproducible (and therefore testable).
package raytrace

import (
	"image"
	"math"
	"runtime"
	"sync"
)

// rng is a tiny, fast, deterministic xorshift64* generator.
type rng struct{ s uint64 }

func newRNG(seed uint64) *rng {
	if seed == 0 {
		seed = 0x9e3779b97f4a7c15
	}
	return &rng{s: seed}
}

func (r *rng) next() uint64 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 7
	r.s ^= r.s << 17
	return r.s
}

// f returns a float64 in [0,1).
func (r *rng) f() float64 { return float64(r.next()>>11) / float64(uint64(1)<<53) }

// unit returns a uniformly random point on the unit sphere.
func (r *rng) unit() Vec3 {
	z := 2*r.f() - 1
	a := 2 * math.Pi * r.f()
	s := math.Sqrt(math.Max(0, 1-z*z))
	return Vec3{X: s * math.Cos(a), Y: s * math.Sin(a), Z: z}
}

// cosineSample draws a cosine-weighted direction in the hemisphere about n.
func cosineSample(n Vec3, r *rng) Vec3 {
	u1, u2 := r.f(), r.f()
	rad := math.Sqrt(u1)
	th := 2 * math.Pi * u2
	x, y := rad*math.Cos(th), rad*math.Sin(th)
	z := math.Sqrt(math.Max(0, 1-u1))
	t, b := basisAround(n)
	return t.Scale(x).Add(b.Scale(y)).Add(n.Scale(z)).Norm()
}

// PathOptions controls the path tracer.
type PathOptions struct {
	Samples  int    // paths per pixel
	MaxDepth int    // maximum bounces (default 6)
	Seed     uint64 // base seed for reproducibility
}

// radiance follows one path and returns the light gathered along it.
func (s *Scene) radiance(r Ray, maxDepth int, rg *rng) Vec3 {
	throughput := Vec3{X: 1, Y: 1, Z: 1}
	var out Vec3
	for d := 0; d < maxDepth; d++ {
		h, ok := s.closest(r, shadowEps, tFar)
		if !ok {
			out = out.Add(throughput.Mul(s.sky(r.Dir)))
			break
		}
		out = out.Add(throughput.Mul(h.Mat.Emit))

		switch {
		case h.Mat.Glass > 0:
			r = s.scatterGlass(r, h, rg)
		case h.Mat.Reflect > 0:
			dir := r.Dir.Reflect(h.N).Norm()
			if h.Mat.Rough > 0 {
				dir = dir.Add(rg.unit().Scale(h.Mat.Rough)).Norm()
				if dir.Dot(h.N) < 0 {
					dir = dir.Reflect(h.N)
				}
			}
			tint := h.Mat.Color
			if tint.LenSq() == 0 {
				tint = Vec3{X: 1, Y: 1, Z: 1}
			}
			throughput = throughput.Mul(tint)
			r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: dir}
		default:
			throughput = throughput.Mul(h.Mat.Color)
			r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: cosineSample(h.N, rg)}
		}

		// Russian roulette after a few bounces.
		if d >= 3 {
			p := math.Max(throughput.X, math.Max(throughput.Y, throughput.Z))
			if p < 1 {
				if rg.f() > p {
					break
				}
				throughput = throughput.Scale(1 / p)
			}
		}
	}
	return out
}

func (s *Scene) scatterGlass(r Ray, h Hit, rg *rng) Ray {
	n := h.N
	ior := h.Mat.Glass
	eta := 1 / ior
	if !h.Front {
		eta = ior
	}
	cosI := math.Min(1, math.Max(0, -r.Dir.Dot(n)))
	k := 1 - eta*eta*(1-cosI*cosI)
	r0 := (1 - ior) / (1 + ior)
	r0 *= r0
	fr := r0 + (1-r0)*math.Pow(1-cosI, 5)
	if k < 0 || rg.f() < fr {
		return Ray{Origin: h.P.Add(n.Scale(shadowEps)), Dir: r.Dir.Reflect(n).Norm()}
	}
	dir := r.Dir.Scale(eta).Add(n.Scale(eta*cosI - math.Sqrt(k))).Norm()
	return Ray{Origin: h.P.Sub(n.Scale(shadowEps)), Dir: dir}
}

// lensRay generates a depth-of-field primary ray through sub-pixel (px,py) using
// the camera aperture/focus; with Aperture 0 it is the pinhole ray.
func (b camBasis) lensRay(px, py float64, cam Camera, rg *rng) Ray {
	base := b.ray(px, py)
	if cam.Aperture <= 0 {
		return base
	}
	focus := cam.Focus
	if focus <= 0 {
		focus = b.origin.Sub(Vec3{0, 0, 0}).Len() // default: focus on the origin
		if focus <= 0 {
			focus = 1
		}
	}
	focal := base.Origin.Add(base.Dir.Scale(focus / base.Dir.Dot(b.forward)))
	// sample a disk on the lens
	rr := cam.Aperture * math.Sqrt(rg.f())
	th := 2 * math.Pi * rg.f()
	off := b.right.Scale(rr * math.Cos(th)).Add(b.up.Scale(rr * math.Sin(th)))
	orig := base.Origin.Add(off)
	return Ray{Origin: orig, Dir: focal.Sub(orig).Norm()}
}

// PathRender renders the scene with Monte-Carlo global illumination.
func PathRender(s *Scene, cam Camera, pxW, pxH int, opt PathOptions) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	if pxW <= 0 || pxH <= 0 {
		return img
	}
	spp := opt.Samples
	if spp < 1 {
		spp = 1
	}
	depth := opt.MaxDepth
	if depth < 1 {
		depth = 6
	}
	b := cam.basis(pxW, pxH)
	inv := 1.0 / float64(spp)

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
					// deterministic per-pixel seed (splitmix-style mix).
					seed := opt.Seed*0x9e3779b97f4a7c15 + uint64(y)*0x100000001b3 + uint64(x)*0x85ebca6b + 1
					rg := newRNG(seed)
					var acc Vec3
					for i := 0; i < spp; i++ {
						u := float64(x) + rg.f()
						v := float64(y) + rg.f()
						acc = acc.Add(s.radiance(b.lensRay(u, v, cam, rg), depth, rg))
					}
					img.SetRGBA(x, y, toRGBA(acc.Scale(inv)))
				}
			}
		}()
	}
	wg.Wait()
	return img
}
