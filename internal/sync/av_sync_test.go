package sync

import (
	"testing"
	"time"
)

func TestNewAVSynchronizer(t *testing.T) {
	avs := NewAVSynchronizer()

	if avs == nil {
		t.Fatal("NewAVSynchronizer() returned nil")
	}

	if avs.audioBuffer == nil {
		t.Error("Audio buffer should be initialized")
	}

	if avs.videoBuffer == nil {
		t.Error("Video buffer should be initialized")
	}
}

func TestAVSynchronizer_AddFrames(t *testing.T) {
	avs := NewAVSynchronizer()

	// Add audio frame
	err := avs.AddAudioFrame([]byte("audio"), 1000, 1)
	if err != nil {
		t.Fatalf("AddAudioFrame() failed: %v", err)
	}

	// Add video frame
	err = avs.AddVideoFrame([]byte("video"), 1000, 1)
	if err != nil {
		t.Fatalf("AddVideoFrame() failed: %v", err)
	}

	stats := avs.GetStatistics()

	if stats.AudioFrames != 1 {
		t.Errorf("AudioFrames = %d, expected 1", stats.AudioFrames)
	}

	if stats.VideoFrames != 1 {
		t.Errorf("VideoFrames = %d, expected 1", stats.VideoFrames)
	}
}

func TestAVSynchronizer_GetFrames(t *testing.T) {
	avs := NewAVSynchronizer()

	// Add frames with same PTS
	avs.AddAudioFrame([]byte("audio"), 1000, 1)
	avs.AddVideoFrame([]byte("video"), 1000, 1)

	// Wait for target delay
	time.Sleep(100 * time.Millisecond)

	// Get frames
	audioFrame, err := avs.GetNextAudioFrame()
	if err != nil {
		t.Logf("GetNextAudioFrame() returned: %v (may need more time)", err)
	}

	videoFrame, err := avs.GetNextVideoFrame()
	if err != nil {
		t.Logf("GetNextVideoFrame() returned: %v (may need more time)", err)
	}

	// At least one should work
	if audioFrame == nil && videoFrame == nil {
		t.Error("Both frames failed to retrieve")
	}
}

func TestAVSynchronizer_SyncOffset(t *testing.T) {
	avs := NewAVSynchronizer()

	// Add frames with different PTS
	avs.AddAudioFrame([]byte("audio"), 1000, 1)   // 1ms
	avs.AddVideoFrame([]byte("video"), 50000, 1) // 50ms

	offset := avs.GetSyncOffset()

	// Audio is 49ms behind video
	expected := -49 * time.Millisecond
	if offset != expected {
		t.Errorf("SyncOffset = %v, expected %v", offset, expected)
	}
}

func TestAVSynchronizer_ClockOffset(t *testing.T) {
	avs := NewAVSynchronizer()

	offset := 10 * time.Millisecond
	avs.UpdateClockOffset(offset)

	stats := avs.GetStatistics()

	if stats.ClockOffset != offset {
		t.Errorf("ClockOffset = %v, expected %v", stats.ClockOffset, offset)
	}
}

func TestAVSynchronizer_Reset(t *testing.T) {
	avs := NewAVSynchronizer()

	// Add frames
	avs.AddAudioFrame([]byte("audio"), 1000, 1)
	avs.AddVideoFrame([]byte("video"), 1000, 1)

	avs.Reset()

	stats := avs.GetStatistics()

	if stats.AudioFrames != 0 {
		t.Errorf("After reset, AudioFrames = %d, expected 0", stats.AudioFrames)
	}

	if stats.VideoFrames != 0 {
		t.Errorf("After reset, VideoFrames = %d, expected 0", stats.VideoFrames)
	}
}

func TestJitterBuffer_Add(t *testing.T) {
	jb := NewJitterBuffer(10, 50*time.Millisecond)

	frame := &Frame{
		Type:      MediaTypeAudio,
		Data:      []byte("test"),
		Timestamp: time.Now(),
		PTS:       1000,
		Sequence:  1,
	}

	err := jb.Add(frame)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if jb.Size() != 1 {
		t.Errorf("Size = %d, expected 1", jb.Size())
	}
}

func TestJitterBuffer_OrderedInsertion(t *testing.T) {
	jb := NewJitterBuffer(10, 50*time.Millisecond)

	// Add frames out of order
	jb.Add(&Frame{PTS: 3000, Sequence: 3})
	jb.Add(&Frame{PTS: 1000, Sequence: 1})
	jb.Add(&Frame{PTS: 2000, Sequence: 2})

	// Peek should return lowest PTS
	frame, err := jb.Peek()
	if err != nil {
		t.Fatalf("Peek() failed: %v", err)
	}

	if frame.PTS != 1000 {
		t.Errorf("First frame PTS = %d, expected 1000", frame.PTS)
	}
}

func TestJitterBuffer_Get(t *testing.T) {
	jb := NewJitterBuffer(10, 10*time.Millisecond)

	frame := &Frame{
		Type:      MediaTypeAudio,
		Timestamp: time.Now().Add(-20 * time.Millisecond), // In the past
		PTS:       1000,
	}

	jb.Add(frame)

	// Should be ready since it's past target delay
	retrieved, err := jb.Get()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.PTS != 1000 {
		t.Errorf("Retrieved PTS = %d, expected 1000", retrieved.PTS)
	}

	if jb.Size() != 0 {
		t.Error("Buffer should be empty after Get()")
	}
}

func TestJitterBuffer_BufferFull(t *testing.T) {
	jb := NewJitterBuffer(3, 50*time.Millisecond)

	// Fill buffer
	for i := 0; i < 3; i++ {
		jb.Add(&Frame{PTS: int64(i * 1000)})
	}

	// Try to add one more
	err := jb.Add(&Frame{PTS: 4000})
	if err == nil {
		t.Error("Add() should fail when buffer is full")
	}
}

func TestJitterBuffer_Clear(t *testing.T) {
	jb := NewJitterBuffer(10, 50*time.Millisecond)

	jb.Add(&Frame{PTS: 1000})
	jb.Add(&Frame{PTS: 2000})

	jb.Clear()

	if jb.Size() != 0 {
		t.Errorf("After Clear(), size = %d, expected 0", jb.Size())
	}
}

func BenchmarkAVSynchronizer_AddAudioFrame(b *testing.B) {
	avs := NewAVSynchronizer()
	data := []byte("audio")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		avs.AddAudioFrame(data, int64(i*1000), uint32(i))
	}
}

func BenchmarkJitterBuffer_Add(b *testing.B) {
	jb := NewJitterBuffer(1000, 50*time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jb.Add(&Frame{PTS: int64(i * 1000), Sequence: uint32(i)})
	}
}
