package group

import (
	"math"
	"sync"
)

// AudioMixer mixes audio from multiple sources
type AudioMixer struct {
	sampleRate     int
	bufferSize     int
	sources        map[string][]int16 // Map of source ID to audio samples
	mutex          sync.RWMutex
	maxAmplitude   int16 // Maximum amplitude to prevent clipping
}

// NewAudioMixer creates a new audio mixer
func NewAudioMixer(sampleRate, bufferSize int) *AudioMixer {
	return &AudioMixer{
		sampleRate:   sampleRate,
		bufferSize:   bufferSize,
		sources:      make(map[string][]int16),
		maxAmplitude: 32767, // Max int16 value
	}
}

// AddSource adds an audio source
func (am *AudioMixer) AddSource(sourceID string, samples []int16) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Pad or truncate to buffer size
	if len(samples) < am.bufferSize {
		padded := make([]int16, am.bufferSize)
		copy(padded, samples)
		am.sources[sourceID] = padded
	} else {
		am.sources[sourceID] = samples[:am.bufferSize]
	}
}

// RemoveSource removes an audio source
func (am *AudioMixer) RemoveSource(sourceID string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	delete(am.sources, sourceID)
}

// Mix mixes all audio sources into a single output buffer
func (am *AudioMixer) Mix() []int16 {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if len(am.sources) == 0 {
		return make([]int16, am.bufferSize)
	}

	// Single source - no mixing needed
	if len(am.sources) == 1 {
		for _, samples := range am.sources {
			result := make([]int16, len(samples))
			copy(result, samples)
			return result
		}
	}

	// Multiple sources - mix with normalization
	output := make([]int32, am.bufferSize) // Use int32 to prevent overflow

	// Sum all sources
	for _, samples := range am.sources {
		for i := 0; i < am.bufferSize && i < len(samples); i++ {
			output[i] += int32(samples[i])
		}
	}

	// Normalize and convert back to int16
	result := make([]int16, am.bufferSize)
	sourceCount := float64(len(am.sources))

	for i := 0; i < am.bufferSize; i++ {
		// Average the samples
		mixed := float64(output[i]) / sourceCount

		// Apply soft clipping to prevent harsh distortion
		mixed = am.softClip(mixed)

		// Clamp to int16 range
		if mixed > float64(am.maxAmplitude) {
			result[i] = am.maxAmplitude
		} else if mixed < float64(-am.maxAmplitude) {
			result[i] = -am.maxAmplitude
		} else {
			result[i] = int16(mixed)
		}
	}

	return result
}

// softClip applies soft clipping to prevent harsh distortion
func (am *AudioMixer) softClip(sample float64) float64 {
	// Soft clipping using tanh function
	// This creates a smooth transition when approaching clipping
	maxFloat := float64(am.maxAmplitude)

	if math.Abs(sample) > maxFloat*0.8 {
		// Apply soft clipping when amplitude exceeds 80% of max
		sign := 1.0
		if sample < 0 {
			sign = -1.0
		}
		normalized := math.Abs(sample) / maxFloat
		clipped := math.Tanh(normalized * 2.0)
		return sign * clipped * maxFloat
	}

	return sample
}

// MixWithVolume mixes all sources with individual volume control
func (am *AudioMixer) MixWithVolume(volumes map[string]float64) []int16 {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	if len(am.sources) == 0 {
		return make([]int16, am.bufferSize)
	}

	output := make([]int32, am.bufferSize)

	// Sum all sources with volume
	for sourceID, samples := range am.sources {
		volume := 1.0
		if v, exists := volumes[sourceID]; exists {
			volume = v
		}

		for i := 0; i < am.bufferSize && i < len(samples); i++ {
			output[i] += int32(float64(samples[i]) * volume)
		}
	}

	// Normalize
	result := make([]int16, am.bufferSize)
	for i := 0; i < am.bufferSize; i++ {
		mixed := float64(output[i])
		mixed = am.softClip(mixed)

		if mixed > float64(am.maxAmplitude) {
			result[i] = am.maxAmplitude
		} else if mixed < float64(-am.maxAmplitude) {
			result[i] = -am.maxAmplitude
		} else {
			result[i] = int16(mixed)
		}
	}

	return result
}

// Clear clears all audio sources
func (am *AudioMixer) Clear() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.sources = make(map[string][]int16)
}

// SourceCount returns the number of audio sources
func (am *AudioMixer) SourceCount() int {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return len(am.sources)
}

// GetSourceIDs returns all source IDs
func (am *AudioMixer) GetSourceIDs() []string {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	ids := make([]string, 0, len(am.sources))
	for id := range am.sources {
		ids = append(ids, id)
	}
	return ids
}

// CalculatePeakLevel calculates the peak audio level (0.0 to 1.0)
func (am *AudioMixer) CalculatePeakLevel(samples []int16) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	maxSample := int16(0)
	for _, sample := range samples {
		abs := sample
		if abs < 0 {
			abs = -abs
		}
		if abs > maxSample {
			maxSample = abs
		}
	}

	return float64(maxSample) / float64(am.maxAmplitude)
}

// CalculateRMSLevel calculates the RMS audio level (0.0 to 1.0)
func (am *AudioMixer) CalculateRMSLevel(samples []int16) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	sum := float64(0)
	for _, sample := range samples {
		sum += float64(sample) * float64(sample)
	}

	rms := math.Sqrt(sum / float64(len(samples)))
	return rms / float64(am.maxAmplitude)
}
