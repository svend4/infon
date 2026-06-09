// bdpt.go is bidirectional path tracing (Veach): the full estimator that completes
// the light tracer (6a, which did the light->camera strategy). It builds an eye
// subpath from the camera and a light subpath from a source, then connects every
// camera vertex t>=2 to every light vertex s>=0 and combines all those strategies
// with multiple importance sampling (balance heuristic). Each complete light path
// is thus sampled many ways and weighted so the strategies sum to one - unbiased -
// while each strategy covers the transport it is best at (s=0/1 = path tracing and
// next-event; larger s = light landing on a surface the eye sees, i.e. caustics on
// diffuse receivers). Implemented for diffuse surfaces + emissive-sphere area
// lights, validated to match the path tracer's mean. Clean-room implementation.
package raytrace

import "math"

// BDPTOptions controls bidirectional path tracing.
type BDPTOptions struct {
	Samples  int // paths per pixel
	MaxDepth int // maximum path length (edges); default 5
	Seed     uint64
}

// bdVert is one path vertex (camera, a diffuse surface, or an area light).
type bdVert struct {
	p, n   Vec3
	albedo Vec3
	beta   Vec3    // subpath throughput up to and including this vertex
	pdfFwd float64 // area-measure pdf, sampled along the subpath direction
	pdfRev float64 // area-measure pdf, sampled in the reverse direction
	emit   Vec3    // emitted radiance (light vertices)
	light  bool
	camera bool
}

// g2 returns the geometry term cos*cos/dist^2 between two vertices and the unit
// direction a->b with the distance.
func bdGeom(a, b bdVert) (float64, Vec3, float64) {
	d := b.p.Sub(a.p)
	dist2 := d.LenSq()
	if dist2 < geomEps {
		return 0, Vec3{}, 0
	}
	dist := math.Sqrt(dist2)
	w := d.Scale(1 / dist)
	ca := 1.0
	if !a.camera {
		ca = math.Abs(a.n.Dot(w))
	}
	cb := math.Abs(b.n.Dot(w))
	return ca * cb / dist2, w, dist
}

// convert turns a solid-angle pdf at `from` (for the direction toward `to`) into an
// area-measure pdf at `to`.
func convertPdf(pdfW float64, from, to bdVert) float64 {
	d := to.p.Sub(from.p)
	dist2 := d.LenSq()
	if dist2 < geomEps {
		return 0
	}
	c := 1.0
	if !to.camera {
		w := d.Scale(1 / math.Sqrt(dist2))
		c = math.Abs(to.n.Dot(w))
	}
	return pdfW * c / dist2
}

// cosPdfTo is the diffuse (cosine) solid-angle pdf at vertex v for the direction
// toward point q (0 if below the hemisphere). Camera vertices use a pinhole pdf
// supplied at generation time, so this is only called on surface vertices.
func (v bdVert) cosPdfTo(q Vec3) float64 {
	w := q.Sub(v.p)
	d := w.Len()
	if d < geomEps {
		return 0
	}
	c := v.n.Dot(w.Scale(1 / d))
	if c <= 0 {
		return 0
	}
	return c / math.Pi
}

// generateLightSubpath builds a light subpath of up to maxDepth vertices.
func (s *Scene) generateLightSubpath(maxDepth int, rg *rng) []bdVert {
	nE := len(s.emit)
	if nE == 0 || maxDepth < 1 {
		return nil
	}
	e := s.emit[int(rg.next()%uint64(nE))]
	nrm := rg.unit()
	p := e.Center.Add(nrm.Scale(e.Radius))
	pdfPos := 1 / (float64(nE) * 4 * math.Pi * e.Radius * e.Radius) // area measure
	// light-vertex throughput is Le/pdfPos (so an s=1 connection matches NEE).
	verts := []bdVert{{p: p, n: nrm, emit: e.Mat.Emit, light: true, pdfFwd: pdfPos, beta: e.Mat.Emit.Scale(1 / pdfPos)}}

	// emit a cosine-distributed direction; the next vertex carries Le*pi/pdfPos.
	dir := cosineSample(nrm, rg)
	beta := verts[0].beta.Scale(math.Pi)
	r := Ray{Origin: p.Add(nrm.Scale(shadowEps)), Dir: dir}
	prevPdfW := math.Max(0, nrm.Dot(dir)) / math.Pi // cosine emission pdf (solid angle)

	for len(verts) < maxDepth {
		hit, ok := s.closest(r, shadowEps, tFar)
		if !ok || !isDiffuseMat(hit.Mat) {
			break // stop at escape or specular (delta connections unsupported here)
		}
		v := bdVert{p: hit.P, n: hit.N, albedo: hit.albedo(), beta: beta}
		v.pdfFwd = convertPdf(prevPdfW, verts[len(verts)-1], v)
		verts = append(verts, v)
		// reverse pdf of the previous vertex, as seen from this one (diffuse cosine).
		pi := len(verts) - 2
		verts[pi].pdfRev = convertPdf(v.cosPdfTo(verts[pi].p), v, verts[pi])

		alb := hit.albedo()
		ndir := cosineSample(hit.N, rg)
		prevPdfW = math.Max(0, hit.N.Dot(ndir)) / math.Pi
		beta = beta.Mul(alb) // diffuse: f*cos/pdf = albedo
		r = Ray{Origin: hit.P.Add(hit.N.Scale(shadowEps)), Dir: ndir}
	}
	return verts
}

// generateEyeSubpath builds an eye subpath from the camera ray.
func (s *Scene) generateEyeSubpath(camPos Vec3, first Ray, pdfCamDir float64, maxDepth int, rg *rng) []bdVert {
	cam := bdVert{p: camPos, camera: true, beta: Vec3{X: 1, Y: 1, Z: 1}, pdfFwd: 1}
	verts := []bdVert{cam}
	beta := Vec3{X: 1, Y: 1, Z: 1}
	r := first
	prevPdfW := pdfCamDir
	for len(verts) < maxDepth {
		hit, ok := s.closest(r, shadowEps, tFar)
		if !ok || !isDiffuseMat(hit.Mat) {
			// record a terminal vertex carrying any emission/sky so s=0 works.
			if ok {
				v := bdVert{p: hit.P, n: hit.N, albedo: hit.albedo(), beta: beta, emit: hit.Mat.Emit, light: hit.Mat.Emit.LenSq() > 0}
				prev := &verts[len(verts)-1]
				v.pdfFwd = convertPdf(prevPdfW, *prev, v)
				verts = append(verts, v)
			}
			break
		}
		v := bdVert{p: hit.P, n: hit.N, albedo: hit.albedo(), beta: beta, emit: hit.Mat.Emit, light: hit.Mat.Emit.LenSq() > 0}
		v.pdfFwd = convertPdf(prevPdfW, verts[len(verts)-1], v)
		verts = append(verts, v)
		pi := len(verts) - 2
		verts[pi].pdfRev = convertPdf(v.cosPdfTo(verts[pi].p), v, verts[pi])
		if v.light {
			break // a light is a terminal eye vertex
		}
		alb := hit.albedo()
		ndir := cosineSample(hit.N, rg)
		prevPdfW = math.Max(0, hit.N.Dot(ndir)) / math.Pi
		beta = beta.Mul(alb)
		r = Ray{Origin: hit.P.Add(hit.N.Scale(shadowEps)), Dir: ndir}
	}
	return verts
}
