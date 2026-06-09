package raytrace

import (
	"strings"
	"testing"
)

// FuzzLoadOBJ checks the .obj parser never panics and that any mesh it returns is
// safe to intersect.
func FuzzLoadOBJ(f *testing.F) {
	f.Add(cubeOBJ)
	f.Add("v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1//1 2//1 3//1\nvn 0 0 1\n")
	f.Add("f 1 2 3\nv x y z\n")
	f.Fuzz(func(t *testing.T, s string) {
		m, err := LoadOBJ(strings.NewReader(s), Material{Color: Vec3{1, 1, 1}})
		if err != nil || m == nil {
			return
		}
		_, _ = m.Intersect(Ray{Origin: Vec3{0, 0, -5}, Dir: Vec3{0, 0, 1}}, 1e-4, 1e9)
	})
}

// FuzzDecodeWire checks the scene-over-RS decoder never panics on arbitrary input.
func FuzzDecodeWire(f *testing.F) {
	f.Add(Encode(sampleWire(), 8))
	f.Add([]byte{0x52, 1, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = Decode(b, 8)
	})
}
