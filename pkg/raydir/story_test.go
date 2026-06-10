package raydir

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// The story turns the page every `perChapter` regions, in order, and stops on the
// last chapter.
func TestStoryAdvances(t *testing.T) {
	st := NewStory("T", []Chapter{
		{Title: "One", Prompt: "p1", Line: "l1"},
		{Title: "Two", Prompt: "p2", Line: "l2"},
		{Title: "Three", Prompt: "p3", Line: "l3"},
	}, 2)

	if st.Prompt() != "p1" {
		t.Fatalf("should start in chapter one, got %q", st.Prompt())
	}
	// region 0: still chapter one
	if entered, _ := st.Advance(); entered {
		t.Error("after one region the page should not turn yet")
	}
	// region 1: page turns to chapter two
	entered, ch := st.Advance()
	if !entered || ch.Title != "Two" {
		t.Errorf("after two regions we should enter chapter two, got entered=%v %q", entered, ch.Title)
	}
	if st.Prompt() != "p2" {
		t.Errorf("prompt should now be chapter two's, got %q", st.Prompt())
	}
	// regions 2,3: into chapter three
	st.Advance()
	entered, ch = st.Advance()
	if !entered || ch.Title != "Three" {
		t.Errorf("should reach chapter three, got entered=%v %q", entered, ch.Title)
	}
	if !st.Done() {
		t.Error("the last chapter should report Done")
	}
	// further growth stays on the final chapter
	for i := 0; i < 5; i++ {
		if entered, _ := st.Advance(); entered {
			t.Error("there is no chapter after the last")
		}
	}
	if n, total := st.Progress(); n != 3 || total != 3 {
		t.Errorf("progress should be 3/3, got %d/%d", n, total)
	}
}

// The built-in arc is well-formed: several chapters, each fully filled in.
func TestDefaultStory(t *testing.T) {
	st := DefaultStory()
	if _, total := st.Progress(); total < 4 {
		t.Errorf("the default arc should have several chapters, got %d", total)
	}
	for i, c := range st.Chapters {
		if strings.TrimSpace(c.Title) == "" || strings.TrimSpace(c.Prompt) == "" || strings.TrimSpace(c.Line) == "" {
			t.Errorf("chapter %d is missing a field: %+v", i, c)
		}
	}
}

// A beacon is a stack of bright orbs at the given place.
func TestBeaconObjects(t *testing.T) {
	objs := BeaconObjects(raytrace.Vec3{X: 5, Z: 9}, ChapterColor(2))
	if len(objs) == 0 {
		t.Fatal("a beacon should have objects")
	}
	for _, o := range objs {
		sp, ok := o.(raytrace.Sphere)
		if !ok {
			t.Fatalf("a beacon orb should be a sphere, got %T", o)
		}
		if sp.Mat.Emit.LenSq() == 0 {
			t.Error("a beacon orb should glow")
		}
		if sp.Center.X != 5 || sp.Center.Z != 9 {
			t.Errorf("a beacon should stand at its place, got %+v", sp.Center)
		}
	}
}

// The world includes placed decor in its scene.
func TestWorldDecor(t *testing.T) {
	w := NewWorld()
	base := len(w.SceneWith(nil).Objects)
	w.AddDecor(BeaconObjects(raytrace.Vec3{Z: 5}, ChapterColor(0))...)
	if got := len(w.SceneWith(nil).Objects); got <= base {
		t.Errorf("decor should add objects to the scene: base %d after %d", base, got)
	}
}
