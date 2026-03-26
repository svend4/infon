package quality

import (
	"testing"
	"time"
)

func TestNewQualityMonitor(t *testing.T) {
	qm := NewQualityMonitor()

	if qm == nil {
		t.Fatal("NewQualityMonitor() returned nil")
	}

	metrics := qm.GetMetrics()

	if metrics.Quality != QualityUnknown {
		t.Errorf("Initial quality = %s, expected unknown", metrics.Quality)
	}

	if metrics.MOS != 0 {
		t.Errorf("Initial MOS = %f, expected 0", metrics.MOS)
	}
}

func TestQualityMonitor_UpdateRTT(t *testing.T) {
	qm := NewQualityMonitor()

	qm.UpdateRTT(50 * time.Millisecond)

	metrics := qm.GetMetrics()

	if metrics.AvgRTT != 50*time.Millisecond {
		t.Errorf("AvgRTT = %v, expected 50ms", metrics.AvgRTT)
	}

	if metrics.MinRTT != 50*time.Millisecond {
		t.Errorf("MinRTT = %v, expected 50ms", metrics.MinRTT)
	}

	if metrics.MaxRTT != 50*time.Millisecond {
		t.Errorf("MaxRTT = %v, expected 50ms", metrics.MaxRTT)
	}
}

func TestQualityMonitor_MultipleRTT(t *testing.T) {
	qm := NewQualityMonitor()

	rtts := []time.Duration{
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}

	for _, rtt := range rtts {
		qm.UpdateRTT(rtt)
	}

	metrics := qm.GetMetrics()

	if metrics.MinRTT != 30*time.Millisecond {
		t.Errorf("MinRTT = %v, expected 30ms", metrics.MinRTT)
	}

	if metrics.MaxRTT != 50*time.Millisecond {
		t.Errorf("MaxRTT = %v, expected 50ms", metrics.MaxRTT)
	}

	expectedAvg := 40 * time.Millisecond
	if metrics.AvgRTT != expectedAvg {
		t.Errorf("AvgRTT = %v, expected %v", metrics.AvgRTT, expectedAvg)
	}

	// Jitter should be calculated
	if metrics.AvgJitter == 0 {
		t.Error("AvgJitter should be > 0 with varying RTT")
	}
}

func TestQualityMonitor_PacketLoss(t *testing.T) {
	qm := NewQualityMonitor()

	qm.UpdatePacketLoss(100, 95) // 5% loss

	metrics := qm.GetMetrics()

	if metrics.PacketsSent != 100 {
		t.Errorf("PacketsSent = %d, expected 100", metrics.PacketsSent)
	}

	if metrics.PacketsReceived != 95 {
		t.Errorf("PacketsReceived = %d, expected 95", metrics.PacketsReceived)
	}

	if metrics.PacketsLost != 5 {
		t.Errorf("PacketsLost = %d, expected 5", metrics.PacketsLost)
	}

	if metrics.LossRate != 5.0 {
		t.Errorf("LossRate = %.2f%%, expected 5.00%%", metrics.LossRate)
	}
}

func TestQualityMonitor_RecordPackets(t *testing.T) {
	qm := NewQualityMonitor()

	// Send 10 packets
	for i := 0; i < 10; i++ {
		qm.RecordPacketSent()
	}

	// Receive 9 packets (1 lost)
	for i := 0; i < 9; i++ {
		qm.RecordPacketReceived()
	}

	metrics := qm.GetMetrics()

	if metrics.PacketsSent != 10 {
		t.Errorf("PacketsSent = %d, expected 10", metrics.PacketsSent)
	}

	if metrics.PacketsReceived != 9 {
		t.Errorf("PacketsReceived = %d, expected 9", metrics.PacketsReceived)
	}

	if metrics.PacketsLost != 1 {
		t.Errorf("PacketsLost = %d, expected 1", metrics.PacketsLost)
	}

	if metrics.LossRate != 10.0 {
		t.Errorf("LossRate = %.2f%%, expected 10.00%%", metrics.LossRate)
	}
}

func TestQualityMonitor_MOS_Excellent(t *testing.T) {
	qm := NewQualityMonitor()

	// Perfect conditions
	qm.UpdateRTT(20 * time.Millisecond)
	qm.UpdatePacketLoss(1000, 1000) // 0% loss

	metrics := qm.GetMetrics()

	if metrics.Quality != QualityExcellent {
		t.Errorf("Quality = %s, expected excellent", metrics.Quality)
	}

	if metrics.MOS < 4.0 {
		t.Errorf("MOS = %.2f, expected >= 4.0", metrics.MOS)
	}
}

func TestQualityMonitor_MOS_Poor(t *testing.T) {
	qm := NewQualityMonitor()

	// Bad conditions
	qm.UpdateRTT(500 * time.Millisecond) // High latency
	qm.UpdatePacketLoss(100, 80)         // 20% loss

	metrics := qm.GetMetrics()

	if metrics.Quality != QualityPoor {
		t.Errorf("Quality = %s, expected poor", metrics.Quality)
	}

	if metrics.MOS >= 2.5 {
		t.Errorf("MOS = %.2f, expected < 2.5", metrics.MOS)
	}
}

func TestQualityMonitor_Jitter(t *testing.T) {
	qm := NewQualityMonitor()

	// Varying RTT to generate jitter
	qm.UpdateRTT(30 * time.Millisecond)
	qm.UpdateRTT(50 * time.Millisecond) // +20ms jitter
	qm.UpdateRTT(35 * time.Millisecond) // -15ms jitter

	metrics := qm.GetMetrics()

	if metrics.AvgJitter == 0 {
		t.Error("AvgJitter should be > 0")
	}

	if metrics.MaxJitter == 0 {
		t.Error("MaxJitter should be > 0")
	}

	// Max jitter should be 20ms
	if metrics.MaxJitter != 20*time.Millisecond {
		t.Errorf("MaxJitter = %v, expected 20ms", metrics.MaxJitter)
	}
}

func TestQualityMonitor_GetSamples(t *testing.T) {
	qm := NewQualityMonitor()

	qm.UpdateRTT(30 * time.Millisecond)
	qm.UpdateRTT(40 * time.Millisecond)
	qm.UpdateRTT(50 * time.Millisecond)

	rttSamples := qm.GetRTTSamples()

	if len(rttSamples) != 3 {
		t.Errorf("RTT samples count = %d, expected 3", len(rttSamples))
	}

	jitterSamples := qm.GetJitterSamples()

	if len(jitterSamples) != 2 {
		t.Errorf("Jitter samples count = %d, expected 2", len(jitterSamples))
	}
}

func TestQualityMonitor_Reset(t *testing.T) {
	qm := NewQualityMonitor()

	qm.UpdateRTT(50 * time.Millisecond)
	qm.UpdatePacketLoss(100, 90)

	qm.Reset()

	metrics := qm.GetMetrics()

	if metrics.AvgRTT != 0 {
		t.Errorf("After reset, AvgRTT = %v, expected 0", metrics.AvgRTT)
	}

	if metrics.PacketsSent != 0 {
		t.Errorf("After reset, PacketsSent = %d, expected 0", metrics.PacketsSent)
	}

	if metrics.Quality != QualityUnknown {
		t.Errorf("After reset, Quality = %s, expected unknown", metrics.Quality)
	}
}

func TestQualityMonitor_IsAcceptable(t *testing.T) {
	qm := NewQualityMonitor()

	// Good conditions
	qm.UpdateRTT(30 * time.Millisecond)
	qm.UpdatePacketLoss(1000, 999)

	if !qm.IsAcceptable() {
		t.Error("Quality should be acceptable")
	}

	// Bad conditions
	qm.Reset()
	qm.UpdateRTT(500 * time.Millisecond)
	qm.UpdatePacketLoss(100, 70)

	if qm.IsAcceptable() {
		t.Error("Quality should not be acceptable")
	}
}

func TestQualityMonitor_GetQualityPercent(t *testing.T) {
	qm := NewQualityMonitor()

	qm.UpdateRTT(30 * time.Millisecond)
	qm.UpdatePacketLoss(1000, 1000)

	percent := qm.GetQualityPercent()

	if percent < 0 || percent > 100 {
		t.Errorf("QualityPercent = %.2f, expected 0-100", percent)
	}

	// Good quality should give high percentage
	if percent < 50 {
		t.Errorf("QualityPercent = %.2f, expected >= 50 for good conditions", percent)
	}
}

func TestFormatMOS(t *testing.T) {
	tests := []struct {
		mos      float64
		contains string
	}{
		{4.5, "Excellent"},
		{4.0, "Excellent"},
		{3.8, "Good"},
		{3.0, "Fair"},
		{2.0, "Poor"},
	}

	for _, tt := range tests {
		result := FormatMOS(tt.mos)
		if !contains(result, tt.contains) {
			t.Errorf("FormatMOS(%.1f) = %s, should contain '%s'", tt.mos, result, tt.contains)
		}
	}
}

func TestFormatQuality(t *testing.T) {
	tests := []struct {
		quality  QualityLevel
		contains string
	}{
		{QualityExcellent, "Excellent"},
		{QualityGood, "Good"},
		{QualityFair, "Fair"},
		{QualityPoor, "Poor"},
		{QualityUnknown, "Unknown"},
	}

	for _, tt := range tests {
		result := FormatQuality(tt.quality)
		if !contains(result, tt.contains) {
			t.Errorf("FormatQuality(%s) = %s, should contain '%s'", tt.quality, result, tt.contains)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s[:len(substr)] == substr || (len(s) > len(substr) && contains(s[1:], substr)))
}

func TestQualityMonitor_MaxSamples(t *testing.T) {
	qm := NewQualityMonitor()
	qm.maxRTTSamples = 10

	// Add more than max
	for i := 0; i < 20; i++ {
		qm.UpdateRTT(time.Duration(i+30) * time.Millisecond)
	}

	samples := qm.GetRTTSamples()

	if len(samples) != 10 {
		t.Errorf("RTT samples = %d, expected 10 (should trim)", len(samples))
	}

	// Verify we kept the most recent
	if samples[len(samples)-1] != 49*time.Millisecond {
		t.Error("Should have kept the most recent samples")
	}
}

func BenchmarkQualityMonitor_UpdateRTT(b *testing.B) {
	qm := NewQualityMonitor()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qm.UpdateRTT(30 * time.Millisecond)
	}
}

func BenchmarkQualityMonitor_GetMetrics(b *testing.B) {
	qm := NewQualityMonitor()

	// Add some data
	for i := 0; i < 50; i++ {
		qm.UpdateRTT(time.Duration(30+i) * time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = qm.GetMetrics()
	}
}
