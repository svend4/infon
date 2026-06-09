// medium.go adds heterogeneous participating media (smoke, clouds, varying fog)
// to the path tracer. The extinction sigma_t varies in space, so free flights
// can't be sampled in closed form; instead it uses delta tracking (Woodcock
// tracking): step with an exponential distance from a constant majorant SigmaMax
// (an upper bound on sigma_t), then accept the collision as "real" with
// probability sigma_t(x)/SigmaMax, otherwise treat it as a fictitious "null"
// collision and continue. The fictitious collisions exactly compensate for the
// homogenised majorant, so the estimator is unbiased and independent of the
// majorant. Shadow/NEE rays use the matching unbiased transmittance (ratio
// tracking). Clean-room implementation.
package raytrace

import "math"

// Medium is a spherical region of heterogeneous scattering medium. Density returns
// the extinction sigma_t at a point (0 <= sigma_t <= SigmaMax); Albedo is the
// single-scatter albedo (scatter vs absorb); G is the Henyey-Greenstein
// anisotropy (0 isotropic, >0 forward, <0 back).
type Medium struct {
	Center   Vec3
	Radius   float64
	SigmaMax float64
	Albedo   Vec3
	G        float64
	Density  func(p Vec3) float64
}

// sigma evaluates the extinction at p, clamped to [0, SigmaMax].
func (m *Medium) sigma(p Vec3) float64 {
	if m.Density == nil {
		return m.SigmaMax
	}
	d := m.Density(p)
	if d < 0 {
		return 0
	}
	if d > m.SigmaMax {
		return m.SigmaMax
	}
	return d
}

// span returns the ray's overlap with the medium sphere, clamped to (0, tMax].
func (m *Medium) span(r Ray, tMax float64) (t0, t1 float64, ok bool) {
	oc := r.Origin.Sub(m.Center)
	b := oc.Dot(r.Dir)
	c := oc.LenSq() - m.Radius*m.Radius
	disc := b*b - c
	if disc < 0 {
		return 0, 0, false
	}
	sq := math.Sqrt(disc)
	t0, t1 = -b-sq, -b+sq
	if t0 < shadowEps {
		t0 = shadowEps
	}
	if t1 > tMax {
		t1 = tMax
	}
	return t0, t1, t1 > t0
}

// sampleCollision delta-tracks the next real collision in [t0,t1]; ok=false means
// the ray passed through the medium without a real interaction.
func (m *Medium) sampleCollision(r Ray, t0, t1 float64, rg *rng) (t float64, ok bool) {
	if m.SigmaMax <= 0 {
		return 0, false
	}
	t = t0
	for {
		t += -math.Log(1-rg.f()) / m.SigmaMax
		if t >= t1 {
			return 0, false
		}
		if m.sigma(r.At(t))/m.SigmaMax > rg.f() {
			return t, true // real collision
		}
		// otherwise a null collision: keep going
	}
}

// transmittance is the unbiased ratio-tracking estimate of exp(-integral sigma_t)
// along [t0,t1].
func (m *Medium) transmittance(r Ray, t0, t1 float64, rg *rng) float64 {
	if m.SigmaMax <= 0 {
		return 1
	}
	tr := 1.0
	t := t0
	for {
		t += -math.Log(1-rg.f()) / m.SigmaMax
		if t >= t1 {
			return tr
		}
		tr *= 1 - m.sigma(r.At(t))/m.SigmaMax
		if tr < 1e-4 { // negligible; stop early (still unbiased in expectation)
			return 0
		}
	}
}

// hgPhase is the Henyey-Greenstein phase function value for the cosine between the
// incoming and outgoing directions.
func hgPhase(cos, g float64) float64 {
	d := 1 + g*g - 2*g*cos
	return (1 - g*g) / (4 * math.Pi * d * math.Sqrt(math.Max(d, 1e-12)))
}

// hgSample draws a scattered direction around the propagation direction w by the
// Henyey-Greenstein distribution (importance-sampled, so the weight is 1).
func hgSample(w Vec3, g float64, rg *rng) Vec3 {
	var cosT float64
	if math.Abs(g) < 1e-3 {
		cosT = 1 - 2*rg.f() // isotropic
	} else {
		// inverse-CDF for the HG cosine, in the convention where cos is measured
		// between the incoming propagation direction w and the scattered direction
		// (so g>0 scatters forward, mean cosine = g), matching hgPhase.
		s := (1 - g*g) / (1 + g - 2*g*rg.f())
		cosT = (1 + g*g - s*s) / (2 * g)
	}
	sinT := math.Sqrt(math.Max(0, 1-cosT*cosT))
	phi := 2 * math.Pi * rg.f()
	t, b := basisAround(w)
	return t.Scale(sinT * math.Cos(phi)).Add(b.Scale(sinT * math.Sin(phi))).Add(w.Scale(cosT)).Norm()
}

// mediumNEE is next-event estimation at a scatter point p (incoming propagation
// wIn): sample an emitter, test surface visibility, attenuate by the medium
// transmittance, and weight by the phase function. The single-scatter albedo is
// applied by the caller.
func (s *Scene) mediumNEE(p, wIn Vec3, rg *rng, time float64) Vec3 {
	n := len(s.emit)
	if n == 0 || s.Medium == nil {
		return Vec3{}
	}
	li := s.emit[int(rg.next()%uint64(n))]
	q := li.Center.Add(rg.unit().Scale(li.Radius))
	d := q.Sub(p)
	dist := d.Len()
	if dist < geomEps {
		return Vec3{}
	}
	wL := d.Scale(1 / dist)
	if s.anyHit(Ray{Origin: p, Dir: wL, Time: time}, shadowEps, dist-2*shadowEps) {
		return Vec3{} // a surface blocks the light
	}
	tr := 1.0
	if t0, t1, in := s.Medium.span(Ray{Origin: p, Dir: wL}, dist); in {
		tr = s.Medium.transmittance(Ray{Origin: p, Dir: wL}, t0, t1, rg)
	}
	cosL := q.Sub(li.Center).Norm().Dot(wL.Neg())
	if cosL <= 0 {
		return Vec3{}
	}
	areaPdf := 1 / (float64(n) * 4 * math.Pi * li.Radius * li.Radius)
	pdfL := areaPdf * dist * dist / cosL
	ph := hgPhase(wIn.Dot(wL), s.Medium.G)
	return li.Mat.Emit.Scale(ph * tr / pdfL)
}
