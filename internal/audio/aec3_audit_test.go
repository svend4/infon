package audio

import (
	"math"
	"testing"
)

// With a pure linear echo of a speech-level reference, the AEC3 adaptive
// filter must stay finite and converge, removing most of the echo instead of
// diverging to Inf/NaN and producing full-scale garbage.
func TestAEC3AuditLMSStableAndConverges(t *testing.T) {
	const frameLen = 160
	aec := NewAEC3(16000, 1)
	seed := uint32(7)

	var inEnergy, outEnergy float64
	for k := 0; k < 100; k++ {
		render := auditNoiseFrame(&seed, frameLen, 8000)
		capture := make([]int16, frameLen)
		for i := range capture {
			capture[i] = render[i] / 2 // echo path gain 0.5, no near-end speech
		}

		out, _ := aec.ProcessCapture(capture, render)
		if len(out) != len(capture) {
			t.Fatalf("frame %d: output length %d, want %d", k, len(out), len(capture))
		}

		aec.mu.RLock()
		for j, c := range aec.filterCoeffs {
			if math.IsNaN(c) || math.IsInf(c, 0) {
				aec.mu.RUnlock()
				t.Fatalf("frame %d: filter coefficient %d diverged to %v", k, j, c)
			}
		}
		aec.mu.RUnlock()

		if k >= 90 {
			for i := range out {
				inEnergy += float64(capture[i]) * float64(capture[i])
				outEnergy += float64(out[i]) * float64(out[i])
			}
		}
	}

	// Require at least 20 dB of echo return loss enhancement once converged.
	if outEnergy > inEnergy/100 {
		t.Fatalf("echo not cancelled: ERLE = %.1f dB, want >= 20 dB",
			10*math.Log10(inEnergy/math.Max(outEnergy, 1e-12)))
	}
}
