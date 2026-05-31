package aisource

import (
	"context"
	"encoding/json"
	"image"
	gocolor "image/color"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/sketch"
)

// directorServer returns a brain that always plans a fixed sketch.
func directorServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sk := sketch.Sketch{Sky: "orange", Ground: "navy", Shapes: []sketch.Shape{
			{Kind: "sun", X: 48, Y: 6, W: 3, Color: "gold"},
			{Kind: "mountain", X: 16, H: 8, Color: "darkgreen"},
		}}
		data, _ := json.Marshal(sk)
		_ = json.NewEncoder(w).Encode(brain.Response{Protocol: brain.Protocol, Kind: "sketch", Sketch: data})
	}))
}

func TestDirectorPainterIsNeuralBackend(t *testing.T) {
	var _ device.NeuralBackend = NewDirectorPainter("http://x", nil)
}

func TestDirectorPainterProceduralName(t *testing.T) {
	if NewDirectorPainter("http://x", nil).Name() != "director+procedural" {
		t.Error("name should be director+procedural when no painter")
	}
}

func TestDirectorRendersPlanProcedurally(t *testing.T) {
	srv := directorServer(t)
	defer srv.Close()

	dp := NewDirectorPainter(srv.URL, nil)
	img, err := dp.Generate(context.Background(), "a calm bay", 128, 96)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 128 || b.Dy() != 96 {
		t.Fatalf("image = %dx%d, want 128x96", b.Dx(), b.Dy())
	}
	if allBlack(img) {
		t.Error("rendered plan should not be all black")
	}
}

// recordingPainter captures the prompt it receives, to prove the director's plan
// enriches what the painter is asked to draw.
type recordingPainter struct{ lastPrompt string }

func (p *recordingPainter) Name() string { return "recording" }
func (p *recordingPainter) Generate(ctx context.Context, prompt string, w, h int) (image.Image, error) {
	p.lastPrompt = prompt
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.SetRGBA(0, 0, gocolor.RGBA{R: 1, A: 255})
	return img, nil
}

func TestDirectorEnrichesPainterPrompt(t *testing.T) {
	srv := directorServer(t)
	defer srv.Close()

	painter := &recordingPainter{}
	dp := NewDirectorPainter(srv.URL, painter)
	if !strings.HasPrefix(dp.Name(), "director+recording") {
		t.Errorf("name = %q", dp.Name())
	}
	if _, err := dp.Generate(context.Background(), "a calm bay", 64, 64); err != nil {
		t.Fatalf("generate: %v", err)
	}
	// The painter must have been asked for the base prompt PLUS the director's
	// planned sky/ground/elements.
	p := painter.lastPrompt
	for _, want := range []string{"a calm bay", "orange sky", "navy ground", "sun", "mountain"} {
		if !strings.Contains(p, want) {
			t.Errorf("enriched prompt %q missing %q", p, want)
		}
	}
}

func TestDirectorFallsBackWhenDirectorDown(t *testing.T) {
	// Director endpoint is dead -> Generate must still return a frame (local
	// procedural fallback), not error.
	dp := NewDirectorPainter("http://127.0.0.1:1/v1/decide", nil)
	img, err := dp.Generate(context.Background(), "a storm at night", 64, 64)
	if err != nil {
		t.Fatalf("should fall back, not error: %v", err)
	}
	if img.Bounds().Dx() != 64 {
		t.Errorf("fallback image = %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestEnrichPrompt(t *testing.T) {
	sk := sketch.Sketch{Sky: "purple", Ground: "teal", Shapes: []sketch.Shape{
		{Kind: "moon"}, {Kind: "star"}, {Kind: "star"}, // duplicate star
	}}
	got := enrichPrompt("base scene", sk)
	if !strings.Contains(got, "purple sky") || !strings.Contains(got, "teal ground") {
		t.Errorf("missing sky/ground: %q", got)
	}
	// Duplicate kinds collapse to one mention.
	if strings.Count(got, "star") != 1 {
		t.Errorf("star should appear once, got %q", got)
	}
}
