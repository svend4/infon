package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

func lum(v raytrace.Vec3) float64 { return 0.2126*v.X + 0.7152*v.Y + 0.0722*v.Z }

// Noon is bright with the sun up; midnight is dark with the sun down; dawn glows
// warm at the horizon.
func TestSkyForTime(t *testing.T) {
	noonTop, _, _, _, noonUp := SkyForTime(0.5)
	midTop, _, _, _, midUp := SkyForTime(0.0)
	if !noonUp {
		t.Error("sun should be up at noon")
	}
	if midUp {
		t.Error("sun should be down at midnight")
	}
	if lum(noonTop) <= lum(midTop) {
		t.Errorf("noon sky (%.3f) should be brighter than midnight (%.3f)", lum(noonTop), lum(midTop))
	}
	_, dawnBot, _, _, _ := SkyForTime(0.25)
	if dawnBot.X <= dawnBot.Z {
		t.Errorf("dawn horizon should be warm (R>B): %v", dawnBot)
	}
}

// The sun arcs across the sky: rising in the east, overhead at noon, setting west.
func TestSunArc(t *testing.T) {
	_, _, rise, _, _ := SkyForTime(0.25)
	_, _, noon, _, _ := SkyForTime(0.5)
	_, _, set, _, _ := SkyForTime(0.75)
	if rise.X <= 0 {
		t.Errorf("sunrise should be toward +X (east), got %v", rise)
	}
	if set.X >= 0 {
		t.Errorf("sunset should be toward -X (west), got %v", set)
	}
	if noon.Y < rise.Y || noon.Y < set.Y {
		t.Errorf("noon sun should be highest: rise %.2f noon %.2f set %.2f", rise.Y, noon.Y, set.Y)
	}
}

func TestEnvRoundTrip(t *testing.T) {
	for _, v := range []float64{0, 0.37, 0.999} {
		got, err := DecodeEnv(EncodeEnv(v))
		if err != nil || math.Abs(got-v) > 1e-12 {
			t.Errorf("env round-trip %v = %v,%v", v, got, err)
		}
	}
	if _, err := DecodeEnv([]byte{1, 2, 3}); err == nil {
		t.Error("short env should error")
	}
}

// A timed world renders a bright daytime sky and a dark night sky, and places a
// sun emitter while the sun is up.
func TestWorldTimeSky(t *testing.T) {
	w := NewWorld()
	w.SetTime(0.5)
	day := w.SceneWith(nil)
	w.SetTime(0.0)
	night := w.SceneWith(nil)
	if lum(day.SkyTop) <= lum(night.SkyTop) {
		t.Errorf("daytime sky should be brighter: day %.3f night %.3f", lum(day.SkyTop), lum(night.SkyTop))
	}
	emitters := func(s *raytrace.Scene) int {
		n := 0
		for _, o := range s.Objects {
			if sp, ok := o.(raytrace.Sphere); ok && sp.Mat.Emit.LenSq() > 0 {
				n++
			}
		}
		return n
	}
	if emitters(day) == 0 {
		t.Error("daytime scene should include a sun emitter")
	}
	if emitters(night) != 0 {
		t.Error("night scene should have no sun emitter")
	}
}
