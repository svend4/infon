package raytrace

import "testing"

// A material override replaces the wrapped object's material on every hit, while
// geometry (t, point, normal) is unchanged — so one shared mesh can be placed in
// many finishes.
func TestInstanceMaterialOverride(t *testing.T) {
	unit := Sphere{Center: Vec3{0, 0, 0}, Radius: 1, Mat: Material{Color: Vec3{1, 0, 0}}}
	xf := Translate(Vec3{0, 0, 5})
	plain := NewInstance(unit, xf)
	tint := NewInstanceMat(unit, xf, Material{Color: Vec3{0, 1, 0}, Reflect: 0.5})

	r := Ray{Origin: Vec3{0, 0, 0}, Dir: Vec3{0, 0, 1}}
	hp, okp := plain.Intersect(r, geomEps, tFar)
	ht, okt := tint.Intersect(r, geomEps, tFar)
	if !okp || !okt {
		t.Fatal("both instances should be hit")
	}
	if hp.Mat.Color != (Vec3{1, 0, 0}) {
		t.Errorf("plain instance should keep red, got %v", hp.Mat.Color)
	}
	if ht.Mat.Color != (Vec3{0, 1, 0}) || ht.Mat.Reflect != 0.5 {
		t.Errorf("override not applied: %+v", ht.Mat)
	}
	if hp.T != ht.T || hp.P != ht.P || hp.N != ht.N {
		t.Error("override must not change geometry")
	}
}
