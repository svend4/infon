package quality

import "testing"

// Receiving more packets than were sent (duplicates, retransmissions, or the
// receiver counting before the sender) must not wrap the unsigned loss counter
// to ~2^64 and pin the call to Poor.
func TestQualityMiscAudit_UpdatePacketLossReceivedExceedsSent(t *testing.T) {
	qm := NewQualityMonitor()
	qm.UpdatePacketLoss(100, 101)

	m := qm.GetMetrics()
	if m.PacketsLost != 0 {
		t.Errorf("PacketsLost = %d, want 0 when received > sent", m.PacketsLost)
	}
	if m.LossRate != 0 {
		t.Errorf("LossRate = %g%%, want 0 when received > sent", m.LossRate)
	}
	if m.Quality == QualityPoor {
		t.Errorf("Quality = %s with no loss and no delay, want better than poor", m.Quality)
	}
}

func TestQualityMiscAudit_RecordPacketReceivedDuplicate(t *testing.T) {
	qm := NewQualityMonitor()
	qm.RecordPacketSent()
	qm.RecordPacketReceived()
	qm.RecordPacketReceived() // duplicate delivery

	m := qm.GetMetrics()
	if m.PacketsLost != 0 {
		t.Errorf("PacketsLost = %d, want 0 after a duplicate delivery", m.PacketsLost)
	}
	if m.LossRate < 0 || m.LossRate > 100 {
		t.Errorf("LossRate = %g%%, want within 0..100", m.LossRate)
	}
	if m.Quality == QualityPoor {
		t.Errorf("Quality = %s after a duplicate delivery, want better than poor", m.Quality)
	}
}

// Packets that are sent but never received are lost; the loss rate must reflect
// that without waiting for a RecordPacketReceived call.
func TestQualityMiscAudit_RecordPacketSentCountsUnreceived(t *testing.T) {
	qm := NewQualityMonitor()
	for i := 0; i < 10; i++ {
		qm.RecordPacketSent()
	}

	m := qm.GetMetrics()
	if m.PacketsLost != 10 {
		t.Errorf("PacketsLost = %d, want 10 (10 sent, 0 received)", m.PacketsLost)
	}
	if m.LossRate != 100 {
		t.Errorf("LossRate = %g%%, want 100%% (10 sent, 0 received)", m.LossRate)
	}
	if m.Quality == QualityExcellent {
		t.Errorf("Quality = %s with 100%% loss, want not excellent", m.Quality)
	}
}
