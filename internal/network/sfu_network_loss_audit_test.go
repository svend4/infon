package network

import "testing"

// Early in a stream expectedSeq is below maxWindow. A receive-buffer cleanup at
// that point must not wrap expectedSeq-maxWindow around and drop every entry,
// including out-of-order packets ahead of expectedSeq.
func TestLossDetectorCleanupEarlyInStreamKeepsFutureEntries(t *testing.T) {
	ld := NewLossDetector()

	// 0..2 arrive in order; 3 and 5 are lost; 4 and 6..110 arrive out of order.
	for seq := uint32(0); seq < 3; seq++ {
		ld.OnPacketReceived(seq)
	}
	ld.OnPacketReceived(4)
	for seq := uint32(6); seq <= 110; seq++ {
		ld.OnPacketReceived(seq)
	}

	// Retransmission of 3 is in order: expectedSeq moves to 5 and, with more than
	// maxWindow entries buffered, the receive buffer is cleaned up.
	if !ld.OnPacketReceived(3) {
		t.Fatal("retransmitted 3 reported as duplicate")
	}

	// 50 was already received, so a repeat is a duplicate.
	if ld.OnPacketReceived(50) {
		t.Error("repeat of already received seq 50 accepted as new")
	}

	// Retransmission of 5 fills the last gap: expectedSeq must skip 6..110.
	if !ld.OnPacketReceived(5) {
		t.Fatal("retransmitted 5 reported as duplicate")
	}
	ld.mu.Lock()
	exp := ld.expectedSeq
	ld.mu.Unlock()
	if exp != 111 {
		t.Errorf("expectedSeq = %d after filling every gap up to 110, want 111", exp)
	}
}
