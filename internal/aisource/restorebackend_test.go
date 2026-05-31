package aisource

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	gocolor "image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/svend4/infon/internal/device"
)

// greenBackend yields a solid green image (the "pre-restoration" frame).
type greenBackend struct{}

func (greenBackend) Name() string { return "green" }
func (greenBackend) Generate(ctx context.Context, prompt string, w, h int) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, gocolor.RGBA{G: 255, A: 255})
		}
	}
	return img, nil
}

func pngBytes(t *testing.T, c gocolor.RGBA, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRestoreIsNeuralBackend(t *testing.T) {
	var _ device.NeuralBackend = NewRestoreBackend(greenBackend{}, RestoreConfig{URL: "http://x"})
}

func TestRestoreNameWraps(t *testing.T) {
	b := NewRestoreBackend(greenBackend{}, RestoreConfig{URL: "http://x"})
	if b.Name() != "restore/green" {
		t.Errorf("name = %q, want restore/green", b.Name())
	}
}

func TestRestoreUsesModelOutput(t *testing.T) {
	// Sidecar returns a solid-RED restored image; the backend must serve that
	// (not the inner green frame).
	red := gocolor.RGBA{R: 255, A: 255}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Confirm the request carried a base64 image + scale.
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["image"].(string); !ok {
			t.Error("request missing base64 image")
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes(t, red, 8, 8))
	}))
	defer srv.Close()

	b := NewRestoreBackend(greenBackend{}, RestoreConfig{URL: srv.URL, Scale: 2})
	img, err := b.Generate(context.Background(), "x", 8, 8)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	r, g, _, _ := img.At(0, 0).RGBA()
	if uint8(r>>8) != 255 || uint8(g>>8) != 0 {
		t.Errorf("expected restored RED, got R=%d G=%d", uint8(r>>8), uint8(g>>8))
	}
}

func TestRestoreFallsBackOnError(t *testing.T) {
	// Sidecar returns 500 -> backend must serve the inner (green) frame, not fail.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	b := NewRestoreBackend(greenBackend{}, RestoreConfig{URL: srv.URL})
	img, err := b.Generate(context.Background(), "x", 8, 8)
	if err != nil {
		t.Fatalf("generate should not error (graceful fallback): %v", err)
	}
	_, g, _, _ := img.At(0, 0).RGBA()
	if uint8(g>>8) != 255 {
		t.Errorf("fallback should be inner green frame, got G=%d", uint8(g>>8))
	}
}

func TestRestoreBase64JSONResponse(t *testing.T) {
	blue := gocolor.RGBA{B: 255, A: 255}
	b64 := base64.StdEncoding.EncodeToString(pngBytes(t, blue, 8, 8))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"b64_json": b64}}})
	}))
	defer srv.Close()

	b := NewRestoreBackend(greenBackend{}, RestoreConfig{URL: srv.URL, B64Path: "data.0.b64_json"})
	img, err := b.Generate(context.Background(), "x", 8, 8)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	_, _, bl, _ := img.At(0, 0).RGBA()
	if uint8(bl>>8) != 255 {
		t.Errorf("expected restored BLUE from JSON b64, got B=%d", uint8(bl>>8))
	}
}

func TestRestoreFromEnvPassthrough(t *testing.T) {
	t.Setenv("RESTORE_API_URL", "")
	got := NewRestoreBackendFromEnv(greenBackend{})
	if got.Name() != "green" {
		t.Errorf("with no RESTORE_API_URL, should return inner unchanged, got %q", got.Name())
	}
	// nil inner stays nil.
	if NewRestoreBackendFromEnv(nil) != nil {
		t.Error("nil inner should stay nil")
	}
	t.Setenv("RESTORE_API_URL", "http://example.invalid")
	if NewRestoreBackendFromEnv(greenBackend{}).Name() != "restore/green" {
		t.Error("with RESTORE_API_URL set, inner should be wrapped")
	}
}
