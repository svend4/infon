package raytrace

import (
	"strings"
	"testing"
)

// a unit cube as six quad faces (exercises fan-triangulation: 6*2 = 12 tris).
const cubeOBJ = `# unit cube
v 0 0 0
v 1 0 0
v 1 1 0
v 0 1 0
v 0 0 1
v 1 0 1
v 1 1 1
v 0 1 1
f 1 2 3 4
f 5 6 7 8
f 1 2 6 5
f 2 3 7 6
f 3 4 8 7
f 4 1 5 8
`

func TestLoadOBJCube(t *testing.T) {
	m, err := LoadOBJ(strings.NewReader(cubeOBJ), Material{Color: Vec3{0.8, 0.8, 0.8}})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Tris) != 12 {
		t.Fatalf("cube tris = %d, want 12", len(m.Tris))
	}
	c, rad := m.Bound()
	if (c.Sub(Vec3{0.5, 0.5, 0.5})).Len() > 1e-9 {
		t.Errorf("bound centre = %v, want (0.5,0.5,0.5)", c)
	}
	if rad < 0.86 || rad > 0.88 { // sqrt(3)/2 ≈ 0.866
		t.Errorf("bound radius = %v, want ~0.866", rad)
	}
	// A ray into the front face (z=0) should hit.
	if _, ok := m.Intersect(Ray{Origin: Vec3{0.5, 0.5, -1}, Dir: Vec3{0, 0, 1}}, 1e-4, 1e9); !ok {
		t.Error("ray into the cube front face should hit")
	}
	// A ray that misses the bounding sphere returns fast with no hit.
	if _, ok := m.Intersect(Ray{Origin: Vec3{50, 50, -1}, Dir: Vec3{0, 0, 1}}, 1e-4, 1e9); ok {
		t.Error("ray far from the cube should miss")
	}
}

func TestLoadOBJFaceFormats(t *testing.T) {
	// vertex/texture/normal and relative indices must all parse to one triangle.
	obj := "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1/1/1 2/2/2 -1//3\n"
	m, err := LoadOBJ(strings.NewReader(obj), Material{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Tris) != 1 {
		t.Fatalf("tris = %d, want 1", len(m.Tris))
	}
}
