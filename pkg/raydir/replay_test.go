package raydir

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// A recording round-trips through encode/decode (and a file), time-sorted.
func TestRecordingRoundTrip(t *testing.T) {
	r := NewRecorder()
	r.Region(500, Region{Index: 1, At: raytrace.Vec3{Z: 20}, Spec: fallbackSpec()})
	r.Region(100, Region{Index: 0, At: raytrace.Vec3{Z: 8}, Spec: fallbackSpec()})
	r.Poses(200, PoseSet{7: {Pos: raytrace.Vec3{X: 1}}})
	r.Chat(300, []ChatMsg{{ID: 1, Sender: "a", Text: "hi"}})
	r.Env(400, 0.42)

	ev := r.Events()
	if len(ev) != 5 || ev[0].TMs != 100 || ev[4].TMs != 500 {
		t.Fatalf("events should be time-sorted, got %d events first=%d last=%d", len(ev), ev[0].TMs, ev[len(ev)-1].TMs)
	}
	got, err := DecodeRecording(EncodeRecording(ev))
	if err != nil || !reflect.DeepEqual(ev, got) {
		t.Fatalf("recording round-trip mismatch: err=%v", err)
	}
	path := filepath.Join(t.TempDir(), "walk.rrec")
	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadRecording(path)
	if err != nil || !reflect.DeepEqual(ev, loaded) {
		t.Fatalf("file round-trip mismatch: err=%v", err)
	}
}

// Decoding rejects corrupt input.
func TestDecodeRecordingBad(t *testing.T) {
	if _, err := DecodeRecording(nil); err == nil {
		t.Error("empty should error")
	}
	if _, err := DecodeRecording([]byte("XXXX\x00\x00\x00\x01")); err == nil {
		t.Error("bad magic should error")
	}
}

// The Player rebuilds the world, poses, chat and time as the timeline advances.
func TestPlayerReconstructs(t *testing.T) {
	r := NewRecorder()
	crystal := brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "mesh", Name: "crystal", Y: 1, R: 1}}}
	r.Region(100, Region{Index: 0, At: raytrace.Vec3{Z: 8}, Spec: crystal})
	r.Poses(200, PoseSet{7: {Pos: raytrace.Vec3{X: 2, Y: 1}}})
	r.Chat(300, []ChatMsg{{ID: 1, Sender: "alice", Text: "hello"}})
	r.Env(400, 0.5)
	r.Region(500, Region{Index: 1, At: raytrace.Vec3{Z: 20}, Spec: crystal})

	p := NewPlayer(r.Events())
	if p.Duration() != 500 {
		t.Fatalf("duration = %d, want 500", p.Duration())
	}
	p.Advance(0)
	if p.World().Chunks() != 0 {
		t.Errorf("at t=0 the world should be empty, got %d chunks", p.World().Chunks())
	}
	p.Advance(250) // first region + poses applied
	if p.World().Chunks() != 1 {
		t.Errorf("at t=250 one region should be applied, got %d", p.World().Chunks())
	}
	if len(p.Poses()) != 1 {
		t.Errorf("at t=250 one walker's pose should be present, got %d", len(p.Poses()))
	}
	p.Advance(600) // everything applied
	if p.World().Chunks() != 2 {
		t.Errorf("at the end both regions should be applied, got %d", p.World().Chunks())
	}
	if len(p.Chat().Lines()) == 0 {
		t.Error("chat should have been replayed")
	}
	if scene := p.World().SceneWith(nil); scene == nil {
		t.Error("replayed world should build a scene")
	}
}
