package raytrace

import "testing"

func benchScene() (*Scene, Camera) {
	sc := &Scene{
		Objects: []Object{
			Plane{Y: 0, Size: 1, C1: Vec3{0.2, 0.2, 0.22}, C2: Vec3{0.8, 0.8, 0.82}},
			Sphere{Center: Vec3{-1.2, 1, 0}, Radius: 1, Mat: Material{Color: Vec3{0.85, 0.25, 0.25}}},
			Sphere{Center: Vec3{1.2, 1, 0.4}, Radius: 1, Mat: Material{Color: Vec3{0.9, 0.9, 0.95}, Reflect: 0.7}},
		},
		Light: Vec3{6, 9, -4}, LightInt: 1.4, Ambient: 0.14,
		SkyTop: Vec3{0.45, 0.62, 0.95}, SkyBottom: Vec3{0.92, 0.95, 1}, MaxBounce: 2, AttenK: 0.03,
	}
	return sc, Camera{Pos: Vec3{0, 2.4, 6}, Yaw: 0, Pitch: -0.2, FOV: 1.0}
}

func BenchmarkRender(b *testing.B) {
	sc, cam := benchScene()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Render(sc, cam, 128, 96, Options{Samples: 1})
	}
}

func BenchmarkMeshBVH(b *testing.B) {
	m := NewMesh(randomTris(4000, 3))
	r := Ray{Origin: Vec3{0, 0, -6}, Dir: Vec3{0, 0, 1}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Intersect(r, 1e-4, 1e9)
	}
}

func BenchmarkMeshScan(b *testing.B) {
	m := &Mesh{Tris: randomTris(4000, 3)} // no BVH
	r := Ray{Origin: Vec3{0, 0, -6}, Dir: Vec3{0, 0, 1}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.scan(r, 1e-4, 1e9)
	}
}
