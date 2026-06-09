// instance.go adds instancing: an Instance wraps any bounded object with an affine
// transform, so one piece of geometry (especially a Mesh with its own BVH) can be
// placed many times — forests, crowds, repeated props — without duplicating the
// triangles. A ray is transformed into the object's local space, intersected
// there, and the hit point and normal are transformed back. The same t parameter
// is valid in both spaces once the local direction is renormalised, so non-uniform
// scaling works; the surface normal is carried by the inverse-transpose, so it
// stays perpendicular under shear/scale. Clean-room implementation.
package raytrace

import "math"

// mat3 is a 3x3 matrix stored as three row vectors.
type mat3 [3]Vec3

func identity3() mat3 {
	return mat3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
}

// mul applies the matrix to a vector (rows dotted with v).
func (m mat3) mul(v Vec3) Vec3 {
	return Vec3{X: m[0].Dot(v), Y: m[1].Dot(v), Z: m[2].Dot(v)}
}

// mulM returns m*n.
func (m mat3) mulM(n mat3) mat3 {
	// row i of m*n is a linear combination of n's rows weighted by m[i].
	row := func(r Vec3) Vec3 {
		return n[0].Scale(r.X).Add(n[1].Scale(r.Y)).Add(n[2].Scale(r.Z))
	}
	return mat3{row(m[0]), row(m[1]), row(m[2])}
}

func (m mat3) transpose() mat3 {
	return mat3{
		{m[0].X, m[1].X, m[2].X},
		{m[0].Y, m[1].Y, m[2].Y},
		{m[0].Z, m[1].Z, m[2].Z},
	}
}

// inverse returns the matrix inverse via the adjugate, and ok=false if singular.
func (m mat3) inverse() (mat3, bool) {
	a, b, c := m[0].X, m[0].Y, m[0].Z
	d, e, f := m[1].X, m[1].Y, m[1].Z
	g, h, i := m[2].X, m[2].Y, m[2].Z
	det := a*(e*i-f*h) - b*(d*i-f*g) + c*(d*h-e*g)
	if math.Abs(det) < 1e-18 {
		return mat3{}, false
	}
	inv := 1 / det
	return mat3{
		{(e*i - f*h) * inv, (c*h - b*i) * inv, (b*f - c*e) * inv},
		{(f*g - d*i) * inv, (a*i - c*g) * inv, (c*d - a*f) * inv},
		{(d*h - e*g) * inv, (b*g - a*h) * inv, (a*e - b*d) * inv},
	}, true
}

// Transform is an affine map x -> M*x + T.
type Transform struct {
	M mat3
	T Vec3
}

// Translate, Scale, ScaleUniform and the Rotate* helpers build basic transforms;
// compose them with Mul (a.Mul(b) applies b first, then a).
func Translate(v Vec3) Transform       { return Transform{identity3(), v} }
func Scale(v Vec3) Transform           { return Transform{mat3{{v.X, 0, 0}, {0, v.Y, 0}, {0, 0, v.Z}}, Vec3{}} }
func ScaleUniform(s float64) Transform { return Scale(Vec3{s, s, s}) }

func RotateX(a float64) Transform {
	c, s := math.Cos(a), math.Sin(a)
	return Transform{mat3{{1, 0, 0}, {0, c, -s}, {0, s, c}}, Vec3{}}
}

func RotateY(a float64) Transform {
	c, s := math.Cos(a), math.Sin(a)
	return Transform{mat3{{c, 0, s}, {0, 1, 0}, {-s, 0, c}}, Vec3{}}
}

func RotateZ(a float64) Transform {
	c, s := math.Cos(a), math.Sin(a)
	return Transform{mat3{{c, -s, 0}, {s, c, 0}, {0, 0, 1}}, Vec3{}}
}

// Mul composes transforms: a.Mul(b) applies b first, then a.
func (a Transform) Mul(b Transform) Transform {
	return Transform{M: a.M.mulM(b.M), T: a.M.mul(b.T).Add(a.T)}
}

// Instance is a transformed placement of a (bounded) object.
type Instance struct {
	obj     Object
	m       mat3      // local -> world linear part
	t       Vec3      // local -> world translation
	minv    mat3      // world -> local linear part
	normalM mat3      // inverse-transpose, for normals
	mat     *Material // optional material override (nil = keep the object's own)
}

// NewInstance places obj under the given local->world transform.
func NewInstance(obj Object, xf Transform) *Instance {
	minv, ok := xf.M.inverse()
	if !ok {
		minv = identity3() // degenerate transform: fall back to identity linear part
	}
	return &Instance{obj: obj, m: xf.M, t: xf.T, minv: minv, normalM: minv.transpose()}
}

// NewInstanceMat is NewInstance with a material override: every hit on this
// placement reports mat instead of the wrapped object's own material, so one
// shared mesh can be placed in many colours/finishes without duplicating geometry.
func NewInstanceMat(obj Object, xf Transform, mat Material) *Instance {
	in := NewInstance(obj, xf)
	in.mat = &mat
	return in
}

// Intersect transforms the ray into local space, intersects the wrapped object,
// and maps the hit back to world space.
func (in *Instance) Intersect(r Ray, tMin, tMax float64) (Hit, bool) {
	lo := in.minv.mul(r.Origin.Sub(in.t))
	ld := in.minv.mul(r.Dir)
	ldLen := ld.Len()
	if ldLen < geomEps {
		return Hit{}, false
	}
	local := Ray{Origin: lo, Dir: ld.Scale(1 / ldLen), Time: r.Time}
	h, ok := in.obj.Intersect(local, tMin*ldLen, tMax*ldLen)
	if !ok {
		return Hit{}, false
	}
	wt := h.T / ldLen // local t (unit dir) -> world t
	wn := in.normalM.mul(h.N).Norm()
	n, front := orient(wn, r.Dir)
	h.T = wt
	h.P = r.At(wt)
	h.N = n
	h.Front = front
	if h.Tan.LenSq() > geomEps { // tangents transform by M (not the inverse-transpose)
		h.Tan = in.m.mul(h.Tan).Norm()
	}
	if in.mat != nil { // per-instance material override
		h.Mat = *in.mat
	}
	return h, true
}
