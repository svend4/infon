// reservoir.go adds ReSTIR-style direct lighting: resampled importance sampling
// (RIS) with a weighted reservoir. Plain next-event estimation picks one light
// uniformly and shadow-tests it; in a scene with many lights of varying
// brightness and distance, most picks contribute almost nothing, so the estimate
// is noisy. RIS instead draws several cheap candidate light samples, weights each
// by its *unshadowed* contribution, resamples one in proportion (a streaming
// weighted reservoir), and casts a shadow ray only for the survivor. It is the
// core estimator behind ReSTIR; spatiotemporal reservoir reuse is the further
// layer on top. Unbiased: with one candidate it is exactly the single-sample NEE
// estimator. Clean-room implementation.
package raytrace

import "math"

// lum3 is the scalar target (luminance) used to weight candidates.
func lum3(v Vec3) float64 { return 0.2126*v.X + 0.7152*v.Y + 0.0722*v.Z }

// lightCand is one resampled candidate: the point on the light, the direction and
// distance to it, the full (unshadowed) contribution there, and the scalar target.
type lightCand struct {
	q     Vec3
	wi    Vec3
	dist  float64
	fArea Vec3
	pHat  float64
}

// reservoir keeps one candidate chosen with probability proportional to its
// weight (weighted reservoir sampling) and the running sum of weights.
type reservoir struct {
	y    lightCand
	wsum float64
	got  bool
}

func (r *reservoir) update(c lightCand, w float64, rg *rng) {
	if w <= 0 {
		return
	}
	r.wsum += w
	if rg.f()*r.wsum < w { // replace with probability w / wsum
		r.y = c
		r.got = true
	}
}

// sampleEmittersRIS is the RIS next-event estimator: resample one of m candidate
// light samples (weighted by unshadowed contribution) and return its visible
// direct contribution. With m == 1 it equals sampleEmitters (NEE, no MIS).
func (s *Scene) sampleEmittersRIS(h Hit, alb Vec3, rg *rng, time float64, m int) Vec3 {
	n := len(s.emit)
	if n == 0 {
		return Vec3{}
	}
	if m < 1 {
		m = 1
	}
	var res reservoir
	for i := 0; i < m; i++ {
		li := s.emit[int(rg.next()%uint64(n))]
		q := li.Center.Add(rg.unit().Scale(li.Radius))
		d := q.Sub(h.P)
		dist := d.Len()
		if dist < geomEps {
			continue
		}
		wi := d.Scale(1 / dist)
		cosS := h.N.Dot(wi)
		if cosS <= 0 {
			continue
		}
		cosL := q.Sub(li.Center).Norm().Dot(wi.Neg())
		if cosL <= 0 {
			continue
		}
		// Full unshadowed contribution f_area = (alb/pi) * Le * cosS * cosL / dist^2,
		// drawn from the area pdf p = 1 / (n * 4pi r^2); candidate weight = pHat/p.
		areaPdf := 1 / (float64(n) * 4 * math.Pi * li.Radius * li.Radius)
		fArea := alb.Scale(1 / math.Pi).Mul(li.Mat.Emit).Scale(cosS * cosL / (dist * dist))
		pHat := lum3(fArea)
		res.update(lightCand{q: q, wi: wi, dist: dist, fArea: fArea, pHat: pHat}, pHat/areaPdf, rg)
	}
	if !res.got || res.y.pHat <= 0 {
		return Vec3{}
	}
	// Shadow-test only the survivor.
	so := h.P.Add(h.N.Scale(shadowEps))
	if s.anyHit(Ray{Origin: so, Dir: res.y.wi, Time: time}, shadowEps, res.y.dist-2*shadowEps) {
		return Vec3{}
	}
	// RIS unbiased contribution weight W = (1/m) * sum(w_i) / pHat(y).
	w := (res.wsum / float64(m)) / res.y.pHat
	return res.y.fArea.Scale(w)
}
