package raytrace

import (
	"math"
	"math/rand"
	"testing"
)

// randomTris builds n deterministic pseudo-random triangles in a box.
func randomTris(n int, seed int64) []Triangle {
	rng := rand.New(rand.NewSource(seed))
	p := func() Vec3 { return Vec3{rng.Float64()*4 - 2, rng.Float64()*4 - 2, rng.Float64()*4 - 2} }
	tris := make([]Triangle, n)
	for i := range tris {
		base := p()
		tris[i] = Triangle{A: base, B: base.Add(p().Scale(0.3)), C: base.Add(p().Scale(0.3)), Mat: Material{Color: Vec3{1, 1, 1}}}
	}
	return tris
}

func TestBVHMatchesLinearScan(t *testing.T) {
	tris := randomTris(400, 7)
	m := NewMesh(tris)         // with BVH
	naive := &Mesh{Tris: tris} // bvh == nil -> linear scan
	if m.bvh == nil {
		t.Fatal("expected a BVH to be built")
	}
	rng := rand.New(rand.NewSource(99))
	for i := 0; i < 300; i++ {
		o := Vec3{rng.Float64()*8 - 4, rng.Float64()*8 - 4, rng.Float64()*8 - 4}
		d := Vec3{rng.Float64()*2 - 1, rng.Float64()*2 - 1, rng.Float64()*2 - 1}.Norm()
		r := Ray{Origin: o, Dir: d}
		hb, okb := m.Intersect(r, 1e-4, 1e9)
		hn, okn := naive.scan(r, 1e-4, 1e9)
		if okb != okn {
			t.Fatalf("ray %d: BVH hit=%v, scan hit=%v", i, okb, okn)
		}
		if okb && math.Abs(hb.T-hn.T) > 1e-6 {
			t.Fatalf("ray %d: BVH t=%v, scan t=%v", i, hb.T, hn.T)
		}
	}
}
