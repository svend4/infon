package audio

import (
	"math"
	"testing"
)

func TestConvertMonoToStereo(t *testing.T) {
	mono := []int16{100, 200, 300}
	stereo := ConvertMonoToStereo(mono)

	expected := []int16{100, 100, 200, 200, 300, 300}

	if len(stereo) != len(expected) {
		t.Fatalf("Length mismatch: got %d, expected %d", len(stereo), len(expected))
	}

	for i, v := range expected {
		if stereo[i] != v {
			t.Errorf("stereo[%d] = %d, expected %d", i, stereo[i], v)
		}
	}
}

func TestConvertStereoToMono(t *testing.T) {
	stereo := []int16{100, 200, 300, 400, 500, 600}
	mono := ConvertStereoToMono(stereo)

	expected := []int16{150, 350, 550} // Averages: (100+200)/2, (300+400)/2, (500+600)/2

	if len(mono) != len(expected) {
		t.Fatalf("Length mismatch: got %d, expected %d", len(mono), len(expected))
	}

	for i, v := range expected {
		if mono[i] != v {
			t.Errorf("mono[%d] = %d, expected %d", i, mono[i], v)
		}
	}
}

func TestNormalizeSamples(t *testing.T) {
	tests := []struct {
		name     string
		input    []int16
		checkPeak bool
	}{
		{
			name:     "quiet samples",
			input:    []int16{100, 200, 300},
			checkPeak: true,
		},
		{
			name:     "already loud",
			input:    []int16{30000, -30000},
			checkPeak: false,
		},
		{
			name:     "silence",
			input:    []int16{0, 0, 0},
			checkPeak: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := NormalizeSamples(tt.input)

			if len(normalized) != len(tt.input) {
				t.Errorf("Length mismatch: got %d, expected %d", len(normalized), len(tt.input))
			}

			if tt.checkPeak {
				peak := CalculatePeak(normalized)
				if peak < 10000 {
					t.Errorf("Peak after normalization too low: %d", peak)
				}
			}

			// Check no overflow
			for i, sample := range normalized {
				if sample > 32767 || sample < -32768 {
					t.Errorf("Sample[%d] = %d, out of range", i, sample)
				}
			}
		})
	}
}

func TestApplyGain(t *testing.T) {
	samples := []int16{1000, -1000, 2000}

	tests := []struct {
		name     string
		gain     float64
		expected []int16
	}{
		{
			name:     "unity gain",
			gain:     1.0,
			expected: []int16{1000, -1000, 2000},
		},
		{
			name:     "double gain",
			gain:     2.0,
			expected: []int16{2000, -2000, 4000},
		},
		{
			name:     "half gain",
			gain:     0.5,
			expected: []int16{500, -500, 1000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyGain(samples, tt.gain)

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("result[%d] = %d, expected %d", i, result[i], expected)
				}
			}
		})
	}
}

func TestApplyGain_NoOverflow(t *testing.T) {
	samples := []int16{30000, -30000}
	result := ApplyGain(samples, 2.0)

	// Should clamp to valid range
	for i, sample := range result {
		if sample > 32767 {
			t.Errorf("result[%d] = %d, overflow", i, sample)
		}
		if sample < -32768 {
			t.Errorf("result[%d] = %d, underflow", i, sample)
		}
	}
}

func TestCalculateRMS(t *testing.T) {
	tests := []struct {
		name     string
		samples  []int16
		expected float64
		tolerance float64
	}{
		{
			name:     "silence",
			samples:  []int16{0, 0, 0, 0},
			expected: 0.0,
			tolerance: 0.1,
		},
		{
			name:     "constant value",
			samples:  []int16{1000, 1000, 1000, 1000},
			expected: 1000.0,
			tolerance: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateRMS(tt.samples)

			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("CalculateRMS() = %f, expected %f", result, tt.expected)
			}
		})
	}
}

func TestCalculatePeak(t *testing.T) {
	tests := []struct {
		name     string
		samples  []int16
		expected int16
	}{
		{
			name:     "silence",
			samples:  []int16{0, 0, 0},
			expected: 0,
		},
		{
			name:     "positive peak",
			samples:  []int16{100, 500, 200},
			expected: 500,
		},
		{
			name:     "negative peak",
			samples:  []int16{100, -500, 200},
			expected: 500,
		},
		{
			name:     "mixed",
			samples:  []int16{100, -1000, 800},
			expected: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePeak(tt.samples)

			if result != tt.expected {
				t.Errorf("CalculatePeak() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestIsSilence(t *testing.T) {
	tests := []struct {
		name      string
		samples   []int16
		threshold int16
		expected  bool
	}{
		{
			name:      "true silence",
			samples:   []int16{0, 0, 0, 0},
			threshold: 100,
			expected:  true,
		},
		{
			name:      "quiet below threshold",
			samples:   []int16{50, -50, 30},
			threshold: 100,
			expected:  true,
		},
		{
			name:      "loud above threshold",
			samples:   []int16{50, 150, 30},
			threshold: 100,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSilence(tt.samples, tt.threshold)

			if result != tt.expected {
				t.Errorf("IsSilence() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateSilence(t *testing.T) {
	silence := GenerateSilence(100)

	if len(silence) != 100 {
		t.Errorf("Length = %d, expected 100", len(silence))
	}

	for i, sample := range silence {
		if sample != 0 {
			t.Errorf("silence[%d] = %d, expected 0", i, sample)
		}
	}
}

func TestGenerateTone(t *testing.T) {
	// Generate 1kHz tone at 16kHz sample rate
	tone := GenerateTone(1000, 16000, 160) // 10ms worth

	if len(tone) != 160 {
		t.Errorf("Length = %d, expected 160", len(tone))
	}

	// Check that it's not silence
	if IsSilence(tone, 100) {
		t.Error("Generated tone is silent")
	}

	// Check peak is reasonable
	peak := CalculatePeak(tone)
	if peak < 20000 {
		t.Errorf("Tone peak too low: %d", peak)
	}
}

func TestMixSamples(t *testing.T) {
	samples1 := []int16{1000, 2000, 3000}
	samples2 := []int16{500, 1000, 1500}

	mixed := MixSamples(samples1, samples2, 1.0, 1.0)

	if len(mixed) != 3 {
		t.Errorf("Length = %d, expected 3", len(mixed))
	}

	// Check no overflow
	for i, sample := range mixed {
		if sample > 32767 || sample < -32768 {
			t.Errorf("mixed[%d] = %d, out of range", i, sample)
		}
	}

	// First sample should be approximately 1500
	if math.Abs(float64(mixed[0])-1500) > 100 {
		t.Errorf("mixed[0] = %d, expected ~1500", mixed[0])
	}
}

func TestMixSamples_DifferentLengths(t *testing.T) {
	samples1 := []int16{1000, 2000}
	samples2 := []int16{500, 1000, 1500, 2000}

	mixed := MixSamples(samples1, samples2, 1.0, 1.0)

	// Should use longer length
	if len(mixed) != 4 {
		t.Errorf("Length = %d, expected 4", len(mixed))
	}
}

func TestFormatToString(t *testing.T) {
	format := AudioFormat{
		SampleRate: 16000,
		BitDepth:   16,
		Channels:   1,
	}

	result := FormatToString(format)
	expected := "16000 Hz, 16-bit, 1 channel(s)"

	if result != expected {
		t.Errorf("FormatToString() = %q, expected %q", result, expected)
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  AudioFormat
		wantErr bool
	}{
		{
			name: "valid format",
			format: AudioFormat{
				SampleRate: 16000,
				BitDepth:   16,
				Channels:   1,
			},
			wantErr: false,
		},
		{
			name: "invalid sample rate",
			format: AudioFormat{
				SampleRate: 0,
				BitDepth:   16,
				Channels:   1,
			},
			wantErr: true,
		},
		{
			name: "invalid channels",
			format: AudioFormat{
				SampleRate: 16000,
				BitDepth:   16,
				Channels:   0,
			},
			wantErr: true,
		},
		{
			name: "invalid bit depth",
			format: AudioFormat{
				SampleRate: 16000,
				BitDepth:   12,
				Channels:   1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFormat(tt.format)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateBufferDuration(t *testing.T) {
	format := AudioFormat{
		SampleRate: 16000,
		Channels:   1,
		BitDepth:   16,
	}

	// 320 samples at 16kHz = 20ms
	duration := CalculateBufferDuration(320, format)

	expected := 20.0
	if math.Abs(duration-expected) > 0.1 {
		t.Errorf("CalculateBufferDuration() = %f, expected %f", duration, expected)
	}
}

func TestCalculateBufferSize(t *testing.T) {
	format := AudioFormat{
		SampleRate: 16000,
		Channels:   1,
		BitDepth:   16,
	}

	// 20ms at 16kHz = 320 samples
	size := CalculateBufferSize(20.0, format)

	expected := 320
	if size != expected {
		t.Errorf("CalculateBufferSize() = %d, expected %d", size, expected)
	}
}

func BenchmarkApplyGain(b *testing.B) {
	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = int16(i * 100)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ApplyGain(samples, 1.5)
	}
}

func BenchmarkMixSamples(b *testing.B) {
	samples1 := make([]int16, 320)
	samples2 := make([]int16, 320)
	for i := range samples1 {
		samples1[i] = int16(i * 100)
		samples2[i] = int16(i * 50)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MixSamples(samples1, samples2, 1.0, 1.0)
	}
}

func BenchmarkCalculateRMS(b *testing.B) {
	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = int16(i * 100)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateRMS(samples)
	}
}
