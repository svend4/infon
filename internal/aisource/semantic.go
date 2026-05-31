// Semantic transport (roadmap B3): "send meaning, not pixels".
//
// Instead of transmitting encoded block frames (hundreds of KB), transmit the
// compact scene description (hundreds of BYTES) and re-render it on the receiver
// at THEIR resolution and quality. This aligns with TVCP's low-bandwidth thesis:
// a scene graph is radically smaller than pixels, and each side renders locally.
//
// The wire format reuses pkg/sketch (a high-level, model-friendly description),
// JSON-encoded. A pixel-frame fallback is used when the receiver has no renderer
// for a frame (handled by the caller); this file provides the encode/decode and
// the bandwidth comparison that justifies the approach.
package aisource

import (
	"encoding/json"

	"github.com/svend4/infon/pkg/scene"
	"github.com/svend4/infon/pkg/sketch"
	"github.com/svend4/infon/pkg/terminal"
)

// SemanticFrame is the transmitted unit: a sketch plus the target canvas the
// sender intended (the receiver may override with its own size).
type SemanticFrame struct {
	Sketch sketch.Sketch `json:"sketch"`
	Cols   int           `json:"cols"`
	Rows   int           `json:"rows"`
}

// EncodeSemantic serializes a semantic frame to JSON bytes for transmission.
func EncodeSemantic(sf SemanticFrame) ([]byte, error) {
	return json.Marshal(sf)
}

// DecodeSemantic parses a semantic frame from bytes.
func DecodeSemantic(data []byte) (SemanticFrame, error) {
	var sf SemanticFrame
	err := json.Unmarshal(data, &sf)
	return sf, err
}

// RenderSemantic re-renders a received semantic frame to a terminal.Frame at the
// receiver's chosen size (cols/rows), falling back to the sender's intent when
// the receiver passes 0. This is the "render at your own resolution" step.
func RenderSemantic(sf SemanticFrame, cols, rows int) *terminal.Frame {
	if cols <= 0 {
		cols = sf.Cols
	}
	if rows <= 0 {
		rows = sf.Rows
	}
	if cols <= 0 {
		cols = 64
	}
	if rows <= 0 {
		rows = 24
	}
	sc := sketch.ToScene(sf.Sketch, cols, rows)
	return scene.RenderScene(&sc)
}

// PixelFrameSize estimates the serialized byte size of transmitting a frame as
// pixels (block cells), for comparison: each cell carries a rune (up to 4 bytes)
// + fg(3) + bg(3) ≈ 10 bytes, matching the repo's ~36-bit-per-block I-frame.
func PixelFrameSize(cols, rows int) int {
	return cols * rows * 10
}
