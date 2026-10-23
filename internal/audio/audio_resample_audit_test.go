package audio

import (
	"fmt"
	"math"
	"testing"
)

// Invalid (zero or negative) or absurd sample rates must not panic in
// makeslice; the samples are returned unchanged, as for equal rates.
func TestConvertSampleRateAuditInvalidRates(t *testing.T) {
	samples := []int16{1, 2, 3, 4, 5, 6, 7, 8}
	cases := []struct{ from, to int }{
		{0, 16000},
		{-8000, 16000},
		{16000, 0},
		{16000, -1},
		{0, 1},
		{1, math.MaxInt},
	}
	for _, c := range cases {
		for name, fn := range map[string]func([]int16, int, int) []int16{
			"ConvertSampleRate": ConvertSampleRate,
			"Resample":          Resample,
		} {
			t.Run(fmt.Sprintf("%s/%d->%d", name, c.from, c.to), func(t *testing.T) {
				var out []int16
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Fatalf("%s(%d -> %d) panicked: %v", name, c.from, c.to, r)
						}
					}()
					out = fn(samples, c.from, c.to)
				}()
				if len(out) != len(samples) {
					t.Fatalf("%s(%d -> %d) returned %d samples, want the %d input samples unchanged",
						name, c.from, c.to, len(out), len(samples))
				}
				for i := range out {
					if out[i] != samples[i] {
						t.Fatalf("sample %d = %d, want %d", i, out[i], samples[i])
					}
				}
			})
		}
	}

	// Valid conversions keep working.
	if got := len(ConvertSampleRate(make([]int16, 160), 8000, 16000)); got != 320 {
		t.Fatalf("8 kHz -> 16 kHz of 160 samples gave %d samples, want 320", got)
	}
	if got := len(ConvertSampleRate(make([]int16, 480), 48000, 16000)); got != 160 {
		t.Fatalf("48 kHz -> 16 kHz of 480 samples gave %d samples, want 160", got)
	}
}
