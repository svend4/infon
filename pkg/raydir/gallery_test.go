package raydir

import (
	"path/filepath"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// A saved world summarises to its region count and place names; a recording to its
// event count and duration.
func TestSummaries(t *testing.T) {
	dir := t.TempDir()
	_, forest, _ := AuthorScene(brain.Local{}, "a quiet forest")
	wpath := filepath.Join(dir, "w.rwld")
	if err := SaveWorld(wpath, []Region{
		{Index: 0, At: raytrace.Vec3{Z: 8}, Spec: forest},
		{Index: 1, At: raytrace.Vec3{Z: 20}, Spec: fallbackSpec()},
	}); err != nil {
		t.Fatal(err)
	}
	wi, err := SummarizeWorld(wpath)
	if err != nil || wi.Regions != 2 {
		t.Fatalf("world summary = %+v, %v", wi, err)
	}
	if wi.Places[0] != "Forest" || wi.Bytes <= 0 {
		t.Errorf("expected a named Forest place and a size, got %+v", wi)
	}

	rpath := filepath.Join(dir, "r.rrec")
	rec := NewRecorder()
	rec.Region(100, Region{Index: 0, Spec: forest})
	rec.Chat(500, []ChatMsg{{ID: 1, Sender: "a", Text: "hi"}})
	if err := rec.Save(rpath); err != nil {
		t.Fatal(err)
	}
	ri, err := SummarizeRecording(rpath)
	if err != nil || ri.Events != 2 || ri.DurationMs != 500 {
		t.Fatalf("recording summary = %+v, %v", ri, err)
	}
}

// The director bench scores the reference author as fully renderable and varied.
func TestBenchDirector(t *testing.T) {
	r := BenchDirector(brain.Local{}, []string{
		"a calm world", "a quiet forest with a house", "a crystal cave", "a calm lake",
	})
	if r.N != 4 || r.Renderable != 4 {
		t.Errorf("reference director should render every prompt, got %d/%d", r.Renderable, r.N)
	}
	if r.Errors != 0 {
		t.Errorf("offline director should have no transport errors, got %d", r.Errors)
	}
	if r.Variety() < 2 || r.AvgObjects() <= 0 {
		t.Errorf("expected varied, non-empty scenes: variety=%d avg=%.1f", r.Variety(), r.AvgObjects())
	}
}
