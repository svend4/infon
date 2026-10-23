package sync

import (
	"errors"
	"testing"
	"time"
)

func readyFrame(pts int64) *Frame {
	return &Frame{Timestamp: clock().Add(-time.Second), PTS: pts}
}

// A frame that arrives after its successors were played must be dropped, not
// played out of order, and must not hold back frames that are already ready.
func TestJitterBuffer_LateFrameDropped(t *testing.T) {
	jb := NewJitterBuffer(10, 10*time.Millisecond)

	_ = jb.Add(readyFrame(1000))
	_ = jb.Add(readyFrame(3000))
	for _, want := range []int64{1000, 3000} {
		f, err := jb.Get()
		if err != nil || f.PTS != want {
			t.Fatalf("Get = %v, %v; want PTS %d", f, err, want)
		}
	}

	_ = jb.Add(readyFrame(4000))
	late := &Frame{Timestamp: clock(), PTS: 2000} // arrives now, after 3000 played
	if err := jb.Add(late); !errors.Is(err, ErrLateFrame) {
		t.Errorf("Add(late) = %v, want ErrLateFrame", err)
	}

	f, err := jb.Get()
	if err != nil || f.PTS != 4000 {
		t.Errorf("Get after late frame = %v, %v; want PTS 4000", f, err)
	}
}

// Duplicates (same PTS) are rejected whether the original is still buffered
// or already played.
func TestJitterBuffer_DuplicateFrameDropped(t *testing.T) {
	jb := NewJitterBuffer(10, 10*time.Millisecond)

	_ = jb.Add(readyFrame(1000))
	if err := jb.Add(readyFrame(1000)); !errors.Is(err, ErrDuplicateFrame) {
		t.Errorf("Add(duplicate of buffered) = %v, want ErrDuplicateFrame", err)
	}
	if n := jb.Size(); n != 1 {
		t.Errorf("Size = %d, want 1", n)
	}

	if _, err := jb.Get(); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if err := jb.Add(readyFrame(1000)); err == nil {
		t.Error("Add(duplicate of played) accepted, want error")
	}
	if n := jb.Size(); n != 0 {
		t.Errorf("Size = %d, want 0", n)
	}
}

// Frames that keep landing behind the last played one mean the sender's clock
// restarted: after restartRun of them in a row the buffer resynchronises, drops
// what it still held from before the restart, and accepts the new timeline.
func TestJitterBuffer_ClockRestartAccepted(t *testing.T) {
	jb := NewJitterBuffer(20, 10*time.Millisecond)

	hour := int64(time.Hour / time.Microsecond)
	_ = jb.Add(readyFrame(hour))
	_ = jb.Add(readyFrame(hour + 20000)) // still buffered when the clock restarts
	if _, err := jb.Get(); err != nil {
		t.Fatalf("Get: %v", err)
	}
	for i := 0; i < restartRun-1; i++ {
		if err := jb.Add(readyFrame(int64(i) * 20000)); !errors.Is(err, ErrLateFrame) {
			t.Fatalf("Add(restart frame %d) = %v, want ErrLateFrame until the run is long enough", i, err)
		}
	}
	for i := restartRun - 1; i < restartRun+2; i++ {
		if err := jb.Add(readyFrame(int64(i) * 20000)); err != nil {
			t.Errorf("Add(PTS %d) after restart = %v, want nil", i*20000, err)
		}
	}
	f, err := jb.Get()
	if err != nil || f.PTS != int64(restartRun-1)*20000 {
		t.Errorf("first frame after restart = %v, %v; want the new timeline, not the stale PTS %d", f, err, hour+20000)
	}
}

// One frame far behind the playout point (a late retransmission) is dropped,
// not taken for a restart: it must not jump the queue or reopen the window for
// duplicates of frames already played.
func TestJitterBuffer_StrayOldFrameDropped(t *testing.T) {
	jb := NewJitterBuffer(200, 10*time.Millisecond)

	for i := 0; i <= 100; i++ { // 0..2s played
		_ = jb.Add(&Frame{Timestamp: clock().Add(-time.Second), PTS: int64(i) * 20000, Sequence: uint32(i + 1)})
		if _, err := jb.Get(); err != nil {
			t.Fatalf("Get %d: %v", i, err)
		}
	}
	_ = jb.Add(&Frame{Timestamp: clock().Add(-time.Second), PTS: 2020000, Sequence: 102})

	if err := jb.Add(&Frame{Timestamp: clock(), PTS: 500000, Sequence: 26}); !errors.Is(err, ErrLateFrame) {
		t.Errorf("Add(PTS 0.5s, 1.5s late) = %v, want ErrLateFrame", err)
	}
	if f, err := jb.Get(); err != nil || f.PTS != 2020000 {
		t.Errorf("Get after stray frame = %v, %v; want the ready PTS 2020000", f, err)
	}
	if err := jb.Add(&Frame{Timestamp: clock(), PTS: 1800000, Sequence: 91}); !errors.Is(err, ErrLateFrame) {
		t.Errorf("Add(duplicate of played PTS 1.8s) = %v, want ErrLateFrame", err)
	}
}

// A frame whose PTS jumped ahead plays when the buffer runs dry, but when the
// sender numbers its frames the ones after it are not late: they must play, not
// be dropped until their PTS catches up with the glitch.
func TestJitterBuffer_OutlierPTSDoesNotDropStream(t *testing.T) {
	jb := NewJitterBuffer(100, 10*time.Millisecond)
	ready := func(pts int64, seq uint32) *Frame {
		return &Frame{Timestamp: clock().Add(-time.Second), PTS: pts, Sequence: seq}
	}

	_ = jb.Add(ready(0, 1))
	_ = jb.Add(ready(620000, 2)) // should have been 20000
	for i := 0; i < 2; i++ {
		if _, err := jb.Get(); err != nil {
			t.Fatalf("Get: %v", err)
		}
	}
	played := 0
	for i := 0; i < 29; i++ {
		if err := jb.Add(ready(int64(i+2)*20000, uint32(i+3))); err != nil {
			t.Errorf("Add(PTS %d, seq %d) = %v", (i+2)*20000, i+3, err)
			continue
		}
		if _, err := jb.Get(); err == nil {
			played++
		}
	}
	if played != 29 {
		t.Errorf("played %d of the 29 frames after the glitch, want all", played)
	}

	// Without sequence numbers the glitch costs at most a restart run.
	jb = NewJitterBuffer(100, 10*time.Millisecond)
	_ = jb.Add(ready(0, 0))
	_ = jb.Add(ready(620000, 0))
	_, _ = jb.Get()
	_, _ = jb.Get()
	lost := 0
	for i := 0; i < 29; i++ {
		if err := jb.Add(ready(int64(i+2)*20000, 0)); err != nil {
			lost++
		}
		_, _ = jb.Get()
	}
	if lost >= restartRun {
		t.Errorf("unnumbered stream lost %d frames after the glitch, want fewer than %d", lost, restartRun)
	}
}

// Clear forgets the playout point.
func TestJitterBuffer_ClearForgetsPlayoutPoint(t *testing.T) {
	jb := NewJitterBuffer(10, 10*time.Millisecond)

	_ = jb.Add(readyFrame(5000))
	_, _ = jb.Get()
	jb.Clear()
	if err := jb.Add(readyFrame(4000)); err != nil {
		t.Errorf("Add after Clear = %v, want nil", err)
	}
}

// The synchronizer counts a late frame as dropped, and a frame it holds back
// to wait for the other stream is neither lost nor treated as played.
func TestAVSync_LateFrameCountedAndWaitKeepsFrame(t *testing.T) {
	advance := fakeClock(t)
	avs := NewAVSynchronizer()

	_ = avs.AddAudioFrame([]byte("a"), 0, 1)
	_ = avs.AddAudioFrame([]byte("a"), 40000, 3)
	advance(60 * time.Millisecond)
	for i := 0; i < 2; i++ {
		if _, err := avs.GetNextAudioFrame(); err != nil {
			t.Fatalf("GetNextAudioFrame: %v", err)
		}
	}
	if err := avs.AddAudioFrame([]byte("late"), 20000, 2); err == nil {
		t.Error("AddAudioFrame(late) accepted, want error")
	}
	if st := avs.GetStatistics(); st.DroppedAudio != 1 {
		t.Errorf("DroppedAudio = %d, want 1", st.DroppedAudio)
	}

	// Audio 300ms ahead of video waits and stays buffered; it plays once the
	// video head that held it back is gone.
	_ = avs.AddAudioFrame([]byte("a"), 340000, 4)
	_ = avs.AddVideoFrame([]byte("v"), 40000, 1)
	advance(120 * time.Millisecond)
	if f, err := avs.GetNextAudioFrame(); err == nil || f != nil {
		t.Fatalf("GetNextAudioFrame = %v, %v; want wait", f, err)
	}
	if n := avs.audioBuffer.Size(); n != 1 {
		t.Fatalf("audio buffer size = %d, want 1 (waiting frame kept)", n)
	}
	_, _ = avs.GetNextVideoFrame() // video is 300ms behind and is skipped
	f, err := avs.GetNextAudioFrame()
	if err != nil || f.PTS != 340000 {
		t.Errorf("GetNextAudioFrame after wait = %v, %v; want PTS 340000", f, err)
	}
}

// Discarding an implausibly-ahead frame does not make it the playout point,
// so the stream's next real frames are still accepted.
func TestAVSync_DiscardedFrameIsNotPlayoutPoint(t *testing.T) {
	advance := fakeClock(t)
	avs := NewAVSynchronizer()

	_ = avs.AddAudioFrame([]byte("bogus"), int64(5500*time.Millisecond/time.Microsecond), 1)
	_ = avs.AddVideoFrame([]byte("v"), 0, 1)
	advance(120 * time.Millisecond)
	if _, err := avs.GetNextAudioFrame(); err == nil {
		t.Fatal("GetNextAudioFrame accepted the bogus frame")
	}
	if err := avs.AddAudioFrame([]byte("a"), int64(4600*time.Millisecond/time.Microsecond), 2); err != nil {
		t.Errorf("AddAudioFrame after discard = %v, want nil", err)
	}
}
