package raydir

import (
	"math"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// Recycling keeps the particle count flat however long it rains.
func TestWeatherCountStable(t *testing.T) {
	w := NewWeather("rain", 1)
	n := len(w.parts)
	if n == 0 {
		t.Fatal("rain should have particles")
	}
	for i := 0; i < 400; i++ {
		w.Step(0.05, raytrace.Vec3{})
	}
	if len(w.parts) != n {
		t.Errorf("particle count drifted: %d -> %d", n, len(w.parts))
	}
}

// The weather band stays around the walker: particles that fall or drift out are
// recycled back into the box around the current centre.
func TestWeatherBanded(t *testing.T) {
	w := NewWeather("rain", 2)
	centre := raytrace.Vec3{X: 50, Z: 50} // walker far from where particles spawned
	for i := 0; i < 300; i++ {
		w.Step(0.05, centre)
	}
	for _, p := range w.parts {
		if math.Abs(p.Pos.X-centre.X) > w.box+1e-6 || math.Abs(p.Pos.Z-centre.Z) > w.box+1e-6 {
			t.Fatalf("particle escaped the band around the walker: %+v", p.Pos)
		}
		if p.Pos.Y < w.bot-1e-6 || p.Pos.Y > w.top+1e-6 {
			t.Fatalf("particle out of the vertical column: y=%.2f", p.Pos.Y)
		}
	}
}

// Precipitation descends over time.
func TestWeatherDescends(t *testing.T) {
	w := NewWeather("snow", 3)
	for i := range w.parts { // lift them high so none recycle during this short run
		w.parts[i].Pos.Y = 12
	}
	meanY := func() float64 {
		s := 0.0
		for _, p := range w.parts {
			s += p.Pos.Y
		}
		return s / float64(len(w.parts))
	}
	before := meanY()
	for i := 0; i < 10; i++ {
		w.Step(0.05, raytrace.Vec3{})
	}
	if after := meanY(); after >= before {
		t.Errorf("snow should descend: mean Y before %.3f after %.3f", before, after)
	}
}

// Wind tilts the fall direction; with no wind it is straight down.
func TestWeatherWindTilt(t *testing.T) {
	w := NewWeather("rain", 4)
	tilted := w.FallDir()
	if tilted.X <= 0 {
		t.Errorf("a +X wind should tilt rain to +X, got dir %+v", tilted)
	}
	w.Wind = raytrace.Vec3{}
	straight := w.FallDir()
	if straight.X != 0 || straight.Y >= 0 {
		t.Errorf("with no wind rain should fall straight down, got %+v", straight)
	}
}

// Fog is a scene property, not particles.
func TestWeatherFog(t *testing.T) {
	fog := NewWeather("fog", 5)
	d, _, on := fog.Fog()
	if !on || d <= 0 {
		t.Errorf("fog weather should report on with positive density, got d=%.3f on=%v", d, on)
	}
	if len(fog.Objects()) != 0 {
		t.Error("fog should have no particle objects")
	}
	if _, _, rainFog := NewWeather("rain", 6).Fog(); rainFog {
		t.Error("rain should not report fog")
	}
}

// Object counts: rain is two-triangle streaks, snow is single flakes.
func TestWeatherObjectCounts(t *testing.T) {
	rain := NewWeather("rain", 7)
	if got, want := len(rain.Objects()), len(rain.parts)*2; got != want {
		t.Errorf("rain objects: got %d want %d", got, want)
	}
	snow := NewWeather("snow", 8)
	if got, want := len(snow.Objects()), len(snow.parts); got != want {
		t.Errorf("snow objects: got %d want %d", got, want)
	}
}

// The world wires weather into the rendered scene: particles add objects and fog
// sets the scene's fog parameters.
func TestWorldWeatherInScene(t *testing.T) {
	w := NewWorld()
	base := len(w.Scene().Objects)
	w.SetWeather("rain")
	if !w.HasWeather() || !w.HasAnimated() {
		t.Fatal("a world with rain should report weather and animation")
	}
	if got := len(w.SceneWith(nil).Objects); got <= base {
		t.Errorf("rain should add objects to the scene: base %d, with rain %d", base, got)
	}
	w.SetWeather("fog")
	s := w.SceneWith(nil)
	if s.FogDensity <= 0 {
		t.Errorf("fog weather should set scene fog density, got %.3f", s.FogDensity)
	}
	w.SetWeather("")
	if w.HasWeather() {
		t.Error("clearing weather should turn it off")
	}
}
