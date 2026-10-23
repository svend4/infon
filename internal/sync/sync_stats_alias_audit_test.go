package sync

import (
	"testing"
	"time"
)

// Callers reuse one receive buffer for every packet; a buffered frame must
// keep the bytes it was added with.
func TestAVSync_FrameDataNotAliased(t *testing.T) {
	advance := fakeClock(t)
	avs := NewAVSynchronizer()

	buf := []byte("audio-1")
	_ = avs.AddAudioFrame(buf, 0, 1)
	copy(buf, "XXXXXXX")
	vbuf := []byte("video-1")
	_ = avs.AddVideoFrame(vbuf, 0, 1)
	copy(vbuf, "YYYYYYY")
	advance(120 * time.Millisecond)

	if f, err := avs.GetNextAudioFrame(); err != nil || string(f.Data) != "audio-1" {
		t.Errorf("audio frame = %v, %v; want data %q", f, err, "audio-1")
	}
	if f, err := avs.GetNextVideoFrame(); err != nil || string(f.Data) != "video-1" {
		t.Errorf("video frame = %v, %v; want data %q", f, err, "video-1")
	}
}
