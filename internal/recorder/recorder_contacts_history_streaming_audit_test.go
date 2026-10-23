package recorder

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// A call can last for hours, so recorded frames must reach the file as they
// are recorded instead of piling up in memory until Stop. Before Stop the file
// must already hold every frame; after Stop it must load back unchanged and no
// temporary files may be left next to it.
func TestRecorderStreamsFramesToDiskAudit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "call.tvcp")
	const w, h, n = 40, 12, 30

	rec := NewRecorder(w, h, 8000)
	if err := rec.Start(path); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var wantFrames []*terminal.Frame
	var wantAudio [][]int16
	buf := make([]int16, 4)
	for i := 0; i < n; i++ {
		f := terminal.NewFrame(w, h)
		f.SetBlock(i%w, i%h, rune('A'+i%26), color.RGB{R: uint8(i), G: 2, B: 3}, color.RGB{R: 4, G: uint8(i), B: 6})
		if err := rec.RecordFrame(f); err != nil {
			t.Fatalf("RecordFrame %d: %v", i, err)
		}
		wantFrames = append(wantFrames, f)

		// The audio goroutine reuses its capture buffer between chunks.
		for j := range buf {
			buf[j] = int16(i*10 + j)
		}
		if err := rec.RecordAudio(buf); err != nil {
			t.Fatalf("RecordAudio %d: %v", i, err)
		}
		wantAudio = append(wantAudio, append([]int16(nil), buf...))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if minSize := int64(34 + n*(12+4+w*h*10)); info.Size() < minSize {
		t.Errorf("before Stop the file holds %d bytes, want at least %d: recorded frames are kept in memory", info.Size(), minSize)
	}

	if err := rec.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if f, a, _ := rec.GetStats(); f != n || a != n {
		t.Errorf("GetStats = %d frames / %d audio, want %d / %d", f, a, n, n)
	}

	p := NewPlayer()
	if err := p.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := p.GetRecording()
	if got.Metadata.FrameCount != n || got.Metadata.AudioChunks != n {
		t.Fatalf("header counts = %d / %d, want %d / %d", got.Metadata.FrameCount, got.Metadata.AudioChunks, n, n)
	}
	var last uint64
	for i, want := range wantFrames {
		if !reflect.DeepEqual(got.Frames[i].Frame, want) {
			t.Errorf("frame %d does not round-trip", i)
		}
		if ts := got.Frames[i].Timestamp; ts < last || ts > uint64(got.Metadata.Duration) {
			t.Errorf("frame %d timestamp %d outside [%d, %d]", i, ts, last, got.Metadata.Duration)
		}
		last = got.Frames[i].Timestamp
	}
	for i, want := range wantAudio {
		if !reflect.DeepEqual(got.Audio[i].Samples, want) {
			t.Errorf("audio %d = %v, want %v", i, got.Audio[i].Samples, want)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("directory holds %v after Stop, want only call.tvcp", names)
	}
}

// Recording must not keep each frame alive: the live heap after recording
// many fresh frames (as cmd/tvcp/call.go does) must stay roughly flat.
func TestRecorderMemoryBoundedAudit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "call.tvcp")
	const w, h, n = 80, 24, 600 // ~23 KB of blocks per frame, ~14 MB in all

	rec := NewRecorder(w, h, 16000)
	if err := rec.Start(path); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	samples := make([]int16, 320)
	for i := 0; i < n; i++ {
		if err := rec.RecordFrame(terminal.NewFrame(w, h)); err != nil {
			t.Fatalf("RecordFrame %d: %v", i, err)
		}
		if err := rec.RecordAudio(samples); err != nil {
			t.Fatalf("RecordAudio %d: %v", i, err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&after)
	if growth := int64(after.HeapAlloc) - int64(before.HeapAlloc); growth > 4<<20 {
		t.Errorf("live heap grew by %d bytes while recording %d frames: the recording is buffered in memory", growth, n)
	}

	if err := rec.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	p := NewPlayer()
	if err := p.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := p.GetRecording(); len(got.Frames) != n || len(got.Audio) != n {
		t.Errorf("loaded %d frames / %d audio, want %d / %d", len(got.Frames), len(got.Audio), n, n)
	}
}
