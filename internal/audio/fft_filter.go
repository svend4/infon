package audio

import (
	"fmt"
	"math"
	"math/cmplx"
	"sync"
)

// FFTBandPassFilter implements frequency-domain bandpass filtering and noise suppression
// Uses FFT/IFFT for spectral analysis and filtering
type FFTBandPassFilter struct {
	mu sync.RWMutex

	// Configuration
	sampleRate   int
	fftSize      int
	hopSize      int
	lowCutoff    float64 // Hz
	highCutoff   float64 // Hz

	// FFT state
	inputBuffer  []float64
	outputBuffer []float64
	window       []float64
	overlapBuffer []float64

	// Noise suppression
	noiseProfile  []float64 // Estimated noise spectrum
	noiseAlpha    float64   // Noise estimation smoothing
	suppressionGain float64 // Amount of suppression (0-1)

	// Statistics
	framesProcessed uint64
}

// FilterConfig holds FFT filter configuration
type FilterConfig struct {
	SampleRate      int     // Sample rate in Hz
	FFTSize         int     // FFT size (power of 2)
	LowCutoffHz     float64 // Low cutoff frequency
	HighCutoffHz    float64 // High cutoff frequency
	NoiseSuppress   bool    // Enable spectral noise suppression
	SuppressionGain float64 // Suppression strength (0-1)
}

// NewFFTBandPassFilter creates a new FFT-based bandpass filter
func NewFFTBandPassFilter(config FilterConfig) (*FFTBandPassFilter, error) {
	// Validate FFT size is power of 2
	if !isPowerOfTwo(config.FFTSize) {
		return nil, fmt.Errorf("FFT size must be power of 2, got %d", config.FFTSize)
	}

	if config.FFTSize < 64 || config.FFTSize > 8192 {
		return nil, fmt.Errorf("FFT size must be 64-8192, got %d", config.FFTSize)
	}

	if config.LowCutoffHz >= config.HighCutoffHz {
		return nil, fmt.Errorf("low cutoff must be < high cutoff")
	}

	hopSize := config.FFTSize / 2 // 50% overlap

	filter := &FFTBandPassFilter{
		sampleRate:      config.SampleRate,
		fftSize:         config.FFTSize,
		hopSize:         hopSize,
		lowCutoff:       config.LowCutoffHz,
		highCutoff:      config.HighCutoffHz,
		inputBuffer:     make([]float64, config.FFTSize),
		outputBuffer:    make([]float64, config.FFTSize),
		window:          makeHannWindow(config.FFTSize),
		overlapBuffer:   make([]float64, config.FFTSize),
		noiseProfile:    make([]float64, config.FFTSize/2+1),
		noiseAlpha:      0.95,
		suppressionGain: config.SuppressionGain,
	}

	return filter, nil
}

// Process applies FFT bandpass filtering and noise suppression
func (f *FFTBandPassFilter) Process(input []int16) []int16 {
	f.mu.Lock()
	defer f.mu.Unlock()

	output := make([]int16, len(input))
	outputIdx := 0

	// Convert int16 to float64
	floatInput := make([]float64, len(input))
	for i, sample := range input {
		floatInput[i] = float64(sample)
	}

	// Process in frames with overlap
	for i := 0; i+f.hopSize <= len(floatInput); i += f.hopSize {
		// Copy input to buffer (handle end of input)
		if i+f.fftSize <= len(floatInput) {
			copy(f.inputBuffer, floatInput[i:i+f.fftSize])
		} else {
			// Partial frame at end - zero pad
			remaining := len(floatInput) - i
			copy(f.inputBuffer[:remaining], floatInput[i:])
			for j := remaining; j < f.fftSize; j++ {
				f.inputBuffer[j] = 0
			}
		}

		// Apply window
		for j := 0; j < f.fftSize; j++ {
			f.inputBuffer[j] *= f.window[j]
		}

		// Forward FFT
		spectrum := fft(f.inputBuffer)

		// Apply bandpass filter and noise suppression
		f.applySpectralFiltering(spectrum)

		// Inverse FFT
		filtered := ifft(spectrum)

		// Overlap-add
		for j := 0; j < f.fftSize; j++ {
			f.overlapBuffer[j] += filtered[j]
		}

		// Copy output
		copyLen := f.hopSize
		if outputIdx+copyLen > len(output) {
			copyLen = len(output) - outputIdx
		}

		for j := 0; j < copyLen; j++ {
			// Clamp to int16 range
			val := f.overlapBuffer[j]
			if val > 32767 {
				output[outputIdx] = 32767
			} else if val < -32768 {
				output[outputIdx] = -32768
			} else {
				output[outputIdx] = int16(val)
			}
			outputIdx++
		}

		// Shift overlap buffer
		copy(f.overlapBuffer, f.overlapBuffer[f.hopSize:])
		for j := f.fftSize - f.hopSize; j < f.fftSize; j++ {
			f.overlapBuffer[j] = 0
		}

		f.framesProcessed++
	}

	return output
}

// applySpectralFiltering applies bandpass filter and noise suppression in frequency domain
func (f *FFTBandPassFilter) applySpectralFiltering(spectrum []complex128) {
	nyquist := float64(f.sampleRate) / 2.0
	binWidth := nyquist / float64(len(spectrum)/2)

	// Process only positive frequencies (0 to N/2)
	// Negative frequencies (N/2+1 to N-1) are mirror of positive
	numBins := len(spectrum)/2 + 1

	for i := 0; i < numBins && i < len(f.noiseProfile); i++ {
		freq := float64(i) * binWidth

		// Bandpass filter
		var bandpassGain float64
		if freq >= f.lowCutoff && freq <= f.highCutoff {
			bandpassGain = 1.0
		} else {
			// Apply smooth rolloff
			if freq < f.lowCutoff {
				distance := (f.lowCutoff - freq) / f.lowCutoff
				bandpassGain = math.Max(0, 1.0-distance*2)
			} else {
				distance := (freq - f.highCutoff) / (nyquist - f.highCutoff)
				bandpassGain = math.Max(0, 1.0-distance*2)
			}
		}

		// Noise suppression
		magnitude := cmplx.Abs(spectrum[i])
		phase := cmplx.Phase(spectrum[i])

		// Update noise profile (during silence)
		if f.noiseProfile[i] == 0 || magnitude < f.noiseProfile[i]*1.5 {
			f.noiseProfile[i] = f.noiseAlpha*f.noiseProfile[i] + (1-f.noiseAlpha)*magnitude
		}

		// Apply spectral subtraction
		suppressedMag := magnitude - f.suppressionGain*f.noiseProfile[i]
		if suppressedMag < 0 {
			suppressedMag = 0 // Floor to zero
		}

		// Apply bandpass gain
		finalMag := suppressedMag * bandpassGain

		// Reconstruct complex value
		spectrum[i] = complex(finalMag*math.Cos(phase), finalMag*math.Sin(phase))

		// Apply same filter to negative frequency (mirror)
		if i > 0 && i < len(spectrum)-i {
			spectrum[len(spectrum)-i] = complex(finalMag*math.Cos(phase), finalMag*math.Sin(phase))
		}
	}
}

// SetBandpass updates the bandpass filter cutoff frequencies
func (f *FFTBandPassFilter) SetBandpass(lowHz, highHz float64) error {
	if lowHz >= highHz {
		return fmt.Errorf("low cutoff must be < high cutoff")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.lowCutoff = lowHz
	f.highCutoff = highHz
	return nil
}

// SetSuppressionGain sets the noise suppression strength (0-1)
func (f *FFTBandPassFilter) SetSuppressionGain(gain float64) error {
	if gain < 0 || gain > 1 {
		return fmt.Errorf("suppression gain must be 0-1, got %f", gain)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.suppressionGain = gain
	return nil
}

// Reset clears all filter state
func (f *FFTBandPassFilter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.inputBuffer {
		f.inputBuffer[i] = 0
	}
	for i := range f.outputBuffer {
		f.outputBuffer[i] = 0
	}
	for i := range f.overlapBuffer {
		f.overlapBuffer[i] = 0
	}
	for i := range f.noiseProfile {
		f.noiseProfile[i] = 0
	}
	f.framesProcessed = 0
}

// GetFramesProcessed returns number of frames processed
func (f *FFTBandPassFilter) GetFramesProcessed() uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.framesProcessed
}

// FFT Implementation (Cooley-Tukey radix-2 algorithm)

// fft performs Fast Fourier Transform
func fft(input []float64) []complex128 {
	n := len(input)
	if n == 1 {
		return []complex128{complex(input[0], 0)}
	}

	// Convert to complex
	x := make([]complex128, n)
	for i := range input {
		x[i] = complex(input[i], 0)
	}

	return fftComplex(x)
}

// fftComplex performs FFT on complex input
func fftComplex(x []complex128) []complex128 {
	n := len(x)

	if n == 1 {
		return x
	}

	if n%2 != 0 {
		// For non-power-of-2, use DFT
		return dft(x)
	}

	// Divide
	even := make([]complex128, n/2)
	odd := make([]complex128, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}

	// Conquer
	fftEven := fftComplex(even)
	fftOdd := fftComplex(odd)

	// Combine
	result := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		t := cmplx.Exp(complex(0, -2*math.Pi*float64(k)/float64(n))) * fftOdd[k]
		result[k] = fftEven[k] + t
		result[k+n/2] = fftEven[k] - t
	}

	return result
}

// ifft performs Inverse Fast Fourier Transform
func ifft(spectrum []complex128) []float64 {
	n := len(spectrum)

	// Conjugate
	conjugated := make([]complex128, n)
	for i := range spectrum {
		conjugated[i] = cmplx.Conj(spectrum[i])
	}

	// Forward FFT
	result := fftComplex(conjugated)

	// Conjugate and scale
	output := make([]float64, n)
	for i := range result {
		output[i] = real(cmplx.Conj(result[i])) / float64(n)
	}

	return output
}

// dft performs Discrete Fourier Transform (fallback for non-power-of-2)
func dft(x []complex128) []complex128 {
	n := len(x)
	result := make([]complex128, n)

	for k := 0; k < n; k++ {
		sum := complex(0, 0)
		for i := 0; i < n; i++ {
			angle := -2 * math.Pi * float64(k) * float64(i) / float64(n)
			sum += x[i] * cmplx.Exp(complex(0, angle))
		}
		result[k] = sum
	}

	return result
}

// Window Functions

// makeHannWindow creates a Hann window
func makeHannWindow(size int) []float64 {
	window := make([]float64, size)
	for i := 0; i < size; i++ {
		window[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(size-1)))
	}
	return window
}

// makeHammingWindow creates a Hamming window
func makeHammingWindow(size int) []float64 {
	window := make([]float64, size)
	for i := 0; i < size; i++ {
		window[i] = 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(size-1))
	}
	return window
}

// makeBlackmanWindow creates a Blackman window
func makeBlackmanWindow(size int) []float64 {
	window := make([]float64, size)
	a0 := 0.42
	a1 := 0.5
	a2 := 0.08

	for i := 0; i < size; i++ {
		window[i] = a0 - a1*math.Cos(2*math.Pi*float64(i)/float64(size-1)) +
			a2*math.Cos(4*math.Pi*float64(i)/float64(size-1))
	}
	return window
}

// Utility functions

// isPowerOfTwo checks if n is a power of 2
func isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// nextPowerOfTwo returns the next power of 2 >= n
func nextPowerOfTwo(n int) int {
	if isPowerOfTwo(n) {
		return n
	}

	power := 1
	for power < n {
		power *= 2
	}
	return power
}

// Preset Filter Configurations

// VoiceBandpassConfig returns a filter optimized for voice (300-3400 Hz)
func VoiceBandpassConfig(sampleRate int) FilterConfig {
	return FilterConfig{
		SampleRate:      sampleRate,
		FFTSize:         512,
		LowCutoffHz:     300,
		HighCutoffHz:    3400,
		NoiseSuppress:   true,
		SuppressionGain: 0.7,
	}
}

// MusicBandpassConfig returns a filter for full-range music (20-20000 Hz)
func MusicBandpassConfig(sampleRate int) FilterConfig {
	return FilterConfig{
		SampleRate:      sampleRate,
		FFTSize:         1024,
		LowCutoffHz:     20,
		HighCutoffHz:    20000,
		NoiseSuppress:   true,
		SuppressionGain: 0.3,
	}
}

// BroadcastBandpassConfig returns a filter for broadcast quality (50-15000 Hz)
func BroadcastBandpassConfig(sampleRate int) FilterConfig {
	return FilterConfig{
		SampleRate:      sampleRate,
		FFTSize:         1024,
		LowCutoffHz:     50,
		HighCutoffHz:    15000,
		NoiseSuppress:   true,
		SuppressionGain: 0.5,
	}
}

// TelephoneBandpassConfig returns a filter for telephone quality (300-3400 Hz)
func TelephoneBandpassConfig(sampleRate int) FilterConfig {
	return FilterConfig{
		SampleRate:      sampleRate,
		FFTSize:         256,
		LowCutoffHz:     300,
		HighCutoffHz:    3400,
		NoiseSuppress:   true,
		SuppressionGain: 0.8,
	}
}
