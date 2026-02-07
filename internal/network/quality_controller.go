package network

import (
	"sync"
	"time"
)

// QualityController manages adaptive bitrate and quality
type QualityController struct {
	mu sync.Mutex

	// Current quality settings
	currentFPS       int
	currentWidth     int
	currentHeight    int

	// Quality constraints
	minFPS           int
	maxFPS           int
	targetLossRate   float64 // Target packet loss rate (%)

	// Network statistics
	recentLossRates  []float64
	recentJitter     []time.Duration
	lastAdjustment   time.Time
	adjustmentCooldown time.Duration

	// Quality change callback
	onQualityChange func(fps, width, height int)
}

// QualityLevel represents current quality settings
type QualityLevel struct {
	FPS    int
	Width  int
	Height int
	Reason string
}

// NewQualityController creates a new adaptive quality controller
func NewQualityController(initialFPS int) *QualityController {
	return &QualityController{
		currentFPS:         initialFPS,
		currentWidth:       40,
		currentHeight:      30,
		minFPS:            5,
		maxFPS:            20,
		targetLossRate:    1.0, // Target < 1% loss
		recentLossRates:   make([]float64, 0),
		recentJitter:      make([]time.Duration, 0),
		lastAdjustment:    time.Now(),
		adjustmentCooldown: 5 * time.Second, // Wait 5s between adjustments
	}
}

// UpdateNetworkStats updates network statistics and may adjust quality
func (qc *QualityController) UpdateNetworkStats(lossRate float64, jitter time.Duration) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	// Keep last 10 measurements (sliding window)
	qc.recentLossRates = append(qc.recentLossRates, lossRate)
	if len(qc.recentLossRates) > 10 {
		qc.recentLossRates = qc.recentLossRates[1:]
	}

	qc.recentJitter = append(qc.recentJitter, jitter)
	if len(qc.recentJitter) > 10 {
		qc.recentJitter = qc.recentJitter[1:]
	}

	// Check if cooldown has passed
	if time.Since(qc.lastAdjustment) < qc.adjustmentCooldown {
		return
	}

	// Calculate average loss rate
	avgLoss := qc.averageLossRate()

	// Adjust quality based on network conditions
	qc.adjustQuality(avgLoss)
}

// averageLossRate calculates average packet loss rate
func (qc *QualityController) averageLossRate() float64 {
	if len(qc.recentLossRates) == 0 {
		return 0
	}

	sum := 0.0
	for _, rate := range qc.recentLossRates {
		sum += rate
	}
	return sum / float64(len(qc.recentLossRates))
}

// adjustQuality adjusts FPS based on network conditions
func (qc *QualityController) adjustQuality(avgLoss float64) {
	oldFPS := qc.currentFPS
	newFPS := qc.currentFPS

	// Network is poor (high loss) - reduce FPS
	if avgLoss > 5.0 {
		// Very bad network (>5% loss) - significantly reduce
		newFPS = qc.currentFPS - 3
	} else if avgLoss > 2.0 {
		// Bad network (>2% loss) - reduce
		newFPS = qc.currentFPS - 2
	} else if avgLoss > qc.targetLossRate {
		// Slightly elevated loss - minor reduction
		newFPS = qc.currentFPS - 1
	} else if avgLoss < 0.5 {
		// Network is good - increase FPS
		newFPS = qc.currentFPS + 1
	}

	// Clamp to min/max
	if newFPS < qc.minFPS {
		newFPS = qc.minFPS
	}
	if newFPS > qc.maxFPS {
		newFPS = qc.maxFPS
	}

	// Only adjust if changed
	if newFPS != oldFPS {
		qc.currentFPS = newFPS
		qc.lastAdjustment = time.Now()

		// Trigger callback if set
		if qc.onQualityChange != nil {
			qc.onQualityChange(qc.currentFPS, qc.currentWidth, qc.currentHeight)
		}
	}
}

// GetCurrentQuality returns current quality settings
func (qc *QualityController) GetCurrentQuality() QualityLevel {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	avgLoss := qc.averageLossRate()
	reason := "Stable"

	if avgLoss > 5.0 {
		reason = "High packet loss - reduced quality"
	} else if avgLoss > 2.0 {
		reason = "Elevated packet loss - adjusted quality"
	} else if avgLoss < 0.5 {
		reason = "Good network - increased quality"
	}

	return QualityLevel{
		FPS:    qc.currentFPS,
		Width:  qc.currentWidth,
		Height: qc.currentHeight,
		Reason: reason,
	}
}

// GetCurrentFPS returns current FPS setting
func (qc *QualityController) GetCurrentFPS() int {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	return qc.currentFPS
}

// SetQualityChangeCallback sets callback for quality changes
func (qc *QualityController) SetQualityChangeCallback(callback func(fps, width, height int)) {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.onQualityChange = callback
}

// ForceQuality manually sets quality (bypasses adaptive control temporarily)
func (qc *QualityController) ForceQuality(fps, width, height int) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	qc.currentFPS = fps
	qc.currentWidth = width
	qc.currentHeight = height
	qc.lastAdjustment = time.Now()
}

// GetStatistics returns quality controller statistics
func (qc *QualityController) GetStatistics() QualityStatistics {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	return QualityStatistics{
		CurrentFPS:     qc.currentFPS,
		AverageLoss:    qc.averageLossRate(),
		Adjustments:    0, // TODO: track this
		LastAdjustment: qc.lastAdjustment,
	}
}

// QualityStatistics holds quality controller statistics
type QualityStatistics struct {
	CurrentFPS     int
	AverageLoss    float64
	Adjustments    int
	LastAdjustment time.Time
}
