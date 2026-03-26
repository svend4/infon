package network

import (
	"testing"
	"time"
)

func TestNewNetworkSimulator(t *testing.T) {
	ns := NewNetworkSimulator()

	if ns == nil {
		t.Fatal("NewNetworkSimulator() returned nil")
	}

	if ns.enabled {
		t.Error("Simulator should start disabled")
	}
}

func TestNetworkSimulator_Enable(t *testing.T) {
	ns := NewNetworkSimulator()

	ns.Enable(Poor4G)

	if !ns.IsEnabled() {
		t.Error("Simulator should be enabled")
	}

	cond := ns.GetCondition()
	if cond != Poor4G {
		t.Error("Condition not set correctly")
	}
}

func TestNetworkSimulator_Disable(t *testing.T) {
	ns := NewNetworkSimulator()

	ns.Enable(Poor4G)
	ns.Disable()

	if ns.IsEnabled() {
		t.Error("Simulator should be disabled")
	}
}

func TestNetworkSimulator_PacketLoss(t *testing.T) {
	ns := NewNetworkSimulator()

	// 100% packet loss
	ns.Enable(NetworkCondition{
		PacketLoss: 100.0,
	})

	dropped := 0
	for i := 0; i < 100; i++ {
		shouldSend, _ := ns.ShouldSendPacket(1024)
		if !shouldSend {
			dropped++
		}
	}

	if dropped < 95 { // Allow some variance
		t.Errorf("Expected ~100 drops, got %d", dropped)
	}
}

func TestNetworkSimulator_Latency(t *testing.T) {
	ns := NewNetworkSimulator()

	ns.Enable(NetworkCondition{
		Latency: 100 * time.Millisecond,
		Jitter:  0,
	})

	shouldSend, delay := ns.ShouldSendPacket(1024)

	if !shouldSend {
		t.Error("Packet should be sent")
	}

	if delay < 90*time.Millisecond || delay > 110*time.Millisecond {
		t.Errorf("Delay = %v, expected ~100ms", delay)
	}
}

func TestNetworkSimulator_BandwidthLimit(t *testing.T) {
	ns := NewNetworkSimulator()

	// 1 KB/s limit
	ns.Enable(NetworkCondition{
		Bandwidth: 1, // 1 KB/s
	})

	// Send 2 KB packet - should fail
	shouldSend, _ := ns.ShouldSendPacket(2048)

	// Give it time to accumulate bandwidth
	time.Sleep(100 * time.Millisecond)

	// Send small packet - should work
	shouldSend2, _ := ns.ShouldSendPacket(100)

	if shouldSend {
		t.Error("Large packet should be dropped due to bandwidth limit")
	}

	if !shouldSend2 {
		t.Error("Small packet should be sent")
	}
}

func TestNetworkSimulator_Corruption(t *testing.T) {
	ns := NewNetworkSimulator()

	ns.Enable(NetworkCondition{
		Corruption: 100.0,
	})

	corrupted := 0
	for i := 0; i < 100; i++ {
		if ns.ShouldCorruptPacket() {
			corrupted++
		}
	}

	if corrupted < 95 {
		t.Errorf("Expected ~100 corruptions, got %d", corrupted)
	}
}

func TestNetworkSimulator_CorruptPacket(t *testing.T) {
	ns := NewNetworkSimulator()

	original := []byte{0x00, 0x00, 0x00, 0x00}
	data := make([]byte, len(original))
	copy(data, original)

	ns.CorruptPacket(data)

	// Should have changed
	changed := false
	for i := range data {
		if data[i] != original[i] {
			changed = true
			break
		}
	}

	if !changed {
		t.Error("Packet should have been corrupted")
	}
}

func TestNetworkSimulator_Statistics(t *testing.T) {
	ns := NewNetworkSimulator()

	ns.Enable(NetworkCondition{
		PacketLoss: 50.0,
	})

	// Send packets
	for i := 0; i < 100; i++ {
		ns.ShouldSendPacket(1024)
	}

	stats := ns.GetStatistics()

	if stats.TotalPackets != 100 {
		t.Errorf("TotalPackets = %d, expected 100", stats.TotalPackets)
	}

	if stats.DroppedPackets == 0 {
		t.Error("Should have dropped packets")
	}

	if stats.DropRate == 0.0 {
		t.Error("DropRate should be > 0")
	}
}

func TestNetworkSimulator_Reset(t *testing.T) {
	ns := NewNetworkSimulator()

	ns.Enable(Poor4G)

	// Generate some stats
	for i := 0; i < 50; i++ {
		ns.ShouldSendPacket(1024)
	}

	ns.Reset()

	stats := ns.GetStatistics()

	if stats.TotalPackets != 0 {
		t.Error("Stats should be reset")
	}
}

func TestNetworkSimulator_Disabled(t *testing.T) {
	ns := NewNetworkSimulator()

	// Disabled simulator should always pass packets through
	shouldSend, delay := ns.ShouldSendPacket(1024)

	if !shouldSend {
		t.Error("Disabled simulator should allow all packets")
	}

	if delay != 0 {
		t.Error("Disabled simulator should have no delay")
	}

	if ns.ShouldCorruptPacket() {
		t.Error("Disabled simulator should not corrupt")
	}
}

func TestGetPresetName(t *testing.T) {
	tests := []struct {
		condition NetworkCondition
		expected  string
	}{
		{PerfectNetwork, "Perfect"},
		{Good4G, "Good 4G"},
		{Regular4G, "Regular 4G"},
		{Poor4G, "Poor 4G"},
		{Good3G, "Good 3G"},
		{EdgeNetwork, "EDGE/2G"},
		{Satellite, "Satellite"},
	}

	for _, tt := range tests {
		name := GetPresetName(tt.condition)
		if name != tt.expected {
			t.Errorf("GetPresetName() = %s, expected %s", name, tt.expected)
		}
	}
}

func TestFormatCondition(t *testing.T) {
	formatted := FormatCondition(Poor4G)

	if formatted == "" {
		t.Error("FormatCondition() returned empty string")
	}

	// Should contain key information
	if !contains(formatted, "Latency") {
		t.Error("Should contain latency info")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}

func BenchmarkNetworkSimulator_ShouldSendPacket(b *testing.B) {
	ns := NewNetworkSimulator()
	ns.Enable(Regular4G)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ns.ShouldSendPacket(1024)
	}
}
