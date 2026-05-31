package aisource

import (
	"context"
	"encoding/json"
	"image"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/sketch"
	"github.com/svend4/infon/pkg/terminal"
)

func TestAICameraIsDeviceCamera(t *testing.T) {
	// Compile-time enforced in camera.go; this exercises it at runtime too.
	var cam device.Camera = NewAICamera(64, 48, 15)
	if err := cam.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !cam.IsOpen() {
		t.Fatal("camera should be open")
	}
	img, err := cam.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 64 || b.Dy() != 48 {
		t.Fatalf("frame = %dx%d, want 64x48", b.Dx(), b.Dy())
	}
	// Successive reads advance the animation, so frames should differ.
	img2, _ := cam.Read()
	if sameImage(img, img2) {
		t.Error("consecutive AICamera frames are identical (no animation)")
	}
	if err := cam.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestBrainBackendRendersSketch(t *testing.T) {
	// A stub tvcp-ai/1 server that always returns a valid sketch.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sk := sketch.Sketch{Sky: "navy", Ground: "darkgreen", Shapes: []sketch.Shape{
			{Kind: "sun", X: 10, Y: 4, W: 3, Color: "gold"},
			{Kind: "text", X: 1, Y: 1, S: "hi", Color: "white"},
		}}
		data, _ := json.Marshal(sk)
		_ = json.NewEncoder(w).Encode(brain.Response{Protocol: brain.Protocol, Kind: "sketch", Sketch: data})
	}))
	defer srv.Close()

	be := NewBrainBackend(srv.URL)
	if be.Name() != "tvcp-ai" {
		t.Errorf("Name() = %q, want tvcp-ai", be.Name())
	}
	img, err := be.Generate(context.Background(), "a calm harbor", 128, 96)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 128 || b.Dy() != 96 {
		t.Fatalf("image = %dx%d, want 128x96", b.Dx(), b.Dy())
	}
	// The sketch paints a sky, so not every pixel can be pure black.
	if allBlack(img) {
		t.Error("rendered sketch image is entirely black")
	}
}

func TestBrainBackendErrorsOnEmptySketch(t *testing.T) {
	// Server returns a response with no usable sketch -> backend must error so the
	// async NeuralGenerator keeps showing the previous (or placeholder) frame.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(brain.Response{Protocol: brain.Protocol, Kind: "sketch", Error: "no idea"})
	}))
	defer srv.Close()

	be := NewBrainBackend(srv.URL)
	if _, err := be.Generate(context.Background(), "x", 64, 64); err == nil {
		t.Error("expected an error when the brain returns no sketch")
	}
}

func TestBrainBackendIsNeuralBackend(t *testing.T) {
	var _ device.NeuralBackend = NewBrainBackend("http://example.invalid")
}

func TestBrainFrameFallsBackOnError(t *testing.T) {
	// Pointed at a dead endpoint, BrainFrame must return a non-nil notice frame.
	f := BrainFrame("http://127.0.0.1:1/v1/decide", "anything", 40, 10)
	if f == nil {
		t.Fatal("BrainFrame returned nil")
	}
	if f.Width != 40 || f.Height != 10 {
		t.Errorf("fallback frame = %dx%d, want 40x10", f.Width, f.Height)
	}
}

// TestFrameToImageUsesCellColors checks the rasterizer: a drawn glyph contributes
// its foreground color; a blank cell contributes its background color.
func TestFrameToImageUsesCellColors(t *testing.T) {
	f := terminal.NewFrame(2, 2)
	f.SetBlock(0, 0, '#', color.NewRGB(200, 0, 0), color.NewRGB(0, 0, 0)) // drawn -> fg
	f.SetBlock(1, 1, ' ', color.NewRGB(0, 0, 0), color.NewRGB(0, 0, 200)) // blank -> bg
	img := frameToImage(f, 2, 2)

	if r, _, _, _ := img.At(0, 0).RGBA(); uint8(r>>8) != 200 {
		t.Errorf("drawn-cell pixel R = %d, want 200", uint8(r>>8))
	}
	if _, _, b, _ := img.At(1, 1).RGBA(); uint8(b>>8) != 200 {
		t.Errorf("blank-cell pixel B = %d, want 200 (background)", uint8(b>>8))
	}
}

// --- helpers ---

func sameImage(a, b image.Image) bool {
	ba, bb := a.Bounds(), b.Bounds()
	if ba != bb {
		return false
	}
	for y := ba.Min.Y; y < ba.Max.Y; y++ {
		for x := ba.Min.X; x < ba.Max.X; x++ {
			r1, g1, b1, _ := a.At(x, y).RGBA()
			r2, g2, b2, _ := b.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 {
				return false
			}
		}
	}
	return true
}

func allBlack(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r != 0 || g != 0 || bl != 0 {
				return false
			}
		}
	}
	return true
}
