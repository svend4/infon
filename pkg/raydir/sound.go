package raydir

import (
	"math"

	"github.com/svend4/infon/pkg/sona"
)

// sound.go gives the Q6 world a VOICE. It maps a world's six axes to music, so every
// hexagram has its own piece, renders the Gray grand tour as one evolving track, and
// sonifies a living world's population so you can HEAR the boom and the crash. It
// reuses the generative composer (ScoreForVector -> the score.go renderer).

// ScoreForVector maps a Q6 world to a score: the sun sets the register (high and
// bright by day, low at night), warmth the major/minor colour, density the tempo, glow
// the timbre brightness, fog a muffling that slows and darkens, and object scale drops
// a grand world an octave.
func ScoreForVector(v SceneVector) ScoreParams {
	fog, warm, dens, sun, scl, glow := v[AxFog], v[AxWarm], v[AxDensity], v[AxSun], v[AxScale], v[AxGlow]
	root := 196.0 + sun*98.0 // ~G3 (196) at night .. ~D4 (294) by day
	if scl >= 0.5 {
		root /= 2 // large forms read as grander — an octave lower
	}
	var sc []int
	switch {
	case warm >= 0.5 && dens < 0.5:
		sc = ScalePentMaj // warm and sparse: open, consonant
	case warm >= 0.5:
		sc = ScaleMajor
	case dens < 0.5:
		sc = ScalePentMin // cool and sparse
	default:
		sc = ScaleMinor
	}
	bpm := int(60 + dens*60 - fog*24)
	if bpm < 48 {
		bpm = 48
	} else if bpm > 140 {
		bpm = 140
	}
	bright := clampf(0.2+glow*0.7-fog*0.3, 0, 1)
	return ScoreParams{Root: root, Scale: sc, BPM: bpm, Bright: bright}
}

// TourPCM renders a walk of hexagrams as one evolving piece: a phrase per world, each
// scored by ScoreForVector — the sound of the grand tour through every world.
func TourPCM(walk []Hexagram, rate int, secPerWorld float64, seed int64) []int16 {
	var out []int16
	for i, h := range walk {
		p := ScoreForVector(VectorFromHexagram(h))
		out = append(out, MelodyPCM(p, rate, secPerWorld, seed+int64(i))...)
	}
	return out
}

// TourWAV is TourPCM as a WAV file (bytes).
func TourWAV(walk []Hexagram, rate int, secPerWorld float64, seed int64) []byte {
	return sona.WAV(TourPCM(walk, rate, secPerWorld, seed), rate)
}

// SonifySeries renders a numeric series as a tone whose pitch tracks each value
// (normalised to the series' own range) — data you can hear, like a population's boom
// and crash. Phase is continuous across points so pitch changes don't click.
func SonifySeries(series []float64, rate int, secPerPoint, baseHz, spanHz float64) []int16 {
	if len(series) == 0 || rate <= 0 {
		return nil
	}
	lo, hi := series[0], series[0]
	for _, x := range series {
		lo, hi = math.Min(lo, x), math.Max(hi, x)
	}
	rng := hi - lo
	if rng == 0 {
		rng = 1
	}
	perN := int(secPerPoint * float64(rate))
	if perN < 1 {
		perN = 1
	}
	out := make([]int16, 0, perN*len(series))
	phase := 0.0
	for _, x := range series {
		f := baseHz + (x-lo)/rng*spanHz
		for i := 0; i < perN; i++ {
			phase += 2 * math.Pi * f / float64(rate)
			out = append(out, int16(math.Sin(phase)*0.3*32767))
		}
	}
	return out
}

// SonifySeriesWAV is SonifySeries as a WAV file (bytes).
func SonifySeriesWAV(series []float64, rate int, secPerPoint, baseHz, spanHz float64) []byte {
	return sona.WAV(SonifySeries(series, rate, secPerPoint, baseHz, spanHz), rate)
}
