package sync

import (
	"testing"
	"time"
)

// A frame whose PTS is implausibly far ahead of the other stream must not be
// put back and kept at the head of its buffer forever: that makes every frame
// of the other stream look late, so the other stream is dropped indefinitely.
func TestAVSync_ImplausibleAudioPTSDoesNotStallVideo(t *testing.T) {
	advance := fakeClock(t)
	avs := NewAVSynchronizer()

	hour := int64(time.Hour / time.Microsecond)
	if err := avs.AddAudioFrame([]byte("bogus"), hour, 1); err != nil {
		t.Fatalf("AddAudioFrame: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := avs.AddVideoFrame([]byte("v"), int64(i)*33333, uint32(i)); err != nil {
			t.Fatalf("AddVideoFrame: %v", err)
		}
	}
	advance(120 * time.Millisecond) // past both jitter buffer target delays

	played := 0
	for i := 0; i < 3; i++ {
		_, _ = avs.GetNextAudioFrame()
		if f, err := avs.GetNextVideoFrame(); err == nil && f != nil {
			played++
		}
	}

	if played != 3 {
		t.Errorf("video frames played = %d, want 3", played)
	}
	if n := avs.audioBuffer.Size(); n != 0 {
		t.Errorf("implausible audio frame still buffered (size %d)", n)
	}
	if st := avs.GetStatistics(); st.DroppedAudio != 1 || st.DroppedVideo != 0 {
		t.Errorf("DroppedAudio=%d DroppedVideo=%d, want 1 and 0", st.DroppedAudio, st.DroppedVideo)
	}
}

func TestAVSync_ImplausibleVideoPTSDoesNotStallAudio(t *testing.T) {
	advance := fakeClock(t)
	avs := NewAVSynchronizer()

	hour := int64(time.Hour / time.Microsecond)
	if err := avs.AddVideoFrame([]byte("bogus"), hour, 1); err != nil {
		t.Fatalf("AddVideoFrame: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := avs.AddAudioFrame([]byte("a"), int64(i)*20000, uint32(i)); err != nil {
			t.Fatalf("AddAudioFrame: %v", err)
		}
	}
	advance(120 * time.Millisecond)

	played := 0
	for i := 0; i < 3; i++ {
		_, _ = avs.GetNextVideoFrame()
		if f, err := avs.GetNextAudioFrame(); err == nil && f != nil {
			played++
		}
	}

	if played != 3 {
		t.Errorf("audio frames played = %d, want 3", played)
	}
	if n := avs.videoBuffer.Size(); n != 0 {
		t.Errorf("implausible video frame still buffered (size %d)", n)
	}
	if st := avs.GetStatistics(); st.DroppedVideo != 1 || st.DroppedAudio != 0 {
		t.Errorf("DroppedVideo=%d DroppedAudio=%d, want 1 and 0", st.DroppedVideo, st.DroppedAudio)
	}
}

// A modest lead within the plausible range still waits for the other stream.
func TestAVSync_PlausibleLeadStillWaits(t *testing.T) {
	advance := fakeClock(t)
	avs := NewAVSynchronizer()

	_ = avs.AddAudioFrame([]byte("a"), int64(300*time.Millisecond/time.Microsecond), 1)
	_ = avs.AddVideoFrame([]byte("v"), 0, 1)
	advance(120 * time.Millisecond)

	if f, err := avs.GetNextAudioFrame(); err == nil || f != nil {
		t.Fatalf("GetNextAudioFrame = %v, %v; want wait error", f, err)
	}
	if n := avs.audioBuffer.Size(); n != 1 {
		t.Errorf("audio buffer size = %d, want 1 (frame put back)", n)
	}
	if st := avs.GetStatistics(); st.DroppedAudio != 0 {
		t.Errorf("DroppedAudio = %d, want 0", st.DroppedAudio)
	}
}
