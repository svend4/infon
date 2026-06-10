package raydir

import (
	"strings"
	"testing"
)

// Choosing walks the forks in order, applies the chosen prompt, records the path,
// and stops when the forks run out.
func TestBranchingChoose(t *testing.T) {
	br := DefaultBranching()
	if br.Prompt() != "a crossroads of paths" {
		t.Errorf("before any choice the prompt should be neutral, got %q", br.Prompt())
	}
	f, ok := br.PendingFork()
	if !ok || f.Question == "" {
		t.Fatal("the first fork should be pending")
	}
	ch, ok := br.Choose(true) // take the right arm of fork 0
	if !ok || ch.Label != f.Right.Label {
		t.Errorf("Choose(right) should return the right arm, got %+v", ch)
	}
	if br.Prompt() != f.Right.Prompt {
		t.Errorf("the active prompt should be the chosen branch's, got %q", br.Prompt())
	}
	// choose through the rest
	for {
		if _, ok := br.PendingFork(); !ok {
			break
		}
		br.Choose(false)
	}
	if !br.Done() {
		t.Error("after choosing every fork, branching should be Done")
	}
	if _, ok := br.Choose(true); ok {
		t.Error("choosing past the last fork should fail")
	}
	if got := br.Path(); len(got) != len(br.Forks) {
		t.Errorf("the path should have one label per fork, got %v", got)
	}
}

// Different choices at the first fork lead to different worlds.
func TestBranchingDiverges(t *testing.T) {
	left, right := DefaultBranching(), DefaultBranching()
	lc, _ := left.Choose(false)
	rc, _ := right.Choose(true)
	if lc.Prompt == rc.Prompt || lc.Label == rc.Label {
		t.Errorf("the two arms of a fork should differ: %q vs %q", lc.Label, rc.Label)
	}
}

// The built-in set is well-formed.
func TestDefaultBranching(t *testing.T) {
	br := DefaultBranching()
	if len(br.Forks) < 3 {
		t.Errorf("expected several forks, got %d", len(br.Forks))
	}
	for i, f := range br.Forks {
		if strings.TrimSpace(f.Question) == "" ||
			f.Left.Label == "" || f.Left.Prompt == "" || f.Right.Label == "" || f.Right.Prompt == "" {
			t.Errorf("fork %d is missing a field: %+v", i, f)
		}
	}
}
