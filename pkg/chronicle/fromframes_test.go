package chronicle

import (
	"testing"

	"github.com/svend4/infon/pkg/arena"
)

// FromFrames over captured snapshots must equal Record over the same match.
func TestFromFramesMatchesRecord(t *testing.T) {
	rec := Record(board(), arena.Planner{}, arena.RefCommander{}, 60)

	a := board()
	frames := [][]arena.Unit{append([]arena.Unit(nil), a.Units...)}
	for tick := 1; tick <= 60; tick++ {
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		a.Step(arena.Planner{}, arena.RefCommander{})
		frames = append(frames, append([]arena.Unit(nil), a.Units...))
	}
	ff := FromFrames(frames)

	if len(ff.Events) != len(rec.Events) {
		t.Fatalf("event count: FromFrames %d vs Record %d", len(ff.Events), len(rec.Events))
	}
	for i := range rec.Events {
		if ff.Events[i] != rec.Events[i] {
			t.Fatalf("event %d differs: %+v vs %+v", i, ff.Events[i], rec.Events[i])
		}
	}
}
