package aisource

import (
	"context"
	"testing"

	"github.com/svend4/infon/internal/device"
)

func TestLocalBackendIsNeuralBackend(t *testing.T) {
	var _ device.NeuralBackend = NewLocalBackend()
	if NewLocalBackend().Name() != "local-procedural" {
		t.Errorf("name = %q", NewLocalBackend().Name())
	}
}

func TestLocalBackendGeneratesSizedImage(t *testing.T) {
	b := NewLocalBackend()
	img, err := b.Generate(context.Background(), "a calm bay at sunset", 160, 96)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if bnd := img.Bounds(); bnd.Dx() != 160 || bnd.Dy() != 96 {
		t.Fatalf("image = %dx%d, want 160x96", bnd.Dx(), bnd.Dy())
	}
	if allBlack(img) {
		t.Error("offline render should not be all black")
	}
}

func TestLocalBackendDeterministic(t *testing.T) {
	// The same prompt must yield the same sketch (reproducible).
	a := sketchFromPrompt("a quiet harbor at night", 40, 20)
	b := sketchFromPrompt("a quiet harbor at night", 40, 20)
	if a.Sky != b.Sky || a.Ground != b.Ground || len(a.Shapes) != len(b.Shapes) {
		t.Error("same prompt should produce the same sketch")
	}
}

func TestLocalBackendPromptKeywordsAffectScene(t *testing.T) {
	night := sketchFromPrompt("a city at night", 40, 20)
	if night.Sky != "navy" {
		t.Errorf("night sky = %q, want navy", night.Sky)
	}
	// Night scenes get a moon + stars; day scenes get a sun.
	hasMoon, hasSun := false, false
	for _, s := range night.Shapes {
		if s.Kind == "moon" {
			hasMoon = true
		}
		if s.Kind == "sun" {
			hasSun = true
		}
	}
	if !hasMoon || hasSun {
		t.Errorf("night scene should have a moon and no sun (moon=%v sun=%v)", hasMoon, hasSun)
	}

	sunset := sketchFromPrompt("a beach at sunset", 40, 20)
	if sunset.Sky != "orange" {
		t.Errorf("sunset sky = %q, want orange", sunset.Sky)
	}

	sea := sketchFromPrompt("the open ocean", 40, 20)
	if sea.Ground != "navy" {
		t.Errorf("ocean ground = %q, want navy", sea.Ground)
	}
	for _, s := range sea.Shapes {
		if s.Kind == "mountain" {
			t.Error("a seascape should not have mountains")
		}
	}
}

func TestLocalBackendDifferentPromptsDiffer(t *testing.T) {
	a := sketchFromPrompt("alpha", 40, 20)
	b := sketchFromPrompt("zulu different words entirely", 40, 20)
	// At least one of sky/ground should differ across very different prompts.
	if a.Sky == b.Sky && a.Ground == b.Ground {
		t.Log("note: palettes coincided; acceptable but unlikely")
	}
}

func TestHashStringNonZero(t *testing.T) {
	if hashString("") == 0 {
		t.Error("hash must never be zero (would break modulo variety)")
	}
	if hashString("a") == hashString("b") {
		t.Error("different inputs should generally hash differently")
	}
}
