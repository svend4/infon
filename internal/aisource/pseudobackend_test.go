package aisource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/pseudo"
)

// End-to-end: PseudoBackend -> HTTP -> reference brain -> pseudo-image spec ->
// rendered raster, for every format. Proves a modelless backend paints pixels.
func TestPseudoBackendAllFormats(t *testing.T) {
	srv := httptest.NewServer(brain.Handler())
	defer srv.Close()

	formats := []pseudo.Format{
		"", pseudo.FormatGrid, pseudo.FormatPixels, pseudo.FormatGlyphs,
		pseudo.FormatSigils, pseudo.FormatVector, pseudo.FormatSketch,
	}
	for _, f := range formats {
		b := NewPseudoBackend(srv.URL, f)
		img, err := b.Generate(context.Background(), "a calm harbor at dawn", 320, 144)
		if err != nil {
			t.Fatalf("format %q: %v", f, err)
		}
		if img == nil {
			t.Fatalf("format %q: nil image", f)
		}
		if bnds := img.Bounds(); bnds.Dx() != 320 || bnds.Dy() != 144 {
			t.Fatalf("format %q: bounds = %v, want 320x144", f, bnds)
		}
	}
}

// TVCP_PROMPT_FILE makes the prompt live-editable: its contents override the
// construction prompt on each frame, so the scene can change without a restart.
func TestPseudoBackendPromptFile(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req brain.Request
		_ = json.NewDecoder(r.Body).Decode(&req)
		got = req.Prompt
		_ = json.NewEncoder(w).Encode(brain.Response{
			Protocol: brain.Protocol, Kind: "image",
			Image: json.RawMessage(`{"format":"grid","grid":{"rows":[["navy","gold"],["teal","teal"]]}}`),
		})
	}))
	defer srv.Close()

	f, err := os.CreateTemp("", "tvcpprompt*.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("live harbor at dusk\n")
	_ = f.Close()
	defer func() { _ = os.Remove(f.Name()) }()
	t.Setenv("TVCP_PROMPT_FILE", f.Name())

	b := NewPseudoBackend(srv.URL, pseudo.FormatGrid)
	if _, err := b.Generate(context.Background(), "ORIGINAL", 320, 144); err != nil {
		t.Fatal(err)
	}
	if got != "live harbor at dusk" {
		t.Fatalf("brain saw prompt %q, want the file contents", got)
	}
}
