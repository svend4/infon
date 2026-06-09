// lighttrace.go is a light (particle) tracer: the complement of the eye-based path
// tracer and the first half of bidirectional path tracing. It shoots paths FROM
// the lights and, at every diffuse vertex, connects directly to the camera -
// projecting the vertex to its pixel, testing visibility, and splatting its
// contribution there. Tracing light->surface->...->eye captures exactly the
// transport the eye tracer struggles with (caustics that reach the camera), and
// is unbiased: averaged, the image matches the path tracer on diffuse scenes.
// Specular vertices are passed through (their delta BRDF can't be connected to a
// point camera); directly-visible lights and mirrors are the eye subpath's job in
// full BDPT. Clean-room implementation.
package raytrace

import (
	"image"
	"math"
)

// LightTraceOptions controls the light tracer.
type LightTraceOptions struct {
	Paths     int // number of light paths to trace
	MaxBounce int // bounce budget per light path (default 8)
	Seed      uint64
}

func isDiffuseMat(m Material) bool {
	return m.Glass == 0 && m.Metal == 0 && m.Reflect == 0 && m.Film == 0 && m.SSS == 0 && !m.Principled
}

// LightTraceRender renders by tracing particles from the lights to the camera.
func LightTraceRender(s *Scene, cam Camera, w, h int, opt LightTraceOptions) image.Image {
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}
	return tonemapSum(lightTraceLinear(s, cam, w, h, opt), w, h, 1)
}

// lightTraceLinear returns the per-pixel linear radiance (pre-tone-map), so the
// estimator can be checked for unbiasedness without the nonlinear tone curve.
func lightTraceLinear(s *Scene, cam Camera, w, h int, opt LightTraceOptions) []Vec3 {
	buf := make([]Vec3, w*h)
	n := opt.Paths
	if n < 1 {
		n = 100000
	}
	maxB := opt.MaxBounce
	if maxB < 1 {
		maxB = 8
	}
	s.gatherEmitters()
	nE := len(s.emit)
	if nE == 0 {
		return buf
	}
	b := cam.basis(w, h)
	area := 4 * b.halfW * b.halfH // image-plane area at unit distance along forward

	rg := newRNG(opt.Seed | 1)
	for k := 0; k < n; k++ {
		e := s.emit[int(rg.next()%uint64(nE))]
		nrm := rg.unit()
		origin := e.Center.Add(nrm.Scale(e.Radius + shadowEps))
		// flux carried by this particle: emitter's emitted power * nE (uniform pick).
		beta := e.Mat.Emit.Scale(4 * math.Pi * math.Pi * e.Radius * e.Radius * float64(nE))
		r := Ray{Origin: origin, Dir: cosineSample(nrm, rg)}
		for d := 0; d < maxB; d++ {
			hit, ok := s.closest(r, shadowEps, tFar)
			if !ok {
				break
			}
			if isDiffuseMat(hit.Mat) {
				s.connectToCamera(cam, b, area, hit, beta, w, h, buf)
				alb := hit.albedo()
				p := math.Max(alb.X, math.Max(alb.Y, alb.Z))
				if p <= 0 || rg.f() > p {
					break
				}
				beta = beta.Mul(alb).Scale(1 / p)
				r = Ray{Origin: hit.P.Add(hit.N.Scale(shadowEps)), Dir: cosineSample(hit.N, rg)}
				continue
			}
			// specular vertex: pass the particle through, no camera connection.
			switch {
			case hit.Mat.Glass > 0:
				r = s.scatterGlass(r, hit, rg)
			default: // mirror / metal
				if a := hit.albedo(); a.LenSq() > 0 {
					beta = beta.Mul(a)
				}
				r = Ray{Origin: hit.P.Add(hit.N.Scale(shadowEps)), Dir: r.Dir.Reflect(hit.N).Norm()}
			}
		}
	}

	// Light tracing accumulates one path per (image area / pixel); normalise so the
	// average matches the eye tracer.
	scale := float64(w*h) / float64(n)
	for i := range buf {
		buf[i] = buf[i].Scale(scale)
	}
	return buf
}

// connectToCamera splats a diffuse light-path vertex's contribution to its pixel.
func (s *Scene) connectToCamera(cam Camera, b camBasis, area float64, hit Hit, beta Vec3, w, h int, buf []Vec3) {
	toCam := cam.Pos.Sub(hit.P)
	dist := toCam.Len()
	if dist < geomEps {
		return
	}
	wc := toCam.Scale(1 / dist) // surface -> camera
	dn := wc.Neg()              // camera -> surface
	cosCam := dn.Dot(b.forward) // angle of the incoming ray at the camera vs forward
	if cosCam <= geomEps {      // behind the camera
		return
	}
	// project onto the image plane at unit forward distance.
	plane := dn.Scale(1 / cosCam)
	uc := plane.Dot(b.right)
	vc := plane.Dot(b.up)
	if math.Abs(uc) > b.halfW || math.Abs(vc) > b.halfH {
		return // outside the frame
	}
	px := int((uc/b.halfW + 1) * 0.5 * float64(w))
	py := int((1 - vc/b.halfH) * 0.5 * float64(h))
	if px < 0 || px >= w || py < 0 || py >= h {
		return
	}
	cosX := hit.N.Dot(wc)
	if cosX <= 0 {
		return
	}
	if s.anyHit(Ray{Origin: hit.P.Add(hit.N.Scale(shadowEps)), Dir: wc}, shadowEps, dist-2*shadowEps) {
		return // camera is occluded from the vertex
	}
	// We = 1/(area * cosCam^4) is the pinhole importance; G = cosX*cosCam/dist^2.
	we := 1 / (area * cosCam * cosCam * cosCam * cosCam)
	g := cosX * cosCam / (dist * dist)
	fr := hit.albedo().Scale(1 / math.Pi) // diffuse BRDF
	buf[py*w+px] = buf[py*w+px].Add(beta.Mul(fr).Scale(g * we))
}
