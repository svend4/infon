// sdf.go renders signed-distance-field objects by sphere tracing (ray marching):
// a Marched wraps a distance estimator and is intersected by stepping along the ray
// by the estimated distance until it reaches the surface. This unlocks shapes that
// triangles can't easily express — fractals (Mandelbulb, Menger sponge), melting
// blended forms (smooth-min, Dali-style), and kaleidoscopic mandalas (domain
// folding) — each a tiny formula rather than a mesh: an entire infinite world
// shipped as a name. A Marched satisfies Object, so it composes with the path
// tracer, materials and the BVH like any other primitive. Clean-room.
package raytrace

import (
	"math"
	"strings"
)

// DistField is a signed distance estimator evaluated at a point relative to the
// object's centre: f(p) approximates the distance to the surface (>=0 outside). It
// must not overestimate (Lipschitz <= 1) for sphere tracing to converge correctly.
type DistField func(p Vec3) float64

// Marched is an SDF object: a distance field inside a bounding sphere, with a
// material and marching parameters.
type Marched struct {
	DE       DistField
	Center   Vec3    // bounding-sphere centre and the field's local origin
	Radius   float64 // bounding-sphere radius (the marched region)
	Mat      Material
	MaxSteps int     // sphere-tracing step budget (default 160)
	Eps      float64 // surface threshold (default Radius*1e-4)
	StepK    float64 // step relaxation in (0,1]; <1 for fractal DEs that overshoot
}

// Intersect sphere-traces the field within its bounding sphere. It works for any
// ray direction (length need not be 1): marching is done in world distance and the
// hit is reported back in the ray's own parameterisation.
func (m Marched) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	dl := r.Dir.Len()
	if dl < geomEps || m.DE == nil {
		return Hit{}, false
	}
	ud := r.Dir.Scale(1 / dl) // unit direction; world distance s = t*dl
	oc := r.Origin.Sub(m.Center)
	b := oc.Dot(ud)
	c := oc.Dot(oc) - m.Radius*m.Radius
	disc := b*b - c
	if disc < 0 {
		return Hit{}, false // ray misses the bounding sphere
	}
	sq := math.Sqrt(disc)
	s0, s1 := -b-sq, -b+sq
	if lo := tMin * dl; s0 < lo {
		s0 = lo
	}
	if hi := tMax * dl; s1 > hi {
		s1 = hi
	}
	if s0 > s1 {
		return Hit{}, false
	}
	steps := m.MaxSteps
	if steps <= 0 {
		steps = 160
	}
	eps := m.Eps
	if eps <= 0 {
		eps = m.Radius * 1e-4
	}
	k := m.StepK
	if k <= 0 {
		k = 1
	}
	s := s0
	for i := 0; i < steps; i++ {
		p := r.Origin.Add(ud.Scale(s))
		d := m.DE(p.Sub(m.Center))
		if d < eps {
			n, front := orient(m.normal(p), r.Dir)
			return Hit{T: s / dl, P: p, N: n, Front: front, Mat: m.Mat}, true
		}
		s += d * k
		if s > s1 {
			break
		}
	}
	return Hit{}, false
}

// normal estimates the surface normal as the gradient of the field (central
// differences).
func (m Marched) normal(p Vec3) Vec3 {
	h := m.Radius * 1e-4
	if h <= 0 {
		h = 1e-4
	}
	lp := p.Sub(m.Center)
	dx := m.DE(lp.Add(Vec3{X: h})) - m.DE(lp.Sub(Vec3{X: h}))
	dy := m.DE(lp.Add(Vec3{Y: h})) - m.DE(lp.Sub(Vec3{Y: h}))
	dz := m.DE(lp.Add(Vec3{Z: h})) - m.DE(lp.Sub(Vec3{Z: h}))
	n := Vec3{X: dx, Y: dy, Z: dz}
	if n.LenSq() < geomEps {
		return Vec3{X: 0, Y: 1, Z: 0}
	}
	return n.Norm()
}

// ---- distance-field primitives and combinators ----

func sdSphere(p Vec3, r float64) float64 { return p.Len() - r }

func sdBox(p, bx Vec3) float64 {
	d := Vec3{X: math.Abs(p.X) - bx.X, Y: math.Abs(p.Y) - bx.Y, Z: math.Abs(p.Z) - bx.Z}
	out := Vec3{X: math.Max(d.X, 0), Y: math.Max(d.Y, 0), Z: math.Max(d.Z, 0)}.Len()
	return out + math.Min(math.Max(d.X, math.Max(d.Y, d.Z)), 0)
}

func sdTorus(p Vec3, ringR, tubeR float64) float64 {
	q := math.Hypot(p.X, p.Z) - ringR
	return math.Hypot(q, p.Y) - tubeR
}

// smin is a smooth (rounded) union: it blends two fields so surfaces merge with a
// fillet of width k — the "melting"/metaball look.
func smin(a, b, k float64) float64 {
	if k <= 0 {
		return math.Min(a, b)
	}
	h := math.Max(k-math.Abs(a-b), 0) / k
	return math.Min(a, b) - h*h*k*0.25
}

func fmod(x, y float64) float64 { // positive modulo
	m := math.Mod(x, y)
	if m < 0 {
		m += y
	}
	return m
}

// mengerDE is the distance estimator for a Menger sponge (recursive cube fold).
func mengerDE(p Vec3, iters int) float64 {
	d := sdBox(p, Vec3{X: 1, Y: 1, Z: 1})
	s := 1.0
	for i := 0; i < iters; i++ {
		a := Vec3{X: fmod(p.X*s, 2) - 1, Y: fmod(p.Y*s, 2) - 1, Z: fmod(p.Z*s, 2) - 1}
		s *= 3
		r := Vec3{X: math.Abs(1 - 3*math.Abs(a.X)), Y: math.Abs(1 - 3*math.Abs(a.Y)), Z: math.Abs(1 - 3*math.Abs(a.Z))}
		da := math.Max(r.X, r.Y)
		db := math.Max(r.Y, r.Z)
		dc := math.Max(r.Z, r.X)
		c := (math.Min(da, math.Min(db, dc)) - 1) / s
		if c > d {
			d = c
		}
	}
	return d
}

// mandelbulbDE is the analytic-derivative distance estimator for the Mandelbulb.
func mandelbulbDE(pos Vec3, power float64, iters int) float64 {
	z := pos
	dr := 1.0
	r := 0.0
	for i := 0; i < iters; i++ {
		r = z.Len()
		if r > 2 {
			break
		}
		theta := math.Acos(z.Z/r) * power
		phi := math.Atan2(z.Y, z.X) * power
		zr := math.Pow(r, power)
		dr = math.Pow(r, power-1)*power*dr + 1
		z = Vec3{
			X: zr * math.Sin(theta) * math.Cos(phi),
			Y: zr * math.Sin(theta) * math.Sin(phi),
			Z: zr * math.Cos(theta),
		}.Add(pos)
	}
	if r < 1e-9 {
		return 0
	}
	return 0.5 * math.Log(r) * r / dr
}

// mandalaDE folds space radially into `sym` mirrored wedges and fills the wedge
// with a small arrangement of rings and beads — a 3-D mandala in relief.
func mandalaDE(p Vec3, sym int) float64 {
	ang := math.Atan2(p.Z, p.X)
	rad := math.Hypot(p.X, p.Z)
	wedge := 2 * math.Pi / float64(sym)
	ang = fmod(ang, wedge)
	ang = math.Abs(ang - wedge/2) // mirror within the wedge
	fp := Vec3{X: math.Cos(ang) * rad, Y: p.Y, Z: math.Sin(ang) * rad}
	d := sdTorus(Vec3{X: fp.X - 0.6, Y: fp.Y, Z: fp.Z}, 0.22, 0.07)
	d = math.Min(d, sdSphere(Vec3{X: fp.X - 1.0, Y: fp.Y, Z: fp.Z}, 0.13))
	d = math.Min(d, sdTorus(fp, 0.95, 0.05))
	d = math.Min(d, sdSphere(fp, 0.22))
	return d
}

// NewMarched builds a named SDF object centred at center within a bounding sphere
// of the given radius: "mandelbulb", "menger", "mandala", or "melt" (two
// smooth-blended blobs). ok is false for an unknown name.
func NewMarched(name string, center Vec3, radius float64, mat Material) (Marched, bool) {
	m := Marched{Center: center, Radius: radius, Mat: mat, StepK: 1}
	switch strings.ToLower(name) {
	case "mandelbulb":
		sc := radius / 1.3
		m.DE = func(p Vec3) float64 { return mandelbulbDE(p.Scale(1/sc), 8, 8) * sc }
		m.StepK = 0.8 // the analytic DE overshoots near the surface
	case "menger":
		sc := radius
		m.DE = func(p Vec3) float64 { return mengerDE(p.Scale(1/sc), 4) * sc }
	case "mandala":
		sc := radius
		m.DE = func(p Vec3) float64 { return mandalaDE(p.Scale(1/sc), 8) * sc }
		m.StepK = 0.9
	case "melt": // a Dali-style metaball: two spheres melting together
		sc := radius
		m.DE = func(p Vec3) float64 {
			a := sdSphere(Vec3{X: p.X + 0.45*sc, Y: p.Y, Z: p.Z}, 0.6*sc)
			b := sdSphere(Vec3{X: p.X - 0.45*sc, Y: p.Y - 0.45*sc, Z: p.Z}, 0.5*sc)
			return smin(a, b, 0.5*sc)
		}
	default:
		return Marched{}, false
	}
	return m, true
}
