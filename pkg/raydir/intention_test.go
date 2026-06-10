package raydir

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/brain"
)

func TestIntentionBuilds(t *testing.T) {
	w := NewWorld()
	w.SetIntention("a forest of crystals")
	if _, s := w.Intention(); s != 0 {
		t.Errorf("a fresh intention starts at zero strength, got %.2f", s)
	}
	for i := 0; i < 100; i++ {
		w.HoldIntention(0.1)
	}
	if _, s := w.Intention(); s < 0.99 {
		t.Errorf("a long-held intention should reach full strength, got %.2f", s)
	}
	// changing the theme resets the strength
	w.SetIntention("a city of glass")
	if _, s := w.Intention(); s != 0 {
		t.Errorf("changing the intention should reset strength, got %.2f", s)
	}
	w.ClearIntention()
	if th, s := w.Intention(); th != "" || s != 0 {
		t.Error("clearing should drop the intention")
	}
}

func TestIntendPrompt(t *testing.T) {
	w := NewWorld()
	if w.IntendPrompt("a plain") != "a plain" {
		t.Error("with no intention the prompt is unchanged")
	}
	w.SetIntention("a forest")
	// a flicker: still mostly the base
	if w.IntendPrompt("a plain") != "a plain" {
		t.Error("a weak intention should not yet change the prompt")
	}
	for i := 0; i < 16; i++ { // build to the colouring band (~0.4)
		w.HoldIntention(0.1)
	}
	if got := w.IntendPrompt("a plain"); !strings.Contains(got, "a forest") || !strings.Contains(got, "a plain") {
		t.Errorf("a building intention should colour the prompt, got %q", got)
	}
	for i := 0; i < 50; i++ { // sustain to full manifestation
		w.HoldIntention(0.1)
	}
	if got := w.IntendPrompt("a plain"); got != "a forest" {
		t.Errorf("a sustained intention should take over the prompt, got %q", got)
	}
}

// A sustained intention manifests: the world the director grows becomes the theme.
func TestIntentionManifests(t *testing.T) {
	w := NewWorld()
	b := brain.Local{}
	hasTrees := func(prompt string) bool {
		_, spec, _ := AuthorScene(b, prompt)
		for _, o := range spec.Objects {
			if o.Kind == "tree" {
				return true
			}
		}
		return false
	}
	base := "a bare plain" // no trees on its own
	if hasTrees(base) {
		t.Skip("base prompt unexpectedly has trees")
	}
	w.SetIntention("a forest with many trees")
	for i := 0; i < 50; i++ {
		w.HoldIntention(0.1) // sustain to full
	}
	if !hasTrees(w.IntendPrompt(base)) {
		t.Error("a sustained intention for a forest should manifest trees in the grown world")
	}
}
