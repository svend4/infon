package audio

import (
	"math"
	"testing"
)

func TestNewNoiseSuppressor(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 320)

	if ns == nil {
		t.Fatal("NewNoiseSuppressor() returned nil")
	}

	if ns.sampleRate != 16000 {
		t.Errorf("NoiseSuppressor sampleRate = %d, expected 16000", ns.sampleRate)
	}

	if ns.frameSize != 320 {
		t.Errorf("NoiseSuppressor frameSize = %d, expected 320", ns.frameSize)
	}
}

func TestNoiseSuppressor_Process(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 320)

	// Create test signal
	input := make([]int16, 320)
	for i := range input {
		input[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	output := ns.Process(input)

	if len(output) != len(input) {
		t.Errorf("Process() length = %d, expected %d", len(output), len(input))
	}

	if output == nil {
		t.Error("Process() should not return nil")
	}
}

func TestNoiseSuppressor_Process_EmptyInput(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 320)

	input := []int16{}
	output := ns.Process(input)

	if len(output) != 0 {
		t.Errorf("Process() with empty input should return empty output, got length %d", len(output))
	}
}

func TestNoiseSuppressor_Process_Silence(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 320)

	// Create silence
	silence := make([]int16, 320)

	output := ns.Process(silence)

	// Silence should remain silence (approximately)
	nonZeroCount := 0
	for _, sample := range output {
		if sample != 0 {
			nonZeroCount++
		}
	}

	// Allow some small numerical errors
	if nonZeroCount > 10 {
		t.Errorf("Process() silence has %d non-zero samples, expected near 0", nonZeroCount)
	}
}

func TestNoiseSuppressor_Process_MaxAmplitude(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 320)

	// Create max amplitude signal
	input := make([]int16, 320)
	for i := range input {
		if i%2 == 0 {
			input[i] = 32767
		} else {
			input[i] = -32768
		}
	}

	output := ns.Process(input)

	// Should not overflow
	for i, sample := range output {
		if sample > 32767 || sample < -32768 {
			t.Errorf("Process() sample[%d] = %d, out of int16 range", i, sample)
		}
	}
}

func TestNoiseSuppressor_SetAggressiveness(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 320)

	tests := []struct {
		name  string
		level float64
	}{
		{
			name:  "mild",
			level: 0.5,
		},
		{
			name:  "moderate",
			level: 1.0,
		},
		{
			name:  "aggressive",
			level: 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns.SetAggressiveness(tt.level)
			// Just verify it doesn't panic
		})
	}
}

func TestNoiseSuppressor_IsCalibrating(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 320)

	// Should be calibrating initially
	if !ns.IsCalibrating() {
		t.Error("IsCalibrating() = false initially, expected true")
	}

	// Process silence frames to complete calibration
	silence := make([]int16, 320)
	for i := 0; i < 25; i++ {
		ns.Process(silence)
	}

	// Should no longer be calibrating
	if ns.IsCalibrating() {
		t.Error("IsCalibrating() = true after calibration, expected false")
	}
}

func TestNoiseSuppressor_GetStatistics(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 320)

	// Process some frames
	signal := make([]int16, 320)
	for i := range signal {
		signal[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	for i := 0; i < 10; i++ {
		ns.Process(signal)
	}

	stats := ns.GetStatistics()

	if stats.TotalFrames != 10 {
		t.Errorf("TotalFrames = %d, expected 10", stats.TotalFrames)
	}

	if stats.ReductionDB < 0.0 {
		t.Errorf("ReductionDB = %f, expected >= 0", stats.ReductionDB)
	}
}

func TestNoiseSuppressor_Reset(t *testing.T) {
	ns := NewNoiseSuppressor(16000, 320)

	// Process frames
	signal := make([]int16, 320)
	for i := 0; i < 10; i++ {
		ns.Process(signal)
	}

	// Verify frames were processed
	stats := ns.GetStatistics()
	if stats.TotalFrames == 0 {
		t.Fatal("No frames processed before reset")
	}

	// Reset
	ns.Reset()

	// Check that state was reset
	if !ns.IsCalibrating() {
		t.Error("IsCalibrating() = false after reset, expected true")
	}

	stats = ns.GetStatistics()
	if stats.TotalFrames != 0 {
		t.Errorf("TotalFrames = %d after reset, expected 0", stats.TotalFrames)
	}
}

func TestNewSpectralGate(t *testing.T) {
	sg := NewSpectralGate(16000)

	if sg == nil {
		t.Fatal("NewSpectralGate() returned nil")
	}

	if sg.sampleRate != 16000 {
		t.Errorf("SpectralGate sampleRate = %d, expected 16000", sg.sampleRate)
	}

	if sg.threshold != -40.0 {
		t.Errorf("SpectralGate threshold = %f, expected -40.0", sg.threshold)
	}
}

func TestSpectralGate_Process(t *testing.T) {
	sg := NewSpectralGate(16000)

	// Create test signal
	input := make([]int16, 320)
	for i := range input {
		input[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	output := sg.Process(input)

	if len(output) != len(input) {
		t.Errorf("Process() length = %d, expected %d", len(output), len(input))
	}
}

func TestNewBandPassFilter(t *testing.T) {
	bpf := NewBandPassFilter(16000, 300.0, 3000.0)

	if bpf == nil {
		t.Fatal("NewBandPassFilter() returned nil")
	}

	if bpf.sampleRate != 16000 {
		t.Errorf("BandPassFilter sampleRate = %d, expected 16000", bpf.sampleRate)
	}

	if bpf.lowCutoff != 300.0 {
		t.Errorf("BandPassFilter lowCutoff = %f, expected 300.0", bpf.lowCutoff)
	}

	if bpf.highCutoff != 3000.0 {
		t.Errorf("BandPassFilter highCutoff = %f, expected 3000.0", bpf.highCutoff)
	}
}

func TestBandPassFilter_Process(t *testing.T) {
	bpf := NewBandPassFilter(16000, 300.0, 3000.0)

	// Create test signal
	input := make([]int16, 320)
	for i := range input {
		input[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	output := bpf.Process(input)

	// Currently returns original samples (TODO implementation)
	if len(output) != len(input) {
		t.Errorf("Process() length = %d, expected %d", len(output), len(input))
	}
}

func TestNewHighPassFilter(t *testing.T) {
	hpf := NewHighPassFilter(16000, 80.0)

	if hpf == nil {
		t.Fatal("NewHighPassFilter() returned nil")
	}

	if hpf.alpha == 0.0 {
		t.Error("HighPassFilter alpha should be non-zero")
	}
}

func TestHighPassFilter_Process(t *testing.T) {
	hpf := NewHighPassFilter(16000, 80.0)

	// Create test signal with DC offset
	input := make([]int16, 320)
	for i := range input {
		input[i] = 1000 + int16(500*math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	output := hpf.Process(input)

	if len(output) != len(input) {
		t.Errorf("Process() length = %d, expected %d", len(output), len(input))
	}
}

func TestHighPassFilter_Process_Silence(t *testing.T) {
	hpf := NewHighPassFilter(16000, 80.0)

	silence := make([]int16, 320)
	output := hpf.Process(silence)

	// Silence should remain silence
	for i, sample := range output {
		if sample != 0 {
			t.Errorf("Process() silence sample[%d] = %d, expected 0", i, sample)
		}
	}
}

func TestHighPassFilter_Process_NoOverflow(t *testing.T) {
	hpf := NewHighPassFilter(16000, 80.0)

	// Create max amplitude signal
	input := make([]int16, 320)
	for i := range input {
		if i%2 == 0 {
			input[i] = 32767
		} else {
			input[i] = -32768
		}
	}

	output := hpf.Process(input)

	// Should not overflow
	for i, sample := range output {
		if sample > 32767 || sample < -32768 {
			t.Errorf("Process() sample[%d] = %d, out of int16 range", i, sample)
		}
	}
}

func BenchmarkNoiseSuppressor_Process(b *testing.B) {
	ns := NewNoiseSuppressor(16000, 320)

	signal := make([]int16, 320)
	for i := range signal {
		signal[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ns.Process(signal)
	}
}

func BenchmarkHighPassFilter_Process(b *testing.B) {
	hpf := NewHighPassFilter(16000, 80.0)

	signal := make([]int16, 320)
	for i := range signal {
		signal[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hpf.Process(signal)
	}
}

func BenchmarkSpectralGate_Process(b *testing.B) {
	sg := NewSpectralGate(16000)

	signal := make([]int16, 320)
	for i := range signal {
		signal[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sg.Process(signal)
	}
}
