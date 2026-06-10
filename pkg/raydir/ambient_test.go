package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func rms(s []int16) float64 {
	var acc float64
	for _, v := range s {
		x := float64(v) / 32767
		acc += x * x
	}
	return math.Sqrt(acc / float64(len(s)))
}

// Silence in, silence out; features make sound; output stays in range.
func TestAmbientLevels(t *testing.T) {
	quiet := AmbientFrame(AmbientFeatures{}, 16000, 0, 4000)
	if rms(quiet) != 0 {
		t.Errorf("no features should be silent, rms=%.4f", rms(quiet))
	}
	windy := AmbientFrame(AmbientFeatures{Wind: 1}, 16000, 0, 4000)
	if rms(windy) <= 0 {
		t.Error("wind should produce sound")
	}
	loud := AmbientFrame(AmbientFeatures{Wind: 1, Water: 1, Hum: 1}, 16000, 0, 4000)
	if rms(loud) <= rms(windy) {
		t.Error("more features should be louder")
	}
}

// The synth is deterministic and seamless: one 2n call equals two back-to-back n
// calls (so streamed frames join without clicks).
func TestAmbientSeamless(t *testing.T) {
	f := AmbientFeatures{Wind: 0.6, Water: 0.5, Hum: 0.3}
	const rate, n = 16000, 320
	whole := AmbientFrame(f, rate, 0, 2*n)
	a := AmbientFrame(f, rate, 0, n)
	b := AmbientFrame(f, rate, float64(n)/rate, n)
	for i := 0; i < n; i++ {
		if whole[i] != a[i] || whole[n+i] != b[i] {
			t.Fatalf("frames should join seamlessly and be deterministic (mismatch at %d)", i)
		}
	}
}

// The soundscape reflects the world: water + forest + day birds turn into the
// matching levels.
func TestWorldAmbientFeatures(t *testing.T) {
	w := NewWorld()
	_, spec, _ := AuthorScene(brain.Local{}, "a forest with a lake and a flock of birds")
	w.AddRegion(Region{Index: 0, At: raytrace.Vec3{Z: 8}, Spec: spec})
	w.SetTime(0.5) // noon
	a := w.Ambient()
	if a.Water <= 0 {
		t.Error("a lake should add water sound")
	}
	if a.Forest <= 0 {
		t.Error("a forest should add forest sound")
	}
	if a.Birds <= 0 {
		t.Error("birds by day should sing")
	}
	w.SetTime(0.0) // midnight
	night := w.Ambient()
	if night.Night <= a.Night {
		t.Error("midnight should be more 'night' than noon")
	}
	if night.Birds >= a.Birds {
		t.Error("birds should quiet down at night")
	}
}
