package raytrace

import (
	"math"
	"testing"
)

func approx(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %v, want %v (±%v)", name, got, want, tol)
	}
}

func sampleWire() SceneWire {
	return SceneWire{
		Cam:   Camera{Pos: Vec3{1.5, 2.5, -3.0}, Yaw: 0.5, Pitch: -0.2, FOV: 1.0},
		Light: Vec3{4, 5, 6},
		Spheres: []Sphere{
			{Center: Vec3{0, 1, 0}, Radius: 1.0, Mat: Material{Color: Vec3{0.8, 0.2, 0.2}, Reflect: 0.3}},
			{Center: Vec3{-2.25, 0.5, 1.0}, Radius: 0.5, Mat: Material{Color: Vec3{0.2, 0.6, 0.9}}},
		},
	}
}

func TestWireRoundTrip(t *testing.T) {
	w := sampleWire()
	got, err := Decode(Encode(w, 8), 8)
	if err != nil {
		t.Fatal(err)
	}
	approx(t, "cam.x", got.Cam.Pos.X, 1.5, 1.0/posScale)
	approx(t, "cam.y", got.Cam.Pos.Y, 2.5, 1.0/posScale)
	approx(t, "cam.z", got.Cam.Pos.Z, -3.0, 1.0/posScale)
	approx(t, "yaw", got.Cam.Yaw, 0.5, 1.0/angScale)
	approx(t, "pitch", got.Cam.Pitch, -0.2, 1.0/angScale)
	approx(t, "fov", got.Cam.FOV, 1.0, fovSpan/255)
	approx(t, "light.z", got.Light.Z, 6, 1.0/posScale)
	if len(got.Spheres) != 2 {
		t.Fatalf("spheres = %d, want 2", len(got.Spheres))
	}
	approx(t, "s0.r", got.Spheres[0].Radius, 1.0, 1.0/radScale)
	approx(t, "s0.reflect", got.Spheres[0].Mat.Reflect, 0.3, 2.0/255)
	approx(t, "s0.color.x", got.Spheres[0].Mat.Color.X, 0.8, 2.0/255)
	approx(t, "s1.center.x", got.Spheres[1].Center.X, -2.25, 1.0/posScale)
}

func TestWireSurvivesBurst(t *testing.T) {
	const ecc = 12
	wire := Encode(sampleWire(), ecc)
	// Corrupt a few scattered bytes (within the ECC budget after interleaving).
	for _, i := range []int{3, 11, 19} {
		if i < len(wire) {
			wire[i] ^= 0xFF
		}
	}
	got, err := Decode(wire, ecc)
	if err != nil {
		t.Fatalf("burst should be correctable: %v", err)
	}
	approx(t, "cam.x after burst", got.Cam.Pos.X, 1.5, 1.0/posScale)
	if len(got.Spheres) != 2 {
		t.Fatalf("spheres after burst = %d, want 2", len(got.Spheres))
	}
}

func TestWireRejectsGarbage(t *testing.T) {
	// Must return an error, never panic, on short/garbage input.
	if _, err := Decode([]byte{1, 2, 3}, 8); err == nil {
		t.Error("expected error on garbage wire")
	}
}

func TestWireBuildsRenderableScene(t *testing.T) {
	w := sampleWire()
	sc := w.Build(0.12)
	if len(sc.Objects) != 1+len(w.Spheres) { // floor + spheres
		t.Errorf("scene objects = %d, want %d", len(sc.Objects), 1+len(w.Spheres))
	}
	img := Render(sc, w.Cam, 16, 16, Options{Samples: 1})
	if img.Bounds().Dx() != 16 || img.Bounds().Dy() != 16 {
		t.Errorf("render bounds = %v", img.Bounds())
	}
}
