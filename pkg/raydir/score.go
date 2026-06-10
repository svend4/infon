package raydir

import (
	"math"
	"math/rand"

	"github.com/svend4/infon/pkg/sona"
)

// score.go gives the world a melody, not just an ambient drone. A small generative
// composer reads the world — day or night, how lively it is, your mood — and picks
// a key and tempo to match: major and quick by day, minor and slow at night,
// pentatonic and unhurried when you're calm. Over a soft bass drone it walks a
// melody through the scale (a biased random walk over scale degrees), each note a
// short plucked tone with harmonics and an envelope. Deterministic from a seed, so
// it's reproducible and testable, and rendered to PCM/WAV for playback.

// Scales as semitone offsets within an octave.
var (
	ScaleMajor   = []int{0, 2, 4, 5, 7, 9, 11}
	ScaleMinor   = []int{0, 2, 3, 5, 7, 8, 10}
	ScalePentMaj = []int{0, 2, 4, 7, 9}
	ScalePentMin = []int{0, 3, 5, 7, 10}
)

// ScoreParams is a piece's key, scale, tempo and timbre.
type ScoreParams struct {
	Root   float64 // root frequency in Hz
	Scale  []int   // semitone offsets within an octave
	BPM    int     // tempo (one melody note per beat)
	Bright float64 // 0..1 timbre brightness (added harmonics)
}

// ScoreForNight derives a score from how dark it is (night 0..1), how lively the
// world is (0..1) and the walker's mood ("calm"/"restless"/"curious"/"").
func ScoreForNight(night, lively float64, mood string) ScoreParams {
	p := ScoreParams{Root: 261.63, Scale: ScaleMajor, BPM: 84, Bright: 0.7} // C4 major, day
	if night > 0.5 {                                                        // night: lower, minor, darker, slower
		p.Root, p.Scale, p.Bright, p.BPM = 196.0, ScaleMinor, 0.3, 60 // G3 minor
	}
	switch mood {
	case "calm": // pentatonic and unhurried — nothing dissonant
		if night > 0.5 {
			p.Scale = ScalePentMin
		} else {
			p.Scale = ScalePentMaj
		}
		p.BPM -= 12
	case "restless": // press on: quicker and brighter
		p.BPM += 16
		p.Bright += 0.1
	case "curious":
		p.Bright += 0.05
	}
	p.BPM += int(lively * 20) // a busier world plays a touch faster
	if p.BPM < 48 {
		p.BPM = 48
	} else if p.BPM > 140 {
		p.BPM = 140
	}
	p.Bright = clampf(p.Bright, 0, 1)
	return p
}

// Score derives a score from the world's current state (time, content, mood).
func (w *World) Score() ScoreParams {
	a := w.Ambient()
	lively := clampf(float64(w.sndForest)/3+b2f(w.sndWater)+b2f(w.sndBirds), 0, 1)
	return ScoreForNight(a.Night, lively, w.MoodName())
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// degreeFreq maps a scale degree (which can span several octaves) to a frequency.
func degreeFreq(p ScoreParams, degree int) float64 {
	scale := p.Scale
	if len(scale) == 0 {
		scale = ScaleMajor
	}
	if degree < 0 {
		degree = 0
	}
	oct := degree / len(scale)
	semi := scale[degree%len(scale)] + 12*oct
	return p.Root * math.Pow(2, float64(semi)/12)
}

// NoteFreqs is the melody as a list of note frequencies: a biased random walk over
// the scale degrees (deterministic from the seed). It is the source of truth the
// PCM renderer plays, so a test of these notes is a test of the music.
func NoteFreqs(p ScoreParams, count int, seed int64) []float64 {
	scale := p.Scale
	if len(scale) == 0 {
		scale = ScaleMajor
	}
	rng := rand.New(rand.NewSource(seed))
	deg, hi := 0, len(scale)*2 // wander within ~two octaves
	out := make([]float64, count)
	for k := 0; k < count; k++ {
		out[k] = degreeFreq(p, deg)
		deg += rng.Intn(5) - 2 // step in [-2,+2]
		if deg < 0 {
			deg = -deg // reflect off the bottom rather than sticking
		}
		if deg > hi {
			deg = hi - (deg - hi)
		}
	}
	return out
}

// noteEnv is a short pluck envelope (fast attack, gentle release) over [0,dur].
func noteEnv(t, dur float64) float64 {
	const a, r = 0.008, 0.12
	switch {
	case t < a:
		return t / a
	case t > dur-r:
		return math.Max(0, (dur-t)/r)
	default:
		return 1
	}
}

// MelodyPCM renders `seconds` of the score to mono 16-bit PCM at `rate`,
// deterministically from the seed.
func MelodyPCM(p ScoreParams, rate int, seconds float64, seed int64) []int16 {
	if rate <= 0 {
		rate = 16000
	}
	if p.BPM <= 0 {
		p.BPM = 72
	}
	n := int(seconds * float64(rate))
	buf := make([]float64, n)
	// a soft bass drone an octave below the root.
	for i := 0; i < n; i++ {
		t := float64(i) / float64(rate)
		buf[i] += math.Sin(2*math.Pi*(p.Root/2)*t) * 0.12
	}
	// the melody: one note per beat.
	noteLen := 60.0 / float64(p.BPM)
	count := int(seconds/noteLen) + 1
	for k, f := range NoteFreqs(p, count, seed) {
		s0 := int(float64(k) * noteLen * float64(rate))
		ns := int(noteLen * 0.92 * float64(rate))
		for i := 0; i < ns; i++ {
			idx := s0 + i
			if idx < 0 || idx >= n {
				continue
			}
			t := float64(i) / float64(rate)
			v := math.Sin(2 * math.Pi * f * t)
			v += p.Bright * 0.5 * math.Sin(2*math.Pi*2*f*t)
			v += p.Bright * 0.25 * math.Sin(2*math.Pi*3*f*t)
			buf[idx] += v * noteEnv(t, noteLen*0.92) * 0.22
		}
	}
	out := make([]int16, n)
	for i, v := range buf {
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		out[i] = int16(v * 32767)
	}
	return out
}

// ScoreWAV renders the score to a WAV file (bytes).
func ScoreWAV(p ScoreParams, rate int, seconds float64, seed int64) []byte {
	return sona.WAV(MelodyPCM(p, rate, seconds, seed), rate)
}
