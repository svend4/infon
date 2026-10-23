package quality

import "testing"

// Before any sample, the documented 0-100 quality percentage must not go
// negative and MinRTT must not leak the internal MaxInt64 sentinel.
func TestQualityMiscAudit_NoSamplesYet(t *testing.T) {
	check := func(t *testing.T, qm *QualityMonitor) {
		t.Helper()
		if p := qm.GetQualityPercent(); p < 0 || p > 100 {
			t.Errorf("GetQualityPercent() = %g before any sample, want within 0..100", p)
		}
		if m := qm.GetMetrics(); m.MinRTT != 0 {
			t.Errorf("MinRTT = %v before any RTT sample, want 0", m.MinRTT)
		}
	}

	qm := NewQualityMonitor()
	t.Run("new", func(t *testing.T) { check(t, qm) })

	qm.UpdatePacketLoss(100, 100)
	qm.Reset()
	t.Run("reset", func(t *testing.T) { check(t, qm) })
}
