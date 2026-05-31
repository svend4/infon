package aisource

import (
	"testing"

	"github.com/svend4/infon/pkg/sketch"
)

func sampleSketch() sketch.Sketch {
	return sketch.Sketch{
		Sky:    "orange",
		Ground: "navy",
		Shapes: []sketch.Shape{
			{Kind: "sun", X: 48, Y: 6, W: 3, Color: "gold"},
			{Kind: "mountain", X: 16, H: 8, Color: "darkgreen"},
			{Kind: "text", X: 2, Y: 1, S: "a calm bay", Color: "white"},
		},
	}
}

func TestSemanticRoundTrip(t *testing.T) {
	sf := SemanticFrame{Sketch: sampleSketch(), Cols: 64, Rows: 24}
	data, err := EncodeSemantic(sf)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := DecodeSemantic(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Cols != 64 || got.Rows != 24 || got.Sketch.Sky != "orange" {
		t.Errorf("round trip mismatch: %+v", got)
	}
	if len(got.Sketch.Shapes) != 3 {
		t.Errorf("shapes lost: %d", len(got.Sketch.Shapes))
	}
}

func TestRenderSemanticAtReceiverSize(t *testing.T) {
	sf := SemanticFrame{Sketch: sampleSketch(), Cols: 64, Rows: 24}
	// Receiver renders at its OWN, larger size.
	f := RenderSemantic(sf, 100, 40)
	if f.Width != 100 || f.Height != 40 {
		t.Fatalf("receiver render = %dx%d, want 100x40", f.Width, f.Height)
	}
	// Falling back to sender intent when receiver passes 0.
	f2 := RenderSemantic(sf, 0, 0)
	if f2.Width != 64 || f2.Height != 24 {
		t.Fatalf("fallback render = %dx%d, want 64x24", f2.Width, f2.Height)
	}
}

// TestSemanticIsMuchSmallerThanPixels is the core B3 claim: a scene description
// is dramatically smaller than the equivalent pixel frame.
func TestSemanticIsMuchSmallerThanPixels(t *testing.T) {
	sf := SemanticFrame{Sketch: sampleSketch(), Cols: 80, Rows: 24}
	data, err := EncodeSemantic(sf)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	semanticBytes := len(data)
	pixelBytes := PixelFrameSize(80, 24)

	ratio := float64(pixelBytes) / float64(semanticBytes)
	t.Logf("semantic=%dB pixel=%dB ratio=%.1fx", semanticBytes, pixelBytes, ratio)
	if ratio < 3 {
		t.Errorf("semantic frame should be >=3x smaller than pixels, got %.1fx", ratio)
	}
}

func TestRenderSemanticEmptyDefaults(t *testing.T) {
	// A zero-size sender intent and zero receiver size must still render at the
	// 64x24 default (no panic, no zero-size frame).
	f := RenderSemantic(SemanticFrame{Sketch: sampleSketch()}, 0, 0)
	if f.Width != 64 || f.Height != 24 {
		t.Fatalf("default render = %dx%d, want 64x24", f.Width, f.Height)
	}
}
