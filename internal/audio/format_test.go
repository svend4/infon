package audio

import (
	"testing"
)

func TestAudioFormat_FrameSize(t *testing.T) {
	tests := []struct {
		name     string
		format   AudioFormat
		expected int
	}{
		{
			name: "mono 16-bit",
			format: AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
			},
			expected: 2, // 1 channel * 16 bits / 8
		},
		{
			name: "stereo 16-bit",
			format: AudioFormat{
				SampleRate: 44100,
				Channels:   2,
				BitDepth:   16,
			},
			expected: 4, // 2 channels * 16 bits / 8
		},
		{
			name: "mono 24-bit",
			format: AudioFormat{
				SampleRate: 48000,
				Channels:   1,
				BitDepth:   24,
			},
			expected: 3, // 1 channel * 24 bits / 8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.format.FrameSize()
			if result != tt.expected {
				t.Errorf("FrameSize() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestAudioFormat_BytesPerSecond(t *testing.T) {
	tests := []struct {
		name     string
		format   AudioFormat
		expected int
	}{
		{
			name: "16kHz mono 16-bit",
			format: AudioFormat{
				SampleRate: 16000,
				Channels:   1,
				BitDepth:   16,
			},
			expected: 32000, // 16000 * 1 * 2
		},
		{
			name: "44.1kHz stereo 16-bit",
			format: AudioFormat{
				SampleRate: 44100,
				Channels:   2,
				BitDepth:   16,
			},
			expected: 176400, // 44100 * 2 * 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.format.BytesPerSecond()
			if result != tt.expected {
				t.Errorf("BytesPerSecond() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestDefaultFormat(t *testing.T) {
	format := DefaultFormat()

	if format.SampleRate != 16000 {
		t.Errorf("DefaultFormat().SampleRate = %d, expected 16000", format.SampleRate)
	}

	if format.Channels != 1 {
		t.Errorf("DefaultFormat().Channels = %d, expected 1", format.Channels)
	}

	if format.BitDepth != 16 {
		t.Errorf("DefaultFormat().BitDepth = %d, expected 16", format.BitDepth)
	}
}

func TestEncodePCM(t *testing.T) {
	tests := []struct {
		name     string
		samples  []int16
		expected int // expected byte length
	}{
		{
			name:     "empty",
			samples:  []int16{},
			expected: 0,
		},
		{
			name:     "single sample",
			samples:  []int16{100},
			expected: 2, // 1 sample * 2 bytes
		},
		{
			name:     "multiple samples",
			samples:  []int16{100, 200, 300, 400},
			expected: 8, // 4 samples * 2 bytes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodePCM(tt.samples)
			if len(result) != tt.expected {
				t.Errorf("EncodePCM() length = %d, expected %d", len(result), tt.expected)
			}
		})
	}
}

func TestDecodePCM(t *testing.T) {
	tests := []struct {
		name     string
		input    []int16
		expected int // expected sample count
	}{
		{
			name:     "empty",
			input:    []int16{},
			expected: 0,
		},
		{
			name:     "single sample",
			input:    []int16{100},
			expected: 1,
		},
		{
			name:     "multiple samples",
			input:    []int16{100, 200, 300, 400},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode first
			encoded := EncodePCM(tt.input)

			// Then decode
			result, err := DecodePCM(encoded)
			if err != nil {
				t.Errorf("DecodePCM() error = %v", err)
				return
			}

			if len(result) != tt.expected {
				t.Errorf("DecodePCM() length = %d, expected %d", len(result), tt.expected)
			}

			// Verify values match
			for i := range tt.input {
				if result[i] != tt.input[i] {
					t.Errorf("DecodePCM() sample[%d] = %d, expected %d", i, result[i], tt.input[i])
				}
			}
		})
	}
}

func TestDecodePCM_InvalidLength(t *testing.T) {
	// Odd number of bytes (invalid for int16)
	invalidData := []byte{0x00, 0x01, 0x02}

	_, err := DecodePCM(invalidData)
	if err == nil {
		t.Error("DecodePCM() with odd byte length should return error")
	}
}

func TestEncodePCM_RoundTrip(t *testing.T) {
	// Test various sample values
	samples := []int16{
		0,      // zero
		1000,   // positive
		-1000,  // negative
		32767,  // max positive
		-32768, // max negative
	}

	encoded := EncodePCM(samples)
	decoded, err := DecodePCM(encoded)

	if err != nil {
		t.Fatalf("Round-trip failed: %v", err)
	}

	if len(decoded) != len(samples) {
		t.Fatalf("Round-trip length mismatch: got %d, expected %d", len(decoded), len(samples))
	}

	for i := range samples {
		if decoded[i] != samples[i] {
			t.Errorf("Round-trip sample[%d] = %d, expected %d", i, decoded[i], samples[i])
		}
	}
}

func BenchmarkEncodePCM(b *testing.B) {
	samples := make([]int16, 320) // 20ms at 16kHz
	for i := range samples {
		samples[i] = int16(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodePCM(samples)
	}
}

func BenchmarkDecodePCM(b *testing.B) {
	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = int16(i)
	}
	data := EncodePCM(samples)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodePCM(data)
	}
}
