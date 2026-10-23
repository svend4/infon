package sync

import (
	"testing"
	"time"
)

// A head frame that the other stream's consumer never takes must not freeze
// this stream forever (e.g. audio is received but not played).
func TestAVSync_UnconsumedAudioDoesNotFreezeVideo(t *testing.T) {
	advance := fakeClock(t)
	avs := NewAVSynchronizer()

	_ = avs.AddAudioFrame([]byte("a"), 0, 1)
	for i := 0; i < 3; i++ {
		_ = avs.AddVideoFrame([]byte("v"), int64(i)*200000, uint32(i))
	}
	advance(400 * time.Millisecond)

	for i := 0; i < 3; i++ {
		if f, err := avs.GetNextVideoFrame(); err != nil || f == nil {
			t.Fatalf("video frame %d: %v", i, err)
		}
	}
}

func TestAVSync_UnconsumedVideoDoesNotFreezeAudio(t *testing.T) {
	advance := fakeClock(t)
	avs := NewAVSynchronizer()

	_ = avs.AddVideoFrame([]byte("v"), 0, 1)
	for i := 0; i < 3; i++ {
		_ = avs.AddAudioFrame([]byte("a"), int64(i)*200000, uint32(i))
	}
	advance(400 * time.Millisecond)

	for i := 0; i < 3; i++ {
		if f, err := avs.GetNextAudioFrame(); err != nil || f == nil {
			t.Fatalf("audio frame %d: %v", i, err)
		}
	}
}
