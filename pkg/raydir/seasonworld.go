package raydir

import "math"

// seasonworld.go deepens the living world with TIME: a SeasonalWorld whose weather both
// cycles (a year of wet and dry) and MOVES (a rain front sweeping across the land), so
// the food follows the seasons and the creatures migrate after the rain — booming in
// the wet, thinning in the dry. It drives the existing ecosystem through its weather
// hook, so the foraging, breeding and dying are unchanged; only the rain moves.

// SeasonalWorld is an ecosystem under a cyclic, moving weather front.
type SeasonalWorld struct {
	eco     *Ecosystem
	yearLen int
	depth   float64 // arena depth (the axis the rain front sweeps along)
}

// NewSeasonalWorld builds a gw×gh world of n creatures whose weather turns over a year
// of yearLen ticks: a global wet/dry cycle times a rain band that sweeps along the
// world, with a thin dry-season baseline so life persists at low density between rains.
func NewSeasonalWorld(gw, gh, n int, seed int64, yearLen int) *SeasonalWorld {
	if yearLen < 1 {
		yearLen = 1
	}
	eco := NewEcosystem(gw, gh, n, nil, seed)
	_, sd := eco.Span()
	s := &SeasonalWorld{eco: eco, yearLen: yearLen, depth: sd}
	eco.SetWeather(func(_, z float64, tick int) float64 {
		phase := 2 * math.Pi * float64(tick) / float64(yearLen)
		wet := 0.5 + 0.5*math.Sin(phase)           // global seasonal wetness 0..1
		bandZ := sd * (0.5 - 0.42*math.Cos(phase)) // the rain front sweeps over the year
		sigma := sd * 0.18
		d := z - bandZ
		band := math.Exp(-(d * d) / (2 * sigma * sigma)) // a gaussian rain band
		return 0.01 + 0.05*wet*band                      // dry baseline + wet moving band
	})
	return s
}

// Step advances the world (and so the season) one tick.
func (s *SeasonalWorld) Step() { s.eco.Step() }

// Population is the living population.
func (s *SeasonalWorld) Population() int { return s.eco.Population() }

// Tick is the number of steps taken.
func (s *SeasonalWorld) Tick() int { return s.eco.Tick() }

// Eco exposes the ecosystem (for rendering the arena).
func (s *SeasonalWorld) Eco() *Ecosystem { return s.eco }

// phase is the year fraction (0..1).
func (s *SeasonalWorld) phase() float64 {
	return math.Mod(float64(s.eco.Tick())/float64(s.yearLen), 1)
}

// Wetness is the current global seasonal wetness (0 dry .. 1 wet).
func (s *SeasonalWorld) Wetness() float64 {
	return 0.5 + 0.5*math.Sin(2*math.Pi*s.phase())
}

// RainBandZ is the world-Z centre of the rain front right now (for drawing it).
func (s *SeasonalWorld) RainBandZ() float64 {
	return s.depth * (0.5 - 0.42*math.Cos(2*math.Pi*s.phase()))
}

// SeasonName names the time of year from the phase.
func (s *SeasonalWorld) SeasonName() string {
	switch int(s.phase() * 4) {
	case 0:
		return "spring"
	case 1:
		return "summer"
	case 2:
		return "autumn"
	default:
		return "winter"
	}
}
