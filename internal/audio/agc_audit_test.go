package audio

import (
	"math"
	"testing"
)

func auditConstFrame(n int, v int16) []int16 {
	frame := make([]int16, n)
	for i := range frame {
		frame[i] = v
	}
	return frame
}

// attackTime (1 ms) is far shorter than a 10 ms frame, so a sudden loud
// signal must be tamed within the first loud frame instead of being driven
// into hard clipping by the gain built up on the quiet signal.
func TestAGCAuditFastAttack(t *testing.T) {
	agc := NewAutomaticGainControl(16000, 1)

	// Settle on a quiet signal for 1 s (100 x 10 ms).
	for k := 0; k < 100; k++ {
		agc.Process(auditConstFrame(160, 1000))
	}

	for k := 0; k < 10; k++ {
		out := agc.Process(auditConstFrame(160, 30000))
		for i, s := range out {
			if s >= math.MaxInt16 || s <= math.MinInt16+1 {
				t.Fatalf("loud frame %d sample %d clipped to %d (gain %.2f)", k, i, s, agc.currentGain)
			}
		}
	}
}

// releaseTime is 100 ms: after exactly one time constant of quiet signal the
// tracked peak must have decayed by 1-1/e of the step, independently of the
// frame size used by the caller.
func TestAGCAuditReleaseTimeConstant(t *testing.T) {
	const (
		loud  = 30000.0
		quiet = 1000.0
	)
	wantPeak := quiet + (loud-quiet)*math.Exp(-1)
	wantGain := 16384.0 / wantPeak

	for _, frameLen := range []int{160, 320} { // 10 ms and 20 ms at 16 kHz
		agc := NewAutomaticGainControl(16000, 1)

		// Settle on a loud signal for 1 s.
		for k := 0; k < 16000/frameLen; k++ {
			agc.Process(auditConstFrame(frameLen, int16(loud)))
		}
		if g := agc.currentGain; math.Abs(g-16384.0/loud)/(16384.0/loud) > 0.01 {
			t.Fatalf("frame %d: settled gain %.3f, want %.3f", frameLen, g, 16384.0/loud)
		}

		// 100 ms of quiet signal.
		for k := 0; k < 1600/frameLen; k++ {
			agc.Process(auditConstFrame(frameLen, int16(quiet)))
		}
		if g := agc.currentGain; math.Abs(g-wantGain)/wantGain > 0.05 {
			t.Errorf("frame %d: gain after one release time constant = %.3f, want %.3f",
				frameLen, g, wantGain)
		}
	}
}
