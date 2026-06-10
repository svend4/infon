package raydir

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// Answers are deterministic, thematic, and the dream question is deflected.
func TestSpriteAnswer(t *testing.T) {
	s := NewSprite("Mira", raytrace.Vec3{Z: 5}, 2)
	a1 := s.Answer("who are you?")
	a2 := s.Answer("who are you?")
	if a1 != a2 {
		t.Error("answers should be deterministic")
	}
	if !strings.Contains(s.Answer("who are you?"), "Mira") {
		t.Error("asked its name, a sprite should say it")
	}
	dream := s.Answer("are we dreaming right now?")
	if !LucidTell(dream) {
		t.Errorf("a sprite should deflect the dream question (the tell), got %q", dream)
	}
	if strings.TrimSpace(s.Answer("tell me anything")) == "" {
		t.Error("a sprite should always answer something")
	}
}

// LucidTell separates a sprite's deflection from a dreamer's admission.
func TestLucidTell(t *testing.T) {
	if !LucidTell("No, this is as real as anything.") {
		t.Error("a denial should read as a sprite")
	}
	if LucidTell("Yes, we're dreaming — I'm lucid.") {
		t.Error("an admission should read as a dreamer, not a sprite")
	}
}

// The sprite drifts about its spot but stays near home.
func TestSpriteWander(t *testing.T) {
	s := NewSprite("Ka", raytrace.Vec3{X: 10, Z: 10}, 1)
	moved := false
	for i := 0; i < 50; i++ {
		s.Step(0.1)
		if s.Pos.Sub(raytrace.Vec3{X: 10, Z: 10}).Len() > 1e-3 {
			moved = true
		}
		if s.Pos.Sub(raytrace.Vec3{X: 10, Z: 10}).Len() > 2 {
			t.Fatalf("a sprite should stay near home, strayed %.2f", s.Pos.Sub(raytrace.Vec3{X: 10, Z: 10}).Len())
		}
	}
	if !moved {
		t.Error("a sprite should drift, not stand frozen")
	}
}

// The world holds sprites, finds the nearest, renders them, and is animated.
func TestWorldSprites(t *testing.T) {
	w := NewWorld()
	base := len(w.SceneWith(nil).Objects)
	w.AddSprite(NewSprite("A", raytrace.Vec3{Z: 5}, 0))
	w.AddSprite(NewSprite("B", raytrace.Vec3{X: 40, Z: 5}, 1))
	if !w.HasSprites() || !w.HasAnimated() {
		t.Fatal("a world with sprites should report them and be animated")
	}
	if got := len(w.SceneWith(nil).Objects); got <= base {
		t.Errorf("sprites should add objects: base %d after %d", base, got)
	}
	near := w.NearestSprite(raytrace.Vec3{Z: 6}, 5)
	if near == nil || near.Name != "A" {
		t.Errorf("the nearest sprite to (0,_,6) should be A, got %v", near)
	}
	if w.NearestSprite(raytrace.Vec3{X: 200}, 5) != nil {
		t.Error("no sprite should be found far from any")
	}
	w.StepSprites(0.1) // should not panic
}
