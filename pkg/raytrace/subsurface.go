// subsurface.go adds volumetric subsurface scattering (SSS) by brute-force random
// walk — the approach modern production path tracers use for skin, wax, marble and
// milk. A ray that strikes an SSS surface either Fresnel-reflects (the boundary
// highlight, handled in radiance) or refracts in; once inside it walks a
// homogeneous medium: at each step it samples an exponential free-flight distance
// from the mean free path, and if that lands before the next boundary it scatters
// isotropically and attenuates by the medium's single-scatter albedo; otherwise it
// reaches the boundary and refracts back out (or totally-internally reflects and
// keeps walking). It is unbiased and reuses the same refract()/closest() machinery
// as the glass path. Clean-room implementation.
package raytrace

import "math"

const sssMaxSteps = 256

// walkSubsurface random-walks the interior medium of an SSS object. The incoming
// ray r has just hit the boundary from outside at h (a transmission was chosen by
// the caller). It returns the ray leaving the surface, the accumulated throughput
// along the walk, and ok=false if the photon was absorbed or the step budget was
// spent (terminate the path).
func (s *Scene) walkSubsurface(r Ray, h Hit, rg *rng) (Ray, Vec3, bool) {
	ior := h.Mat.SSS

	// Refract into the medium (front face: eta = 1/ior). Guard the rare failure.
	dir, ok := refract(r.Dir, h.N, 1/ior)
	if !ok {
		out := Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: r.Dir.Reflect(h.N).Norm(), Time: r.Time}
		return out, Vec3{X: 1, Y: 1, Z: 1}, true
	}

	albedo := h.Mat.SSSColor
	if albedo.LenSq() == 0 {
		albedo = h.albedo()
	}
	sigmaT := 1.0
	if h.Mat.SSSRadius > 0 {
		sigmaT = 1 / h.Mat.SSSRadius
	}

	pos := h.P.Sub(h.N.Scale(shadowEps)) // step just inside the surface
	tput := Vec3{X: 1, Y: 1, Z: 1}

	for step := 0; step < sssMaxSteps; step++ {
		ir := Ray{Origin: pos, Dir: dir, Time: r.Time}
		hit, hok := s.closest(ir, shadowEps, tFar)
		boundDist := tFar
		if hok {
			boundDist = hit.T
		}

		// Sample an exponential free-flight distance (mean free path 1/sigmaT).
		d := -math.Log(1-rg.f()) / sigmaT
		if d < boundDist {
			// Real scatter event inside the medium.
			pos = ir.At(d)
			tput = tput.Mul(albedo)
			dir = rg.unit() // isotropic phase function
			if step >= 4 {  // Russian roulette once throughput has dropped.
				p := math.Max(tput.X, math.Max(tput.Y, tput.Z))
				if p < 1 {
					if rg.f() > p {
						return Ray{}, Vec3{}, false
					}
					tput = tput.Scale(1 / p)
				}
			}
			continue
		}

		if !hok {
			return Ray{}, Vec3{}, false // open medium (non-closed object): give up
		}

		// Reached the boundary: hit.N faces against ir.Dir, i.e. back into the
		// medium. Try to refract out (inside -> outside, eta = ior).
		outDir, refrOK := refract(ir.Dir, hit.N, ior)
		cosI := math.Min(1, math.Max(0, -ir.Dir.Dot(hit.N)))
		r0 := (ior - 1) / (ior + 1)
		r0 *= r0
		fr := r0 + (1-r0)*math.Pow(1-cosI, 5)
		if !refrOK || rg.f() < fr {
			// Total internal reflection / Fresnel reflection: stay inside.
			pos = hit.P.Add(hit.N.Scale(shadowEps)) // hit.N points inward
			dir = ir.Dir.Reflect(hit.N).Norm()
			continue
		}
		// Exit the medium.
		out := Ray{Origin: hit.P.Add(outDir.Scale(shadowEps)), Dir: outDir, Time: r.Time}
		return out, tput, true
	}
	return Ray{}, Vec3{}, false // step budget exhausted
}
