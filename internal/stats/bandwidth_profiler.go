package stats

import (
	"fmt"
	"sync"
	"time"
)

// BandwidthSample represents a bandwidth measurement at a point in time
type BandwidthSample struct {
	Timestamp      time.Time
	BytesSent      uint64
	BytesReceived  uint64
	PacketsSent    uint64
	PacketsReceived uint64
}

// BandwidthProfiler tracks bandwidth usage over time
type BandwidthProfiler struct {
	mu sync.RWMutex

	// Cumulative totals
	totalBytesSent      uint64
	totalBytesReceived  uint64
	totalPacketsSent    uint64
	totalPacketsReceived uint64

	// Current session
	sessionStart time.Time
	sessionEnd   time.Time

	// Samples for graphing/analysis
	samples        []BandwidthSample
	maxSamples     int
	sampleInterval time.Duration

	// Peak measurements
	peakUploadRate   float64 // KB/s
	peakDownloadRate float64 // KB/s
	peakTotalRate    float64 // KB/s

	// Current rate tracking
	lastSampleTime    time.Time
	lastBytesSent     uint64
	lastBytesReceived uint64

	// Running averages
	avgUploadRate   float64
	avgDownloadRate float64

	// Per-stream tracking
	streams map[string]*StreamStats
}

// StreamStats tracks per-stream bandwidth
type StreamStats struct {
	StreamID       string
	BytesSent      uint64
	BytesReceived  uint64
	PacketsSent    uint64
	PacketsReceived uint64
	StartTime      time.Time
	LastUpdate     time.Time
}

// NewBandwidthProfiler creates a new bandwidth profiler
func NewBandwidthProfiler() *BandwidthProfiler {
	return &BandwidthProfiler{
		sessionStart:   time.Now(),
		samples:        make([]BandwidthSample, 0),
		maxSamples:     1000, // Keep last 1000 samples
		sampleInterval: 1 * time.Second,
		lastSampleTime: time.Now(),
		streams:        make(map[string]*StreamStats),
	}
}

// RecordSent records data sent
func (bp *BandwidthProfiler) RecordSent(bytes uint64, packets uint64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.totalBytesSent += bytes
	bp.totalPacketsSent += packets
}

// RecordReceived records data received
func (bp *BandwidthProfiler) RecordReceived(bytes uint64, packets uint64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.totalBytesReceived += bytes
	bp.totalPacketsReceived += packets
}

// RecordStreamSent records data sent on a specific stream
func (bp *BandwidthProfiler) RecordStreamSent(streamID string, bytes uint64, packets uint64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.totalBytesSent += bytes
	bp.totalPacketsSent += packets

	stream, ok := bp.streams[streamID]
	if !ok {
		stream = &StreamStats{
			StreamID:  streamID,
			StartTime: time.Now(),
		}
		bp.streams[streamID] = stream
	}

	stream.BytesSent += bytes
	stream.PacketsSent += packets
	stream.LastUpdate = time.Now()
}

// RecordStreamReceived records data received on a specific stream
func (bp *BandwidthProfiler) RecordStreamReceived(streamID string, bytes uint64, packets uint64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.totalBytesReceived += bytes
	bp.totalPacketsReceived += packets

	stream, ok := bp.streams[streamID]
	if !ok {
		stream = &StreamStats{
			StreamID:  streamID,
			StartTime: time.Now(),
		}
		bp.streams[streamID] = stream
	}

	stream.BytesReceived += bytes
	stream.PacketsReceived += packets
	stream.LastUpdate = time.Now()
}

// Sample takes a bandwidth sample (call periodically, e.g., every second)
func (bp *BandwidthProfiler) Sample() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	now := time.Now()
	timeDelta := now.Sub(bp.lastSampleTime).Seconds()

	if timeDelta == 0 {
		return
	}

	// Calculate rates (KB/s)
	bytesSentDelta := bp.totalBytesSent - bp.lastBytesSent
	bytesReceivedDelta := bp.totalBytesReceived - bp.lastBytesReceived

	uploadRate := float64(bytesSentDelta) / timeDelta / 1024.0
	downloadRate := float64(bytesReceivedDelta) / timeDelta / 1024.0
	totalRate := uploadRate + downloadRate

	// Update peaks
	if uploadRate > bp.peakUploadRate {
		bp.peakUploadRate = uploadRate
	}
	if downloadRate > bp.peakDownloadRate {
		bp.peakDownloadRate = downloadRate
	}
	if totalRate > bp.peakTotalRate {
		bp.peakTotalRate = totalRate
	}

	// Update running averages
	sessionDuration := now.Sub(bp.sessionStart).Seconds()
	if sessionDuration > 0 {
		bp.avgUploadRate = float64(bp.totalBytesSent) / sessionDuration / 1024.0
		bp.avgDownloadRate = float64(bp.totalBytesReceived) / sessionDuration / 1024.0
	}

	// Add sample
	sample := BandwidthSample{
		Timestamp:      now,
		BytesSent:      bp.totalBytesSent,
		BytesReceived:  bp.totalBytesReceived,
		PacketsSent:    bp.totalPacketsSent,
		PacketsReceived: bp.totalPacketsReceived,
	}

	bp.samples = append(bp.samples, sample)

	// Trim if exceeds max
	if len(bp.samples) > bp.maxSamples {
		bp.samples = bp.samples[len(bp.samples)-bp.maxSamples:]
	}

	// Update last values
	bp.lastSampleTime = now
	bp.lastBytesSent = bp.totalBytesSent
	bp.lastBytesReceived = bp.totalBytesReceived
}

// GetCurrentRates returns current upload/download rates in KB/s
func (bp *BandwidthProfiler) GetCurrentRates() (upload, download float64) {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	now := time.Now()
	timeDelta := now.Sub(bp.lastSampleTime).Seconds()

	if timeDelta == 0 {
		return 0, 0
	}

	bytesSentDelta := bp.totalBytesSent - bp.lastBytesSent
	bytesReceivedDelta := bp.totalBytesReceived - bp.lastBytesReceived

	upload = float64(bytesSentDelta) / timeDelta / 1024.0
	download = float64(bytesReceivedDelta) / timeDelta / 1024.0

	return upload, download
}

// GetStatistics returns comprehensive bandwidth statistics
func (bp *BandwidthProfiler) GetStatistics() Statistics {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	sessionDuration := time.Since(bp.sessionStart)
	if !bp.sessionEnd.IsZero() {
		sessionDuration = bp.sessionEnd.Sub(bp.sessionStart)
	}

	return Statistics{
		TotalBytesSent:      bp.totalBytesSent,
		TotalBytesReceived:  bp.totalBytesReceived,
		TotalPacketsSent:    bp.totalPacketsSent,
		TotalPacketsReceived: bp.totalPacketsReceived,
		SessionDuration:     sessionDuration,
		PeakUploadRate:      bp.peakUploadRate,
		PeakDownloadRate:    bp.peakDownloadRate,
		PeakTotalRate:       bp.peakTotalRate,
		AvgUploadRate:       bp.avgUploadRate,
		AvgDownloadRate:     bp.avgDownloadRate,
		SampleCount:         len(bp.samples),
	}
}

// Statistics represents bandwidth statistics
type Statistics struct {
	TotalBytesSent      uint64
	TotalBytesReceived  uint64
	TotalPacketsSent    uint64
	TotalPacketsReceived uint64
	SessionDuration     time.Duration
	PeakUploadRate      float64 // KB/s
	PeakDownloadRate    float64 // KB/s
	PeakTotalRate       float64 // KB/s
	AvgUploadRate       float64 // KB/s
	AvgDownloadRate     float64 // KB/s
	SampleCount         int
}

// GetSamples returns bandwidth samples for graphing
func (bp *BandwidthProfiler) GetSamples() []BandwidthSample {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	// Return a copy
	samples := make([]BandwidthSample, len(bp.samples))
	copy(samples, bp.samples)
	return samples
}

// GetStreamStats returns statistics for all streams
func (bp *BandwidthProfiler) GetStreamStats() map[string]*StreamStats {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	// Return a copy
	stats := make(map[string]*StreamStats)
	for id, stream := range bp.streams {
		streamCopy := *stream
		stats[id] = &streamCopy
	}
	return stats
}

// Reset resets all statistics
func (bp *BandwidthProfiler) Reset() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.totalBytesSent = 0
	bp.totalBytesReceived = 0
	bp.totalPacketsSent = 0
	bp.totalPacketsReceived = 0
	bp.sessionStart = time.Now()
	bp.sessionEnd = time.Time{}
	bp.samples = make([]BandwidthSample, 0)
	bp.peakUploadRate = 0
	bp.peakDownloadRate = 0
	bp.peakTotalRate = 0
	bp.lastSampleTime = time.Now()
	bp.lastBytesSent = 0
	bp.lastBytesReceived = 0
	bp.avgUploadRate = 0
	bp.avgDownloadRate = 0
	bp.streams = make(map[string]*StreamStats)
}

// EndSession marks the end of the current session
func (bp *BandwidthProfiler) EndSession() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.sessionEnd = time.Now()
}

// FormatBytes formats bytes in human-readable format
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatRate formats rate in human-readable format
func FormatRate(kbps float64) string {
	if kbps < 1 {
		return fmt.Sprintf("%.0f B/s", kbps*1024)
	}
	if kbps < 1024 {
		return fmt.Sprintf("%.2f KB/s", kbps)
	}
	return fmt.Sprintf("%.2f MB/s", kbps/1024)
}
