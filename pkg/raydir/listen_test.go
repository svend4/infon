package raydir

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// The director's prompt follows the latest usable chat line, but trivial chatter
// falls through to the fallback.
func TestDirectorPrompt(t *testing.T) {
	if got := DirectorPrompt("a forest at night with fireflies", "fallback"); got != "a forest at night with fireflies" {
		t.Errorf("a real wish should steer the director, got %q", got)
	}
	if got := DirectorPrompt("hi", "fallback"); got != "fallback" {
		t.Errorf("idle chatter should fall through, got %q", got)
	}
	if got := DirectorPrompt("   ", "fallback"); got != "fallback" {
		t.Errorf("blank should fall through, got %q", got)
	}
}

// Conversational steering end-to-end: a chat wish, used as the prompt, makes the
// reference director author the matching scene (trees + night + fireflies).
func TestChatSteersDirector(t *testing.T) {
	prompt := DirectorPrompt("a forest at night with fireflies", "a calm world")
	_, spec, _ := AuthorScene(brain.Local{}, prompt)
	hasTree, night, firefly := false, false, false
	for _, o := range spec.Objects {
		if o.Kind == "tree" {
			hasTree = true
		}
		if o.Anim == "wander" {
			firefly = true
		}
	}
	if spec.SkyTop[2] < 0.2 { // night sky is dark
		night = true
	}
	if !hasTree || !night || !firefly {
		t.Errorf("the world should become a night forest with fireflies: tree=%v night=%v firefly=%v", hasTree, night, firefly)
	}
}

// Authored regions are named and remembered as landmarks; fast-travel finds them.
func TestWorldLandmarks(t *testing.T) {
	w := NewWorld()
	_, spec, _ := AuthorScene(brain.Local{}, "a quiet forest path")
	w.AddRegion(Region{Index: 0, At: raytrace.Vec3{X: 0, Z: 10}, Spec: spec})
	w.AddRegion(Region{Index: 1, At: raytrace.Vec3{X: 5, Z: 30}, Spec: brain.SceneSpec{Objects: spec.Objects}}) // unnamed -> default

	lm := w.Landmarks()
	if len(lm) != 2 {
		t.Fatalf("expected 2 landmarks, got %d", len(lm))
	}
	if lm[0].Name != "Forest" {
		t.Errorf("named region should be 'Forest', got %q", lm[0].Name)
	}
	if lm[1].Name != "Region 1" {
		t.Errorf("unnamed region should default-name, got %q", lm[1].Name)
	}
	if l, ok := w.FindLandmark("forest"); !ok || l.At.Z != 10 {
		t.Errorf("fast-travel should find the Forest at z=10, got %+v ok=%v", l, ok)
	}
	if _, ok := w.FindLandmark("nowhere"); ok {
		t.Error("an unknown place should not be found")
	}
}

// The minimap shows the landmarks (by name), other walkers and you.
func TestMinimap(t *testing.T) {
	marks := []Landmark{{Index: 0, At: raytrace.Vec3{X: 0, Z: 10}, Name: "Forest"}, {Index: 1, At: raytrace.Vec3{X: 8, Z: 30}, Name: "Crystal Cave"}}
	others := []raytrace.Vec3{{X: 2, Z: 12}}
	m := Minimap(marks, others, raytrace.Vec3{X: 0, Z: 0}, 24, 10)
	if !strings.Contains(m, "@") {
		t.Error("the map should mark you with '@'")
	}
	if !strings.Contains(m, "Forest") || !strings.Contains(m, "Crystal Cave") {
		t.Error("the legend should list the named places")
	}
	if strings.Count(m, "\n") < 10 {
		t.Error("the map grid should have the requested rows")
	}
}
