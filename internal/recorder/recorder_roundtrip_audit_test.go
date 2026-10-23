package recorder

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// TestRecorderRoundTripAudit records frames and audio, stops (which writes
// the file) and loads the file back: everything recorded must come back.
func TestRecorderRoundTripAudit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "call.tvcp")

	rec := NewRecorder(3, 2, 8000)
	if err := rec.Start(path); err != nil {
		t.Fatalf("Start: %v", err)
	}

	f1 := terminal.NewFrame(3, 2)
	f1.SetBlock(0, 0, 'A', color.RGB{R: 1, G: 2, B: 3}, color.RGB{R: 4, G: 5, B: 6})
	f1.SetBlock(2, 1, '█', color.RGB{R: 255, G: 0, B: 128}, color.RGB{R: 7, G: 8, B: 9})
	f2 := terminal.NewFrame(3, 2)
	f2.SetBlock(1, 0, 'z', color.RGB{R: 10, G: 20, B: 30}, color.RGB{R: 40, G: 50, B: 60})

	if err := rec.RecordFrame(f1); err != nil {
		t.Fatalf("RecordFrame: %v", err)
	}
	if err := rec.RecordAudio([]int16{0, 1, -1, 32767, -32768}); err != nil {
		t.Fatalf("RecordAudio: %v", err)
	}
	if err := rec.RecordFrame(f2); err != nil {
		t.Fatalf("RecordFrame: %v", err)
	}
	if err := rec.RecordAudio([]int16{42}); err != nil {
		t.Fatalf("RecordAudio: %v", err)
	}
	if err := rec.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	p := NewPlayer()
	if err := p.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := p.GetRecording()

	// The file stores the start time with second precision.
	want := rec.metadata
	gm := got.Metadata
	if gm.StartTime.Unix() != want.StartTime.Unix() {
		t.Errorf("StartTime = %d, want %d", gm.StartTime.Unix(), want.StartTime.Unix())
	}
	gm.StartTime, want.StartTime = time.Time{}, time.Time{}
	if gm != want {
		t.Errorf("metadata = %+v, want %+v", gm, want)
	}
	if gm.FrameCount != 2 || gm.AudioChunks != 2 {
		t.Errorf("counts = %d frames / %d audio, want 2 / 2", gm.FrameCount, gm.AudioChunks)
	}

	if len(got.Frames) != len(rec.frames) {
		t.Fatalf("got %d frames, want %d", len(got.Frames), len(rec.frames))
	}
	for i := range rec.frames {
		if got.Frames[i].Timestamp != rec.frames[i].Timestamp {
			t.Errorf("frame %d timestamp = %d, want %d", i, got.Frames[i].Timestamp, rec.frames[i].Timestamp)
		}
		if !reflect.DeepEqual(got.Frames[i].Frame, rec.frames[i].Frame) {
			t.Errorf("frame %d = %+v, want %+v", i, got.Frames[i].Frame, rec.frames[i].Frame)
		}
	}

	if !reflect.DeepEqual(got.Audio, rec.audio) {
		t.Errorf("audio = %+v, want %+v", got.Audio, rec.audio)
	}
}
