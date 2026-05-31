package reactive

import (
	"math"
	"testing"
)

// tone generates n int16 samples of a sine wave at freqHz and given amplitude
// (0..1 of full scale).
func tone(n, sampleRate int, freqHz, amp float64) []int16 {
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		v := amp * math.Sin(2*math.Pi*freqHz*float64(i)/float64(sampleRate))
		out[i] = int16(v * 32767)
	}
	return out
}

func TestExtractEmpty(t *testing.T) {
	f := Extract(nil, 16000)
	if f != (Features{}) {
		t.Errorf("empty input should give zero features, got %+v", f)
	}
	if Extract([]int16{1, 2, 3}, 0) != (Features{}) {
		t.Error("zero sample rate should give zero features")
	}
}

func TestSilenceHasLowLevel(t *testing.T) {
	f := Extract(make([]int16, 1024), 16000)
	if f.Level > 0.01 {
		t.Errorf("silence level = %.3f, want ~0", f.Level)
	}
}

func TestLouderMeansHigherLevel(t *testing.T) {
	quiet := Extract(tone(1024, 16000, 440, 0.1), 16000)
	loud := Extract(tone(1024, 16000, 440, 0.9), 16000)
	if loud.Level <= quiet.Level {
		t.Errorf("louder tone should have higher level: quiet=%.3f loud=%.3f", quiet.Level, loud.Level)
	}
}

func TestBassToneLandsInBassBand(t *testing.T) {
	// A 100 Hz tone should put most energy in Bass, not Treble.
	f := Extract(tone(2048, 16000, 100, 0.8), 16000)
	if f.Bass <= f.Treble {
		t.Errorf("100Hz tone: bass (%.3f) should exceed treble (%.3f)", f.Bass, f.Treble)
	}
}

func TestTrebleToneLandsInTrebleBand(t *testing.T) {
	// A 6 kHz tone should put most energy in Treble, not Bass.
	f := Extract(tone(2048, 16000, 6000, 0.8), 16000)
	if f.Treble <= f.Bass {
		t.Errorf("6kHz tone: treble (%.3f) should exceed bass (%.3f)", f.Treble, f.Bass)
	}
}

func TestMidToneLandsInMidBand(t *testing.T) {
	// A 1 kHz tone should land predominantly in Mid.
	f := Extract(tone(2048, 16000, 1000, 0.8), 16000)
	if f.Mid <= f.Bass || f.Mid <= f.Treble {
		t.Errorf("1kHz tone should be mid-dominant: %+v", f)
	}
}

func TestFeaturesAreNormalized(t *testing.T) {
	// Even a full-scale tone must keep features within [0,1].
	f := Extract(tone(2048, 16000, 440, 1.0), 16000)
	for _, v := range []float64{f.Level, f.Bass, f.Mid, f.Treble} {
		if v < 0 || v > 1 {
			t.Errorf("feature out of range: %+v", f)
		}
	}
}

func TestNextPow2(t *testing.T) {
	cases := map[int]int{1: 1, 2: 2, 3: 4, 5: 8, 1000: 1024, 1024: 1024}
	for in, want := range cases {
		if got := nextPow2(in); got != want {
			t.Errorf("nextPow2(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestFFTMatchesDFTForImpulse(t *testing.T) {
	// FFT of a unit impulse is all-ones spectrum — a basic correctness check.
	x := make([]float64, 8)
	x[0] = 1
	spec := fftReal(x)
	for i, c := range spec {
		if math.Abs(real(c)-1) > 1e-9 || math.Abs(imag(c)) > 1e-9 {
			t.Errorf("bin %d = %v, want 1+0i", i, c)
		}
	}
}
