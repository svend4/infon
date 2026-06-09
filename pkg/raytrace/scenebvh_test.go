package raytrace

import (
	"math"
	"math/rand"
	"testing"
)

func manyObjects(n int, seed int64) []Object {
	rng := rand.New(rand.NewSource(seed))
	objs := make([]Object, 0, n+1)
	for i := 0; i < n; i++ {
		objs = append(objs, Sphere{
			Center: Vec3{X: rng.Float64()*8 - 4, Y: rng.Float64()*8 - 4, Z: rng.Float64()*8 - 4},
			Radius: 0.2 + rng.Float64()*0.5,
			Mat:    Material{Color: Vec3{1, 1, 1}},
		})
	}
	objs = append(objs, Plane{Y: -5, Size: 1, C1: Vec3{1, 1, 1}, C2: Vec3{1, 1, 1}}) // unbounded -> rest list
	return objs
}

func TestSceneBVHMatchesLinear(t *testing.T) {
	objs := manyObjects(120, 11)
	lin := &Scene{Objects: objs}
	acc := &Scene{Objects: objs}
	acc.BuildBVH()
	rq := rand.New(rand.NewSource(22))
	for i := 0; i < 500; i++ {
		o := Vec3{X: rq.Float64()*10 - 5, Y: rq.Float64()*10 - 5, Z: rq.Float64()*10 - 5}
		d := Vec3{X: rq.Float64()*2 - 1, Y: rq.Float64()*2 - 1, Z: rq.Float64()*2 - 1}.Norm()
		r := Ray{Origin: o, Dir: d}
		h1, ok1 := lin.closest(r, 1e-4, 1e9)
		h2, ok2 := acc.closest(r, 1e-4, 1e9)
		if ok1 != ok2 {
			t.Fatalf("ray %d: linear hit=%v, bvh hit=%v", i, ok1, ok2)
		}
		if ok1 && math.Abs(h1.T-h2.T) > 1e-6 {
			t.Fatalf("ray %d: linear t=%v, bvh t=%v", i, h1.T, h2.T)
		}
	}
}

func BenchmarkSceneClosestLinear(b *testing.B) {
	s := &Scene{Objects: manyObjects(300, 1)}
	r := Ray{Origin: Vec3{0, 0, -8}, Dir: Vec3{0, 0, 1}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.closest(r, 1e-4, 1e9)
	}
}

func BenchmarkSceneClosestBVH(b *testing.B) {
	s := &Scene{Objects: manyObjects(300, 1)}
	s.BuildBVH()
	r := Ray{Origin: Vec3{0, 0, -8}, Dir: Vec3{0, 0, 1}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.closest(r, 1e-4, 1e9)
	}
}
