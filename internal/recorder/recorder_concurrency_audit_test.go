package recorder

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/svend4/infon/pkg/terminal"
)

// In a call the video and audio goroutines keep recording while the main
// goroutine stops the recording. Run under -race: the recorder must be free of
// data races, and the file must load with header counts that match its entries.
func TestRecorderConcurrentStopAudit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "call.tvcp")
	rec := NewRecorder(4, 2, 8000)
	if err := rec.Start(path); err != nil {
		t.Fatalf("Start: %v", err)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	record := func(fn func()) {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				fn()
			}
		}
	}
	wg.Add(3)
	go record(func() {
		if rec.IsRecording() {
			_ = rec.RecordFrame(terminal.NewFrame(4, 2))
		}
	})
	go record(func() { _ = rec.RecordAudio([]int16{1, 2, 3, 4}) })
	go record(func() { _, _, _ = rec.GetStats() })

	for {
		if f, a, _ := rec.GetStats(); f >= 200 && a >= 200 {
			break
		}
	}
	if err := rec.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	close(stop)
	wg.Wait()

	p := NewPlayer()
	if err := p.Load(path); err != nil {
		t.Fatalf("Load after concurrent Stop: %v", err)
	}
	meta := p.GetInfo()
	if int(meta.FrameCount) != len(p.recording.Frames) || int(meta.AudioChunks) != len(p.recording.Audio) {
		t.Fatalf("header says %d frames / %d audio, file holds %d / %d",
			meta.FrameCount, meta.AudioChunks, len(p.recording.Frames), len(p.recording.Audio))
	}
}
