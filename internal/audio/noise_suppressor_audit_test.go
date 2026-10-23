package audio

import (
	"math"
	"testing"
)

// auditNoiseFrame returns a deterministic pseudo-random frame with samples
// uniformly spread over [-amp, amp].
func auditNoiseFrame(seed *uint32, n int, amp float64) []int16 {
	frame := make([]int16, n)
	for i := range frame {
		*seed = *seed*1664525 + 1013904223
		u := float64(*seed)/float64(math.MaxUint32)*2 - 1
		frame[i] = int16(u * amp)
	}
	return frame
}

// auditSineFrame returns frame number k of a continuous sine wave.
func auditSineFrame(k, n, sampleRate int, freq, amp float64) []int16 {
	frame := make([]int16, n)
	for i := range frame {
		t := float64(k*n+i) / float64(sampleRate)
		frame[i] = int16(amp * math.Sin(2*math.Pi*freq*t))
	}
	return frame
}

func auditRMS(frame []int16) float64 {
	var sum float64
	for _, s := range frame {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(frame)))
}

// Sustained speech must not be absorbed into the noise floor and attenuated,
// while the background noise between speech must still be suppressed.
func TestNoiseSuppressorAuditSustainedSpeechNotSuppressed(t *testing.T) {
	const (
		sampleRate = 16000
		frameLen   = 320 // 20 ms, as used by cmd/tvcp/call.go
	)
	ns := NewNoiseSuppressor(sampleRate, 1)
	seed := uint32(1)

	// 2 s of quiet background noise.
	var lastNoiseReduced bool
	for k := 0; k < 100; k++ {
		_, lastNoiseReduced = ns.Process(auditNoiseFrame(&seed, frameLen, 40))
	}
	if !lastNoiseReduced {
		t.Fatalf("steady background noise is not suppressed after 2 s")
	}

	// 3 s of continuous speech-level signal.
	for k := 0; k < 150; k++ {
		in := auditSineFrame(k, frameLen, sampleRate, 220, 8000)
		out, reduced := ns.Process(in)
		if reduced {
			t.Fatalf("speech frame %d (%.2f s) classified as noise", k, float64(k*frameLen)/sampleRate)
		}
		if ratio := auditRMS(out) / auditRMS(in); ratio < 0.99 {
			t.Fatalf("speech frame %d attenuated: output/input RMS = %.3f", k, ratio)
		}
	}

	// Background noise after the speech must again be suppressed.
	for k := 0; k < 50; k++ {
		in := auditNoiseFrame(&seed, frameLen, 40)
		out, reduced := ns.Process(in)
		if k >= 5 {
			if !reduced {
				t.Fatalf("noise frame %d after speech not suppressed", k)
			}
			if auditRMS(out) >= auditRMS(in) {
				t.Fatalf("noise frame %d after speech not attenuated", k)
			}
		}
	}
}
