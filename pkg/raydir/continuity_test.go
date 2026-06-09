package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func countKind(s brain.SceneSpec, kind string) int {
	n := 0
	for _, o := range s.Objects {
		if o.Kind == kind {
			n++
		}
	}
	return n
}

// Authored regions connect into one place: a later region inherits the previous
// region's sky and lays a path of stepping stones leading back toward it; the
// first region has neither.
func TestRegionContinuity(t *testing.T) {
	w := NewWorld()
	r0, _, err := w.AuthorRegion(brain.Local{}, "a calm world at night", 0, raytrace.Vec3{X: 0, Y: 0, Z: 0})
	if err != nil {
		t.Fatal(err)
	}
	r1, _, err := w.AuthorRegion(brain.Local{}, "a calm world", 1, raytrace.Vec3{X: 0, Y: 0, Z: 12})
	if err != nil {
		t.Fatal(err)
	}

	// sky carried: region 1 (no "night" in its prompt) still inherits region 0's
	// night sky, instead of reverting to the default bright sky.
	if r1.Spec.SkyTop != r0.Spec.SkyTop {
		t.Errorf("region 1 should inherit region 0's sky, got %v want %v", r1.Spec.SkyTop, r0.Spec.SkyTop)
	}
	// the first region has no path; the second has a stepping-stone path back.
	if countKind(r0.Spec, "cylinder") != 0 {
		t.Error("the first region should have no path connector")
	}
	if countKind(r1.Spec, "cylinder") == 0 {
		t.Error("a later region should lay a path back toward the previous one")
	}
	// the path leads back (toward -Z, since the walker moved +Z).
	backward := false
	for _, o := range r1.Spec.Objects {
		if o.Kind == "cylinder" && o.Z < -0.5 {
			backward = true
		}
	}
	if !backward {
		t.Error("the path stones should lead back toward the previous region")
	}
}

// Continuity context never makes the scene unrenderable, and AuthorScene (no
// context) is unchanged — first regions author exactly as before.
func TestContinuityKeepsRenderable(t *testing.T) {
	_, spec, err := AuthorScene(brain.Local{}, "a calm world")
	if err != nil {
		t.Fatal(err)
	}
	if !hasRenderable(spec) {
		t.Error("a contextless author should still be renderable")
	}
	if countKind(spec, "cylinder") != 0 {
		t.Error("without a heading there should be no path connector")
	}
}
