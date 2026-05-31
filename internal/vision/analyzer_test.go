package vision

import (
	"context"
	"encoding/json"
	"image"
	gocolor "image/color"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func TestPixelBox(t *testing.T) {
	d := Detection{X: 0.25, Y: 0.5, W: 0.5, H: 0.25}
	x0, y0, x1, y1 := d.PixelBox(40, 20)
	if x0 != 10 || y0 != 10 || x1 != 30 || y1 != 15 {
		t.Errorf("box = (%d,%d,%d,%d), want (10,10,30,15)", x0, y0, x1, y1)
	}
}

func TestPixelBoxClamps(t *testing.T) {
	// Out-of-range normalized coords clamp into the grid.
	d := Detection{X: -0.5, Y: -0.5, W: 5, H: 5}
	x0, y0, x1, y1 := d.PixelBox(10, 10)
	if x0 < 0 || y0 < 0 || x1 > 9 || y1 > 9 {
		t.Errorf("box not clamped: (%d,%d,%d,%d)", x0, y0, x1, y1)
	}
}

func solid(w, h int, c gocolor.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestHTTPAnalyzerParsesDetections(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Confirm the request carried a base64 image.
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["image"].(string); !ok {
			t.Error("request missing base64 image")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"detections": []map[string]any{
				{"label": "face", "confidence": 0.9, "x": 0.3, "y": 0.2, "w": 0.4, "h": 0.5,
					"points": [][2]float64{{0.5, 0.4}}},
				{"label": "person", "confidence": 0.8, "x": 0.1, "y": 0.1, "w": 0.8, "h": 0.8},
			},
		})
	}))
	defer srv.Close()

	a := NewHTTPAnalyzer(srv.URL, "")
	if a.Name() != "http-vision" {
		t.Errorf("name = %q", a.Name())
	}
	dets, err := a.Analyze(context.Background(), solid(32, 32, gocolor.RGBA{A: 255}))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(dets) != 2 {
		t.Fatalf("got %d detections, want 2", len(dets))
	}
	if dets[0].Label != "face" || len(dets[0].Points) != 1 {
		t.Errorf("first detection wrong: %+v", dets[0])
	}
}

func TestHTTPAnalyzerHandlesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	a := NewHTTPAnalyzer(srv.URL, "")
	if _, err := a.Analyze(context.Background(), solid(8, 8, gocolor.RGBA{A: 255})); err == nil {
		t.Error("expected error on HTTP 500")
	}
}

func TestNewHTTPAnalyzerFromEnv(t *testing.T) {
	t.Setenv("VISION_API_URL", "")
	if _, ok := NewHTTPAnalyzerFromEnv(); ok {
		t.Error("unset VISION_API_URL should give ok=false")
	}
	t.Setenv("VISION_API_URL", "http://x")
	if _, ok := NewHTTPAnalyzerFromEnv(); !ok {
		t.Error("set VISION_API_URL should give ok=true")
	}
}

func TestDetectionOverlayDrawsBox(t *testing.T) {
	f := terminal.NewFrame(20, 20)
	f.Fill('.', color.White, color.NewRGB(10, 10, 10))
	dets := []Detection{
		{Label: "face", X: 0.25, Y: 0.25, W: 0.5, H: 0.5, Points: [][2]float64{{0.5, 0.5}}},
	}
	DetectionOverlay(f, dets, color.NewRGB(0, 255, 0))

	// Corners of the box (5,5)-(15,15) should be box-drawing chars.
	if f.Blocks[5][5].Glyph != '┌' {
		t.Errorf("top-left corner = %q, want ┌", f.Blocks[5][5].Glyph)
	}
	if f.Blocks[15][15].Glyph != '┘' {
		t.Errorf("bottom-right corner = %q, want ┘", f.Blocks[15][15].Glyph)
	}
	// Label drawn above the box.
	if f.Blocks[4][5].Glyph != 'f' {
		t.Errorf("label start = %q, want 'f'", f.Blocks[4][5].Glyph)
	}
	// Landmark marker at center.
	if f.Blocks[10][10].Glyph != '•' {
		t.Errorf("landmark = %q, want •", f.Blocks[10][10].Glyph)
	}
	// Interior (not on an edge) stays the base '.'.
	if f.Blocks[8][8].Glyph != '.' {
		t.Errorf("box interior was overwritten: %q", f.Blocks[8][8].Glyph)
	}
}
