package quality

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// QualityLevel represents call quality rating
type QualityLevel string

const (
	QualityExcellent QualityLevel = "excellent" // MOS >= 4.0
	QualityGood      QualityLevel = "good"      // MOS >= 3.5
	QualityFair      QualityLevel = "fair"      // MOS >= 2.5
	QualityPoor      QualityLevel = "poor"      // MOS < 2.5
	QualityUnknown   QualityLevel = "unknown"
)

// QualityMonitor tracks call quality metrics
type QualityMonitor struct {
	mu sync.RWMutex

	// RTT (Round-Trip Time) measurements
	rttSamples    []time.Duration
	maxRTTSamples int
	avgRTT        time.Duration
	minRTT        time.Duration
	maxRTT        time.Duration

	// Jitter measurements
	jitterSamples    []time.Duration
	maxJitterSamples int
	avgJitter        time.Duration
	maxJitter        time.Duration

	// Packet loss
	packetsSent     uint64
	packetsReceived uint64
	packetsLost     uint64
	lossRate        float64 // Percentage

	// MOS (Mean Opinion Score)
	currentMOS float64

	// Quality level
	currentQuality QualityLevel

	// Timestamps
	lastUpdate time.Time
	startTime  time.Time

	// Statistics
	updateCount uint64
}

// NewQualityMonitor creates a new quality monitor
func NewQualityMonitor() *QualityMonitor {
	return &QualityMonitor{
		rttSamples:       make([]time.Duration, 0),
		maxRTTSamples:    100,
		jitterSamples:    make([]time.Duration, 0),
		maxJitterSamples: 100,
		minRTT:           time.Duration(math.MaxInt64),
		maxRTT:           0,
		maxJitter:        0,
		currentMOS:       0,
		currentQuality:   QualityUnknown,
		lastUpdate:       time.Now(),
		startTime:        time.Now(),
	}
}

// UpdateRTT updates RTT measurement
func (qm *QualityMonitor) UpdateRTT(rtt time.Duration) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Add sample
	qm.rttSamples = append(qm.rttSamples, rtt)

	// Trim if exceeds max
	if len(qm.rttSamples) > qm.maxRTTSamples {
		qm.rttSamples = qm.rttSamples[len(qm.rttSamples)-qm.maxRTTSamples:]
	}

	// Update min/max
	if rtt < qm.minRTT {
		qm.minRTT = rtt
	}
	if rtt > qm.maxRTT {
		qm.maxRTT = rtt
	}

	// Calculate average RTT
	var sum time.Duration
	for _, r := range qm.rttSamples {
		sum += r
	}
	qm.avgRTT = sum / time.Duration(len(qm.rttSamples))

	// Calculate jitter (RTT variation)
	if len(qm.rttSamples) > 1 {
		prevRTT := qm.rttSamples[len(qm.rttSamples)-2]
		jitter := absDuration(rtt - prevRTT)

		qm.jitterSamples = append(qm.jitterSamples, jitter)

		if len(qm.jitterSamples) > qm.maxJitterSamples {
			qm.jitterSamples = qm.jitterSamples[len(qm.jitterSamples)-qm.maxJitterSamples:]
		}

		if jitter > qm.maxJitter {
			qm.maxJitter = jitter
		}

		// Calculate average jitter
		var jitterSum time.Duration
		for _, j := range qm.jitterSamples {
			jitterSum += j
		}
		qm.avgJitter = jitterSum / time.Duration(len(qm.jitterSamples))
	}

	qm.lastUpdate = time.Now()
	qm.updateCount++

	// Recalculate MOS
	qm.calculateMOS()
}

// UpdatePacketLoss updates packet loss statistics
func (qm *QualityMonitor) UpdatePacketLoss(sent, received uint64) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	qm.packetsSent = sent
	qm.packetsReceived = received

	if sent > 0 {
		qm.packetsLost = sent - received
		qm.lossRate = float64(qm.packetsLost) / float64(sent) * 100.0
	}

	qm.lastUpdate = time.Now()
	qm.updateCount++

	// Recalculate MOS
	qm.calculateMOS()
}

// RecordPacketSent records a packet sent
func (qm *QualityMonitor) RecordPacketSent() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	qm.packetsSent++

	if qm.packetsSent > 0 {
		qm.lossRate = float64(qm.packetsLost) / float64(qm.packetsSent) * 100.0
	}

	qm.updateCount++
	qm.calculateMOS()
}

// RecordPacketReceived records a packet received
func (qm *QualityMonitor) RecordPacketReceived() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	qm.packetsReceived++

	if qm.packetsSent > 0 {
		qm.packetsLost = qm.packetsSent - qm.packetsReceived
		qm.lossRate = float64(qm.packetsLost) / float64(qm.packetsSent) * 100.0
	}

	qm.updateCount++
	qm.calculateMOS()
}

// calculateMOS calculates Mean Opinion Score based on E-model (ITU-T G.107)
// Simplified version - actual MOS calculation is very complex
func (qm *QualityMonitor) calculateMOS() {
	// E-model simplified formula
	// R = 93.2 - delay_factor - loss_factor - jitter_factor
	// MOS = 1 + 0.035*R + 7*10^-6*R*(R-60)*(100-R)

	// Delay factor (RTT impact)
	delayMs := float64(qm.avgRTT.Milliseconds())
	delayFactor := 0.0
	if delayMs < 150 {
		delayFactor = 0
	} else if delayMs < 300 {
		delayFactor = (delayMs - 150) * 0.1
	} else {
		delayFactor = 15 + (delayMs-300)*0.2
	}

	// Loss factor (packet loss impact)
	lossFactor := qm.lossRate * 2.5

	// Jitter factor
	jitterMs := float64(qm.avgJitter.Milliseconds())
	jitterFactor := jitterMs * 0.1

	// Calculate R-factor
	R := 93.2 - delayFactor - lossFactor - jitterFactor

	// Clamp R to valid range
	if R < 0 {
		R = 0
	}
	if R > 100 {
		R = 100
	}

	// Calculate MOS from R-factor
	if R < 0 {
		qm.currentMOS = 1.0
	} else if R > 100 {
		qm.currentMOS = 4.5
	} else {
		qm.currentMOS = 1 + 0.035*R + 7e-6*R*(R-60)*(100-R)
	}

	// Clamp MOS to 1.0-5.0
	if qm.currentMOS < 1.0 {
		qm.currentMOS = 1.0
	}
	if qm.currentMOS > 5.0 {
		qm.currentMOS = 5.0
	}

	// Determine quality level
	if qm.currentMOS >= 4.0 {
		qm.currentQuality = QualityExcellent
	} else if qm.currentMOS >= 3.5 {
		qm.currentQuality = QualityGood
	} else if qm.currentMOS >= 2.5 {
		qm.currentQuality = QualityFair
	} else {
		qm.currentQuality = QualityPoor
	}
}

// GetMetrics returns current quality metrics
func (qm *QualityMonitor) GetMetrics() Metrics {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	return Metrics{
		AvgRTT:         qm.avgRTT,
		MinRTT:         qm.minRTT,
		MaxRTT:         qm.maxRTT,
		AvgJitter:      qm.avgJitter,
		MaxJitter:      qm.maxJitter,
		PacketsSent:    qm.packetsSent,
		PacketsReceived: qm.packetsReceived,
		PacketsLost:    qm.packetsLost,
		LossRate:       qm.lossRate,
		MOS:            qm.currentMOS,
		Quality:        qm.currentQuality,
		LastUpdate:     qm.lastUpdate,
		MonitoringTime: time.Since(qm.startTime),
		UpdateCount:    qm.updateCount,
	}
}

// Metrics represents call quality metrics
type Metrics struct {
	AvgRTT          time.Duration
	MinRTT          time.Duration
	MaxRTT          time.Duration
	AvgJitter       time.Duration
	MaxJitter       time.Duration
	PacketsSent     uint64
	PacketsReceived uint64
	PacketsLost     uint64
	LossRate        float64 // Percentage
	MOS             float64 // 1.0-5.0
	Quality         QualityLevel
	LastUpdate      time.Time
	MonitoringTime  time.Duration
	UpdateCount     uint64
}

// GetRTTSamples returns recent RTT samples
func (qm *QualityMonitor) GetRTTSamples() []time.Duration {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	samples := make([]time.Duration, len(qm.rttSamples))
	copy(samples, qm.rttSamples)
	return samples
}

// GetJitterSamples returns recent jitter samples
func (qm *QualityMonitor) GetJitterSamples() []time.Duration {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	samples := make([]time.Duration, len(qm.jitterSamples))
	copy(samples, qm.jitterSamples)
	return samples
}

// Reset resets all metrics
func (qm *QualityMonitor) Reset() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	qm.rttSamples = make([]time.Duration, 0)
	qm.jitterSamples = make([]time.Duration, 0)
	qm.avgRTT = 0
	qm.minRTT = time.Duration(math.MaxInt64)
	qm.maxRTT = 0
	qm.avgJitter = 0
	qm.maxJitter = 0
	qm.packetsSent = 0
	qm.packetsReceived = 0
	qm.packetsLost = 0
	qm.lossRate = 0
	qm.currentMOS = 0
	qm.currentQuality = QualityUnknown
	qm.lastUpdate = time.Now()
	qm.startTime = time.Now()
	qm.updateCount = 0
}

// FormatMOS formats MOS score with rating
func FormatMOS(mos float64) string {
	quality := ""
	if mos >= 4.0 {
		quality = "Excellent"
	} else if mos >= 3.5 {
		quality = "Good"
	} else if mos >= 2.5 {
		quality = "Fair"
	} else {
		quality = "Poor"
	}
	return fmt.Sprintf("%.2f (%s)", mos, quality)
}

// FormatQuality returns emoji + text for quality level
func FormatQuality(quality QualityLevel) string {
	switch quality {
	case QualityExcellent:
		return "🟢 Excellent"
	case QualityGood:
		return "🟡 Good"
	case QualityFair:
		return "🟠 Fair"
	case QualityPoor:
		return "🔴 Poor"
	default:
		return "⚪ Unknown"
	}
}

// absDuration returns absolute value of duration
func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// IsAcceptable returns true if quality is acceptable (MOS >= 3.5)
func (qm *QualityMonitor) IsAcceptable() bool {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	return qm.currentMOS >= 3.5
}

// GetQualityPercent returns quality as percentage (0-100)
func (qm *QualityMonitor) GetQualityPercent() float64 {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	// Convert MOS (1-5) to percentage (0-100)
	return (qm.currentMOS - 1.0) / 4.0 * 100.0
}
