package raytrace

import "testing"

func TestGodRaysShadowedMediumIsDarker(t *testing.T) {
	scene := &Scene{
		Objects:     []Object{Sphere{Center: Vec3{0, 5, 0}, Radius: 2, Mat: Material{Color: Vec3{0.5, 0.5, 0.5}}}},
		Light:       Vec3{0, 10, 0},
		LightInt:    1,
		Volume:      0.2,
		VolumeColor: Vec3{1, 1, 1},
	}
	// Straight up under the occluder: the medium is shadowed from the light above.
	insShadow, trShadow := scene.marchVolume(Ray{Origin: Vec3{0, 0, 0}, Dir: Vec3{0, 1, 0}}, 3)
	// Straight up off to the side: the medium is lit.
	insLit, _ := scene.marchVolume(Ray{Origin: Vec3{6, 0, 0}, Dir: Vec3{0, 1, 0}}, 3)

	if insLit.X <= insShadow.X {
		t.Errorf("lit medium (%.4f) should out-glow shadowed medium (%.4f)", insLit.X, insShadow.X)
	}
	if trShadow >= 1 || trShadow <= 0 {
		t.Errorf("transmittance through the medium should be in (0,1), got %.4f", trShadow)
	}
	// With no medium there is no in-scatter and full transmittance.
	noVol := &Scene{Light: Vec3{0, 10, 0}}
	if ins, tr := noVol.marchVolume(Ray{Origin: Vec3{0, 0, 0}, Dir: Vec3{0, 1, 0}}, 3); ins.Len() != 0 || tr != 1 {
		t.Errorf("no medium should give zero in-scatter and full transmittance, got %v %v", ins, tr)
	}
}
