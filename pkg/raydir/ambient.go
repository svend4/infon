package raydir

import "math"

// ambient.go lets you HEAR the world. The scene's content and time of day become a
// procedural soundscape — wind that rises at dusk, water lapping where there's
// water, a forest rustle, birds by day, crickets at night, a low hum near glowing
// fractals — synthesised locally from a handful of 0..1 levels. The sound is
// reconstructed from meaning (the world's features), never streamed: another sense
// for "meaning, not pixels".

// AmbientFeatures are the levels (0..1) that drive the soundscape.
type AmbientFeatures struct {
	Wind   float64
	Water  float64
	Forest float64
	Birds  float64
	Hum    float64
	Night  float64
}

// hashf maps an integer to a stable value in [-1,1].
func hashf(i int64) float64 {
	u := uint64(i)*2654435761 + 0x9e3779b97f4a7c15
	u ^= u >> 33
	u *= 0xff51afd7ed558ccd
	u ^= u >> 33
	return float64(u>>11)/float64(uint64(1)<<53)*2 - 1
}

// vnoise is smooth value noise in [-1,1] (interpolated hash), for wind/water.
func vnoise(x float64) float64 {
	xi := math.Floor(x)
	fr := x - xi
	a, b := hashf(int64(xi)), hashf(int64(xi)+1)
	fr = fr * fr * (3 - 2*fr) // smoothstep
	return a + (b-a)*fr
}

// AmbientFrame synthesises n mono 16-bit samples of the soundscape starting at time
// t (seconds), deterministically — so consecutive frames join seamlessly (every
// term is a function of absolute time).
func AmbientFrame(f AmbientFeatures, rate int, t float64, n int) []int16 {
	if rate <= 0 {
		rate = 16000
	}
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		ti := t + float64(i)/float64(rate)
		var s float64
		s += (vnoise(ti*5)*0.6 + vnoise(ti*13)*0.3) * f.Wind * 0.35    // wind
		s += (vnoise(ti*45)*0.5 + vnoise(ti*110)*0.3) * f.Water * 0.30 // water babble
		s += (vnoise(ti*80) * vnoise(ti*9)) * f.Forest * 0.18          // leafy rustle
		s += math.Sin(2*math.Pi*70*ti) * f.Hum * 0.16                  // low hum
		if f.Night > 0 {                                               // crickets
			if 0.5+0.5*math.Sin(2*math.Pi*7*ti) > 0.7 {
				s += math.Sin(2*math.Pi*4200*ti) * f.Night * 0.05
			}
		}
		if f.Birds > 0 && vnoise(ti*0.7) > 0.6 { // occasional chirp (frequency sweep)
			sweep := 2000 + 1400*math.Sin(2*math.Pi*8*ti)
			s += math.Sin(2*math.Pi*sweep*ti) * f.Birds * 0.07
		}
		if s > 1 {
			s = 1
		} else if s < -1 {
			s = -1
		}
		out[i] = int16(s * 32767)
	}
	return out
}

// Ambient derives the soundscape levels from the world's content and time of day.
func (w *World) Ambient() AmbientFeatures {
	night := 0.0
	if w.timeSet {
		a := 2 * math.Pi * (w.Time - 0.25)
		h := math.Sin(a)        // sun altitude
		day := (h + 0.15) / 0.3 // 0 night .. 1 day
		night = clampf(1-day, 0, 1)
	}
	forest := clampf(float64(w.sndForest)/3, 0, 1)
	birds := 0.0
	if w.sndBirds {
		birds = 1 - night // birds sing by day
	}
	hum := 0.0
	if w.sndHum {
		hum = 1
	}
	water := 0.0
	if w.sndWater {
		water = 1
	}
	return AmbientFeatures{
		Wind:   0.30 + 0.30*night, // wind picks up at dusk/night
		Water:  water,
		Forest: forest,
		Birds:  birds,
		Hum:    hum,
		Night:  night,
	}
}
