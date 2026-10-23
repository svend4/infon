package audio

import (
	"math"
	"testing"
)

// auditERLE returns the echo return loss enhancement in dB.
func auditERLE(inEnergy, outEnergy float64) float64 {
	return 10 * math.Log10(inEnergy/math.Max(outEnergy, 1e-12))
}

// The documented flow feeds the speaker audio with ProcessRender and then
// processes the microphone frame; the render history must be used as the
// echo reference.
func TestAEC3AuditProcessRenderFeedsCapture(t *testing.T) {
	const frameLen = 160
	aec := NewAEC3(16000, 1)
	seed := uint32(13)

	var inEnergy, outEnergy float64
	for k := 0; k < 100; k++ {
		render := auditNoiseFrame(&seed, frameLen, 8000)
		aec.ProcessRender(render)
		capture := make([]int16, frameLen)
		for i := range capture {
			capture[i] = render[i] / 2 // echo path gain 0.5, no near-end speech
		}
		out, _ := aec.ProcessCapture(capture, nil)
		if k >= 90 {
			for i := range out {
				inEnergy += float64(capture[i]) * float64(capture[i])
				outEnergy += float64(out[i]) * float64(out[i])
			}
		}
	}
	if erle := auditERLE(inEnergy, outEnergy); erle < 20 {
		t.Fatalf("echo fed through ProcessRender not cancelled: ERLE = %.1f dB, want >= 20 dB", erle)
	}
}

// An echo delayed by 40 samples reaches back into the previous 160-sample
// render frame for the first 40 samples of every capture frame; the filter
// must see that history to cancel it.
func TestAEC3AuditEchoAcrossFrameBoundary(t *testing.T) {
	const (
		frameLen = 160
		delay    = 40
	)
	aec := NewAEC3(16000, 1)
	seed := uint32(17)
	prev := make([]int16, frameLen)

	var inEnergy, outEnergy float64
	for k := 0; k < 100; k++ {
		render := auditNoiseFrame(&seed, frameLen, 8000)
		capture := make([]int16, frameLen)
		for i := range capture {
			if i >= delay {
				capture[i] = render[i-delay] / 2
			} else {
				capture[i] = prev[frameLen+i-delay] / 2
			}
		}
		out, _ := aec.ProcessCapture(capture, render)
		if k >= 90 {
			for i := range out {
				inEnergy += float64(capture[i]) * float64(capture[i])
				outEnergy += float64(out[i]) * float64(out[i])
			}
		}
		prev = render
	}
	if erle := auditERLE(inEnergy, outEnergy); erle < 20 {
		t.Fatalf("delayed echo not cancelled: ERLE = %.1f dB, want >= 20 dB", erle)
	}
}
