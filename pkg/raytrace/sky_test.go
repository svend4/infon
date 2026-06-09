package raytrace

import (
	"math"
	"testing"
)

func finiteV(v Vec3) bool {
	return !math.IsNaN(v.X) && !math.IsInf(v.X, 0) && !math.IsNaN(v.Y) && !math.IsInf(v.Y, 0) && !math.IsNaN(v.Z) && !math.IsInf(v.Z, 0)
}

// The physical sky is finite and non-negative everywhere, including at and below
// the horizon and toward the sun.
func TestPreethamFiniteNonNeg(t *testing.T) {
	sky := NewPreethamSky(Vec3{X: 0.3, Y: 0.6, Z: 0.5}, 2.5)
	for _, d := range []Vec3{
		{X: 0, Y: 1, Z: 0}, {X: 1, Y: 0.01, Z: 0}, {X: 0, Y: -0.5, Z: 1},
		{X: 0.3, Y: 0.6, Z: 0.5}, {X: -1, Y: 0.2, Z: -1},
	} {
		c := sky.At(d.Norm())
		if !finiteV(c) {
			t.Fatalf("sky(%v) not finite: %v", d, c)
		}
		if c.X < 0 || c.Y < 0 || c.Z < 0 {
			t.Errorf("sky(%v) negative: %v", d, c)
		}
	}
}

// The sky is brighter toward the sun than away from it.
func TestPreethamBrightTowardSun(t *testing.T) {
	sun := Vec3{X: 0, Y: 0.5, Z: 1}.Norm()
	sky := NewPreethamSky(sun, 2.5)
	toward := lumOf(sky.At(sun))
	away := lumOf(sky.At(Vec3{X: 0, Y: 0.5, Z: -1}.Norm()))
	if toward <= away {
		t.Errorf("sky should be brighter toward the sun: toward %.3f away %.3f", toward, away)
	}
}

// With a low sun, the horizon toward the sun is warmer (red>blue) than the cool
// zenith — automatic sunset reddening.
func TestPreethamSunsetWarmHorizon(t *testing.T) {
	sun := Vec3{X: 1, Y: 0.06, Z: 0}.Norm() // just above the horizon
	sky := NewPreethamSky(sun, 3.0)
	horizon := sky.At(Vec3{X: 1, Y: 0.05, Z: 0}.Norm())
	zenith := sky.At(Vec3{X: 0, Y: 1, Z: 0})
	hWarm := horizon.X - horizon.Z // red minus blue
	zWarm := zenith.X - zenith.Z
	if hWarm <= zWarm {
		t.Errorf("low-sun horizon should be warmer than zenith: horizon %.3f zenith %.3f", hWarm, zWarm)
	}
}

// A scene with Sky set uses the physical sky (overriding the gradient).
func TestSceneUsesPhysicalSky(t *testing.T) {
	s := &Scene{SkyTop: Vec3{X: 1, Y: 1, Z: 1}, SkyBottom: Vec3{X: 1, Y: 1, Z: 1}}
	plain := s.sky(Vec3{X: 0, Y: 1, Z: 0})
	s.Sky = NewPreethamSky(Vec3{X: 0, Y: 0.7, Z: 0.7}, 2.5)
	phys := s.sky(Vec3{X: 0, Y: 1, Z: 0})
	if plain == phys {
		t.Error("setting Scene.Sky should change the sky colour")
	}
}
