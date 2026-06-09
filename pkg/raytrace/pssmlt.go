// pssmlt.go is Metropolis Light Transport in primary sample space (Kelemen et al.
// 2002) - the tractable, correct MLT. The path tracer is a deterministic map from a
// vector of uniforms (the "primary samples") to a path contribution; PSSMLT runs a
// Markov chain over that vector, proposing large steps (fresh vector, global
// exploration) and small steps (a local perturbation), and accepting them so the
// chain visits samples in proportion to their contribution. Bright, hard-to-find
// paths, once found, are explored by nearby mutations instead of being rediscovered
// from scratch - so it shines where independent sampling struggles. A bootstrap
// phase estimates the normalisation b = E[contribution] so the result is unbiased:
// the stationary image matches the path tracer. Clean-room implementation.
package raytrace

import (
	"image"
	"math"
)

// pssState holds the mutatable primary-sample vector that drives one path
// evaluation. f()/next() on an rng with a non-nil pss read from here, generating
// coordinates lazily (fresh on a large step or beyond the base length; a local
// perturbation of the base otherwise).
type pssState struct {
	u, base []float64
	i       int
	large   bool
	sigma   float64
	src     uint64
}

func (p *pssState) rnd() float64 {
	p.src ^= p.src << 13
	p.src ^= p.src >> 7
	p.src ^= p.src << 17
	return float64(p.src>>11) / float64(uint64(1)<<53)
}

// next returns the next primary sample, extending the vector on demand.
func (p *pssState) next() float64 {
	idx := p.i
	p.i++
	for len(p.u) <= idx {
		j := len(p.u)
		if p.large || j >= len(p.base) {
			p.u = append(p.u, p.rnd())
		} else {
			v := p.base[j] + p.sigma*(2*p.rnd()-1) // symmetric perturbation
			p.u = append(p.u, v-math.Floor(v))     // wrap to [0,1)
		}
	}
	return p.u[idx]
}

// MLTOptions controls Metropolis light transport.
type MLTOptions struct {
	Mutations     int     // chain length (total mutations)
	Bootstrap     int     // bootstrap samples for the normalisation b
	MaxDepth      int     // path depth
	LargeStepProb float64 // probability of a large step (default 0.3)
	Sigma         float64 // small-step size (default 0.01)
	NEE, MIS      bool
	Seed          uint64
}

// mltSample evaluates the path tracer for a primary-sample vector, returning the
// pixel it lands on and its radiance.
func (s *Scene) mltSample(cam Camera, b camBasis, w, h, depth, mode int, ps *pssState) (int, int, Vec3) {
	ps.i = 0
	rg := &rng{pss: ps}
	fx := rg.f() * float64(w)
	fy := rg.f() * float64(h)
	px := int(fx)
	py := int(fy)
	if px < 0 {
		px = 0
	} else if px >= w {
		px = w - 1
	}
	if py < 0 {
		py = 0
	} else if py >= h {
		py = h - 1
	}
	c := s.radiance(b.lensRay(fx, fy, cam, rg), depth, rg, mode)
	return px, py, c
}

// MLTRender renders with primary-sample-space Metropolis light transport.
func MLTRender(s *Scene, cam Camera, w, h int, opt MLTOptions) image.Image {
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}
	return tonemapSum(s.mltLinear(cam, w, h, opt), w, h, 1)
}

// mltLinear runs the chain and returns the per-pixel linear radiance (pre-tone-map).
func (s *Scene) mltLinear(cam Camera, w, h int, opt MLTOptions) []Vec3 {
	n := opt.Mutations
	if n < 1 {
		n = 1 << 20
	}
	m := opt.Bootstrap
	if m < 1 {
		m = 10000
	}
	depth := opt.MaxDepth
	if depth < 1 {
		depth = 6
	}
	pLarge := opt.LargeStepProb
	if pLarge <= 0 {
		pLarge = 0.3
	}
	sigma := opt.Sigma
	if sigma <= 0 {
		sigma = 0.01
	}
	mode := 0
	if opt.MIS {
		mode = 2
	} else if opt.NEE {
		mode = 1
	}
	s.gatherEmitters()
	b := cam.basis(w, h)
	buf := make([]Vec3, w*h)
	src := newRNG(opt.Seed | 1)

	// bootstrap: estimate b = E[contribution] and keep candidate chain starts.
	type boot struct {
		u []float64
		f float64
	}
	samples := make([]boot, 0, m)
	var bsum float64
	for k := 0; k < m; k++ {
		ps := &pssState{large: true, sigma: sigma, src: src.next() | 1}
		_, _, c := s.mltSample(cam, b, w, h, depth, mode, ps)
		fl := lum3(c)
		bsum += fl
		samples = append(samples, boot{append([]float64(nil), ps.u...), fl})
	}
	bConst := bsum / float64(m)
	if bConst <= 0 {
		return buf // nothing emits
	}

	// start the chain at a bootstrap sample chosen proportional to contribution.
	target := src.f() * bsum
	startU := samples[len(samples)-1].u
	var acc float64
	for _, sm := range samples {
		acc += sm.f
		if acc >= target {
			startU = sm.u
			break
		}
	}

	cur := &pssState{u: append([]float64(nil), startU...), sigma: sigma, src: src.next() | 1}
	cpx, cpy, cC := s.mltSample(cam, b, w, h, depth, mode, cur)
	cF := lum3(cC)

	for step := 0; step < n; step++ {
		prop := &pssState{base: cur.u, large: src.f() < pLarge, sigma: sigma, src: src.next() | 1}
		ppx, ppy, pC := s.mltSample(cam, b, w, h, depth, mode, prop)
		pF := lum3(pC)
		a := 1.0
		if cF > 0 {
			a = math.Min(1, pF/cF)
		}
		// expected-value splatting: deposit at both states by their dwell weight.
		if cF > 0 {
			buf[cpy*w+cpx] = buf[cpy*w+cpx].Add(cC.Scale((1 - a) / cF))
		}
		if pF > 0 {
			buf[ppy*w+ppx] = buf[ppy*w+ppx].Add(pC.Scale(a / pF))
		}
		if src.f() < a {
			cur, cpx, cpy, cC, cF = prop, ppx, ppy, pC, pF
		}
	}

	scale := bConst * float64(w*h) / float64(n)
	for i := range buf {
		buf[i] = buf[i].Scale(scale)
	}
	return buf
}
