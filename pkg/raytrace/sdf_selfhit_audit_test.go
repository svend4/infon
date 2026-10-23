package raytrace

import (
	"math"
	"testing"
)

// A secondary ray spawned the way the renderers do it (origin offset by shadowEps
// along the oriented normal, tMin = shadowEps) must not re-hit the convex surface it
// leaves, and every reported hit must lie strictly after tMin. Radius > 1 makes the
// default eps (Radius*1e-4) wider than shadowEps.
func TestMarchedSecondaryRayNoSelfHit(t *testing.T) {
	const R = 2.0
	m := Marched{
		DE:     func(p Vec3) float64 { return p.Len() - R },
		Center: Vec3{X: 0, Y: 0, Z: 5}, Radius: R * 1.25,
	}
	cam := Ray{Origin: Vec3{}, Dir: Vec3{X: 0, Y: 0, Z: 1}}
	h, ok := m.Intersect(cam, shadowEps, tFar)
	if !ok {
		t.Fatal("camera ray should hit the field")
	}
	tu, tv := basisAround(h.N)
	for i := 0; i < 64; i++ {
		// directions over the outgoing hemisphere, from grazing to straight out
		cz := 0.05 + 0.95*float64(i%8)/7
		ph := float64(i/8) * 2 * math.Pi / 8
		sz := math.Sqrt(1 - cz*cz)
		dir := tu.Scale(sz * math.Cos(ph)).Add(tv.Scale(sz * math.Sin(ph))).Add(h.N.Scale(cz)).Norm()
		r := Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: dir}
		if h2, ok := m.Intersect(r, shadowEps, tFar); ok {
			t.Fatalf("dir %d (cos %.2f): leaving ray re-hit its own surface at t=%g (tMin=%g)", i, cz, h2.T, shadowEps)
		}
	}
	// the mirror reflection of the camera ray, exactly as the path tracer spawns it
	refl := Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: cam.Dir.Reflect(h.N).Norm()}
	if h2, ok := m.Intersect(refl, shadowEps, tFar); ok {
		t.Fatalf("reflected ray re-hit its own surface at t=%g", h2.T)
	}
}

// A ray spawned just inside the surface (refraction / subsurface walk) reports the
// next crossing after tMin — the exit on the far side — not its own origin.
func TestMarchedInwardRayFindsExit(t *testing.T) {
	const R = 2.0
	m := Marched{
		DE:     func(p Vec3) float64 { return p.Len() - R },
		Center: Vec3{X: 0, Y: 0, Z: 5}, Radius: R * 1.25,
	}
	cam := Ray{Origin: Vec3{}, Dir: Vec3{X: 0, Y: 0, Z: 1}}
	h, ok := m.Intersect(cam, shadowEps, tFar)
	if !ok {
		t.Fatal("camera ray should hit the field")
	}
	in := Ray{Origin: h.P.Sub(h.N.Scale(shadowEps)), Dir: cam.Dir}
	h2, ok := m.Intersect(in, shadowEps, tFar)
	if !ok {
		t.Fatal("inward ray should hit the far side of the field")
	}
	if h2.T <= shadowEps {
		t.Fatalf("hit at t=%g is not strictly after tMin=%g", h2.T, shadowEps)
	}
	if math.Abs(h2.P.Z-(5+R)) > 2e-3 {
		t.Errorf("exit at z=%.5f, want %.5f", h2.P.Z, 5+R)
	}
	if h2.Front {
		t.Error("exit hit should be a back-face hit")
	}
}
