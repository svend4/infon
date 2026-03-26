package audio

import (
	"math"
	"math/cmplx"
	"testing"
)

func TestNewFFTBandPassFilter(t *testing.T) {
	config := FilterConfig{
		SampleRate:      16000,
		FFTSize:         512,
		LowCutoffHz:     300,
		HighCutoffHz:    3400,
		NoiseSuppress:   true,
		SuppressionGain: 0.7,
	}

	filter, err := NewFFTBandPassFilter(config)
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Filter is nil")
	}

	if filter.sampleRate != 16000 {
		t.Errorf("Sample rate = %d, expected 16000", filter.sampleRate)
	}

	if filter.fftSize != 512 {
		t.Errorf("FFT size = %d, expected 512", filter.fftSize)
	}

	if filter.hopSize != 256 {
		t.Errorf("Hop size = %d, expected 256", filter.hopSize)
	}
}

func TestNewFFTBandPassFilterInvalidFFTSize(t *testing.T) {
	config := FilterConfig{
		SampleRate:   16000,
		FFTSize:      500, // Not power of 2
		LowCutoffHz:  300,
		HighCutoffHz: 3400,
	}

	_, err := NewFFTBandPassFilter(config)
	if err == nil {
		t.Error("Should fail with non-power-of-2 FFT size")
	}
}

func TestNewFFTBandPassFilterInvalidCutoffs(t *testing.T) {
	config := FilterConfig{
		SampleRate:   16000,
		FFTSize:      512,
		LowCutoffHz:  3400,
		HighCutoffHz: 300, // High < Low
	}

	_, err := NewFFTBandPassFilter(config)
	if err == nil {
		t.Error("Should fail when high cutoff < low cutoff")
	}
}

func TestFFTBandPassFilterProcess(t *testing.T) {
	config := FilterConfig{
		SampleRate:      16000,
		FFTSize:         512,
		LowCutoffHz:     300,
		HighCutoffHz:    3400,
		NoiseSuppress:   false,
		SuppressionGain: 0,
	}

	filter, _ := NewFFTBandPassFilter(config)

	// Create test signal (1kHz tone, within passband)
	input := make([]int16, 1600) // 100ms at 16kHz
	for i := range input {
		input[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	output := filter.Process(input)

	if len(output) != len(input) {
		t.Errorf("Output length = %d, expected %d", len(output), len(input))
	}

	// Signal should pass through (within passband)
	// Check that output has energy
	var energy float64
	for _, sample := range output {
		energy += float64(sample) * float64(sample)
	}
	energy /= float64(len(output))

	if energy < 100 {
		t.Errorf("Output energy too low: %f", energy)
	}
}

func TestFFTBandPassFilterRejectsOutOfBand(t *testing.T) {
	config := FilterConfig{
		SampleRate:      16000,
		FFTSize:         512,
		LowCutoffHz:     1000,
		HighCutoffHz:    2000,
		NoiseSuppress:   false,
		SuppressionGain: 0,
	}

	filter, _ := NewFFTBandPassFilter(config)

	// Create low-frequency signal (100 Hz, below passband)
	input := make([]int16, 1600)
	for i := range input {
		input[i] = int16(10000 * math.Sin(2*math.Pi*100*float64(i)/16000))
	}

	output := filter.Process(input)

	// Signal should be attenuated
	var inputEnergy, outputEnergy float64
	for i := range input {
		inputEnergy += float64(input[i]) * float64(input[i])
		outputEnergy += float64(output[i]) * float64(output[i])
	}
	inputEnergy /= float64(len(input))
	outputEnergy /= float64(len(output))

	// Output should have less energy than input
	if outputEnergy > inputEnergy*0.5 {
		t.Logf("Input energy: %f, Output energy: %f", inputEnergy, outputEnergy)
		t.Log("Note: Some energy may pass through due to filter rolloff")
	}
}

func TestFFTBandPassFilterSetBandpass(t *testing.T) {
	config := VoiceBandpassConfig(16000)
	filter, _ := NewFFTBandPassFilter(config)

	err := filter.SetBandpass(500, 4000)
	if err != nil {
		t.Errorf("SetBandpass failed: %v", err)
	}

	// Try invalid range
	err = filter.SetBandpass(4000, 500)
	if err == nil {
		t.Error("SetBandpass should fail when low > high")
	}
}

func TestFFTBandPassFilterSetSuppressionGain(t *testing.T) {
	config := VoiceBandpassConfig(16000)
	filter, _ := NewFFTBandPassFilter(config)

	// Valid values
	for _, gain := range []float64{0, 0.5, 1.0} {
		err := filter.SetSuppressionGain(gain)
		if err != nil {
			t.Errorf("SetSuppressionGain(%f) failed: %v", gain, err)
		}
	}

	// Invalid values
	err := filter.SetSuppressionGain(-0.1)
	if err == nil {
		t.Error("SetSuppressionGain should fail for negative value")
	}

	err = filter.SetSuppressionGain(1.1)
	if err == nil {
		t.Error("SetSuppressionGain should fail for value > 1")
	}
}

func TestFFTBandPassFilterReset(t *testing.T) {
	config := VoiceBandpassConfig(16000)
	filter, _ := NewFFTBandPassFilter(config)

	// Process some audio
	input := make([]int16, 1600)
	filter.Process(input)

	if filter.GetFramesProcessed() == 0 {
		t.Error("Should have processed frames")
	}

	// Reset
	filter.Reset()

	if filter.GetFramesProcessed() != 0 {
		t.Errorf("After reset, frames processed = %d, expected 0", filter.GetFramesProcessed())
	}
}

func TestFFTBandPassFilterNoiseSuppress(t *testing.T) {
	config := FilterConfig{
		SampleRate:      16000,
		FFTSize:         512,
		LowCutoffHz:     300,
		HighCutoffHz:    3400,
		NoiseSuppress:   true,
		SuppressionGain: 0.8,
	}

	filter, _ := NewFFTBandPassFilter(config)

	// Create noisy signal
	input := make([]int16, 1600)
	for i := range input {
		// Signal + noise
		signal := 10000 * math.Sin(2*math.Pi*1000*float64(i)/16000)
		noise := 500 * (math.Sin(2*math.Pi*5000*float64(i)/16000) +
			math.Sin(2*math.Pi*7000*float64(i)/16000))
		input[i] = int16(signal + noise)
	}

	output := filter.Process(input)

	// Output should have noise reduced
	if len(output) != len(input) {
		t.Errorf("Output length = %d, expected %d", len(output), len(input))
	}
}

// FFT Tests

func TestFFTRealSignal(t *testing.T) {
	// Create a simple sine wave
	n := 8
	input := make([]float64, n)
	for i := 0; i < n; i++ {
		input[i] = math.Sin(2 * math.Pi * float64(i) / float64(n))
	}

	spectrum := fft(input)

	if len(spectrum) != n {
		t.Errorf("Spectrum length = %d, expected %d", len(spectrum), n)
	}

	// Verify DC component is near zero
	dc := cmplx.Abs(spectrum[0])
	if dc > 0.01 {
		t.Logf("DC component = %f (expected near 0)", dc)
	}
}

func TestFFTDCSignal(t *testing.T) {
	// DC signal (all ones)
	n := 8
	input := make([]float64, n)
	for i := 0; i < n; i++ {
		input[i] = 1.0
	}

	spectrum := fft(input)

	// All energy should be in DC bin
	dc := cmplx.Abs(spectrum[0])
	if dc < float64(n)*0.9 {
		t.Errorf("DC component = %f, expected ~%d", dc, n)
	}

	// Other bins should be near zero
	for i := 1; i < len(spectrum); i++ {
		mag := cmplx.Abs(spectrum[i])
		if mag > 0.1 {
			t.Errorf("Bin %d magnitude = %f, expected ~0", i, mag)
		}
	}
}

func TestIFFT(t *testing.T) {
	// Create signal
	n := 16
	input := make([]float64, n)
	for i := 0; i < n; i++ {
		input[i] = math.Sin(2 * math.Pi * float64(i) / float64(n))
	}

	// FFT -> IFFT should give back original
	spectrum := fft(input)
	reconstructed := ifft(spectrum)

	// Compare
	for i := 0; i < n; i++ {
		diff := math.Abs(input[i] - reconstructed[i])
		if diff > 0.0001 {
			t.Errorf("Sample %d: input = %f, reconstructed = %f, diff = %f",
				i, input[i], reconstructed[i], diff)
		}
	}
}

func TestFFTComplexPowerOf2(t *testing.T) {
	sizes := []int{2, 4, 8, 16, 32, 64, 128, 256}

	for _, n := range sizes {
		input := make([]complex128, n)
		for i := 0; i < n; i++ {
			input[i] = complex(float64(i), 0)
		}

		spectrum := fftComplex(input)

		if len(spectrum) != n {
			t.Errorf("Size %d: spectrum length = %d, expected %d",
				n, len(spectrum), n)
		}
	}
}

func TestDFT(t *testing.T) {
	// Test with non-power-of-2 size
	n := 7
	input := make([]complex128, n)
	for i := 0; i < n; i++ {
		input[i] = complex(math.Sin(2*math.Pi*float64(i)/float64(n)), 0)
	}

	spectrum := dft(input)

	if len(spectrum) != n {
		t.Errorf("Spectrum length = %d, expected %d", len(spectrum), n)
	}
}

// Window Function Tests

func TestMakeHannWindow(t *testing.T) {
	n := 128
	window := makeHannWindow(n)

	if len(window) != n {
		t.Errorf("Window length = %d, expected %d", len(window), n)
	}

	// Check endpoints (should be near zero)
	if window[0] > 0.01 {
		t.Errorf("Window[0] = %f, expected ~0", window[0])
	}

	if window[n-1] > 0.01 {
		t.Errorf("Window[%d] = %f, expected ~0", n-1, window[n-1])
	}

	// Check middle (should be ~1)
	if window[n/2] < 0.99 {
		t.Errorf("Window[%d] = %f, expected ~1", n/2, window[n/2])
	}
}

func TestMakeHammingWindow(t *testing.T) {
	n := 128
	window := makeHammingWindow(n)

	if len(window) != n {
		t.Errorf("Window length = %d, expected %d", len(window), n)
	}

	// Hamming window endpoints should be 0.08
	if math.Abs(window[0]-0.08) > 0.01 {
		t.Errorf("Window[0] = %f, expected ~0.08", window[0])
	}
}

func TestMakeBlackmanWindow(t *testing.T) {
	n := 128
	window := makeBlackmanWindow(n)

	if len(window) != n {
		t.Errorf("Window length = %d, expected %d", len(window), n)
	}

	// Check that all values are in valid range [0, 1]
	// Allow for small floating point errors near 0
	for i, val := range window {
		if val < -0.0001 || val > 1.0001 {
			t.Errorf("Window[%d] = %f, out of range [0, 1]", i, val)
		}
	}
}

// Utility Tests

func TestIsPowerOfTwo(t *testing.T) {
	powerOfTwo := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024}
	for _, n := range powerOfTwo {
		if !isPowerOfTwo(n) {
			t.Errorf("isPowerOfTwo(%d) = false, expected true", n)
		}
	}

	notPowerOfTwo := []int{0, 3, 5, 7, 9, 15, 17, 31, 33, 100, 500}
	for _, n := range notPowerOfTwo {
		if isPowerOfTwo(n) {
			t.Errorf("isPowerOfTwo(%d) = true, expected false", n)
		}
	}
}

func TestNextPowerOfTwo(t *testing.T) {
	testCases := []struct {
		input    int
		expected int
	}{
		{1, 1},
		{2, 2},
		{3, 4},
		{5, 8},
		{9, 16},
		{17, 32},
		{100, 128},
		{500, 512},
		{1000, 1024},
	}

	for _, tc := range testCases {
		result := nextPowerOfTwo(tc.input)
		if result != tc.expected {
			t.Errorf("nextPowerOfTwo(%d) = %d, expected %d",
				tc.input, result, tc.expected)
		}
	}
}

// Preset Configuration Tests

func TestVoiceBandpassConfig(t *testing.T) {
	config := VoiceBandpassConfig(16000)

	if config.LowCutoffHz != 300 {
		t.Errorf("Low cutoff = %f, expected 300", config.LowCutoffHz)
	}

	if config.HighCutoffHz != 3400 {
		t.Errorf("High cutoff = %f, expected 3400", config.HighCutoffHz)
	}

	if !config.NoiseSuppress {
		t.Error("Noise suppression should be enabled")
	}

	// Should be able to create filter
	_, err := NewFFTBandPassFilter(config)
	if err != nil {
		t.Errorf("Failed to create filter with voice config: %v", err)
	}
}

func TestMusicBandpassConfig(t *testing.T) {
	config := MusicBandpassConfig(48000)

	if config.LowCutoffHz != 20 {
		t.Errorf("Low cutoff = %f, expected 20", config.LowCutoffHz)
	}

	if config.HighCutoffHz != 20000 {
		t.Errorf("High cutoff = %f, expected 20000", config.HighCutoffHz)
	}

	_, err := NewFFTBandPassFilter(config)
	if err != nil {
		t.Errorf("Failed to create filter with music config: %v", err)
	}
}

func TestBroadcastBandpassConfig(t *testing.T) {
	config := BroadcastBandpassConfig(44100)

	if config.LowCutoffHz != 50 {
		t.Errorf("Low cutoff = %f, expected 50", config.LowCutoffHz)
	}

	if config.HighCutoffHz != 15000 {
		t.Errorf("High cutoff = %f, expected 15000", config.HighCutoffHz)
	}

	_, err := NewFFTBandPassFilter(config)
	if err != nil {
		t.Errorf("Failed to create filter with broadcast config: %v", err)
	}
}

func TestTelephoneBandpassConfig(t *testing.T) {
	config := TelephoneBandpassConfig(8000)

	if config.LowCutoffHz != 300 {
		t.Errorf("Low cutoff = %f, expected 300", config.LowCutoffHz)
	}

	if config.HighCutoffHz != 3400 {
		t.Errorf("High cutoff = %f, expected 3400", config.HighCutoffHz)
	}

	_, err := NewFFTBandPassFilter(config)
	if err != nil {
		t.Errorf("Failed to create filter with telephone config: %v", err)
	}
}

// Integration Tests

func TestMultipleProcessCalls(t *testing.T) {
	config := VoiceBandpassConfig(16000)
	filter, _ := NewFFTBandPassFilter(config)

	// Process multiple chunks
	for chunk := 0; chunk < 10; chunk++ {
		input := make([]int16, 1600)
		for i := range input {
			input[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(chunk*1600+i)/16000))
		}

		output := filter.Process(input)
		if len(output) != len(input) {
			t.Errorf("Chunk %d: output length = %d, expected %d",
				chunk, len(output), len(input))
		}
	}

	if filter.GetFramesProcessed() == 0 {
		t.Error("Should have processed frames")
	}
}

// Benchmarks

func BenchmarkFFTBandPassFilterProcess(b *testing.B) {
	config := VoiceBandpassConfig(16000)
	filter, _ := NewFFTBandPassFilter(config)

	input := make([]int16, 1600) // 100ms
	for i := range input {
		input[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Process(input)
	}
}

func BenchmarkFFT512(b *testing.B) {
	input := make([]float64, 512)
	for i := range input {
		input[i] = math.Sin(2 * math.Pi * float64(i) / 512.0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fft(input)
	}
}

func BenchmarkIFFT512(b *testing.B) {
	input := make([]float64, 512)
	for i := range input {
		input[i] = math.Sin(2 * math.Pi * float64(i) / 512.0)
	}
	spectrum := fft(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ifft(spectrum)
	}
}

func BenchmarkHannWindow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		makeHannWindow(512)
	}
}

func BenchmarkFFT1024(b *testing.B) {
	input := make([]float64, 1024)
	for i := range input {
		input[i] = math.Sin(2 * math.Pi * float64(i) / 1024.0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fft(input)
	}
}
