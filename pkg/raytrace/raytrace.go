// Package raytrace is a small, dependency-light CPU ray tracer that renders to an
// image.Image — the same output type as the isometric pkg/fold renderer — so it
// drops straight into every existing consumer: PNG/GIF export, the
// half-block/sextant/octant/braille/sixel/kitty terminal encoders, and the orbit
// demos. It implements analytic spheres, an infinite checkerboard floor and
// triangle meshes (Möller–Trumbore, smooth vertex normals, a BVH acceleration
// structure), Lambert + Blinn-Phong shading with hard or soft shadow rays,
// ambient occlusion, distance attenuation, emissive and dielectric (glass)
// materials with Schlick-Fresnel reflection/refraction, supersampled
// anti-aliasing, ACES tone mapping and a parallel per-row render loop.
//
// It is a clean-room implementation written from scratch (no third-party or
// copied code), so it carries the repository's own licence.
package raytrace

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
)

// Ray is a half-line Origin + t*Dir. Dir is expected to be unit length.
// Ray is a half-line. Time is the shutter fraction in [0,1] the ray is sampled
// at; static objects ignore it, MovingSphere uses it for motion blur.
type Ray struct {
	Origin, Dir Vec3
	Time        float64
}

// At returns the point at parameter t along the ray.
func (r Ray) At(t float64) Vec3 { return r.Origin.Add(r.Dir.Scale(t)) }

// Material describes a surface.
type Material struct {
	Color    Vec3    // base (diffuse / metal tint) colour, channels 0..1
	Reflect  float64 // mirror fraction, 0 = matte .. 1 = perfect mirror
	Spec     float64 // Blinn-Phong specular strength (raster shader; 0 = none)
	Shine    float64 // specular exponent (default 32 when Spec > 0)
	Emit     Vec3    // emissive colour (also an area light in the path tracer)
	Glass    float64 // refractive index (0 = opaque; ~1.5 = glass)
	Rough    float64 // GGX roughness (0 = sharp/mirror .. 1 = very rough); also the glossy spread
	Metal    float64 // metalness for the path tracer's GGX lobe (0 = dielectric .. 1 = metal)
	Disperse float64 // glass only: per-channel IOR spread (chromatic dispersion); 0 = none
	Tex      Texture // optional surface texture; overrides Color when set
	Bump     Texture // optional tangent-space normal map (RGB-encoded); perturbs the shading normal for surface detail without geometry

	// Principled selects the unified Disney-style BSDF (path tracer): one surface
	// blending a Lambert diffuse lobe and a GGX specular lobe, driven by Metal and
	// Rough, instead of choosing a single Reflect/Metal/diffuse behaviour.
	Principled bool

	// Subsurface scattering (volumetric random walk). SSS is the boundary index of
	// refraction (0 = off; ~1.3-1.4 for skin/wax/marble). When set, light refracts
	// into the surface, random-walks a homogeneous interior medium with
	// single-scatter albedo SSSColor (falls back to the base colour), stepping a
	// mean free path of SSSRadius world units between isotropic scatter events,
	// then refracts back out. Smaller SSSRadius = denser / more opaque (marble);
	// larger = more translucent (wax, skin). A Fresnel boundary reflection (the
	// shiny highlight) is kept alongside the soft subsurface term.
	SSS       float64
	SSSColor  Vec3
	SSSRadius float64

	// Thin-film interference (iridescence): Film is the film thickness in
	// nanometres (0 = off; ~100-600 for soap/oil), FilmIOR its refractive index
	// (default 1.33). The surface reflects like a mirror, tinted by the
	// angle/thickness-dependent interference colour.
	Film    float64
	FilmIOR float64
}

// Hit records the nearest intersection along a ray.
type Hit struct {
	T     float64 // ray parameter
	P     Vec3    // world-space hit point
	N     Vec3    // unit shading normal, oriented against the incoming ray
	U, V  float64 // surface texture coordinates (0..1) where defined
	Front bool    // true if the ray struck the outward-facing side
	Mat   Material
	Tan   Vec3    // UV-aligned tangent (set only when a normal map needs it; zero = none)
	TanW  float64 // tangent-space handedness (+1/-1) for the bitangent N x Tan
}

// albedo is the surface's base colour at the hit: the material texture if one is
// set, otherwise the flat Material.Color.
func (h Hit) albedo() Vec3 {
	if h.Mat.Tex != nil {
		return h.Mat.Tex.At(h.U, h.V, h.P)
	}
	return h.Mat.Color
}

// Object is anything a ray can intersect. Intersect reports the nearest hit with
// t in (tMin, tMax], or ok=false.
type Object interface {
	Intersect(r Ray, tMin, tMax float64) (Hit, bool)
}

const (
	geomEps   = 1e-9
	shadowEps = 1e-4
	tFar      = 1e9
	goldenAng = 2.399963229728653 // radians, for low-discrepancy disk/hemisphere sampling
)

// orient flips n to face against the ray direction and reports whether it was
// already front-facing.
func orient(n, dir Vec3) (Vec3, bool) {
	if n.Dot(dir) > 0 {
		return n.Neg(), false
	}
	return n, true
}

// ---------- sphere ----------

// Sphere is an analytic sphere.
type Sphere struct {
	Center Vec3
	Radius float64
	Mat    Material
}

// Intersect implements Object for a sphere (unit-direction quadratic, a=1).
func (s Sphere) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	return intersectSphere(s.Center, s.Radius, s.Mat, r, tMin, tMax)
}

// intersectSphere is the shared ray/sphere test used by Sphere and MovingSphere
// (the latter passes the center evaluated at the ray's time).
func intersectSphere(center Vec3, radius float64, mat Material, r Ray, tMin, tMax float64) (Hit, bool) {
	oc := r.Origin.Sub(center)
	b := oc.Dot(r.Dir)
	c := oc.LenSq() - radius*radius
	disc := b*b - c
	if disc < 0 {
		return Hit{}, false
	}
	sq := math.Sqrt(disc)
	t := -b - sq
	if t <= tMin || t > tMax {
		if t = -b + sq; t <= tMin || t > tMax {
			return Hit{}, false
		}
	}
	p := r.At(t)
	gn := p.Sub(center).Norm()
	n, front := orient(gn, r.Dir)
	u := 0.5 + math.Atan2(gn.Z, gn.X)/(2*math.Pi)
	v := 0.5 - math.Asin(math.Max(-1, math.Min(1, gn.Y)))/math.Pi
	hit := Hit{T: t, P: p, N: n, U: u, V: v, Front: front, Mat: mat}
	if mat.Bump != nil { // east-pointing (increasing-U) tangent; degenerate at poles
		if east := (Vec3{X: 0, Y: 1, Z: 0}).Cross(gn); east.LenSq() > geomEps {
			hit.Tan, hit.TanW = east.Norm(), 1
		}
	}
	return hit, true
}

// sphereOverlaps reports whether ray r's [tMin,tMax] segment crosses the sphere.
func sphereOverlaps(r Ray, center Vec3, radius, tMin, tMax float64) bool {
	oc := r.Origin.Sub(center)
	b := oc.Dot(r.Dir)
	disc := b*b - (oc.LenSq() - radius*radius)
	if disc < 0 {
		return false
	}
	sq := math.Sqrt(disc)
	return -b-sq <= tMax && -b+sq >= tMin
}

// MovingSphere is a sphere whose center travels linearly from C0 (shutter open,
// time 0) to C1 (shutter close, time 1). Sampling many rays with times spread
// over [0,1] (the camera does this automatically) renders motion blur.
type MovingSphere struct {
	C0, C1 Vec3
	Radius float64
	Mat    Material
}

// center returns the sphere's center at shutter fraction t in [0,1].
func (m MovingSphere) center(t float64) Vec3 {
	return m.C0.Add(m.C1.Sub(m.C0).Scale(t))
}

// Intersect evaluates the sphere at the ray's time, then does the static test.
func (m MovingSphere) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	return intersectSphere(m.center(r.Time), m.Radius, m.Mat, r, tMin, tMax)
}

// ---------- triangle ----------

// Triangle is a single triangle. If Na/Nb/Nc are non-zero the shading normal is
// barycentrically interpolated (smooth shading); otherwise the flat face normal
// is used.
type Triangle struct {
	A, B, C       Vec3
	Na, Nb, Nc    Vec3       // optional vertex normals (smooth shading)
	UVa, UVb, UVc [2]float64 // optional texture coordinates
	Mat           Material
}

func (tr Triangle) smooth() bool {
	return tr.Na.LenSq() > 0 && tr.Nb.LenSq() > 0 && tr.Nc.LenSq() > 0
}

// Intersect implements the Möller–Trumbore ray/triangle test.
func (tr Triangle) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	e1 := tr.B.Sub(tr.A)
	e2 := tr.C.Sub(tr.A)
	pv := r.Dir.Cross(e2)
	det := e1.Dot(pv)
	if det > -geomEps && det < geomEps {
		return Hit{}, false
	}
	inv := 1 / det
	tv := r.Origin.Sub(tr.A)
	u := tv.Dot(pv) * inv
	if u < 0 || u > 1 {
		return Hit{}, false
	}
	qv := tv.Cross(e1)
	v := r.Dir.Dot(qv) * inv
	if v < 0 || u+v > 1 {
		return Hit{}, false
	}
	t := e2.Dot(qv) * inv
	if t <= tMin || t > tMax {
		return Hit{}, false
	}
	w0 := 1 - u - v
	var ng Vec3
	if tr.smooth() {
		ng = tr.Na.Scale(w0).Add(tr.Nb.Scale(u)).Add(tr.Nc.Scale(v)).Norm()
	} else {
		ng = e1.Cross(e2).Norm()
	}
	hu := w0*tr.UVa[0] + u*tr.UVb[0] + v*tr.UVc[0]
	hv := w0*tr.UVa[1] + u*tr.UVb[1] + v*tr.UVc[1]
	n, front := orient(ng, r.Dir)
	hit := Hit{T: t, P: r.At(t), N: n, U: hu, V: hv, Front: front, Mat: tr.Mat}
	if tr.Mat.Bump != nil { // UV-aligned tangent frame for the normal map
		if tan, w, ok := triTangent(tr); ok {
			hit.Tan, hit.TanW = tan, w
		}
	}
	return hit, true
}

// triTangent computes the UV-aligned tangent (the world direction of increasing
// U) and the tangent-space handedness from the triangle's edges and UV deltas, or
// ok=false if the UVs are degenerate.
func triTangent(tr Triangle) (Vec3, float64, bool) {
	e1 := tr.B.Sub(tr.A)
	e2 := tr.C.Sub(tr.A)
	du1, dv1 := tr.UVb[0]-tr.UVa[0], tr.UVb[1]-tr.UVa[1]
	du2, dv2 := tr.UVc[0]-tr.UVa[0], tr.UVc[1]-tr.UVa[1]
	denom := du1*dv2 - du2*dv1
	if math.Abs(denom) < 1e-12 {
		return Vec3{}, 0, false
	}
	inv := 1 / denom
	tan := e1.Scale(dv2).Sub(e2.Scale(dv1)).Scale(inv)
	bit := e2.Scale(du1).Sub(e1.Scale(du2)).Scale(inv)
	if tan.LenSq() < geomEps {
		return Vec3{}, 0, false
	}
	w := 1.0
	if e1.Cross(e2).Cross(tan).Dot(bit) < 0 {
		w = -1
	}
	return tan, w, true
}

// ---------- plane (checkerboard floor) ----------

// Plane is an infinite horizontal floor at height Y with a two-colour
// checkerboard whose cells are Size wide.
type Plane struct {
	Y      float64
	Size   float64
	C1, C2 Vec3
	Mat    Material // Reflect/Spec/etc. apply; Color is taken from the checker
}

// Intersect implements Object for the floor plane.
func (p Plane) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	if math.Abs(r.Dir.Y) < geomEps {
		return Hit{}, false
	}
	t := (p.Y - r.Origin.Y) / r.Dir.Y
	if t <= tMin || t > tMax {
		return Hit{}, false
	}
	hp := r.At(t)
	sz := p.Size
	if sz <= 0 {
		sz = 1
	}
	cx := int(math.Floor(hp.X / sz))
	cz := int(math.Floor(hp.Z / sz))
	mat := p.Mat
	mat.Color = p.C1
	if (cx+cz)&1 == 0 {
		mat.Color = p.C2
	}
	n, front := orient(Vec3{0, 1, 0}, r.Dir)
	return Hit{T: t, P: hp, N: n, Front: front, Mat: mat}, true
}

// ---------- camera ----------

// Camera is a thin-lens camera: a position, a yaw/pitch look direction, a
// vertical field of view, and an optional aperture/focus for depth of field
// (used by the path tracer; the raster Render treats it as a pinhole).
type Camera struct {
	Pos      Vec3
	Yaw      float64 // around the y (up) axis
	Pitch    float64 // up (+) / down (-)
	FOV      float64 // vertical field of view, radians (default pi/3 if <= 0)
	Aperture float64 // lens radius for depth of field (0 = pinhole, sharp)
	Focus    float64 // focus distance (defaults to looking at infinity if 0)
}

type camBasis struct {
	origin, forward, right, up Vec3
	halfW, halfH               float64
	pxW, pxH                   float64
}

func (c Camera) basis(pxW, pxH int) camBasis {
	fov := c.FOV
	if fov <= 0 {
		fov = math.Pi / 3
	}
	cp := math.Cos(c.Pitch)
	forward := Vec3{X: cp * math.Sin(c.Yaw), Y: math.Sin(c.Pitch), Z: cp * math.Cos(c.Yaw)}.Norm()
	right := forward.Cross(Vec3{0, 1, 0})
	if right.LenSq() < geomEps {
		right = Vec3{1, 0, 0}
	}
	right = right.Norm()
	up := right.Cross(forward).Norm()
	halfH := math.Tan(fov / 2)
	halfW := halfH * float64(pxW) / float64(pxH)
	return camBasis{c.Pos, forward, right, up, halfW, halfH, float64(pxW), float64(pxH)}
}

func (b camBasis) ray(px, py float64) Ray {
	ndcX := (2*px/b.pxW - 1) * b.halfW
	ndcY := (1 - 2*py/b.pxH) * b.halfH
	dir := b.forward.Add(b.right.Scale(ndcX)).Add(b.up.Scale(ndcY)).Norm()
	return Ray{Origin: b.origin, Dir: dir}
}

// ---------- scene & shading ----------

// Scene is the world.
type Scene struct {
	Objects   []Object
	Light     Vec3
	LightInt  float64      // light strength (default 1 if 0)
	Ambient   float64      // 0..1 ambient term
	SkyTop    Vec3         // looking up
	SkyBottom Vec3         // looking toward the horizon/down
	SkyTex    Texture      // optional equirectangular environment map (overrides the gradient)
	Sky       *PreethamSky // optional physical sky model (overrides gradient/env map)
	Env       *EnvSampler  // optional importance sampler for the environment (path-tracer env-NEE)
	Caustics  *PhotonMap   // optional caustic photon map (additive caustic term at diffuse hits)
	Medium    *Medium      // optional heterogeneous participating medium (delta tracking; path tracer)
	MaxBounce int          // reflection/refraction bounce budget (0 = none)
	AttenK    float64      // linear distance attenuation coefficient (0 = none)

	// Soft shadows: a light of radius LightRadius sampled ShadowSamples times.
	LightRadius   float64
	ShadowSamples int

	// Ambient occlusion: AOSamples hemisphere rays of length AORadius darken the
	// ambient term in creases (0 samples = off).
	AOSamples int
	AORadius  float64

	// Distance fog (aerial perspective): a hit fades toward FogColor with the
	// ray-segment length (Beer-Lambert). FogDensity <= 0 disables it.
	FogDensity float64
	FogColor   Vec3

	// Volumetric single-scatter ("god rays"): a homogeneous medium of density
	// Volume scatters Scene.Light along the view ray; shadowed medium stays dark,
	// lit medium glows. Volume <= 0 disables it (raster Render only).
	Volume      float64
	VolumeColor Vec3

	// RISCandidates enables ReSTIR-style direct lighting (resampled importance
	// sampling): each NEE draw resamples one of this many candidate light samples
	// weighted by their unshadowed contribution, then shadow-tests only the
	// survivor. 0/1 = the plain single-sample estimator; higher cuts noise in
	// many-light scenes. Requires NEE on and MIS off (it replaces light-side MIS).
	RISCandidates int

	sbvh     *objBVH    // optional top-level acceleration structure (see BuildBVH)
	emit     []Sphere   // emissive spheres, cached for next-event estimation
	emitTris []Triangle // emissive triangles (area lights), cached for NEE

	// Power-weighted light selection (importance sampling of many lights): per
	// emitter power (luminance x area), its inclusive cumulative sum and the total,
	// in the order spheres then triangles. NEE picks a light proportional to power.
	emitPow   []float64
	emitCum   []float64
	emitTotal float64
}

// lumOf is the photometric luminance of a colour (used to weight light power).
func lumOf(v Vec3) float64 { return 0.2126*v.X + 0.7152*v.Y + 0.0722*v.Z }

// gatherEmitters caches the scene's emissive geometry so the path tracer can sample
// it directly (next-event estimation), and precomputes power weights so brighter,
// bigger lights are sampled more often (importance sampling of many lights). It
// gathers emissive spheres, top-level emissive triangles, and emissive triangles
// inside meshes/instances (transformed to world space, honouring an instance's
// material override). Called by PathRender.
func (s *Scene) gatherEmitters() {
	s.emit = s.emit[:0]
	s.emitTris = s.emitTris[:0]
	addTri := func(t Triangle) {
		if _, a := triNormalArea(t); a > 0 { // skip degenerate (zero-area) triangles
			s.emitTris = append(s.emitTris, t)
		}
	}
	for _, o := range s.Objects {
		switch g := o.(type) {
		case Sphere:
			if g.Mat.Emit.LenSq() > 0 {
				s.emit = append(s.emit, g)
			}
		case Triangle:
			if g.Mat.Emit.LenSq() > 0 {
				addTri(g)
			}
		case *Mesh:
			for _, t := range g.Tris {
				if t.Mat.Emit.LenSq() > 0 {
					addTri(t)
				}
			}
		case *Instance:
			if mesh, ok := g.obj.(*Mesh); ok {
				for _, t := range mesh.Tris {
					mat := t.Mat
					if g.mat != nil { // an instance material override can make it emit
						mat = *g.mat
					}
					if mat.Emit.LenSq() > 0 {
						addTri(Triangle{A: g.m.mul(t.A).Add(g.t), B: g.m.mul(t.B).Add(g.t), C: g.m.mul(t.C).Add(g.t), Mat: mat})
					}
				}
			}
		}
	}
	// power weights: luminance x surface area, in the order spheres then triangles.
	s.emitPow = s.emitPow[:0]
	s.emitCum = s.emitCum[:0]
	s.emitTotal = 0
	addPow := func(p float64) {
		if p <= 0 {
			p = 1e-6 // keep every emitter selectable (and its MIS weight finite)
		}
		s.emitTotal += p
		s.emitPow = append(s.emitPow, p)
		s.emitCum = append(s.emitCum, s.emitTotal)
	}
	for i := range s.emit {
		e := s.emit[i]
		addPow(lumOf(e.Mat.Emit) * 4 * math.Pi * e.Radius * e.Radius)
	}
	for i := range s.emitTris {
		_, a := triNormalArea(s.emitTris[i])
		addPow(lumOf(s.emitTris[i].Mat.Emit) * a)
	}
}

func (s *Scene) closest(r Ray, tMin, tMax float64) (Hit, bool) {
	h, ok := s.closestRaw(r, tMin, tMax)
	if ok && h.Mat.Bump != nil {
		h.N = perturbNormal(h)
		if h.N.Dot(r.Dir) > 0 { // keep the perturbed normal facing the ray
			h.N = h.N.Neg()
		}
	}
	return h, ok
}

func (s *Scene) closestRaw(r Ray, tMin, tMax float64) (Hit, bool) {
	if s.sbvh != nil {
		return s.sbvh.closest(r, tMin, tMax)
	}
	var best Hit
	found := false
	near := tMax
	for _, o := range s.Objects {
		if h, ok := o.Intersect(r, tMin, near); ok {
			best, found, near = h, true, h.T
		}
	}
	return best, found
}

// perturbNormal applies a material's tangent-space normal map at the hit: it
// decodes the RGB-encoded normal, builds a tangent frame from the geometric
// normal, and returns the perturbed shading normal. A flat map (0.5,0.5,1)
// leaves the normal unchanged.
func perturbNormal(h Hit) Vec3 {
	c := h.Mat.Bump.At(h.U, h.V, h.P)
	tn := Vec3{X: 2*c.X - 1, Y: 2*c.Y - 1, Z: 2*c.Z - 1}
	var t, b Vec3
	if h.Tan.LenSq() > geomEps {
		// UV-aligned frame: orthonormalise the tangent against the shading normal
		// and rebuild the bitangent with the stored handedness.
		t = h.Tan.Sub(h.N.Scale(h.N.Dot(h.Tan)))
		if t.LenSq() < geomEps {
			t, b = basisAround(h.N)
		} else {
			t = t.Norm()
			b = h.N.Cross(t).Scale(h.TanW)
		}
	} else {
		t, b = basisAround(h.N)
	}
	n := t.Scale(tn.X).Add(b.Scale(tn.Y)).Add(h.N.Scale(tn.Z))
	if n.LenSq() < geomEps {
		return h.N
	}
	return n.Norm()
}

func (s *Scene) anyHit(r Ray, tMin, tMax float64) bool {
	if s.sbvh != nil {
		return s.sbvh.anyHit(r, tMin, tMax)
	}
	for _, o := range s.Objects {
		if _, ok := o.Intersect(r, tMin, tMax); ok {
			return true
		}
	}
	return false
}

// marchVolume integrates single-scattering of Scene.Light through a homogeneous
// medium along ray r over [0, tMax], ray-marching with shadow rays at each step.
// Returns the accumulated in-scatter and the medium transmittance over the segment.
func (s *Scene) marchVolume(r Ray, tMax float64) (Vec3, float64) {
	if s.Volume <= 0 {
		return Vec3{}, 1
	}
	if tMax > 40 {
		tMax = 40 // cap for rays that miss everything
	}
	const steps = 32
	dt := tMax / steps
	li := s.LightInt
	if li == 0 {
		li = 1
	}
	var inscatter Vec3
	transmittance := 1.0
	stepTr := math.Exp(-s.Volume * dt)
	for i := 0; i < steps; i++ {
		p := r.At((float64(i) + 0.5) * dt)
		toL := s.Light.Sub(p)
		dist := toL.Len()
		if dist > geomEps && !s.anyHit(Ray{Origin: p, Dir: toL.Scale(1 / dist)}, shadowEps, dist-shadowEps) {
			atten := li / (1 + s.AttenK*dist)
			inscatter = inscatter.Add(s.VolumeColor.Scale(transmittance * s.Volume * dt * atten))
		}
		transmittance *= stepTr
	}
	return inscatter, transmittance
}

// fog blends a colour toward FogColor by the transmittance over distance t.
func (s *Scene) fog(col Vec3, t float64) Vec3 {
	if s.FogDensity <= 0 {
		return col
	}
	tr := math.Exp(-s.FogDensity * t)
	return col.Scale(tr).Add(s.FogColor.Scale(1 - tr))
}

func (s *Scene) sky(dir Vec3) Vec3 {
	if s.Sky != nil {
		return s.Sky.At(dir)
	}
	if s.SkyTex != nil {
		u := 0.5 + math.Atan2(dir.Z, dir.X)/(2*math.Pi)
		v := 0.5 - math.Asin(math.Max(-1, math.Min(1, dir.Y)))/math.Pi
		return s.SkyTex.At(u, v, dir)
	}
	t := 0.5 * (dir.Y + 1)
	return s.SkyBottom.Scale(1 - t).Add(s.SkyTop.Scale(t))
}

// basisAround returns two unit vectors tangent and bitangent to n.
func basisAround(n Vec3) (Vec3, Vec3) {
	up := Vec3{0, 1, 0}
	if math.Abs(n.Y) > 0.99 {
		up = Vec3{1, 0, 0}
	}
	t := up.Cross(n).Norm()
	return t, n.Cross(t).Norm()
}

// visibility returns the fraction of the light reaching p (1 = fully lit). With
// LightRadius>0 and ShadowSamples>1 it integrates a disk light (soft shadows).
func (s *Scene) visibility(p, n Vec3) float64 {
	sorig := p.Add(n.Scale(shadowEps))
	if s.LightRadius <= 0 || s.ShadowSamples <= 1 {
		toL := s.Light.Sub(sorig)
		d := toL.Len()
		if s.anyHit(Ray{Origin: sorig, Dir: toL.Scale(1 / d)}, shadowEps, d-2*shadowEps) {
			return 0
		}
		return 1
	}
	ld := s.Light.Sub(sorig).Norm()
	tu, tv := basisAround(ld)
	k := s.ShadowSamples
	vis := 0.0
	for i := 0; i < k; i++ {
		rr := math.Sqrt((float64(i) + 0.5) / float64(k))
		th := float64(i) * goldenAng
		off := tu.Scale(rr * math.Cos(th) * s.LightRadius).Add(tv.Scale(rr * math.Sin(th) * s.LightRadius))
		target := s.Light.Add(off)
		toL := target.Sub(sorig)
		d := toL.Len()
		if !s.anyHit(Ray{Origin: sorig, Dir: toL.Scale(1 / d)}, shadowEps, d-2*shadowEps) {
			vis++
		}
	}
	return vis / float64(k)
}

// ao returns an ambient-occlusion factor in [0,1] (1 = open) from AOSamples
// cosine-weighted hemisphere rays.
func (s *Scene) ao(p, n Vec3) float64 {
	if s.AOSamples <= 0 || s.AORadius <= 0 {
		return 1
	}
	tu, tv := basisAround(n)
	orig := p.Add(n.Scale(shadowEps))
	open := 0.0
	k := s.AOSamples
	for i := 0; i < k; i++ {
		rr := math.Sqrt((float64(i) + 0.5) / float64(k))
		th := float64(i) * goldenAng
		x, y := rr*math.Cos(th), rr*math.Sin(th)
		z := math.Sqrt(math.Max(0, 1-rr*rr))
		dir := tu.Scale(x).Add(tv.Scale(y)).Add(n.Scale(z)).Norm()
		if !s.anyHit(Ray{Origin: orig, Dir: dir}, shadowEps, s.AORadius) {
			open++
		}
	}
	return open / float64(k)
}

func (s *Scene) shade(r Ray, depth int) Vec3 {
	h, ok := s.closest(r, shadowEps, tFar)
	if !ok {
		return s.sky(r.Dir)
	}
	li := s.LightInt
	if li == 0 {
		li = 1
	}

	// Dielectric (glass): mix Fresnel reflection and refraction.
	if depth > 0 && h.Mat.Glass > 0 {
		return s.fog(s.dielectric(r, h, depth), h.T)
	}

	alb := h.albedo()
	col := alb.Scale(s.Ambient * s.ao(h.P, h.N)).Add(h.Mat.Emit)

	toL := s.Light.Sub(h.P)
	dist := toL.Len()
	if dist > geomEps {
		L := toL.Scale(1 / dist)
		diff := h.N.Dot(L)
		if diff > 0 {
			if vis := s.visibility(h.P, h.N); vis > 0 {
				atten := vis * li / (1 + s.AttenK*dist)
				col = col.Add(alb.Scale((1 - s.Ambient) * diff * atten))
				if h.Mat.Spec > 0 {
					shine := h.Mat.Shine
					if shine <= 0 {
						shine = 32
					}
					half := L.Sub(r.Dir).Norm() // L + viewDir, viewDir = -r.Dir
					spec := math.Pow(math.Max(0, h.N.Dot(half)), shine)
					col = col.Add(Vec3{1, 1, 1}.Scale(h.Mat.Spec * spec * atten))
				}
			}
		}
	}

	if depth > 0 && h.Mat.Reflect > 0 {
		rdir := r.Dir.Reflect(h.N).Norm()
		refl := s.shade(Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: rdir}, depth-1)
		col = col.Scale(1 - h.Mat.Reflect).Add(refl.Scale(h.Mat.Reflect))
	}
	return s.fog(col, h.T)
}

// dielectric shades a glass surface: a Fresnel-weighted mix of the reflected and
// refracted rays (Schlick approximation), with total internal reflection handled.
func (s *Scene) dielectric(r Ray, h Hit, depth int) Vec3 {
	n := h.N // faces the ray
	ior := h.Mat.Glass
	eta := 1 / ior
	if !h.Front {
		eta = ior
	}
	cosI := math.Min(1, math.Max(0, -r.Dir.Dot(n)))

	reflDir := r.Dir.Reflect(n).Norm()
	refl := s.shade(Ray{Origin: h.P.Add(n.Scale(shadowEps)), Dir: reflDir}, depth-1)

	k := 1 - eta*eta*(1-cosI*cosI)
	if k < 0 {
		return refl.Add(h.Mat.Emit) // total internal reflection
	}
	refrDir := r.Dir.Scale(eta).Add(n.Scale(eta*cosI - math.Sqrt(k))).Norm()
	refr := s.shade(Ray{Origin: h.P.Sub(n.Scale(shadowEps)), Dir: refrDir}, depth-1)

	r0 := (1 - ior) / (1 + ior)
	r0 *= r0
	fr := r0 + (1-r0)*math.Pow(1-cosI, 5)
	return refl.Scale(fr).Add(refr.Scale(1 - fr)).Add(h.Mat.Emit)
}

// Trace returns the (clamped) colour seen along a single ray. Exposed for tests.
func (s *Scene) Trace(r Ray) Vec3 { return clampVec(s.shade(r, s.MaxBounce)) }

// ---------- render ----------

// Options controls render quality.
type Options struct {
	Samples int // anti-aliasing samples per axis (1 = off; 2 = 4 rays/pixel)
}

// Render renders the scene from cam into a pxW x pxH image using all CPU cores.
// Pixels are written to disjoint locations, so the parallelism is race-free and
// the output is deterministic.
func Render(s *Scene, cam Camera, pxW, pxH int, opt Options) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, pxW, pxH))
	if pxW <= 0 || pxH <= 0 {
		return img
	}
	ss := opt.Samples
	if ss < 1 {
		ss = 1
	}
	b := cam.basis(pxW, pxH)
	depth := s.MaxBounce
	inv := 1.0 / float64(ss*ss)

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
					var acc Vec3
					for sy := 0; sy < ss; sy++ {
						for sx := 0; sx < ss; sx++ {
							u := float64(x) + (float64(sx)+0.5)/float64(ss)
							v := float64(y) + (float64(sy)+0.5)/float64(ss)
							r := b.ray(u, v)
							col := s.shade(r, depth)
							if s.Volume > 0 {
								tMax := tFar
								if h, ok := s.closest(r, shadowEps, tFar); ok {
									tMax = h.T
								}
								ins, tr := s.marchVolume(r, tMax)
								col = col.Scale(tr).Add(ins)
							}
							acc = acc.Add(clampVec(col))
						}
					}
					img.SetRGBA(x, y, toRGBA(acc.Scale(inv)))
				}
			}
		}()
	}
	wg.Wait()
	return img
}

// ---------- colour helpers ----------

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func clampVec(v Vec3) Vec3 { return Vec3{clamp01(v.X), clamp01(v.Y), clamp01(v.Z)} }

// aces is the Narkowicz ACES filmic tone-mapping curve (compresses highlights).
func aces(x float64) float64 {
	const a, b, c, d, e = 2.51, 0.03, 2.43, 0.59, 0.14
	return clamp01((x * (a*x + b)) / (x*(c*x+d) + e))
}

// toRGBA tone-maps (ACES) and applies an approximate gamma (2.0).
func toRGBA(v Vec3) color.RGBA {
	g := func(x float64) uint8 { return uint8(math.Sqrt(aces(x))*255 + 0.5) }
	return color.RGBA{R: g(v.X), G: g(v.Y), B: g(v.Z), A: 255}
}
