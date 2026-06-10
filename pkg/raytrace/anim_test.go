package raytrace

import (
	"math"
	"testing"
)

func movingAnim() SceneAnim {
	a := SceneAnim{N: 2}
	for f := 0; f < 8; f++ {
		ang := float64(f) * 0.3
		a.Frames = append(a.Frames, SceneWire{
			Cam:   Camera{Pos: Vec3{0, 2, -5}, FOV: 1.0},
			Light: Vec3{4, 6, -3},
			Spheres: []Sphere{
				{Center: Vec3{math.Cos(ang) * 2, 1, math.Sin(ang) * 2}, Radius: 1, Mat: Material{Color: Vec3{0.8, 0.3, 0.3}}},
				{Center: Vec3{0, 0.5, 0}, Radius: 0.5, Mat: Material{Color: Vec3{0.3, 0.6, 0.9}}},
			},
		})
	}
	return a
}

func TestAnimRoundTrip(t *testing.T) {
	a := movingAnim()
	got, err := DecodeAnim(EncodeAnim(a, 10), 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.N != a.N || len(got.Frames) != len(a.Frames) {
		t.Fatalf("anim shape = N%d/%d frames, want N%d/%d", got.N, len(got.Frames), a.N, len(a.Frames))
	}
	last := a.Frames[len(a.Frames)-1].Spheres[0].Center
	dec := got.Frames[len(got.Frames)-1].Spheres[0].Center
	if dec.Sub(last).Len() > 2.0/posScale {
		t.Errorf("final moving sphere = %+v, want %+v", dec, last)
	}
}

func TestAnimDeltaIsSmall(t *testing.T) {
	// Only one sphere moves, so the delta stream must be far smaller than sending
	// every frame in full.
	a := movingAnim()
	raw := AnimRawSize(a)
	full := frameWidth(a.N) * len(a.Frames)
	if raw >= full {
		t.Errorf("delta size %d should be < full size %d", raw, full)
	}
}

func TestAnimSurvivesBurst(t *testing.T) {
	const ecc = 12
	wire := EncodeAnim(movingAnim(), ecc)
	for _, i := range []int{2, 9, 17} {
		if i < len(wire) {
			wire[i] ^= 0xFF
		}
	}
	if _, err := DecodeAnim(wire, ecc); err != nil {
		t.Fatalf("burst should be correctable: %v", err)
	}
}
