package stats

import (
	"testing"
	"time"
)

func TestNewBandwidthProfiler(t *testing.T) {
	bp := NewBandwidthProfiler()

	if bp == nil {
		t.Fatal("NewBandwidthProfiler() returned nil")
	}

	if bp.maxSamples != 1000 {
		t.Errorf("maxSamples = %d, expected 1000", bp.maxSamples)
	}

	if bp.sampleInterval != time.Second {
		t.Errorf("sampleInterval = %v, expected 1s", bp.sampleInterval)
	}
}

func TestBandwidthProfiler_RecordSent(t *testing.T) {
	bp := NewBandwidthProfiler()

	bp.RecordSent(1024, 10)

	stats := bp.GetStatistics()

	if stats.TotalBytesSent != 1024 {
		t.Errorf("TotalBytesSent = %d, expected 1024", stats.TotalBytesSent)
	}

	if stats.TotalPacketsSent != 10 {
		t.Errorf("TotalPacketsSent = %d, expected 10", stats.TotalPacketsSent)
	}
}

func TestBandwidthProfiler_RecordReceived(t *testing.T) {
	bp := NewBandwidthProfiler()

	bp.RecordReceived(2048, 20)

	stats := bp.GetStatistics()

	if stats.TotalBytesReceived != 2048 {
		t.Errorf("TotalBytesReceived = %d, expected 2048", stats.TotalBytesReceived)
	}

	if stats.TotalPacketsReceived != 20 {
		t.Errorf("TotalPacketsReceived = %d, expected 20", stats.TotalPacketsReceived)
	}
}

func TestBandwidthProfiler_Sample(t *testing.T) {
	bp := NewBandwidthProfiler()

	// Record some data
	bp.RecordSent(10240, 10) // 10 KB
	time.Sleep(10 * time.Millisecond)

	bp.Sample()

	stats := bp.GetStatistics()

	if stats.SampleCount != 1 {
		t.Errorf("SampleCount = %d, expected 1", stats.SampleCount)
	}

	if stats.AvgUploadRate == 0 {
		t.Error("AvgUploadRate should be > 0")
	}
}

func TestBandwidthProfiler_GetCurrentRates(t *testing.T) {
	bp := NewBandwidthProfiler()

	// Simulate traffic
	bp.RecordSent(10240, 10)     // 10 KB sent
	bp.RecordReceived(20480, 20) // 20 KB received

	time.Sleep(50 * time.Millisecond)

	upload, download := bp.GetCurrentRates()

	// Rates should be positive
	if upload <= 0 {
		t.Error("Upload rate should be > 0")
	}

	if download <= 0 {
		t.Error("Download rate should be > 0")
	}

	// Download should be roughly 2x upload
	if download < upload {
		t.Error("Download rate should be higher than upload rate")
	}
}

func TestBandwidthProfiler_PeakRates(t *testing.T) {
	bp := NewBandwidthProfiler()

	// First burst
	bp.RecordSent(10240, 10)
	time.Sleep(10 * time.Millisecond)
	bp.Sample()

	// Bigger burst
	bp.RecordSent(51200, 50) // 50 KB
	time.Sleep(10 * time.Millisecond)
	bp.Sample()

	stats := bp.GetStatistics()

	if stats.PeakUploadRate == 0 {
		t.Error("PeakUploadRate should be > 0")
	}
}

func TestBandwidthProfiler_StreamStats(t *testing.T) {
	bp := NewBandwidthProfiler()

	// Record stream data
	bp.RecordStreamSent("stream1", 1024, 10)
	bp.RecordStreamReceived("stream1", 2048, 20)

	bp.RecordStreamSent("stream2", 512, 5)

	streamStats := bp.GetStreamStats()

	if len(streamStats) != 2 {
		t.Errorf("Stream count = %d, expected 2", len(streamStats))
	}

	stream1 := streamStats["stream1"]
	if stream1 == nil {
		t.Fatal("stream1 not found")
	}

	if stream1.BytesSent != 1024 {
		t.Errorf("stream1 BytesSent = %d, expected 1024", stream1.BytesSent)
	}

	if stream1.BytesReceived != 2048 {
		t.Errorf("stream1 BytesReceived = %d, expected 2048", stream1.BytesReceived)
	}
}

func TestBandwidthProfiler_Reset(t *testing.T) {
	bp := NewBandwidthProfiler()

	// Record some data
	bp.RecordSent(10240, 10)
	bp.RecordReceived(20480, 20)
	bp.Sample()

	// Reset
	bp.Reset()

	stats := bp.GetStatistics()

	if stats.TotalBytesSent != 0 {
		t.Errorf("After reset, TotalBytesSent = %d, expected 0", stats.TotalBytesSent)
	}

	if stats.TotalBytesReceived != 0 {
		t.Errorf("After reset, TotalBytesReceived = %d, expected 0", stats.TotalBytesReceived)
	}

	if stats.SampleCount != 0 {
		t.Errorf("After reset, SampleCount = %d, expected 0", stats.SampleCount)
	}

	streamStats := bp.GetStreamStats()
	if len(streamStats) != 0 {
		t.Errorf("After reset, stream count = %d, expected 0", len(streamStats))
	}
}

func TestBandwidthProfiler_GetSamples(t *testing.T) {
	bp := NewBandwidthProfiler()

	// Take multiple samples
	for i := 0; i < 5; i++ {
		bp.RecordSent(1024, 1)
		time.Sleep(10 * time.Millisecond)
		bp.Sample()
	}

	samples := bp.GetSamples()

	if len(samples) != 5 {
		t.Errorf("Sample count = %d, expected 5", len(samples))
	}

	// Verify samples are in chronological order
	for i := 1; i < len(samples); i++ {
		if samples[i].Timestamp.Before(samples[i-1].Timestamp) {
			t.Error("Samples should be in chronological order")
		}
	}

	// Verify cumulative nature
	for i := 1; i < len(samples); i++ {
		if samples[i].BytesSent < samples[i-1].BytesSent {
			t.Error("BytesSent should be cumulative")
		}
	}
}

func TestBandwidthProfiler_SessionDuration(t *testing.T) {
	bp := NewBandwidthProfiler()

	time.Sleep(100 * time.Millisecond)

	stats := bp.GetStatistics()

	if stats.SessionDuration < 100*time.Millisecond {
		t.Errorf("SessionDuration = %v, expected >= 100ms", stats.SessionDuration)
	}

	bp.EndSession()

	stats = bp.GetStatistics()

	// After ending, duration should be fixed
	endedDuration := stats.SessionDuration

	time.Sleep(50 * time.Millisecond)

	stats = bp.GetStatistics()

	if stats.SessionDuration != endedDuration {
		t.Error("SessionDuration should not change after EndSession()")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KiB"},
		{1536, "1.50 KiB"},
		{1048576, "1.00 MiB"},
		{1073741824, "1.00 GiB"},
	}

	for _, tt := range tests {
		result := FormatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestFormatRate(t *testing.T) {
	tests := []struct {
		kbps float64
		want string
	}{
		{0.5, "512 B/s"},
		{1.0, "1.00 KB/s"},
		{10.5, "10.50 KB/s"},
		{1024.0, "1.00 MB/s"},
		{2048.5, "2.00 MB/s"},
	}

	for _, tt := range tests {
		result := FormatRate(tt.kbps)
		if result != tt.want {
			t.Errorf("FormatRate(%.2f) = %s, expected %s", tt.kbps, result, tt.want)
		}
	}
}

func TestBandwidthProfiler_MaxSamples(t *testing.T) {
	bp := NewBandwidthProfiler()
	bp.maxSamples = 10 // Set low limit for testing

	// Add more samples than max
	for i := 0; i < 20; i++ {
		bp.RecordSent(1024, 1)
		time.Sleep(5 * time.Millisecond)
		bp.Sample()
	}

	samples := bp.GetSamples()

	if len(samples) != 10 {
		t.Errorf("Sample count = %d, expected 10 (should trim to maxSamples)", len(samples))
	}

	// Verify we kept the most recent samples
	stats := bp.GetStatistics()
	if samples[len(samples)-1].BytesSent != stats.TotalBytesSent {
		t.Error("Should have kept the most recent samples")
	}
}

func BenchmarkBandwidthProfiler_RecordSent(b *testing.B) {
	bp := NewBandwidthProfiler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.RecordSent(1024, 1)
	}
}

func BenchmarkBandwidthProfiler_Sample(b *testing.B) {
	bp := NewBandwidthProfiler()

	// Add some data
	bp.RecordSent(10240, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.Sample()
	}
}

func BenchmarkBandwidthProfiler_GetStatistics(b *testing.B) {
	bp := NewBandwidthProfiler()

	// Add some samples
	for i := 0; i < 100; i++ {
		bp.RecordSent(1024, 1)
		bp.Sample()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bp.GetStatistics()
	}
}
