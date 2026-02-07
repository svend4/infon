package audio

import (
	"math"
	"testing"
)

func TestNewVAD(t *testing.T) {
	vad := NewVAD(16000, 320)

	if vad == nil {
		t.Fatal("NewVAD() returned nil")
	}

	if vad.sampleRate != 16000 {
		t.Errorf("VAD sampleRate = %d, expected 16000", vad.sampleRate)
	}

	if vad.frameSize != 320 {
		t.Errorf("VAD frameSize = %d, expected 320", vad.frameSize)
	}

	if vad.threshold != 0.7 {
		t.Errorf("VAD threshold = %f, expected 0.7", vad.threshold)
	}

	if vad.onsetDelay != 2 {
		t.Errorf("VAD onsetDelay = %d, expected 2", vad.onsetDelay)
	}

	if vad.hangoverFrames != 10 {
		t.Errorf("VAD hangoverFrames = %d, expected 10", vad.hangoverFrames)
	}
}

func TestVAD_SetThreshold(t *testing.T) {
	vad := NewVAD(16000, 320)

	tests := []struct {
		name      string
		threshold float64
		expected  float64
	}{
		{
			name:      "normal threshold",
			threshold: 0.5,
			expected:  0.5,
		},
		{
			name:      "low threshold",
			threshold: 0.1,
			expected:  0.1,
		},
		{
			name:      "high threshold",
			threshold: 0.9,
			expected:  0.9,
		},
		{
			name:      "clamped to minimum",
			threshold: -0.5,
			expected:  0.01,
		},
		{
			name:      "clamped to maximum",
			threshold: 1.5,
			expected:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vad.SetThreshold(tt.threshold)
			if math.Abs(vad.threshold-tt.expected) > 0.001 {
				t.Errorf("SetThreshold(%f) resulted in %f, expected %f", tt.threshold, vad.threshold, tt.expected)
			}
		})
	}
}

func TestVAD_IsSpeaking_Silence(t *testing.T) {
	vad := NewVAD(16000, 320)

	// Create silence (all zeros)
	silence := make([]int16, 320)

	// Test multiple frames of silence
	for i := 0; i < 20; i++ {
		result := vad.IsSpeaking(silence)
		if result {
			t.Errorf("IsSpeaking() with silence = true at frame %d, expected false", i)
		}
	}
}

func TestVAD_IsSpeaking_LoudSignal(t *testing.T) {
	vad := NewVAD(16000, 320)
	vad.SetThreshold(0.5)

	// Create loud signal (high amplitude sine wave)
	loudSignal := make([]int16, 320)
	for i := range loudSignal {
		// 1000 Hz sine wave with amplitude 10000
		loudSignal[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	// Process several frames to build up energy
	for i := 0; i < 5; i++ {
		vad.IsSpeaking(loudSignal)
	}

	// Should be speaking after onset delay
	result := vad.IsSpeaking(loudSignal)
	if !result {
		t.Error("IsSpeaking() with loud signal = false, expected true")
	}
}

func TestVAD_IsSpeaking_QuietSignal(t *testing.T) {
	vad := NewVAD(16000, 320)
	vad.SetThreshold(0.7)

	// Create quiet signal (low amplitude)
	quietSignal := make([]int16, 320)
	for i := range quietSignal {
		quietSignal[i] = int16(100 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	// Test multiple frames
	for i := 0; i < 20; i++ {
		result := vad.IsSpeaking(quietSignal)
		if result {
			t.Errorf("IsSpeaking() with quiet signal = true at frame %d, expected false", i)
		}
	}
}

func TestVAD_HangoverPeriod(t *testing.T) {
	vad := NewVAD(16000, 320)
	vad.SetThreshold(0.5)

	// Create loud signal
	loudSignal := make([]int16, 320)
	for i := range loudSignal {
		loudSignal[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	// Create silence
	silence := make([]int16, 320)

	// Process loud signal to start speaking
	for i := 0; i < 5; i++ {
		vad.IsSpeaking(loudSignal)
	}

	// Verify speaking
	if !vad.IsSpeaking(loudSignal) {
		t.Fatal("Failed to start speaking")
	}

	// Process silence - should still be speaking due to hangover
	hangoverFrames := 0
	for i := 0; i < 15; i++ {
		if vad.IsSpeaking(silence) {
			hangoverFrames++
		}
	}

	// Should have hangover period (default 10 frames)
	if hangoverFrames == 0 {
		t.Error("No hangover period detected")
	}

	if hangoverFrames > 12 {
		t.Errorf("Hangover period too long: %d frames", hangoverFrames)
	}
}

func TestVAD_OnsetDelay(t *testing.T) {
	vad := NewVAD(16000, 320)
	vad.SetThreshold(0.5)

	// Create loud signal
	loudSignal := make([]int16, 320)
	for i := range loudSignal {
		loudSignal[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	// First frame should not immediately trigger speaking (onset delay)
	result := vad.IsSpeaking(loudSignal)
	if result {
		t.Error("IsSpeaking() = true on first frame, expected false due to onset delay")
	}

	// After onset delay frames, should be speaking
	for i := 0; i < 5; i++ {
		vad.IsSpeaking(loudSignal)
	}

	result = vad.IsSpeaking(loudSignal)
	if !result {
		t.Error("IsSpeaking() = false after onset delay, expected true")
	}
}

func TestVAD_GetStats(t *testing.T) {
	vad := NewVAD(16000, 320)

	// Create loud signal
	loudSignal := make([]int16, 320)
	for i := range loudSignal {
		loudSignal[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	// Create silence
	silence := make([]int16, 320)

	// Process some frames
	for i := 0; i < 5; i++ {
		vad.IsSpeaking(loudSignal)
	}

	for i := 0; i < 15; i++ {
		vad.IsSpeaking(silence)
	}

	stats := vad.GetStats()

	if stats.TotalFrames != 20 {
		t.Errorf("TotalFrames = %d, expected 20", stats.TotalFrames)
	}

	if stats.ActiveFrames == 0 {
		t.Error("ActiveFrames = 0, expected > 0")
	}

	if stats.ActivityRate < 0.0 || stats.ActivityRate > 1.0 {
		t.Errorf("ActivityRate = %f, expected 0.0-1.0", stats.ActivityRate)
	}

	if stats.CurrentEnergy < 0.0 {
		t.Errorf("CurrentEnergy = %f, expected >= 0.0", stats.CurrentEnergy)
	}

	if stats.NoiseFloor < 0.0 {
		t.Errorf("NoiseFloor = %f, expected >= 0.0", stats.NoiseFloor)
	}
}

func TestVAD_EnergyCalculation(t *testing.T) {
	vad := NewVAD(16000, 320)

	// Test with known signals
	tests := []struct {
		name            string
		samples         []int16
		expectHighEnergy bool
	}{
		{
			name:            "silence",
			samples:         make([]int16, 320),
			expectHighEnergy: false,
		},
		{
			name: "loud signal",
			samples: func() []int16 {
				s := make([]int16, 320)
				for i := range s {
					s[i] = 10000
				}
				return s
			}(),
			expectHighEnergy: true,
		},
		{
			name: "medium signal",
			samples: func() []int16 {
				s := make([]int16, 320)
				for i := range s {
					s[i] = 1000
				}
				return s
			}(),
			expectHighEnergy: false, // Should be lower than loud signal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vad.IsSpeaking(tt.samples)
			stats := vad.GetStats()

			if tt.expectHighEnergy && stats.CurrentEnergy < 0.5 {
				t.Errorf("Expected high energy, got %f", stats.CurrentEnergy)
			}

			if !tt.expectHighEnergy && stats.CurrentEnergy > 0.9 {
				t.Errorf("Expected low energy, got %f", stats.CurrentEnergy)
			}
		})
	}
}

func TestVAD_NoiseFloorAdaptation(t *testing.T) {
	vad := NewVAD(16000, 320)

	// Create low-level noise
	noise := make([]int16, 320)
	for i := range noise {
		noise[i] = 50
	}

	// Process many frames of noise
	for i := 0; i < 100; i++ {
		vad.IsSpeaking(noise)
	}

	stats := vad.GetStats()

	// Noise floor should have adapted
	if stats.NoiseFloor == 0.0 {
		t.Error("NoiseFloor not adapted")
	}

	// Should not be speaking with consistent low noise
	result := vad.IsSpeaking(noise)
	if result {
		t.Error("IsSpeaking() = true with low noise, expected false")
	}
}

func BenchmarkVAD_IsSpeaking(b *testing.B) {
	vad := NewVAD(16000, 320)

	// Create test signal
	signal := make([]int16, 320)
	for i := range signal {
		signal[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vad.IsSpeaking(signal)
	}
}

func BenchmarkVAD_IsSpeaking_Parallel(b *testing.B) {
	// Create test signal
	signal := make([]int16, 320)
	for i := range signal {
		signal[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		vad := NewVAD(16000, 320)
		for pb.Next() {
			vad.IsSpeaking(signal)
		}
	})
}
