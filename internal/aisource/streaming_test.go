package aisource

import (
	"context"
	"image"
	gocolor "image/color"
	"testing"

	"github.com/svend4/infon/internal/device"
)

// constBackend always returns a solid image of a fixed color.
type constBackend struct{ c gocolor.RGBA }

func (constBackend) Name() string { return "const" }
func (b constBackend) Generate(ctx context.Context, prompt string, w, h int) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, b.c)
		}
	}
	return img, nil
}

func TestStreamingIsNeuralBackend(t *testing.T) {
	var _ device.NeuralBackend = NewStreamingBackend(constBackend{}, 0.5)
}

func TestStreamingNameWraps(t *testing.T) {
	s := NewStreamingBackend(constBackend{}, 0.5)
	if s.Name() != "streaming/const" {
		t.Errorf("name = %q, want streaming/const", s.Name())
	}
}

func TestStreamingFirstFrameIsTarget(t *testing.T) {
	// With no history, the first frame should equal the backend's output.
	red := gocolor.RGBA{R: 255, A: 255}
	s := NewStreamingBackend(constBackend{c: red}, 0.5)
	img, err := s.Generate(context.Background(), "x", 8, 8)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	r, _, _, _ := img.At(0, 0).RGBA()
	if uint8(r>>8) != 255 {
		t.Errorf("first frame R = %d, want 255 (target)", uint8(r>>8))
	}
}

func TestStreamingBlendsTowardTarget(t *testing.T) {
	// Start the history at black by generating once with a black backend, then
	// switch to white and confirm the next frame is BETWEEN black and white
	// (cross-faded), not full white.
	s := NewStreamingBackend(constBackend{c: gocolor.RGBA{A: 255}}, 0.5) // black
	_, _ = s.Generate(context.Background(), "x", 8, 8)                   // history = black

	// Swap inner to white.
	s.inner = constBackend{c: gocolor.RGBA{R: 255, G: 255, B: 255, A: 255}}
	img, _ := s.Generate(context.Background(), "x", 8, 8)
	r, _, _, _ := img.At(4, 4).RGBA()
	r8 := uint8(r >> 8)
	if r8 == 0 || r8 == 255 {
		t.Errorf("blended frame R = %d, expected strictly between black and white", r8)
	}
}

func TestStreamingConvergesToTarget(t *testing.T) {
	// Repeatedly generating the same white target must converge toward white.
	s := NewStreamingBackend(constBackend{c: gocolor.RGBA{A: 255}}, 0.6) // black history
	_, _ = s.Generate(context.Background(), "x", 4, 4)
	s.inner = constBackend{c: gocolor.RGBA{R: 255, G: 255, B: 255, A: 255}}

	var last uint8
	for i := 0; i < 12; i++ {
		img, _ := s.Generate(context.Background(), "x", 4, 4)
		r, _, _, _ := img.At(0, 0).RGBA()
		last = uint8(r >> 8)
	}
	if last < 240 {
		t.Errorf("after many frames R = %d, expected near 255 (converged)", last)
	}
}

func TestStreamingResetDropsHistory(t *testing.T) {
	s := NewStreamingBackend(constBackend{c: gocolor.RGBA{A: 255}}, 0.5)
	_, _ = s.Generate(context.Background(), "x", 4, 4)
	s.Reset()
	// After reset, switching to white should yield full white immediately (no
	// history to blend from).
	s.inner = constBackend{c: gocolor.RGBA{R: 255, G: 255, B: 255, A: 255}}
	img, _ := s.Generate(context.Background(), "x", 4, 4)
	r, _, _, _ := img.At(0, 0).RGBA()
	if uint8(r>>8) != 255 {
		t.Errorf("after reset R = %d, want 255", uint8(r>>8))
	}
}

func TestStreamingClampsCoherence(t *testing.T) {
	if NewStreamingBackend(constBackend{}, 0).Coherence != 0.5 {
		t.Error("coherence 0 should default to 0.5")
	}
	if NewStreamingBackend(constBackend{}, 2).Coherence != 0.5 {
		t.Error("coherence >1 should default to 0.5")
	}
	if NewStreamingBackend(constBackend{}, 0.3).Coherence != 0.3 {
		t.Error("valid coherence should be kept")
	}
}
