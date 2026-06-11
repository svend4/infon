package raydir

import (
	"image"
	"math"

	"github.com/svend4/infon/pkg/raytrace"
)

// climate.go drives the world's weather from a hexagram cellular automaton (HexCA)
// instead of an RNG. The automaton's mean yang-count picks the weather — yin
// (still, cold) tends to snow, balance to fog, yang (charged, active) to rain — and
// stepping it advances the weather deterministically.
//
// Two technical wins beyond "it looks alive": because the automaton is seeded and
// deterministic, every peer in a shared world derives the SAME weather from the
// same seed and step count — nothing to synchronise over the wire (true to the
// project's "meaning, not pixels") — and the future is computable by running the
// automaton forward (a forecast). It also gives HexCA, which shipped with tests but
// no caller, a job. (The yang->storm mapping is a design choice, not physics.)

// Climate is a HexCA whose aggregate state selects the weather.
type Climate struct {
	ca *HexCA
}

// NewClimate seeds a climate automaton.
func NewClimate(seed int64) *Climate { return &Climate{ca: NewHexCA(8, 8, seed)} }

// MeanYang is the average number of yang lines per cell (0..6) — the automaton's
// overall "charge".
func (c *Climate) MeanYang() float64 {
	g := c.ca.Grid()
	if len(g) == 0 {
		return 0
	}
	sum := 0
	for _, cell := range g {
		sum += yangCount(int(cell))
	}
	return float64(sum) / float64(len(g))
}

// Kind is the global weather from the automaton's mean yang-count (the "season").
func (c *Climate) Kind() string { return WeatherFromYang(c.MeanYang()) }

// KindAt is the weather at world (x,z): the position indexes a cell of the
// automaton (scale world units per cell, toroidal), and that cell's yang-count
// picks the kind — so weather varies by REGION, the automaton's drifting domains
// giving coherent zones. Deterministic in position, so peers agree without syncing.
func (c *Climate) KindAt(x, z, scale float64) string {
	if scale <= 0 {
		scale = 12
	}
	cx := int(math.Floor(x / scale))
	cz := int(math.Floor(z / scale))
	return WeatherFromYang(float64(yangCount(int(c.ca.At(cx, cz)))))
}

// Step advances the automaton one tick (the weather evolves).
func (c *Climate) Step() { c.ca.Step() }

// Render draws the automaton's current grid (see HexCA.Render), px per cell.
func (c *Climate) Render(px int) image.Image { return c.ca.Render(px) }

// Forecast returns the next n weather kinds without disturbing the climate: it runs
// a copy of the automaton forward, so the future is computable because the
// automaton is deterministic.
func (c *Climate) Forecast(n int) []string {
	clone := &Climate{ca: &HexCA{W: c.ca.W, H: c.ca.H, cells: append([]uint8(nil), c.ca.cells...)}}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		clone.Step()
		out = append(out, clone.Kind())
	}
	return out
}

// WeatherFromYang maps a mean yang-count (0..6) to a weather kind: yin -> snow,
// balance -> fog, yang -> rain.
func WeatherFromYang(y float64) string {
	switch {
	case y < 2.7:
		return "snow"
	case y < 3.3:
		return "fog"
	default:
		return "rain"
	}
}

// climateScale is the world-units-per-cell of the climate map.
const climateScale = 12.0

// SetClimate drives the world's weather from a HexCA seeded by seed, and sets the
// initial weather from the cell at the origin.
func (w *World) SetClimate(seed int64) {
	w.climate = NewClimate(seed)
	w.SetWeather(w.climate.KindAt(0, 0, climateScale))
}

// UpdateClimate sets the weather from the climate cell under pos, changing it only
// when you cross into a region with a different kind. Deterministic in position, so
// every peer in a shared world sees the same weather where they stand.
func (w *World) UpdateClimate(pos raytrace.Vec3) {
	if w.climate == nil {
		return
	}
	if k := w.climate.KindAt(pos.X, pos.Z, climateScale); w.weather == nil || w.weather.Kind != k {
		w.SetWeather(k)
	}
}

// StepClimate advances the climate automaton and updates the weather when its kind
// changes (no-op without a climate). Driven by a shared clock, every peer in a
// shared world stays in step without exchanging weather state.
func (w *World) StepClimate() {
	if w.climate == nil {
		return
	}
	prev := w.climate.Kind()
	w.climate.Step()
	if k := w.climate.Kind(); k != prev {
		w.SetWeather(k)
	}
}

// HasClimate reports whether a climate automaton drives the weather.
func (w *World) HasClimate() bool { return w.climate != nil }

// Climate returns the world's climate automaton (or nil).
func (w *World) Climate() *Climate { return w.climate }
