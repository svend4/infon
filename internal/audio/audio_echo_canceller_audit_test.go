package audio

import (
	"fmt"
	"testing"
)

// The far-end (reference) frame is routinely shorter than the capture frame
// or missing (no playback yet, jitter-buffer underrun); Process must treat the
// missing reference as silence instead of panicking.
func TestEchoCancellerAuditShortReference(t *testing.T) {
	seed := uint32(5)
	input := auditNoiseFrame(&seed, 320, 4000)
	for _, refLen := range []int{0, 1, 160, 319} {
		t.Run(fmt.Sprintf("ref%d", refLen), func(t *testing.T) {
			ec := NewEchoCanceller(16000, 320)
			var ref []int16
			if refLen > 0 {
				ref = auditNoiseFrame(&seed, refLen, 4000)
			}
			var out []int16
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("Process(input[320], reference[%d]) panicked: %v", refLen, r)
					}
				}()
				out = ec.Process(input, ref)
			}()
			if len(out) != len(input) {
				t.Fatalf("output length %d, want %d", len(out), len(input))
			}
			if refLen == 0 {
				for i := range out {
					if out[i] != input[i] {
						t.Fatalf("sample %d = %d with no reference, want input %d", i, out[i], input[i])
					}
				}
			}
		})
	}
}

// Near-end speech while the far end is nearly silent (a single LSB of
// dither) must not drive the NLMS weights to the clamp: the next far-end
// frame would otherwise come out of the canceller as full-scale noise.
func TestEchoCancellerAuditNearSilentReferenceStable(t *testing.T) {
	const (
		sampleRate = 16000
		frameLen   = 320
	)
	ec := NewEchoCanceller(sampleRate, frameLen)
	seed := uint32(9)

	// 0.5 s of near-end speech over +-1 LSB far-end dither.
	for k := 0; k < 25; k++ {
		ref := auditNoiseFrame(&seed, frameLen, 1.999)
		ec.Process(auditSineFrame(k, frameLen, sampleRate, 1000, 10000), ref)
	}

	// Far-end speech with a linear echo and a silent near end.
	ref := auditNoiseFrame(&seed, frameLen, 3000)
	input := make([]int16, frameLen)
	for i := range input {
		input[i] = ref[i] * 3 / 10
	}
	out := ec.Process(input, ref)
	if in, got := auditRMS(input), auditRMS(out); got > 1.5*in {
		t.Fatalf("echo canceller amplified the echo after near-end speech: output RMS %.0f, input RMS %.0f", got, in)
	}
}
