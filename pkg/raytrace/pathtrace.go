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

// rng is a tiny, fast, deterministic xorshift64* generator. When qmc is set it
// also serves Owen-scrambled Sobol (0,2) samples for the primary 2-D domains
// (pixel AA, lens) via qmc2; everything else still draws from the PRNG.
type rng struct {
	s     uint64 // xorshift state
	qmc   bool   // low-discrepancy primary sampling enabled
	qidx  uint32 // sample index within the pixel (the Sobol point index)
	qseed uint32 // per-pixel scramble base (decorrelates pixels)
}

func newRNG(seed uint64) *rng {
	if seed == 0 {
		seed = 0x9e3779b97f4a7c15
	}
	return &rng{s: seed}
}

// qmc2 returns the current sample's 2-D point for the given padding group
// (0 = pixel AA, 1 = lens). With qmc off it falls back to two PRNG draws, so the
// stream is byte-for-byte the legacy behaviour. Each group gets an independent
// Owen scramble, so groups are decorrelated but each is individually stratified.
func (r *rng) qmc2(group uint32) (float64, float64) {
	if !r.qmc {
		return r.f(), r.f()
	}
	s0 := hashU32(r.qseed + group*0x9e3779b9)
	s1 := hashU32(r.qseed + group*0x9e3779b9 + 1)
	a := owenScramble(sobolDim0(r.qidx), s0)
	b := owenScramble(sobolDim1(r.qidx), s1)
	return float64(a) * sobolInv, float64(b) * sobolInv
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

// sampleGGX importance-samples a microfacet half-vector from the GGX distribution
// (roughness alpha = rough^2) around the normal n.
func sampleGGX(n Vec3, rough float64, r *rng) Vec3 {
	a := rough * rough
	if a < 1e-3 {
		a = 1e-3
	}
	u1, u2 := r.f(), r.f()
	cosT := math.Sqrt((1 - u1) / (1 + (a*a-1)*u1))
	sinT := math.Sqrt(math.Max(0, 1-cosT*cosT))
	phi := 2 * math.Pi * u2
	t, b := basisAround(n)
	return t.Scale(sinT * math.Cos(phi)).Add(b.Scale(sinT * math.Sin(phi))).Add(n.Scale(cosT)).Norm()
}

// g1ggx is the Smith-GGX masking term for one direction (a = roughness^2).
func g1ggx(cosX, a float64) float64 {
	a2 := a * a
	return 2 * cosX / (cosX + math.Sqrt(a2+(1-a2)*cosX*cosX))
}

// schlickV is the per-channel Schlick Fresnel for a coloured F0.
func schlickV(f0 Vec3, cosT float64) Vec3 {
	m := math.Pow(1-cosT, 5)
	return f0.Add(Vec3{X: 1, Y: 1, Z: 1}.Sub(f0).Scale(m))
}

// PathOptions controls the path tracer.
type PathOptions struct {
	Samples  int    // paths per pixel
	MaxDepth int    // maximum bounces (default 6)
	Seed     uint64 // base seed for reproducibility
	NEE      bool   // next-event estimation: sample emissive spheres directly
	MIS      bool   // combine light- and BSDF-sampling with the power heuristic
	Sobol    bool   // Owen-scrambled Sobol (0,2) sampling of pixel AA + lens (less noise per spp)
}

func (o PathOptions) mode() int {
	switch {
	case o.MIS:
		return 2
	case o.NEE:
		return 1
	default:
		return 0
	}
}

// emitterRadius finds the emissive sphere whose surface contains p (used to weight
// BSDF-sampled emitter hits under MIS).
func (s *Scene) emitterRadius(p Vec3) (float64, bool) {
	for _, e := range s.emit {
		if math.Abs(p.Sub(e.Center).Len()-e.Radius) < 1e-3 {
			return e.Radius, true
		}
	}
	return 0, false
}

// radiance follows one path. mode 0 = naive (BSDF only), 1 = NEE only, 2 = MIS
// (light- and BSDF-sampling combined with the power heuristic).
func (s *Scene) radiance(r Ray, maxDepth int, rg *rng, mode int) Vec3 {
	throughput := Vec3{X: 1, Y: 1, Z: 1}
	var out Vec3
	specularPrev := true // the camera ray counts as a specular connection
	var prevPdfB float64
	nEmit := len(s.emit)
	tm := r.Time // shutter time, constant along the whole path (motion blur)
pathLoop:
	for d := 0; d < maxDepth; d++ {
		h, ok := s.closest(r, shadowEps, tFar)

		// Heterogeneous medium: delta-track a volume collision before the surface.
		// A real collision in front of the surface scatters in the medium; "no
		// collision" implicitly carries the transmittance to the surface/sky.
		if s.Medium != nil && (mode >= 1) {
			tMax := tFar
			if ok {
				tMax = h.T
			}
			if t0, t1, in := s.Medium.span(r, tMax); in {
				if tc, scattered := s.Medium.sampleCollision(r, t0, t1, rg); scattered {
					p := r.At(tc)
					a := s.Medium.Albedo
					out = out.Add(throughput.Mul(a).Mul(s.mediumNEE(p, r.Dir, rg, tm)))
					throughput = throughput.Mul(a)
					if d >= 2 { // Russian roulette so dense media stay bounded
						pr := math.Max(throughput.X, math.Max(throughput.Y, throughput.Z))
						if pr < 1 {
							if rg.f() > pr {
								break
							}
							throughput = throughput.Scale(1 / pr)
						}
					}
					r = Ray{Origin: p, Dir: hgSample(r.Dir, s.Medium.G, rg), Time: tm}
					specularPrev = false
					prevPdfB = 0 // medium direct light is counted by mediumNEE only
					continue pathLoop
				}
			}
		}

		if !ok {
			sky := s.sky(r.Dir)
			if mode >= 1 && s.Env != nil && !specularPrev { // MIS against env-NEE
				pdfE := s.Env.Pdf(r.Dir)
				sky = sky.Scale(prevPdfB * prevPdfB / (prevPdfB*prevPdfB + pdfE*pdfE))
			}
			out = out.Add(throughput.Mul(sky))
			break
		}
		// Emission reaching the eye along this path (BSDF-sampling side).
		if h.Mat.Emit.LenSq() > 0 {
			switch {
			case mode == 0 || specularPrev:
				out = out.Add(throughput.Mul(h.Mat.Emit))
			case mode == 2 && nEmit > 0:
				if rad, okE := s.emitterRadius(h.P); okE {
					if cosL := -h.N.Dot(r.Dir); cosL > geomEps {
						areaPdf := 1 / (float64(nEmit) * 4 * math.Pi * rad * rad)
						pdfL := areaPdf * h.T * h.T / cosL
						w := prevPdfB * prevPdfB / (prevPdfB*prevPdfB + pdfL*pdfL)
						out = out.Add(throughput.Mul(h.Mat.Emit).Scale(w))
					}
				}
				// mode 1 (NEE only): the emitter is already counted via NEE.
			}
		}

		switch {
		case h.Mat.SSS > 0:
			// Dielectric boundary: Fresnel-reflect (the highlight), or transmit and
			// random-walk the subsurface medium. Branch is chosen by the Fresnel
			// term without reweighting (the branch pdf cancels it), as for glass.
			cosI := math.Min(1, math.Max(0, -r.Dir.Dot(h.N)))
			r0 := (h.Mat.SSS - 1) / (h.Mat.SSS + 1)
			r0 *= r0
			fr := r0 + (1-r0)*math.Pow(1-cosI, 5)
			if rg.f() < fr {
				r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: r.Dir.Reflect(h.N).Norm(), Time: tm}
				specularPrev = true
				break
			}
			outRay, tput, okS := s.walkSubsurface(r, h, rg)
			if !okS {
				break pathLoop
			}
			throughput = throughput.Mul(tput)
			r = outRay
			specularPrev = true
		case h.Mat.Glass > 0:
			if h.Mat.Disperse > 0 {
				// spectral: one of R/G/B per ray, each with its own IOR, scaled by
				// 3 so the channels average to the achromatic result.
				c := int(rg.next() % 3)
				ior := h.Mat.Glass + (float64(c)-1)*h.Mat.Disperse
				r = s.scatterGlassIOR(r, h, rg, ior)
				tint := Vec3{}
				switch c {
				case 0:
					tint.X = 3
				case 1:
					tint.Y = 3
				default:
					tint.Z = 3
				}
				throughput = throughput.Mul(tint)
			} else {
				r = s.scatterGlass(r, h, rg)
			}
			specularPrev = true
		case h.Mat.Principled:
			// Disney-style: stochastically pick the diffuse or the GGX specular
			// lobe and divide by its selection probability (unbiased).
			m := h.Mat.Metal
			alb := h.albedo()
			f0 := Vec3{X: 0.04, Y: 0.04, Z: 0.04}.Scale(1 - m).Add(alb.Scale(m))
			pSpec := 0.2 + 0.7*m
			if pSpec < 0.1 {
				pSpec = 0.1
			} else if pSpec > 0.95 {
				pSpec = 0.95
			}
			if rg.f() < pSpec {
				V := r.Dir.Neg()
				hv := sampleGGX(h.N, h.Mat.Rough, rg)
				vh := V.Dot(hv)
				wi := r.Dir.Reflect(hv).Norm()
				nl := h.N.Dot(wi)
				nv := h.N.Dot(V)
				nh := h.N.Dot(hv)
				if vh <= 0 || nl <= 0 || nv*nh < geomEps {
					break pathLoop
				}
				a := h.Mat.Rough * h.Mat.Rough
				if a < 1e-3 {
					a = 1e-3
				}
				g := g1ggx(nv, a) * g1ggx(nl, a)
				throughput = throughput.Mul(schlickV(f0, vh)).Scale(g * vh / (nv * nh) / pSpec)
				r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: wi, Time: tm}
				specularPrev = true
			} else {
				throughput = throughput.Mul(alb.Scale((1 - m) / (1 - pSpec)))
				r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: cosineSample(h.N, rg), Time: tm}
				specularPrev = false
			}
		case h.Mat.Metal > 0:
			// GGX microfacet metal: importance-sample the half-vector, reflect,
			// and weight by F*G*(V·H)/((N·V)(N·H)) (the BRDF*cos/pdf for D-sampling).
			V := r.Dir.Neg()
			hv := sampleGGX(h.N, h.Mat.Rough, rg)
			vh := V.Dot(hv)
			wi := r.Dir.Reflect(hv).Norm()
			nl := h.N.Dot(wi)
			nv := h.N.Dot(V)
			nh := h.N.Dot(hv)
			if vh <= 0 || nl <= 0 || nv*nh < geomEps {
				break pathLoop // microfacet points away: end the path
			}
			f0 := Vec3{X: 0.04, Y: 0.04, Z: 0.04}.Scale(1 - h.Mat.Metal).Add(h.albedo().Scale(h.Mat.Metal))
			a := h.Mat.Rough * h.Mat.Rough
			if a < 1e-3 {
				a = 1e-3
			}
			g := g1ggx(nv, a) * g1ggx(nl, a)
			throughput = throughput.Mul(schlickV(f0, vh)).Scale(g * vh / (nv * nh))
			r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: wi, Time: tm}
			specularPrev = true
		case h.Mat.Reflect > 0:
			dir := r.Dir.Reflect(h.N).Norm()
			if h.Mat.Rough > 0 {
				dir = dir.Add(rg.unit().Scale(h.Mat.Rough)).Norm()
				if dir.Dot(h.N) < 0 {
					dir = dir.Reflect(h.N)
				}
			}
			tint := h.albedo()
			if tint.LenSq() == 0 {
				tint = Vec3{X: 1, Y: 1, Z: 1}
			}
			throughput = throughput.Mul(tint)
			r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: dir, Time: tm}
			specularPrev = true
		default:
			alb := h.albedo()
			if mode >= 1 {
				if s.RISCandidates > 1 && mode == 1 {
					out = out.Add(throughput.Mul(s.sampleEmittersRIS(h, alb, rg, tm, s.RISCandidates)))
				} else {
					out = out.Add(throughput.Mul(s.sampleEmitters(h, alb, rg, mode == 2, tm)))
				}
				if s.Env != nil {
					out = out.Add(throughput.Mul(s.sampleEnv(h, alb, rg, tm)))
				}
			}
			if s.Caustics != nil {
				out = out.Add(throughput.Mul(s.Caustics.Estimate(h.P, h.N, alb)))
			}
			dir := cosineSample(h.N, rg)
			prevPdfB = math.Max(0, h.N.Dot(dir)) / math.Pi
			throughput = throughput.Mul(alb)
			r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: dir, Time: tm}
			specularPrev = false
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

// sampleEmitters is the next-event estimator: pick an emissive sphere, draw a
// point on it, and return the unoccluded direct diffuse contribution (area-light
// form, normalised for the number of emitters). With mis it is weighted against
// the BSDF-sampling pdf (power heuristic).
func (s *Scene) sampleEmitters(h Hit, alb Vec3, rg *rng, mis bool, time float64) Vec3 {
	n := len(s.emit)
	if n == 0 {
		return Vec3{}
	}
	li := s.emit[int(rg.next()%uint64(n))]
	q := li.Center.Add(rg.unit().Scale(li.Radius))
	d := q.Sub(h.P)
	dist := d.Len()
	if dist < geomEps {
		return Vec3{}
	}
	wi := d.Scale(1 / dist)
	cosS := h.N.Dot(wi)
	if cosS <= 0 {
		return Vec3{}
	}
	cosL := q.Sub(li.Center).Norm().Dot(wi.Neg())
	if cosL <= 0 {
		return Vec3{}
	}
	so := h.P.Add(h.N.Scale(shadowEps))
	if s.anyHit(Ray{Origin: so, Dir: wi, Time: time}, shadowEps, dist-2*shadowEps) {
		return Vec3{}
	}
	areaPdf := 1 / (float64(n) * 4 * math.Pi * li.Radius * li.Radius)
	pdfL := areaPdf * dist * dist / cosL
	// diffuse: f = alb/pi, estimator = f * cosS * Le / pdfL.
	contrib := alb.Scale(1 / math.Pi).Mul(li.Mat.Emit).Scale(cosS / pdfL)
	if mis {
		pdfB := cosS / math.Pi
		contrib = contrib.Scale(pdfL * pdfL / (pdfL*pdfL + pdfB*pdfB))
	}
	return contrib
}

// sampleEnv is next-event estimation for the environment: importance-sample a
// bright env direction, test visibility to infinity, and return the MIS-weighted
// (power heuristic vs the cosine BSDF) direct contribution. Radiance is read from
// s.sky so it stays consistent with the BSDF-sampling side.
func (s *Scene) sampleEnv(h Hit, alb Vec3, rg *rng, time float64) Vec3 {
	we, _, pdfE := s.Env.Sample(rg)
	if pdfE <= 0 {
		return Vec3{}
	}
	ndl := h.N.Dot(we)
	if ndl <= 0 {
		return Vec3{}
	}
	if s.anyHit(Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: we, Time: time}, shadowEps, tFar) {
		return Vec3{}
	}
	pdfB := ndl / math.Pi
	w := pdfE * pdfE / (pdfE*pdfE + pdfB*pdfB)
	return alb.Scale(1 / math.Pi).Mul(s.sky(we)).Scale(ndl / pdfE * w)
}

// refract returns the refracted direction of unit d through the unit normal n
// (which faces against d) for the relative index eta = n_in/n_out, or ok=false on
// total internal reflection.
func refract(d, n Vec3, eta float64) (Vec3, bool) {
	cosI := math.Min(1, math.Max(0, -d.Dot(n)))
	k := 1 - eta*eta*(1-cosI*cosI)
	if k < 0 {
		return Vec3{}, false
	}
	return d.Scale(eta).Add(n.Scale(eta*cosI - math.Sqrt(k))).Norm(), true
}

func (s *Scene) scatterGlass(r Ray, h Hit, rg *rng) Ray {
	return s.scatterGlassIOR(r, h, rg, h.Mat.Glass)
}

// scatterGlassIOR reflects or refracts at a dielectric of the given index of
// refraction, choosing the branch stochastically by the Schlick Fresnel term.
func (s *Scene) scatterGlassIOR(r Ray, h Hit, rg *rng, ior float64) Ray {
	n := h.N
	eta := 1 / ior
	if !h.Front {
		eta = ior
	}
	cosI := math.Min(1, math.Max(0, -r.Dir.Dot(n)))
	tdir, ok := refract(r.Dir, n, eta)
	r0 := (1 - ior) / (1 + ior)
	r0 *= r0
	fr := r0 + (1-r0)*math.Pow(1-cosI, 5)
	if !ok || rg.f() < fr {
		return Ray{Origin: h.P.Add(n.Scale(shadowEps)), Dir: r.Dir.Reflect(n).Norm(), Time: r.Time}
	}
	return Ray{Origin: h.P.Sub(n.Scale(shadowEps)), Dir: tdir, Time: r.Time}
}

// lensRay generates a depth-of-field primary ray through sub-pixel (px,py) using
// the camera aperture/focus; with Aperture 0 it is the pinhole ray. The shutter
// time is sampled here (static objects ignore it).
func (b camBasis) lensRay(px, py float64, cam Camera, rg *rng) Ray {
	return b.lensRayAt(px, py, rg.f(), cam, rg)
}

// lensRayAt is lensRay with the shutter time supplied by the caller, so camera
// motion blur can pick the interpolated camera pose at that same time.
func (b camBasis) lensRayAt(px, py, time float64, cam Camera, rg *rng) Ray {
	base := b.ray(px, py)
	base.Time = time
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
	// sample a disk on the lens (group 1; stratified under Sobol)
	lu, lv := rg.qmc2(1)
	rr := cam.Aperture * math.Sqrt(lu)
	th := 2 * math.Pi * lv
	off := b.right.Scale(rr * math.Cos(th)).Add(b.up.Scale(rr * math.Sin(th)))
	orig := base.Origin.Add(off)
	return Ray{Origin: orig, Dir: focal.Sub(orig).Norm(), Time: base.Time}
}

// pathSum renders opt.Samples paths per pixel and returns the SUM of linear
// radiance per pixel (row-major), pre-tone-map. Parallel and deterministic given
// Seed. PathRender and Accumulator are both built on it.
func pathSum(s *Scene, cam Camera, pxW, pxH int, opt PathOptions) []Vec3 {
	b := cam.basis(pxW, pxH)
	return pathSumRays(s, pxW, pxH, opt, func(u, v float64, rg *rng) Ray {
		return b.lensRay(u, v, cam, rg)
	})
}

// pathSumRays is the shared parallel sampler: for each pixel it draws spp paths,
// generating each primary ray via gen, and returns the per-pixel radiance sum.
// The static camera, the moving camera (pathSumMotion) and any future primary-ray
// scheme all share this one worker loop.
func pathSumRays(s *Scene, pxW, pxH int, opt PathOptions, gen func(u, v float64, rg *rng) Ray) []Vec3 {
	buf := make([]Vec3, pxW*pxH)
	if pxW <= 0 || pxH <= 0 {
		return buf
	}
	spp := opt.Samples
	if spp < 1 {
		spp = 1
	}
	depth := opt.MaxDepth
	if depth < 1 {
		depth = 6
	}
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
					// deterministic per-pixel seed (splitmix-style mix).
					seed := opt.Seed*0x9e3779b97f4a7c15 + uint64(y)*0x100000001b3 + uint64(x)*0x85ebca6b + 1
					rg := newRNG(seed)
					rg.qmc = opt.Sobol
					rg.qseed = uint32(seed) ^ uint32(seed>>32)
					var acc Vec3
					for i := 0; i < spp; i++ {
						rg.qidx = uint32(i)
						du, dv := rg.qmc2(0) // stratified pixel anti-aliasing
						u := float64(x) + du
						v := float64(y) + dv
						acc = acc.Add(s.radiance(gen(u, v, rg), depth, rg, mode))
					}
					buf[y*pxW+x] = acc
				}
			}
		}()
	}
	wg.Wait()
	return buf
}

// tonemapSum tone-maps a per-pixel radiance sum (spp samples each) to an image.
func tonemapSum(buf []Vec3, pxW, pxH, spp int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	if spp < 1 {
		spp = 1
	}
	inv := 1.0 / float64(spp)
	for y := 0; y < pxH; y++ {
		for x := 0; x < pxW; x++ {
			img.SetRGBA(x, y, toRGBA(buf[y*pxW+x].Scale(inv)))
		}
	}
	return img
}

// PathRender renders the scene with Monte-Carlo global illumination.
func PathRender(s *Scene, cam Camera, pxW, pxH int, opt PathOptions) image.Image {
	if pxW <= 0 || pxH <= 0 {
		return image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	}
	return tonemapSum(pathSum(s, cam, pxW, pxH, opt), pxW, pxH, opt.Samples)
}

// Accumulator progressively averages path-traced samples, so a viewer can refine
// an image over time (more samples = less noise) without restarting. Each
// AddSamples call offsets the seed so batches are decorrelated.
type Accumulator struct {
	w, h    int
	sum     []Vec3
	samples int
}

// NewAccumulator makes an accumulator for a w x h image.
func NewAccumulator(w, h int) *Accumulator { return &Accumulator{w: w, h: h, sum: make([]Vec3, w*h)} }

// Reset clears it (call when the camera or scene changes).
func (a *Accumulator) Reset() {
	for i := range a.sum {
		a.sum[i] = Vec3{}
	}
	a.samples = 0
}

// Samples reports how many samples per pixel have been accumulated so far.
func (a *Accumulator) Samples() int { return a.samples }

// AddSamples folds another opt.Samples paths per pixel into the running average.
func (a *Accumulator) AddSamples(s *Scene, cam Camera, opt PathOptions) {
	o := opt
	o.Seed = opt.Seed + uint64(a.samples) + 1
	batch := pathSum(s, cam, a.w, a.h, o)
	for i := range a.sum {
		a.sum[i] = a.sum[i].Add(batch[i])
	}
	spp := opt.Samples
	if spp < 1 {
		spp = 1
	}
	a.samples += spp
}

// Image tone-maps the current running average into an RGBA image.
func (a *Accumulator) Image() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, a.w, a.h))
	inv := 1.0
	if a.samples > 0 {
		inv = 1.0 / float64(a.samples)
	}
	for y := 0; y < a.h; y++ {
		for x := 0; x < a.w; x++ {
			img.SetRGBA(x, y, toRGBA(a.sum[y*a.w+x].Scale(inv)))
		}
	}
	return img
}
