package audio

import (
	"math"
)

// NoiseSuppressor reduces background noise in audio using spectral subtraction
type NoiseSuppressor struct {
	sampleRate int
	frameSize  int

	// Noise profile
	noiseProfile []float64
	noiseEstimate []float64

	// Parameters
	alpha          float64 // Over-subtraction factor
	beta           float64 // Spectral floor
	smoothingAlpha float64 // Smoothing factor

	// State
	initialized    bool
	frameCount     int
	calibrationFrames int

	// Statistics
	totalFrames    uint64
	noisyFrames    uint64
	cleanFrames    uint64
}

// NewNoiseSuppressor creates a new noise suppressor
func NewNoiseSuppressor(sampleRate, frameSize int) *NoiseSuppressor {
	spectrumSize := frameSize/2 + 1

	return &NoiseSuppressor{
		sampleRate:        sampleRate,
		frameSize:         frameSize,
		noiseProfile:      make([]float64, spectrumSize),
		noiseEstimate:     make([]float64, spectrumSize),
		alpha:             2.0,  // Over-subtraction factor
		beta:              0.01, // Spectral floor (1%)
		smoothingAlpha:    0.9,  // Temporal smoothing
		initialized:       false,
		frameCount:        0,
		calibrationFrames: 20,   // ~400ms calibration @ 20ms frames
	}
}

// Process applies noise suppression to audio samples
func (ns *NoiseSuppressor) Process(samples []int16) []int16 {
	ns.frameCount++
	ns.totalFrames++

	// Convert int16 to float64
	floatSamples := make([]float64, len(samples))
	for i, s := range samples {
		floatSamples[i] = float64(s)
	}

	// Apply noise suppression
	enhanced := ns.spectralSubtraction(floatSamples)

	// Convert back to int16
	output := make([]int16, len(enhanced))
	for i, s := range enhanced {
		// Clamp to int16 range
		val := s
		if val > 32767.0 {
			val = 32767.0
		}
		if val < -32768.0 {
			val = -32768.0
		}
		output[i] = int16(val)
	}

	return output
}

// spectralSubtraction performs spectral subtraction noise reduction
func (ns *NoiseSuppressor) spectralSubtraction(samples []float64) []float64 {
	// Simple time-domain approach (for real-time performance)
	// Full spectral subtraction would require FFT

	// Estimate current frame energy
	energy := ns.calculateEnergy(samples)

	// Build/update noise profile during initial frames
	if ns.frameCount <= ns.calibrationFrames {
		ns.updateNoiseProfile(samples, energy)
		ns.noisyFrames++
		return samples // Return original during calibration
	}

	if !ns.initialized {
		ns.initialized = true
	}

	// Estimate noise level
	noiseLevel := ns.estimateNoiseLevel(samples)

	// Apply simple spectral subtraction
	enhanced := make([]float64, len(samples))

	for i, sample := range samples {
		// Simple gain-based reduction
		gain := ns.calculateGain(energy, noiseLevel)
		enhanced[i] = sample * gain
	}

	// Count frame types
	if energy > noiseLevel*2.0 {
		ns.cleanFrames++
	} else {
		ns.noisyFrames++
	}

	return enhanced
}

// calculateEnergy computes frame energy
func (ns *NoiseSuppressor) calculateEnergy(samples []float64) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	var sum float64
	for _, s := range samples {
		sum += s * s
	}

	return math.Sqrt(sum / float64(len(samples)))
}

// updateNoiseProfile updates noise estimation during calibration
func (ns *NoiseSuppressor) updateNoiseProfile(samples []float64, energy float64) {
	// Simple approach: average energy of initial frames
	alpha := 0.1
	for i := range ns.noiseProfile {
		idx := i * len(samples) / len(ns.noiseProfile)
		if idx < len(samples) {
			ns.noiseProfile[i] = ns.noiseProfile[i]*(1.0-alpha) + math.Abs(samples[idx])*alpha
		}
	}
}

// estimateNoiseLevel estimates current noise level
func (ns *NoiseSuppressor) estimateNoiseLevel(samples []float64) float64 {
	// Use calibrated noise profile
	var avgNoise float64
	for _, n := range ns.noiseProfile {
		avgNoise += n
	}
	if len(ns.noiseProfile) > 0 {
		avgNoise /= float64(len(ns.noiseProfile))
	}

	return avgNoise
}

// calculateGain computes suppression gain
func (ns *NoiseSuppressor) calculateGain(signalEnergy, noiseEnergy float64) float64 {
	if noiseEnergy <= 0.0 {
		return 1.0
	}

	// Signal-to-Noise Ratio
	snr := signalEnergy / noiseEnergy

	// Spectral subtraction gain
	var gain float64
	if snr > ns.alpha {
		// Clean signal: full gain
		gain = 1.0 - ns.alpha/snr
	} else {
		// Noisy signal: apply floor
		gain = ns.beta
	}

	// Apply smoothing
	gain = math.Max(gain, ns.beta)
	gain = math.Min(gain, 1.0)

	return gain
}

// SetAggressiveness adjusts noise suppression level (0.0 = off, 1.0 = max)
func (ns *NoiseSuppressor) SetAggressiveness(level float64) {
	// Clamp to 0.0-1.0
	if level < 0.0 {
		level = 0.0
	}
	if level > 1.0 {
		level = 1.0
	}

	// Adjust alpha (over-subtraction factor)
	ns.alpha = 1.0 + level*2.0 // 1.0-3.0

	// Adjust beta (spectral floor)
	ns.beta = 0.1 - level*0.09 // 0.01-0.10
}

// IsCalibrating returns true if still in calibration phase
func (ns *NoiseSuppressor) IsCalibrating() bool {
	return ns.frameCount <= ns.calibrationFrames
}

// GetStatistics returns noise suppression statistics
func (ns *NoiseSuppressor) GetStatistics() NoiseSuppressionStatistics {
	var cleanRatio float64
	if ns.totalFrames > 0 {
		cleanRatio = float64(ns.cleanFrames) / float64(ns.totalFrames) * 100.0
	}

	return NoiseSuppressionStatistics{
		TotalFrames:  ns.totalFrames,
		CleanFrames:  ns.cleanFrames,
		NoisyFrames:  ns.noisyFrames,
		CleanRatio:   cleanRatio,
		Calibrated:   ns.initialized,
	}
}

// Reset resets noise suppression state
func (ns *NoiseSuppressor) Reset() {
	ns.frameCount = 0
	ns.initialized = false

	// Clear noise profile
	for i := range ns.noiseProfile {
		ns.noiseProfile[i] = 0.0
	}
	for i := range ns.noiseEstimate {
		ns.noiseEstimate[i] = 0.0
	}
}

// NoiseSuppressionStatistics contains statistics
type NoiseSuppressionStatistics struct {
	TotalFrames uint64  // Total frames processed
	CleanFrames uint64  // Frames classified as clean
	NoisyFrames uint64  // Frames classified as noisy
	CleanRatio  float64 // Clean signal ratio (%)
	Calibrated  bool    // Calibration complete
}

// SimpleNoiseSuppressor provides basic noise gating
type SimpleNoiseSuppressor struct {
	threshold float64
}

// NewSimpleNoiseSuppressor creates a simple noise gate
func NewSimpleNoiseSuppressor(threshold float64) *SimpleNoiseSuppressor {
	return &SimpleNoiseSuppressor{
		threshold: threshold,
	}
}

// Process applies simple noise gating
func (sns *SimpleNoiseSuppressor) Process(samples []int16) []int16 {
	// Calculate RMS energy
	var sum float64
	for _, s := range samples {
		val := float64(s)
		sum += val * val
	}
	rms := math.Sqrt(sum / float64(len(samples)))

	// If below threshold, mute
	if rms < sns.threshold {
		output := make([]int16, len(samples))
		// Return silence
		return output
	}

	// Otherwise return original
	return samples
}

// BandPassFilter applies simple band-pass filtering for speech
type BandPassFilter struct {
	sampleRate int
	lowCutoff  float64 // Hz
	highCutoff float64 // Hz
}

// NewBandPassFilter creates a band-pass filter (e.g., 300-3400 Hz for speech)
func NewBandPassFilter(sampleRate int, lowCutoff, highCutoff float64) *BandPassFilter {
	return &BandPassFilter{
		sampleRate: sampleRate,
		lowCutoff:  lowCutoff,
		highCutoff: highCutoff,
	}
}

// Process applies band-pass filtering (simplified approach)
func (bpf *BandPassFilter) Process(samples []int16) []int16 {
	// Simplified: just apply a gain reduction for out-of-band frequencies
	// A real implementation would use FFT and frequency-domain filtering

	// For now, return original samples
	// TODO: Implement proper FFT-based filtering
	return samples
}

// HighPassFilter removes DC offset and very low frequencies
type HighPassFilter struct {
	prevInput  float64
	prevOutput float64
	alpha      float64
}

// NewHighPassFilter creates a high-pass filter
func NewHighPassFilter(sampleRate int, cutoffHz float64) *HighPassFilter {
	// Calculate alpha from cutoff frequency
	rc := 1.0 / (2.0 * math.Pi * cutoffHz)
	dt := 1.0 / float64(sampleRate)
	alpha := rc / (rc + dt)

	return &HighPassFilter{
		alpha: alpha,
	}
}

// Process applies high-pass filtering
func (hpf *HighPassFilter) Process(samples []int16) []int16 {
	output := make([]int16, len(samples))

	for i, sample := range samples {
		input := float64(sample)

		// High-pass filter equation
		filtered := hpf.alpha * (hpf.prevOutput + input - hpf.prevInput)

		hpf.prevInput = input
		hpf.prevOutput = filtered

		// Clamp and convert to int16
		if filtered > 32767.0 {
			filtered = 32767.0
		}
		if filtered < -32768.0 {
			filtered = -32768.0
		}

		output[i] = int16(filtered)
	}

	return output
}
