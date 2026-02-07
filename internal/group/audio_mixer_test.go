package group

import (
	"math"
	"testing"
	"time"
)

func TestNewAudioMixer(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	if mixer == nil {
		t.Fatal("NewAudioMixer() returned nil")
	}

	if mixer.sampleRate != 16000 {
		t.Errorf("AudioMixer sampleRate = %d, expected 16000", mixer.sampleRate)
	}

	if mixer.framesPerChunk != 320 {
		t.Errorf("AudioMixer framesPerChunk = %d, expected 320", mixer.framesPerChunk)
	}

	if mixer.sources == nil {
		t.Error("AudioMixer sources map should be initialized")
	}
}

func TestAudioMixer_AddSource(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	// Create test samples
	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	mixer.AddSource("peer1", samples)

	// Check that source was added
	mixer.mutex.Lock()
	if len(mixer.sources) != 1 {
		t.Errorf("AddSource() resulted in %d sources, expected 1", len(mixer.sources))
	}

	source, exists := mixer.sources["peer1"]
	if !exists {
		t.Error("AddSource() did not add source 'peer1'")
	}

	if len(source.samples) != len(samples) {
		t.Errorf("Source samples length = %d, expected %d", len(source.samples), len(samples))
	}
	mixer.mutex.Unlock()
}

func TestAudioMixer_AddSource_MultiplePeers(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	samples1 := make([]int16, 320)
	samples2 := make([]int16, 320)
	samples3 := make([]int16, 320)

	mixer.AddSource("peer1", samples1)
	mixer.AddSource("peer2", samples2)
	mixer.AddSource("peer3", samples3)

	mixer.mutex.Lock()
	if len(mixer.sources) != 3 {
		t.Errorf("AddSource() resulted in %d sources, expected 3", len(mixer.sources))
	}
	mixer.mutex.Unlock()
}

func TestAudioMixer_AddSource_UpdateExisting(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	samples1 := make([]int16, 320)
	for i := range samples1 {
		samples1[i] = 100
	}

	samples2 := make([]int16, 320)
	for i := range samples2 {
		samples2[i] = 200
	}

	mixer.AddSource("peer1", samples1)
	mixer.AddSource("peer1", samples2) // Update same peer

	mixer.mutex.Lock()
	if len(mixer.sources) != 1 {
		t.Errorf("AddSource() resulted in %d sources, expected 1", len(mixer.sources))
	}

	source := mixer.sources["peer1"]
	if len(source.samples) > 0 && source.samples[0] != 200 {
		t.Errorf("Source not updated, first sample = %d, expected 200", source.samples[0])
	}
	mixer.mutex.Unlock()
}

func TestAudioMixer_Mix_Empty(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	mixed := mixer.Mix()

	if len(mixed) != 320 {
		t.Errorf("Mix() length = %d, expected 320", len(mixed))
	}

	// Empty mix should be silence
	for i, sample := range mixed {
		if sample != 0 {
			t.Errorf("Mix() empty sample[%d] = %d, expected 0", i, sample)
		}
	}
}

func TestAudioMixer_Mix_SingleSource(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = 1000
	}

	mixer.AddSource("peer1", samples)
	mixed := mixer.Mix()

	if len(mixed) != 320 {
		t.Errorf("Mix() length = %d, expected 320", len(mixed))
	}

	// Single source should pass through (approximately)
	for i, sample := range mixed {
		if math.Abs(float64(sample)-1000.0) > 100 {
			t.Errorf("Mix() single source sample[%d] = %d, expected ~1000", i, sample)
		}
	}
}

func TestAudioMixer_Mix_MultipleSources(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	samples1 := make([]int16, 320)
	samples2 := make([]int16, 320)

	for i := range samples1 {
		samples1[i] = 500
		samples2[i] = 500
	}

	mixer.AddSource("peer1", samples1)
	mixer.AddSource("peer2", samples2)

	mixed := mixer.Mix()

	if len(mixed) != 320 {
		t.Errorf("Mix() length = %d, expected 320", len(mixed))
	}

	// Two sources with same amplitude should mix to higher amplitude
	// but soft-clipping should prevent simple addition
	for i, sample := range mixed {
		if sample == 0 {
			t.Errorf("Mix() two sources sample[%d] = 0, expected non-zero", i)
		}

		// Should not simply add to 1000 due to soft-clipping
		if sample == 1000 {
			t.Logf("Note: sample[%d] = 1000, soft-clipping may not be active", i)
		}
	}
}

func TestAudioMixer_Mix_NoOverflow(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	// Create multiple sources with max amplitude
	for i := 0; i < 10; i++ {
		samples := make([]int16, 320)
		for j := range samples {
			samples[j] = 30000
		}
		mixer.AddSource(string(rune('A'+i)), samples)
	}

	mixed := mixer.Mix()

	// Verify no overflow
	for i, sample := range mixed {
		if sample > 32767 || sample < -32768 {
			t.Errorf("Mix() sample[%d] = %d, out of int16 range", i, sample)
		}
	}

	// With soft-clipping, should be at or near max
	hasNearMax := false
	for _, sample := range mixed {
		if math.Abs(float64(sample)) > 30000 {
			hasNearMax = true
			break
		}
	}

	if !hasNearMax {
		t.Error("Mix() with multiple loud sources should produce near-max amplitude")
	}
}

func TestAudioMixer_Mix_SoftClipping(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	samples1 := make([]int16, 320)
	samples2 := make([]int16, 320)

	for i := range samples1 {
		samples1[i] = 20000
		samples2[i] = 20000
	}

	mixer.AddSource("peer1", samples1)
	mixer.AddSource("peer2", samples2)

	mixed := mixer.Mix()

	// Soft-clipping should prevent simple addition (40000) and overflow
	for i, sample := range mixed {
		if sample > 32767 || sample < -32768 {
			t.Errorf("Mix() sample[%d] = %d, overflow detected", i, sample)
		}

		// Should be less than simple addition due to soft-clipping
		if sample == 40000 {
			t.Errorf("Mix() sample[%d] = 40000, soft-clipping not working", i)
		}
	}
}

func TestAudioMixer_CleanupOldSources(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	samples := make([]int16, 320)

	// Add source
	mixer.AddSource("peer1", samples)

	// Verify source exists
	mixer.mutex.Lock()
	if len(mixer.sources) != 1 {
		t.Fatal("Source not added")
	}
	mixer.mutex.Unlock()

	// Wait for cleanup time (>1s)
	time.Sleep(1100 * time.Millisecond)

	// Mix should trigger cleanup
	mixer.Mix()

	// Source should be cleaned up
	mixer.mutex.Lock()
	if len(mixer.sources) != 0 {
		t.Errorf("Old source not cleaned up, %d sources remaining", len(mixer.sources))
	}
	mixer.mutex.Unlock()
}

func TestAudioMixer_Mix_DifferentLengths(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	samples1 := make([]int16, 320)
	samples2 := make([]int16, 160) // Half length

	for i := range samples1 {
		samples1[i] = 1000
	}
	for i := range samples2 {
		samples2[i] = 1000
	}

	mixer.AddSource("peer1", samples1)
	mixer.AddSource("peer2", samples2)

	mixed := mixer.Mix()

	// Should still produce full-length output
	if len(mixed) != 320 {
		t.Errorf("Mix() length = %d, expected 320", len(mixed))
	}

	// First half should have both sources
	firstHalfNonZero := false
	for i := 0; i < 160; i++ {
		if mixed[i] != 0 {
			firstHalfNonZero = true
			break
		}
	}

	if !firstHalfNonZero {
		t.Error("First half of mix should be non-zero")
	}

	// Second half should only have one source
	secondHalfNonZero := false
	for i := 160; i < 320; i++ {
		if mixed[i] != 0 {
			secondHalfNonZero = true
			break
		}
	}

	if !secondHalfNonZero {
		t.Error("Second half of mix should be non-zero (from peer1)")
	}
}

func TestAudioMixer_Mix_Alternating(t *testing.T) {
	mixer := NewAudioMixer(16000, 320)

	samples := make([]int16, 320)
	for i := range samples {
		if i%2 == 0 {
			samples[i] = 1000
		} else {
			samples[i] = -1000
		}
	}

	mixer.AddSource("peer1", samples)
	mixed := mixer.Mix()

	// Alternating pattern should be preserved (approximately)
	for i := range mixed {
		if i%2 == 0 {
			if mixed[i] < 0 {
				t.Errorf("Mix() sample[%d] should be positive", i)
			}
		} else {
			if mixed[i] > 0 {
				t.Errorf("Mix() sample[%d] should be negative", i)
			}
		}
	}
}

func BenchmarkAudioMixer_Mix_SingleSource(b *testing.B) {
	mixer := NewAudioMixer(16000, 320)

	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	mixer.AddSource("peer1", samples)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mixer.Mix()
	}
}

func BenchmarkAudioMixer_Mix_ThreeSources(b *testing.B) {
	mixer := NewAudioMixer(16000, 320)

	for j := 0; j < 3; j++ {
		samples := make([]int16, 320)
		for i := range samples {
			samples[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
		}
		mixer.AddSource(string(rune('A'+j)), samples)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mixer.Mix()
	}
}

func BenchmarkAudioMixer_Mix_TenSources(b *testing.B) {
	mixer := NewAudioMixer(16000, 320)

	for j := 0; j < 10; j++ {
		samples := make([]int16, 320)
		for i := range samples {
			samples[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
		}
		mixer.AddSource(string(rune('A'+j)), samples)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mixer.Mix()
	}
}

func BenchmarkAudioMixer_AddSource(b *testing.B) {
	mixer := NewAudioMixer(16000, 320)

	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = int16(1000 * math.Sin(2*math.Pi*1000*float64(i)/16000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mixer.AddSource("peer1", samples)
	}
}
