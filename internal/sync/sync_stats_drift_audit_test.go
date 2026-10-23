package sync

import (
	"testing"
	"time"
)

// DriftEstimate is the change in clock offset per second. Dividing one
// Duration by another gave a dimensionless ratio (0 for any offset smaller
// than the elapsed time), so the estimate was always 0.
func TestAVSync_DriftEstimateIsOffsetChangePerSecond(t *testing.T) {
	avs := NewAVSynchronizer()

	avs.UpdateClockOffset(10 * time.Millisecond)
	avs.mu.Lock()
	avs.lastDriftCalc = clock().Add(-20 * time.Second)
	avs.mu.Unlock()
	avs.UpdateClockOffset(210 * time.Millisecond) // +200ms over 20s

	got := avs.GetStatistics().DriftEstimate
	want := 10 * time.Millisecond
	if got < want*99/100 || got > want*101/100 {
		t.Errorf("DriftEstimate = %v, want about %v per second", got, want)
	}
}

// Reset forgets the drift reference so the next estimate does not mix offsets
// from before and after the reset.
func TestAVSync_DriftEstimateAfterReset(t *testing.T) {
	avs := NewAVSynchronizer()

	avs.UpdateClockOffset(time.Second)
	avs.Reset()
	avs.UpdateClockOffset(50 * time.Millisecond)
	avs.mu.Lock()
	avs.lastDriftCalc = clock().Add(-20 * time.Second)
	avs.mu.Unlock()
	avs.UpdateClockOffset(50 * time.Millisecond)

	if got := avs.GetStatistics().DriftEstimate; got != 0 {
		t.Errorf("DriftEstimate = %v, want 0 for a constant offset", got)
	}
}
