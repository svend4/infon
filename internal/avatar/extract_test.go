package avatar

import (
	"context"
	"encoding/json"
	"image"
	gocolor "image/color"
	"net/http"
	"net/http/httptest"
	"testing"
)

func frame(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, gocolor.RGBA{R: 180, G: 160, B: 140, A: 255})
		}
	}
	return img
}

func TestExtractParsesLandmarks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["image"].(string); !ok {
			t.Error("request missing base64 image")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"points": [][2]float32{{0.5, 0.5}, {0.25, 0.75}},
			"yaw":    0.1, "pitch": -0.2, "roll": 0.0,
		})
	}))
	defer srv.Close()

	kp, err := NewExtractor(srv.URL).Extract(context.Background(), frame(64, 64), 7)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if kp.Frame != 7 {
		t.Errorf("frame = %d, want 7", kp.Frame)
	}
	if len(kp.Points) != 2 {
		t.Fatalf("got %d points, want 2", len(kp.Points))
	}
	if kp.Yaw != 0.1 || kp.Pitch != -0.2 {
		t.Errorf("pose mismatch: %+v", kp)
	}
}

func TestExtractHandlesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := NewExtractor(srv.URL).Extract(context.Background(), frame(8, 8), 0); err == nil {
		t.Error("expected error on HTTP 500")
	}
}

// TestExtractEncodeRoundTrip ties the sender path together: extract -> encode ->
// decode yields the same landmarks (the data that travels on the wire).
func TestExtractEncodeRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"points": [][2]float32{{0.1, 0.2}, {0.9, 0.8}, {0.5, 0.5}},
		})
	}))
	defer srv.Close()

	kp, err := NewExtractor(srv.URL).Extract(context.Background(), frame(32, 32), 3)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	wire := kp.Encode()
	got, err := DecodeKeypoints(wire)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Points) != 3 || got.Frame != 3 {
		t.Errorf("round trip mismatch: %+v", got)
	}
	t.Logf("3 landmarks travel in %d bytes", len(wire))
}
