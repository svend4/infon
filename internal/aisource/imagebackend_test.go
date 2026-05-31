package aisource

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/svend4/infon/internal/device"
)

// tinyPNG returns a 2x2 solid-red PNG as bytes.
func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.Set(x, y, color.RGBA{R: 220, G: 20, B: 20, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestImageBackendRawImageResponse(t *testing.T) {
	png := tinyPNG(t)
	var gotPrompt string
	var gotW, gotH float64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotPrompt, _ = body["prompt"].(string)
		gotW, _ = body["width"].(float64)
		gotH, _ = body["height"].(float64)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	defer srv.Close()

	be := NewImageBackend(ImageConfig{URL: srv.URL})
	img, err := be.Generate(context.Background(), "a red square", 2, 2)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 2 || b.Dy() != 2 {
		t.Fatalf("image = %dx%d, want 2x2", b.Dx(), b.Dy())
	}
	r, _, _, _ := img.At(0, 0).RGBA()
	if uint8(r>>8) != 220 {
		t.Errorf("pixel R = %d, want 220", uint8(r>>8))
	}
	// The request must carry the prompt and the requested size.
	if gotPrompt != "a red square" {
		t.Errorf("prompt sent = %q", gotPrompt)
	}
	if gotW != 2 || gotH != 2 {
		t.Errorf("size sent = %vx%v, want 2x2", gotW, gotH)
	}
}

func TestImageBackendBase64JSONResponse(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString(tinyPNG(t))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// OpenAI-style envelope: {"data":[{"b64_json":"..."}]}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{map[string]any{"b64_json": b64}},
		})
	}))
	defer srv.Close()

	be := NewImageBackend(ImageConfig{URL: srv.URL, B64Path: "data.0.b64_json"})
	img, err := be.Generate(context.Background(), "x", 2, 2)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 2 {
		t.Fatalf("decoded image width = %d, want 2", b.Dx())
	}
}

func TestImageBackendDataURL(t *testing.T) {
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(tinyPNG(t))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"image": dataURL})
	}))
	defer srv.Close()

	be := NewImageBackend(ImageConfig{URL: srv.URL, B64Path: "image"})
	if _, err := be.Generate(context.Background(), "x", 2, 2); err != nil {
		t.Fatalf("Generate with data URL: %v", err)
	}
}

func TestImageBackendHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	be := NewImageBackend(ImageConfig{URL: srv.URL})
	if _, err := be.Generate(context.Background(), "x", 8, 8); err == nil {
		t.Error("expected an error on HTTP 500")
	}
}

func TestImageBackendBadB64Path(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"nothing": "here"})
	}))
	defer srv.Close()

	be := NewImageBackend(ImageConfig{URL: srv.URL, B64Path: "data.0.b64_json"})
	if _, err := be.Generate(context.Background(), "x", 8, 8); err == nil {
		t.Error("expected an error when the b64 path is missing")
	}
}

func TestImageBackendCustomFieldNames(t *testing.T) {
	png := tinyPNG(t)
	var keys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		for k := range body {
			keys = append(keys, k)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	defer srv.Close()

	be := NewImageBackend(ImageConfig{
		URL:         srv.URL,
		PromptField: "text",
		WidthField:  "w",
		HeightField: "h",
		Extra:       map[string]any{"model": "demo"},
	})
	if _, err := be.Generate(context.Background(), "hello", 16, 16); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	want := map[string]bool{"text": true, "w": true, "h": true, "model": true}
	for _, k := range keys {
		delete(want, k)
	}
	if len(want) != 0 {
		t.Errorf("request missing fields: %v (got keys %v)", want, keys)
	}
}

func TestNewImageBackendFromEnv(t *testing.T) {
	t.Setenv("IMAGE_API_URL", "")
	if _, ok := NewImageBackendFromEnv(); ok {
		t.Error("expected ok=false when IMAGE_API_URL is unset")
	}
	t.Setenv("IMAGE_API_URL", "http://example.invalid/v1/images")
	t.Setenv("IMAGE_B64_PATH", "data.0.b64_json")
	be, ok := NewImageBackendFromEnv()
	if !ok || be == nil {
		t.Fatal("expected a backend when IMAGE_API_URL is set")
	}
	if be.cfg.B64Path != "data.0.b64_json" {
		t.Errorf("B64Path = %q, want data.0.b64_json", be.cfg.B64Path)
	}
}

func TestImageBackendIsNeuralBackend(t *testing.T) {
	var _ device.NeuralBackend = NewImageBackend(ImageConfig{URL: "http://x"})
}

func TestLookupString(t *testing.T) {
	var doc any
	_ = json.Unmarshal([]byte(`{"a":{"b":[{"c":"deep"}]}}`), &doc)
	got, err := lookupString(doc, "a.b.0.c")
	if err != nil {
		t.Fatalf("lookupString: %v", err)
	}
	if got != "deep" {
		t.Errorf("got %q, want deep", got)
	}
	if _, err := lookupString(doc, "a.b.9.c"); err == nil {
		t.Error("expected out-of-range error")
	}
	if _, err := lookupString(doc, "a.x"); err == nil {
		t.Error("expected missing-key error")
	}
}
