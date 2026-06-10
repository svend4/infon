package raytrace

import (
	"image"
	"math"
)

// pdfLightOrigin is the area-measure pdf of sampling the light vertex v as a light
// origin (uniform light choice * uniform point), for the s=0 MIS term.
func (s *Scene) pdfLightOrigin(v bdVert) float64 {
	r, ok := s.emitterRadius(v.p)
	if !ok || len(s.emit) == 0 {
		return 0
	}
	return 1 / (float64(len(s.emit)) * 4 * math.Pi * r * r)
}

// bdConnect returns the unweighted contribution of connecting the light subpath
// (first s vertices) to the eye subpath (first t vertices), with visibility.
func (s *Scene) bdConnect(light, eye []bdVert, sIdx, tIdx int) Vec3 {
	pt := eye[tIdx-1]
	if sIdx == 0 {
		if !pt.light {
			return Vec3{}
		}
		return pt.beta.Mul(pt.emit) // eye path reached a light (path-tracing strategy)
	}
	if pt.light {
		return Vec3{} // a light eye-vertex is only valid via s=0
	}
	qs := light[sIdx-1]
	g, w, dist := bdGeom(pt, qs)
	if g <= 0 {
		return Vec3{}
	}
	if pt.n.Dot(w) <= 0 || qs.n.Dot(w.Neg()) <= 0 {
		return Vec3{}
	}
	if s.anyHit(Ray{Origin: pt.p.Add(pt.n.Scale(shadowEps)), Dir: w}, shadowEps, dist-2*shadowEps) {
		return Vec3{}
	}
	fe := pt.albedo.Scale(1 / math.Pi) // diffuse BSDF at the eye vertex
	if sIdx == 1 {
		// qs is the light source: it emits qs.beta (= Le/pdfPos) toward pt.
		return pt.beta.Mul(fe).Mul(qs.beta).Scale(g)
	}
	fl := qs.albedo.Scale(1 / math.Pi) // diffuse BSDF at the light vertex
	return pt.beta.Mul(fe).Mul(fl).Mul(qs.beta).Scale(g)
}

// bdMIS returns the balance-heuristic MIS weight for strategy (sIdx,tIdx). It
// temporarily rewrites the reverse pdfs of the four vertices around the
// connection, runs the standard ratio recurrence, then restores them.
func (s *Scene) bdMIS(light, eye []bdVert, sIdx, tIdx int) float64 {
	if sIdx+tIdx == 2 {
		return 1
	}
	type sv struct {
		v   *bdVert
		rev float64
	}
	var saves []sv
	setRev := func(v *bdVert, rev float64) {
		saves = append(saves, sv{v, v.pdfRev})
		v.pdfRev = rev
	}

	pt := &eye[tIdx-1]
	var ptMinus *bdVert
	if tIdx >= 2 {
		ptMinus = &eye[tIdx-2]
	}
	var qs, qsMinus *bdVert
	if sIdx >= 1 {
		qs = &light[sIdx-1]
	}
	if sIdx >= 2 {
		qsMinus = &light[sIdx-2]
	}

	// pt.pdfRev: pt as seen from the light side.
	if sIdx == 0 {
		setRev(pt, s.pdfLightOrigin(*pt))
	} else {
		setRev(pt, convertPdf(qs.cosPdfTo(pt.p), *qs, *pt))
	}
	// ptMinus.pdfRev: from pt toward ptMinus.
	if ptMinus != nil {
		setRev(ptMinus, convertPdf(pt.cosPdfTo(ptMinus.p), *pt, *ptMinus))
	}
	// qs.pdfRev: from pt toward qs.
	if qs != nil {
		setRev(qs, convertPdf(pt.cosPdfTo(qs.p), *pt, *qs))
	}
	// qsMinus.pdfRev: from qs toward qsMinus.
	if qsMinus != nil {
		setRev(qsMinus, convertPdf(qs.cosPdfTo(qsMinus.p), *qs, *qsMinus))
	}

	remap0 := func(f float64) float64 {
		if f == 0 {
			return 1
		}
		return f
	}
	sumRi := 0.0
	ri := 1.0
	// Stop at i>1: this BDPT does not sample the t=1 (light->camera) strategy -- that
	// is the separate light tracer -- so it must not appear in the partition either.
	for i := tIdx - 1; i > 1; i-- {
		ri *= remap0(eye[i].pdfRev) / remap0(eye[i].pdfFwd)
		sumRi += ri
	}
	ri = 1.0
	for i := sIdx - 1; i >= 0; i-- {
		ri *= remap0(light[i].pdfRev) / remap0(light[i].pdfFwd)
		sumRi += ri
	}

	for i := len(saves) - 1; i >= 0; i-- {
		saves[i].v.pdfRev = saves[i].rev
	}
	return 1 / (1 + sumRi)
}

// BDPTRender renders with bidirectional path tracing.
func BDPTRender(s *Scene, cam Camera, w, h int, opt BDPTOptions) image.Image {
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}
	return tonemapSum(s.bdptLinear(cam, w, h, opt), w, h, 1)
}

// bdptLinear returns the per-pixel linear radiance (pre-tone-map) for unbiasedness
// checks without the nonlinear tone curve.
func (s *Scene) bdptLinear(cam Camera, w, h int, opt BDPTOptions) []Vec3 {
	spp := opt.Samples
	if spp < 1 {
		spp = 1
	}
	maxDepth := opt.MaxDepth
	if maxDepth < 1 {
		maxDepth = 5
	}
	s.gatherEmitters()
	b := cam.basis(w, h)
	area := 4 * b.halfW * b.halfH
	buf := make([]Vec3, w*h)

	parallelRows(h, func(y int) {
		for x := 0; x < w; x++ {
			seed := opt.Seed*0x9e3779b97f4a7c15 + uint64(y)*0x100000001b3 + uint64(x)*0x85ebca6b + 1
			rg := newRNG(seed)
			var acc Vec3
			for i := 0; i < spp; i++ {
				u := float64(x) + rg.f()
				v := float64(y) + rg.f()
				ray := b.ray(u, v)
				cosCam := ray.Dir.Dot(b.forward)
				if cosCam <= 0 {
					continue
				}
				pdfCamDir := 1 / (area * cosCam * cosCam * cosCam)
				eye := s.generateEyeSubpath(cam.Pos, ray, pdfCamDir, maxDepth+1, rg)
				lt := s.generateLightSubpath(maxDepth, rg)
				var L Vec3
				for t := 2; t <= len(eye); t++ {
					for ss := 0; ss <= len(lt); ss++ {
						if ss+t-2 > maxDepth { // path length (edges) budget
							continue
						}
						c := s.bdConnect(lt, eye, ss, t)
						if c.LenSq() == 0 {
							continue
						}
						L = L.Add(c.Scale(s.bdMIS(lt, eye, ss, t)))
					}
				}
				acc = acc.Add(L)
			}
			buf[y*w+x] = acc.Scale(1 / float64(spp))
		}
	})
	return buf
}
