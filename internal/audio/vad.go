package audio

import (
	"math"
)

// VAD (Voice Activity Detection) detects speech in audio
type VAD struct {
	sampleRate      int
	energyThreshold float64

	// Adaptive thresholds
	minEnergy       float64
	maxEnergy       float64
	noiseFloor      float64

	// State
	frameCount      int
	speechFrames    int
	silenceFrames   int
	isSpeaking      bool

	// Configuration
	hangoverFrames  int  // Frames to keep sending after speech stops
	onsetFrames     int  // Frames needed to start speech

	// Statistics
	totalFrames     uint64
	speechTotal     uint64
	silenceTotal    uint64
}

// NewVAD creates a new Voice Activity Detector
func NewVAD(sampleRate int) *VAD {
	return &VAD{
		sampleRate:      sampleRate,
		energyThreshold: 300.0,  // Initial threshold
		minEnergy:       1000.0,
		maxEnergy:       0.0,
		noiseFloor:      100.0,
		hangoverFrames:  10,     // ~200ms hangover at 20ms frames
		onsetFrames:     2,      // 2 frames to start speech
		frameCount:      0,
		speechFrames:    0,
		silenceFrames:   0,
		isSpeaking:      false,
	}
}

// Process analyzes audio samples and returns true if speech detected
func (v *VAD) Process(samples []int16) bool {
	v.frameCount++
	v.totalFrames++

	// Calculate frame energy
	energy := v.calculateEnergy(samples)

	// Update adaptive thresholds
	v.updateThresholds(energy)

	// Determine if this frame contains speech
	hasSpeech := energy > v.energyThreshold

	// State machine for speech detection
	if hasSpeech {
		v.speechFrames++
		v.silenceFrames = 0

		// Start speaking if we have enough onset frames
		if !v.isSpeaking && v.speechFrames >= v.onsetFrames {
			v.isSpeaking = true
		}
	} else {
		v.silenceFrames++
		v.speechFrames = 0

		// Stop speaking after hangover period
		if v.isSpeaking && v.silenceFrames >= v.hangoverFrames {
			v.isSpeaking = false
		}
	}

	// Update statistics
	if v.isSpeaking {
		v.speechTotal++
	} else {
		v.silenceTotal++
	}

	return v.isSpeaking
}

// calculateEnergy computes RMS energy of audio samples
func (v *VAD) calculateEnergy(samples []int16) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	var sum float64
	for _, sample := range samples {
		val := float64(sample)
		sum += val * val
	}

	rms := math.Sqrt(sum / float64(len(samples)))
	return rms
}

// updateThresholds adapts thresholds based on observed energy levels
func (v *VAD) updateThresholds(energy float64) {
	// Update min/max energy
	if energy < v.minEnergy {
		v.minEnergy = energy
	}
	if energy > v.maxEnergy {
		v.maxEnergy = energy
	}

	// Adaptive noise floor (slowly tracks minimum energy)
	alpha := 0.01 // Smoothing factor
	if energy < v.noiseFloor*2.0 {
		v.noiseFloor = v.noiseFloor*(1.0-alpha) + energy*alpha
	}

	// Adaptive threshold (noise floor + margin)
	margin := 3.0 // 3x noise floor
	v.energyThreshold = v.noiseFloor * margin

	// Clamp threshold to reasonable range
	if v.energyThreshold < 100.0 {
		v.energyThreshold = 100.0
	}
	if v.energyThreshold > 2000.0 {
		v.energyThreshold = 2000.0
	}
}

// IsSpeaking returns true if speech is currently detected
func (v *VAD) IsSpeaking() bool {
	return v.isSpeaking
}

// GetStatistics returns VAD statistics
func (v *VAD) GetStatistics() VADStatistics {
	var activityRate float64
	if v.totalFrames > 0 {
		activityRate = float64(v.speechTotal) / float64(v.totalFrames) * 100.0
	}

	return VADStatistics{
		TotalFrames:     v.totalFrames,
		SpeechFrames:    v.speechTotal,
		SilenceFrames:   v.silenceTotal,
		ActivityRate:    activityRate,
		CurrentEnergy:   v.energyThreshold,
		NoiseFloor:      v.noiseFloor,
		IsSpeaking:      v.isSpeaking,
	}
}

// SetSensitivity adjusts VAD sensitivity (0.0 = less sensitive, 1.0 = more sensitive)
func (v *VAD) SetSensitivity(sensitivity float64) {
	// Clamp to 0.0-1.0
	if sensitivity < 0.0 {
		sensitivity = 0.0
	}
	if sensitivity > 1.0 {
		sensitivity = 1.0
	}

	// Adjust onset frames (fewer frames = more sensitive)
	v.onsetFrames = int(5.0 - sensitivity*3.0) // 2-5 frames
	if v.onsetFrames < 1 {
		v.onsetFrames = 1
	}

	// Adjust hangover frames
	v.hangoverFrames = int(15.0 - sensitivity*5.0) // 10-15 frames
	if v.hangoverFrames < 5 {
		v.hangoverFrames = 5
	}
}

// Reset resets VAD state
func (v *VAD) Reset() {
	v.frameCount = 0
	v.speechFrames = 0
	v.silenceFrames = 0
	v.isSpeaking = false
	v.minEnergy = 1000.0
	v.maxEnergy = 0.0
	v.noiseFloor = 100.0
}

// VADStatistics contains VAD performance statistics
type VADStatistics struct {
	TotalFrames   uint64  // Total frames processed
	SpeechFrames  uint64  // Frames classified as speech
	SilenceFrames uint64  // Frames classified as silence
	ActivityRate  float64 // Speech activity rate (%)
	CurrentEnergy float64 // Current energy threshold
	NoiseFloor    float64 // Estimated noise floor
	IsSpeaking    bool    // Current speaking state
}

// SimpleVAD provides a simple energy-based VAD
type SimpleVAD struct {
	threshold float64
}

// NewSimpleVAD creates a simple VAD with fixed threshold
func NewSimpleVAD(threshold float64) *SimpleVAD {
	return &SimpleVAD{
		threshold: threshold,
	}
}

// Process checks if audio energy exceeds threshold
func (v *SimpleVAD) Process(samples []int16) bool {
	if len(samples) == 0 {
		return false
	}

	var sum float64
	for _, sample := range samples {
		val := float64(sample)
		sum += val * val
	}

	rms := math.Sqrt(sum / float64(len(samples)))
	return rms > v.threshold
}

// ZeroCrossingRate calculates zero-crossing rate (useful for speech detection)
func ZeroCrossingRate(samples []int16) float64 {
	if len(samples) < 2 {
		return 0.0
	}

	crossings := 0
	for i := 1; i < len(samples); i++ {
		if (samples[i-1] >= 0 && samples[i] < 0) || (samples[i-1] < 0 && samples[i] >= 0) {
			crossings++
		}
	}

	return float64(crossings) / float64(len(samples)-1)
}

// SpectralCentroid calculates spectral centroid (useful for speech detection)
func SpectralCentroid(samples []int16) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	// Simple approximation using time-domain samples
	var weightedSum float64
	var totalMagnitude float64

	for i, sample := range samples {
		magnitude := math.Abs(float64(sample))
		weightedSum += float64(i) * magnitude
		totalMagnitude += magnitude
	}

	if totalMagnitude == 0.0 {
		return 0.0
	}

	return weightedSum / totalMagnitude
}
