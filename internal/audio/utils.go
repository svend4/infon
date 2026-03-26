package audio

import (
	"fmt"
	"math"
)

// ConvertSampleRate converts audio samples from one sample rate to another
// using simple linear interpolation
func ConvertSampleRate(samples []int16, fromRate, toRate int) []int16 {
	if fromRate == toRate {
		return samples
	}

	ratio := float64(fromRate) / float64(toRate)
	newLength := int(float64(len(samples)) / ratio)
	result := make([]int16, newLength)

	for i := range result {
		srcPos := float64(i) * ratio
		srcIdx := int(srcPos)

		if srcIdx >= len(samples)-1 {
			result[i] = samples[len(samples)-1]
			continue
		}

		// Linear interpolation
		frac := srcPos - float64(srcIdx)
		sample1 := float64(samples[srcIdx])
		sample2 := float64(samples[srcIdx+1])
		interpolated := sample1 + (sample2-sample1)*frac

		result[i] = int16(interpolated)
	}

	return result
}

// ConvertMonoToStereo converts mono audio to stereo by duplicating channels
func ConvertMonoToStereo(mono []int16) []int16 {
	stereo := make([]int16, len(mono)*2)
	for i, sample := range mono {
		stereo[i*2] = sample     // Left
		stereo[i*2+1] = sample   // Right
	}
	return stereo
}

// ConvertStereoToMono converts stereo audio to mono by averaging channels
func ConvertStereoToMono(stereo []int16) []int16 {
	if len(stereo)%2 != 0 {
		// Invalid stereo data
		return stereo
	}

	mono := make([]int16, len(stereo)/2)
	for i := range mono {
		left := int32(stereo[i*2])
		right := int32(stereo[i*2+1])
		mono[i] = int16((left + right) / 2)
	}
	return mono
}

// NormalizeSamples normalizes audio samples to maximize volume
// without clipping
func NormalizeSamples(samples []int16) []int16 {
	if len(samples) == 0 {
		return samples
	}

	// Find peak amplitude
	var peak int16
	for _, sample := range samples {
		abs := sample
		if abs < 0 {
			abs = -abs
		}
		if abs > peak {
			peak = abs
		}
	}

	// If already at max or silent, no normalization needed
	if peak == 0 || peak >= 32000 {
		return samples
	}

	// Calculate gain factor
	gain := 32000.0 / float64(peak)

	// Apply gain
	result := make([]int16, len(samples))
	for i, sample := range samples {
		normalized := float64(sample) * gain
		if normalized > 32767 {
			normalized = 32767
		}
		if normalized < -32768 {
			normalized = -32768
		}
		result[i] = int16(normalized)
	}

	return result
}

// ApplyGain applies a gain factor to audio samples
// gain > 1.0 amplifies, gain < 1.0 attenuates
func ApplyGain(samples []int16, gain float64) []int16 {
	if gain == 1.0 {
		return samples
	}

	result := make([]int16, len(samples))
	for i, sample := range samples {
		amplified := float64(sample) * gain
		if amplified > 32767 {
			amplified = 32767
		}
		if amplified < -32768 {
			amplified = -32768
		}
		result[i] = int16(amplified)
	}

	return result
}

// CalculateRMS calculates the Root Mean Square (RMS) of audio samples
// Returns value in range [0, 32768]
func CalculateRMS(samples []int16) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	var sum float64
	for _, sample := range samples {
		f := float64(sample)
		sum += f * f
	}

	return math.Sqrt(sum / float64(len(samples)))
}

// CalculatePeak calculates the peak amplitude of audio samples
// Returns value in range [0, 32768]
func CalculatePeak(samples []int16) int16 {
	var peak int16
	for _, sample := range samples {
		abs := sample
		if abs < 0 {
			abs = -abs
		}
		if abs > peak {
			peak = abs
		}
	}
	return peak
}

// IsSilence checks if audio samples are effectively silent
// threshold is the maximum absolute value considered silent (typically 100-500)
func IsSilence(samples []int16, threshold int16) bool {
	for _, sample := range samples {
		abs := sample
		if abs < 0 {
			abs = -abs
		}
		if abs > threshold {
			return false
		}
	}
	return true
}

// GenerateSilence generates silent audio samples
func GenerateSilence(length int) []int16 {
	return make([]int16, length)
}

// GenerateTone generates a sine wave tone
// frequency in Hz, sampleRate in Hz, duration in samples
func GenerateTone(frequency float64, sampleRate int, length int) []int16 {
	samples := make([]int16, length)
	for i := range samples {
		t := float64(i) / float64(sampleRate)
		sample := math.Sin(2 * math.Pi * frequency * t)
		samples[i] = int16(sample * 32000)
	}
	return samples
}

// MixSamples mixes two audio sample arrays with optional gain for each
// Returns mixed samples with soft clipping to prevent overflow
func MixSamples(samples1, samples2 []int16, gain1, gain2 float64) []int16 {
	maxLen := len(samples1)
	if len(samples2) > maxLen {
		maxLen = len(samples2)
	}

	result := make([]int16, maxLen)

	for i := 0; i < maxLen; i++ {
		var sample1, sample2 float64

		if i < len(samples1) {
			sample1 = float64(samples1[i]) * gain1
		}
		if i < len(samples2) {
			sample2 = float64(samples2[i]) * gain2
		}

		// Mix and apply soft clipping
		mixed := sample1 + sample2

		// Soft clipping using tanh
		if math.Abs(mixed) > 16384 {
			mixed = math.Tanh(mixed/32768.0) * 32768.0
		}

		// Clamp to int16 range
		if mixed > 32767 {
			mixed = 32767
		}
		if mixed < -32768 {
			mixed = -32768
		}

		result[i] = int16(mixed)
	}

	return result
}

// Resample resamples audio using higher quality interpolation
// This is a simple implementation - for production use, consider
// a proper resampler library
func Resample(samples []int16, fromRate, toRate int) []int16 {
	if fromRate == toRate {
		return samples
	}

	// Use ConvertSampleRate for now
	// TODO: Implement higher quality resampling (e.g., sinc interpolation)
	return ConvertSampleRate(samples, fromRate, toRate)
}

// FormatToString returns a human-readable string for audio format
func FormatToString(format AudioFormat) string {
	return fmt.Sprintf("%d Hz, %d-bit, %d channel(s)",
		format.SampleRate,
		format.BitDepth,
		format.Channels)
}

// ValidateFormat checks if an audio format is valid
func ValidateFormat(format AudioFormat) error {
	if format.SampleRate <= 0 {
		return fmt.Errorf("invalid sample rate: %d", format.SampleRate)
	}
	if format.Channels <= 0 || format.Channels > 8 {
		return fmt.Errorf("invalid channels: %d", format.Channels)
	}
	if format.BitDepth != 8 && format.BitDepth != 16 && format.BitDepth != 24 && format.BitDepth != 32 {
		return fmt.Errorf("invalid bit depth: %d", format.BitDepth)
	}
	return nil
}

// CalculateBufferDuration calculates buffer duration in milliseconds
func CalculateBufferDuration(bufferSize int, format AudioFormat) float64 {
	if format.SampleRate == 0 {
		return 0
	}
	return float64(bufferSize) / float64(format.SampleRate) * 1000.0
}

// CalculateBufferSize calculates buffer size from duration
func CalculateBufferSize(durationMs float64, format AudioFormat) int {
	return int(float64(format.SampleRate) * durationMs / 1000.0)
}
