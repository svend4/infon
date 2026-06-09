// Package raytrace is a small, dependency-light CPU ray tracer that renders to an
// image.Image — the same output type as the isometric pkg/fold renderer — so it
// drops straight into every existing consumer: PNG/GIF export, the
// half-block/sixel/kitty terminal encoders, and the orbit demos. It implements
// analytic spheres, an infinite checkerboard floor and triangle meshes
// (Möller–Trumbore with a bounding-sphere reject), Lambertian shading with hard
// shadow rays and distance attenuation, optional one-bounce-or-more reflection
// and supersampled anti-aliasing, and a parallel per-row render loop.
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
type Ray struct{ Origin, Dir Vec3 }

// At returns the point at parameter t along the ray.
func (r Ray) At(t float64) Vec3 { return r.Origin.Add(r.Dir.Scale(t)) }

// Material describes a surface: a base colour (channels 0..1) and a mirror
// fraction (0 = matte, 1 = perfect mirror).
type Material struct {
	Color   Vec3
	Reflect float64
}

// Hit records the nearest intersection along a ray.
type Hit struct {
	T   float64 // ray parameter
	P   Vec3    // world-space hit point
	N   Vec3    // unit normal, oriented against the incoming ray
	Mat Material
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
)

// ---------- sphere ----------

// Sphere is an analytic sphere.
type Sphere struct {
	Center Vec3
	Radius float64
	Mat    Material
}

// Intersect implements Object for a sphere (unit-direction quadratic, a=1).
func (s Sphere) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	oc := r.Origin.Sub(s.Center)
	b := oc.Dot(r.Dir)
	c := oc.LenSq() - s.Radius*s.Radius
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
	n := p.Sub(s.Center).Norm()
	if n.Dot(r.Dir) > 0 {
		n = n.Neg()
	}
	return Hit{T: t, P: p, N: n, Mat: s.Mat}, true
}

// sphereOverlaps reports whether ray r's [tMin,tMax] segment crosses the sphere
// (center,radius). Used for bounding-sphere rejection of meshes.
func sphereOverlaps(r Ray, center Vec3, radius, tMin, tMax float64) bool {
	oc := r.Origin.Sub(center)
	b := oc.Dot(r.Dir)
	c := oc.LenSq() - radius*radius
	disc := b*b - c
	if disc < 0 {
		return false
	}
	sq := math.Sqrt(disc)
	t0, t1 := -b-sq, -b+sq
	return t0 <= tMax && t1 >= tMin
}

// ---------- triangle ----------

// Triangle is a single flat triangle.
type Triangle struct {
	A, B, C Vec3
	Mat     Material
}

// Intersect implements the Möller–Trumbore ray/triangle test.
func (tr Triangle) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	e1 := tr.B.Sub(tr.A)
	e2 := tr.C.Sub(tr.A)
	pv := r.Dir.Cross(e2)
	det := e1.Dot(pv)
	if det > -geomEps && det < geomEps {
		return Hit{}, false // ray parallel to the triangle plane
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
	n := e1.Cross(e2).Norm()
	if n.Dot(r.Dir) > 0 {
		n = n.Neg()
	}
	return Hit{T: t, P: r.At(t), N: n, Mat: tr.Mat}, true
}

// ---------- mesh ----------

// Mesh is a triangle soup with a precomputed bounding sphere for quick rejection.
type Mesh struct {
	Tris   []Triangle
	center Vec3
	radius float64
}

// NewMesh builds a mesh and computes its bounding sphere.
func NewMesh(tris []Triangle) *Mesh {
	m := &Mesh{Tris: tris}
	if len(tris) == 0 {
		return m
	}
	var sum Vec3
	for _, t := range tris {
		sum = sum.Add(t.A).Add(t.B).Add(t.C)
	}
	c := sum.Scale(1 / float64(3*len(tris)))
	var rad float64
	for _, t := range tris {
		for _, v := range [3]Vec3{t.A, t.B, t.C} {
			if d := v.Sub(c).Len(); d > rad {
				rad = d
			}
		}
	}
	m.center, m.radius = c, rad
	return m
}

// Bound returns the mesh bounding sphere (centre, radius).
func (m *Mesh) Bound() (Vec3, float64) { return m.center, m.radius }

// Intersect implements Object: reject against the bounding sphere, then test
// every triangle, keeping the nearest.
func (m *Mesh) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	if len(m.Tris) == 0 || !sphereOverlaps(r, m.center, m.radius, tMin, tMax) {
		return Hit{}, false
	}
	var best Hit
	found := false
	closest := tMax
	for _, t := range m.Tris {
		if h, ok := t.Intersect(r, tMin, closest); ok {
			best, found, closest = h, true, h.T
		}
	}
	return best, found
}

// ---------- plane (checkerboard floor) ----------

// Plane is an infinite horizontal floor at height Y with a two-colour
// checkerboard whose cells are Size wide.
type Plane struct {
	Y      float64
	Size   float64
	C1, C2 Vec3
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
	col := p.C1
	if (cx+cz)&1 == 0 {
		col = p.C2
	}
	n := Vec3{0, 1, 0}
	if r.Dir.Y > 0 {
		n = Vec3{0, -1, 0}
	}
	return Hit{T: t, P: hp, N: n, Mat: Material{Color: col}}, true
}

// ---------- camera ----------

// Camera is a pinhole camera: a position, a yaw/pitch look direction and a
// vertical field of view (radians).
type Camera struct {
	Pos   Vec3
	Yaw   float64 // around the y (up) axis
	Pitch float64 // up (+) / down (-)
	FOV   float64 // vertical field of view, radians (default pi/3 if <= 0)
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

// ray generates the primary ray through sub-pixel coordinate (px,py).
func (b camBasis) ray(px, py float64) Ray {
	ndcX := (2*px/b.pxW - 1) * b.halfW
	ndcY := (1 - 2*py/b.pxH) * b.halfH
	dir := b.forward.Add(b.right.Scale(ndcX)).Add(b.up.Scale(ndcY)).Norm()
	return Ray{Origin: b.origin, Dir: dir}
}

// ---------- scene & shading ----------

// Scene is the world: objects, one point light, an ambient floor, a sky gradient
// and a reflection-bounce budget.
type Scene struct {
	Objects   []Object
	Light     Vec3
	LightInt  float64 // light strength (default 1 if 0)
	Ambient   float64 // 0..1 ambient term
	SkyTop    Vec3    // looking up
	SkyBottom Vec3    // looking toward the horizon/down
	MaxBounce int     // reflection bounces (0 = none)
	AttenK    float64 // linear distance attenuation coefficient (0 = none)
}

func (s *Scene) closest(r Ray, tMin, tMax float64) (Hit, bool) {
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

func (s *Scene) sky(dir Vec3) Vec3 {
	t := 0.5 * (dir.Y + 1)
	return s.SkyBottom.Scale(1 - t).Add(s.SkyTop.Scale(t))
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
	col := h.Mat.Color.Scale(s.Ambient)

	toL := s.Light.Sub(h.P)
	dist := toL.Len()
	if dist > geomEps {
		L := toL.Scale(1 / dist)
		diff := h.N.Dot(L)
		if diff > 0 {
			sorig := h.P.Add(h.N.Scale(shadowEps))
			if _, blocked := s.closest(Ray{Origin: sorig, Dir: L}, shadowEps, dist-2*shadowEps); !blocked {
				atten := li / (1 + s.AttenK*dist)
				col = col.Add(h.Mat.Color.Scale((1 - s.Ambient) * diff * atten))
			}
		}
	}

	if depth > 0 && h.Mat.Reflect > 0 {
		rdir := r.Dir.Reflect(h.N).Norm()
		refl := s.shade(Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: rdir}, depth-1)
		col = col.Scale(1 - h.Mat.Reflect).Add(refl.Scale(h.Mat.Reflect))
	}
	return col
}

// Trace returns the colour seen along a single ray (channels 0..1, unclamped
// callers should clamp). Exposed for testing and custom pipelines.
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
							acc = acc.Add(clampVec(s.shade(b.ray(u, v), depth)))
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

// toRGBA clamps and applies an approximate gamma (2.0) for pleasant midtones.
func toRGBA(v Vec3) color.RGBA {
	v = clampVec(v)
	g := func(x float64) uint8 { return uint8(math.Sqrt(x)*255 + 0.5) }
	return color.RGBA{R: g(v.X), G: g(v.Y), B: g(v.Z), A: 255}
}
