package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

func TestWeatherFromYang(t *testing.T) {
	cases := map[float64]string{0: "snow", 1.5: "snow", 3.0: "fog", 4.5: "rain", 6: "rain"}
	for y, want := range cases {
		if got := WeatherFromYang(y); got != want {
			t.Errorf("WeatherFromYang(%.1f) = %q, want %q", y, got, want)
		}
	}
}

func TestClimateDeterministic(t *testing.T) {
	a, b := NewClimate(42), NewClimate(42)
	for i := 0; i < 25; i++ {
		if a.Kind() != b.Kind() || a.MeanYang() != b.MeanYang() {
			t.Fatalf("step %d: same seed diverged (%s/%.3f vs %s/%.3f)", i, a.Kind(), a.MeanYang(), b.Kind(), b.MeanYang())
		}
		a.Step()
		b.Step()
	}
}

func TestClimateForecastMatchesSteppingAndIsPure(t *testing.T) {
	c := NewClimate(7)
	before := c.MeanYang()
	fc := c.Forecast(6)
	if c.MeanYang() != before {
		t.Error("Forecast must not mutate the climate")
	}
	d := NewClimate(7)
	for i, k := range fc {
		d.Step()
		if d.Kind() != k {
			t.Errorf("forecast[%d]=%q but stepping gives %q", i, k, d.Kind())
		}
	}
}

func TestKindAtIsRegionalAndDeterministic(t *testing.T) {
	c := NewClimate(42)
	valid := map[string]bool{"rain": true, "snow": true, "fog": true}
	seen := map[string]bool{}
	for x := 0; x < 8; x++ {
		for z := 0; z < 8; z++ {
			k := c.KindAt(float64(x)*12+1, float64(z)*12+1, 12)
			if !valid[k] {
				t.Fatalf("cell (%d,%d) gave invalid kind %q", x, z, k)
			}
			seen[k] = true
			// same position is stable
			if k2 := c.KindAt(float64(x)*12+1, float64(z)*12+1, 12); k2 != k {
				t.Errorf("KindAt not deterministic at (%d,%d): %q vs %q", x, z, k, k2)
			}
		}
	}
	if len(seen) < 2 {
		t.Errorf("an 8x8 climate map should hold more than one weather zone, got %v", seen)
	}
}

func TestWorldClimateDrivesWeatherByRegion(t *testing.T) {
	w := NewWorld()
	if w.HasWeather() || w.HasClimate() {
		t.Fatal("a fresh world has no weather/climate")
	}
	w.SetClimate(42)
	if !w.HasClimate() || !w.HasWeather() {
		t.Fatal("SetClimate should install a climate and weather")
	}
	valid := map[string]bool{"rain": true, "snow": true, "fog": true}
	changes := 0
	prev := w.weather.Kind
	for z := 0; z < 100; z++ { // walk down the world, crossing climate zones
		pos := raytrace.Vec3{Z: float64(z) * 3}
		w.UpdateClimate(pos)
		if !valid[w.weather.Kind] {
			t.Fatalf("z=%d invalid weather %q", z, w.weather.Kind)
		}
		if w.weather.Kind != w.climate.KindAt(pos.X, pos.Z, 12) {
			t.Fatalf("z=%d weather %q should match the cell %q", z, w.weather.Kind, w.climate.KindAt(pos.X, pos.Z, 12))
		}
		if w.weather.Kind != prev {
			changes++
			prev = w.weather.Kind
		}
	}
	if changes == 0 {
		t.Error("walking across the climate map should change the weather at least once")
	}
}
