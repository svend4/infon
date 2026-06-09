package raytrace

import (
	"math"
	"testing"
)

func vecClose(a, b Vec3, eps float64) bool { return a.Sub(b).Len() <= eps }

// An instanced unit sphere under translate+uniform-scale must intersect exactly
// like an explicit sphere at the transformed center and radius.
func TestInstanceEqualsMovedSphere(t *testing.T) {
	unit := Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{1, 0, 0}}}
	inst := NewInstance(unit, Translate(Vec3{2, 1, -3}).Mul(ScaleUniform(1.5)))
	ref := Sphere{Center: Vec3{2, 1, -3}, Radius: 1.5, Mat: Material{Color: Vec3{1, 0, 0}}}

	origins := []Vec3{{2, 1, -8}, {-3, 1, -3}, {2, 6, -3}, {6, 3, -7}}
	for _, o := range origins {
		dir := ref.Center.Sub(o).Norm()
		r := Ray{Origin: o, Dir: dir}
		hi, oki := inst.Intersect(r, geomEps, tFar)
		hr, okr := ref.Intersect(r, geomEps, tFar)
		if oki != okr {
			t.Fatalf("origin %v: instance ok=%v but reference ok=%v", o, oki, okr)
		}
		if !okr {
			continue
		}
		if math.Abs(hi.T-hr.T) > 1e-6 || !vecClose(hi.P, hr.P, 1e-6) || !vecClose(hi.N, hr.N, 1e-6) {
			t.Errorf("origin %v: instance hit (t=%.5f P=%v N=%v) != reference (t=%.5f P=%v N=%v)",
				o, hi.T, hi.P, hi.N, hr.T, hr.P, hr.N)
		}
		if hi.Front != hr.Front {
			t.Errorf("origin %v: front mismatch %v vs %v", o, hi.Front, hr.Front)
		}
	}
}

// A translated instance is hit toward its new location and misses the original.
func TestInstanceTranslateMoves(t *testing.T) {
	inst := NewInstance(Sphere{Center: Vec3{0, 0, 0}, Radius: 1}, Translate(Vec3{5, 0, 0}))
	toMoved := Ray{Origin: Vec3{5, 0, -5}, Dir: Vec3{0, 0, 1}}
	if _, ok := inst.Intersect(toMoved, geomEps, tFar); !ok {
		t.Error("should hit the instance at its translated position")
	}
	toOrigin := Ray{Origin: Vec3{0, 0, -5}, Dir: Vec3{0, 0, 1}}
	if _, ok := inst.Intersect(toOrigin, geomEps, tFar); ok {
		t.Error("should miss where the original (untranslated) sphere was")
	}
}

// Under non-uniform scale the surface normal must follow the inverse-transpose, so
// it equals the analytic ellipsoid gradient (and stays perpendicular to the
// surface), not the naively transformed normal.
func TestInstanceNonUniformScaleNormal(t *testing.T) {
	// unit sphere scaled to an ellipsoid with semi-axes (2,1,1).
	inst := NewInstance(Sphere{Center: Vec3{0, 0, 0}, Radius: 1}, Scale(Vec3{2, 1, 1}))
	r := Ray{Origin: Vec3{3, 1.2, 0}, Dir: Vec3{-1, -0.4, 0}.Norm()}
	h, ok := inst.Intersect(r, geomEps, tFar)
	if !ok {
		t.Fatal("ray should hit the ellipsoid")
	}
	// F(x,y,z) = (x/2)^2 + y^2 + z^2 - 1; gradient = (x/2, 2y, 2z).
	grad := Vec3{X: h.P.X / 2, Y: 2 * h.P.Y, Z: 2 * h.P.Z}.Norm()
	// h.N is oriented against the ray (front face), so it should match the outward
	// gradient direction.
	if d := h.N.Dot(grad); d < 0.999 {
		t.Errorf("normal should match the ellipsoid gradient (dot=%.4f): N=%v grad=%v", d, h.N, grad)
	}
	if l := h.N.Len(); math.Abs(l-1) > 1e-9 {
		t.Errorf("normal must be unit length, got %.6f", l)
	}
}

// The scene BVH bounds of an instance must enclose its transformed geometry.
func TestInstanceBoundsEnclose(t *testing.T) {
	inst := NewInstance(Sphere{Center: Vec3{0, 0, 0}, Radius: 1}, Translate(Vec3{4, -2, 1}).Mul(ScaleUniform(2)))
	box, ok := objectBounds(inst)
	if !ok {
		t.Fatal("instance should be bounded")
	}
	// transformed sphere: center (4,-2,1), radius 2 -> min (2,-4,-1), max (6,0,3).
	const eps = 1e-3
	encloses := box.min.X <= 2+eps && box.max.X >= 6-eps &&
		box.min.Y <= -4+eps && box.max.Y >= 0-eps &&
		box.min.Z <= -1+eps && box.max.Z >= 3-eps
	if !encloses {
		t.Errorf("bounds %v..%v do not enclose the transformed sphere (want min<=(2,-4,-1), max>=(6,0,3))", box.min, box.max)
	}
}

// An instance is a drop-in scene object: a BVH over instances matches a brute
// force scan.
func TestInstanceInSceneBVH(t *testing.T) {
	mk := func() []Object {
		return []Object{
			NewInstance(Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{1, 0, 0}}}, Translate(Vec3{-2, 0, 0})),
			NewInstance(Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{0, 1, 0}}}, Translate(Vec3{2, 0, 0}).Mul(ScaleUniform(1.3))),
		}
	}
	scan := &Scene{Objects: mk()}
	bvh := &Scene{Objects: mk()}
	bvh.BuildBVH()
	for _, o := range []Vec3{{-2, 0, -5}, {2, 0, -5}, {0, 0, -5}} {
		r := Ray{Origin: o, Dir: Vec3{0, 0, 1}}
		hs, oks := scan.closest(r, geomEps, tFar)
		hb, okb := bvh.closest(r, geomEps, tFar)
		if oks != okb || (oks && math.Abs(hs.T-hb.T) > 1e-9) {
			t.Errorf("origin %v: scan(ok=%v t=%.4f) != bvh(ok=%v t=%.4f)", o, oks, hs.T, okb, hb.T)
		}
	}
}
