package audio

import (
	"math"
	"testing"
)

func TestNewVAD(t *testing.T) {
	vad := NewVAD(16000)

	if vad == nil {
		t.Fatal("NewVAD() returned nil")
	}

	if vad.sampleRate != 16000 {
		t.Errorf("VAD sampleRate = %d, expected 16000", vad.sampleRate)
	}

	if vad.energyThreshold <= 0 {
		t.Errorf("VAD energyThreshold = %f, expected > 0", vad.energyThreshold)
	}

	if vad.onsetFrames != 2 {
		t.Errorf("VAD onsetFrames = %d, expected 2", vad.onsetFrames)
	}

	if vad.hangoverFrames != 10 {
		t.Errorf("VAD hangoverFrames = %d, expected 10", vad.hangoverFrames)
	}
}

func TestVAD_SetSensitivity(t *testing.T) {
	vad := NewVAD(16000)

	tests := []struct {
		name        string
		sensitivity float64
	}{
		{
			name:        "normal sensitivity",
			sensitivity: 0.5,
		},
		{
			name:        "low sensitivity",
			sensitivity: 0.0,
		},
		{
			name:        "high sensitivity",
			sensitivity: 1.0,
		},
		{
			name:        "clamped negative",
			sensitivity: -0.5,
		},
		{
			name:        "clamped above 1.0",
			sensitivity: 1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vad.SetSensitivity(tt.sensitivity)
			// Just verify it doesn't panic and onset/hangover frames are set
			if vad.onsetFrames < 1 {
				t.Errorf("SetSensitivity(%f) resulted in onsetFrames = %d, expected >= 1", tt.sensitivity, vad.onsetFrames)
			}
			if vad.hangoverFrames < 5 {
				t.Errorf("SetSensitivity(%f) resulted in hangoverFrames = %d, expected >= 5", tt.sensitivity, vad.hangoverFrames)
			}
		})
	}
}

func TestVAD_IsSpeaking_Silence(t *testing.T) {
	vad := NewVAD(16000)

	// Create silence (all zeros)
	silence := make([]int16, 320)

	// Test multiple frames of silence
	for i := 0; i < 20; i++ {
		result := vad.Process(silence)
		if result {
			t.Errorf("IsSpeaking() with silence = true at frame %d, expected false", i)
		}
	}
}

func TestVAD_IsSpeaking_LoudSignal(t *testing.T) {
	vad := NewVAD(16000)
	vad.SetSensitivity(0.5)

	// Create loud signal (high amplitude sine wave)
	loudSignal := make([]int16, 320)
	for i := range loudSignal {
		// 1000 Hz sine wave with amplitude 10000
		loudSignal[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	// Process several frames to build up energy
	for i := 0; i < 5; i++ {
		vad.Process(loudSignal)
	}

	// Should be speaking after onset delay
	result := vad.Process(loudSignal)
	if !result {
		t.Error("IsSpeaking() with loud signal = false, expected true")
	}
}

func TestVAD_IsSpeaking_QuietSignal(t *testing.T) {
	vad := NewVAD(16000)
	vad.SetSensitivity(0.7)

	// Create quiet signal (low amplitude)
	quietSignal := make([]int16, 320)
	for i := range quietSignal {
		quietSignal[i] = int16(100 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	// Test multiple frames
	for i := 0; i < 20; i++ {
		result := vad.Process(quietSignal)
		if result {
			t.Errorf("IsSpeaking() with quiet signal = true at frame %d, expected false", i)
		}
	}
}

func TestVAD_HangoverPeriod(t *testing.T) {
	vad := NewVAD(16000)
	vad.SetSensitivity(0.5)

	// Create loud signal
	loudSignal := make([]int16, 320)
	for i := range loudSignal {
		loudSignal[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	// Create silence
	silence := make([]int16, 320)

	// Process loud signal to start speaking
	for i := 0; i < 5; i++ {
		vad.Process(loudSignal)
	}

	// Verify speaking
	if !vad.Process(loudSignal) {
		t.Fatal("Failed to start speaking")
	}

	// Process silence - should still be speaking due to hangover
	hangoverFrames := 0
	for i := 0; i < 15; i++ {
		if vad.Process(silence) {
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
	vad := NewVAD(16000)
	vad.SetSensitivity(0.5)

	// Create loud signal
	loudSignal := make([]int16, 320)
	for i := range loudSignal {
		loudSignal[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	// First frame should not immediately trigger speaking (onset delay)
	result := vad.Process(loudSignal)
	if result {
		t.Error("IsSpeaking() = true on first frame, expected false due to onset delay")
	}

	// After onset delay frames, should be speaking
	for i := 0; i < 5; i++ {
		vad.Process(loudSignal)
	}

	result = vad.Process(loudSignal)
	if !result {
		t.Error("IsSpeaking() = false after onset delay, expected true")
	}
}

func TestVAD_GetStats(t *testing.T) {
	vad := NewVAD(16000)

	// Create loud signal
	loudSignal := make([]int16, 320)
	for i := range loudSignal {
		loudSignal[i] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	// Create silence
	silence := make([]int16, 320)

	// Process some frames
	for i := 0; i < 5; i++ {
		vad.Process(loudSignal)
	}

	for i := 0; i < 15; i++ {
		vad.Process(silence)
	}

	stats := vad.GetStatistics()

	if stats.TotalFrames != 20 {
		t.Errorf("TotalFrames = %d, expected 20", stats.TotalFrames)
	}

	if stats.SpeechFrames == 0 {
		t.Error("SpeechFrames = 0, expected > 0")
	}

	if stats.ActivityRate < 0.0 || stats.ActivityRate > 100.0 {
		t.Errorf("ActivityRate = %f, expected 0.0-100.0", stats.ActivityRate)
	}

	if stats.CurrentEnergy < 0.0 {
		t.Errorf("CurrentEnergy = %f, expected >= 0.0", stats.CurrentEnergy)
	}

	if stats.NoiseFloor < 0.0 {
		t.Errorf("NoiseFloor = %f, expected >= 0.0", stats.NoiseFloor)
	}
}

func TestVAD_EnergyCalculation(t *testing.T) {
	// Test with known signals
	tests := []struct {
		name         string
		samples      []int16
		expectSpeech bool
	}{
		{
			name:         "silence",
			samples:      make([]int16, 320),
			expectSpeech: false,
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
			expectSpeech: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh VAD for each test
			vad := NewVAD(16000)

			// Process several frames to get past onset delay
			for i := 0; i < 5; i++ {
				vad.Process(tt.samples)
			}

			stats := vad.GetStatistics()

			if tt.expectSpeech && stats.SpeechFrames == 0 {
				t.Error("Expected speech to be detected")
			}

			if !tt.expectSpeech && stats.SpeechFrames > 1 {
				t.Errorf("Expected no speech, but %d frames detected as speech", stats.SpeechFrames)
			}
		})
	}
}

func TestVAD_NoiseFloorAdaptation(t *testing.T) {
	vad := NewVAD(16000)

	// Create low-level noise
	noise := make([]int16, 320)
	for i := range noise {
		noise[i] = 50
	}

	// Process many frames of noise
	for i := 0; i < 100; i++ {
		vad.Process(noise)
	}

	stats := vad.GetStatistics()

	// Noise floor should have adapted
	if stats.NoiseFloor == 0.0 {
		t.Error("NoiseFloor not adapted")
	}

	// Should not be speaking with consistent low noise
	result := vad.Process(noise)
	if result {
		t.Error("IsSpeaking() = true with low noise, expected false")
	}
}

func BenchmarkVAD_IsSpeaking(b *testing.B) {
	vad := NewVAD(16000)

	// Create test signal
	signal := make([]int16, 320)
	for i := range signal {
		signal[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vad.Process(signal)
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
		vad := NewVAD(16000)
		for pb.Next() {
			vad.Process(signal)
		}
	})
}
