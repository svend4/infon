// PseudoBackend bridges the open tvcp-ai/1 brain protocol to the
// device.NeuralBackend seam using pkg/pseudo. Where BrainBackend asks for a
// sketch, PseudoBackend asks for a "pseudo-image" (kind:"image") — a compact
// spec that ANY ordinary text model can emit (a coarse color grid for
// pseudo-diffusion, glyph art, sigils, or vector shapes). No diffusion model,
// no image head: the model speaks a few tokens of JSON and we render the pixels.
package aisource

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"net/http"
	"time"

	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/pseudo"
)

// PseudoBackend renders frames from a tvcp-ai/1 brain via pseudo-image specs.
type PseudoBackend struct {
	URL    string
	Format pseudo.Format // optional format hint sent to the model ("" = grid / model's choice)
	HTTP   *http.Client
}

// NewPseudoBackend creates a backend that asks a tvcp-ai/1 endpoint for
// pseudo-images in the given format ("" lets the brain default to grid).
func NewPseudoBackend(url string, format pseudo.Format) *PseudoBackend {
	return &PseudoBackend{URL: url, Format: format, HTTP: &http.Client{Timeout: 120 * time.Second}}
}

// Name implements device.NeuralBackend.
func (b *PseudoBackend) Name() string {
	if b.Format != "" {
		return "tvcp-ai/pseudo:" + string(b.Format)
	}
	return "tvcp-ai/pseudo"
}

// Generate implements device.NeuralBackend: request a pseudo-image spec for the
// prompt and render it to a raster (downsampled later by the BABE renderer).
func (b *PseudoBackend) Generate(ctx context.Context, prompt string, width, height int) (image.Image, error) {
	_ = ctx // the HTTP client carries its own timeout
	cols, rows := width/8, height/8
	if cols < 24 {
		cols = 24
	}
	if rows < 12 {
		rows = 12
	}
	hb := brain.HTTPBrain{URL: b.URL, HTTP: b.HTTP}
	resp, err := hb.Decide(brain.Request{
		Kind:   "image",
		Prompt: prompt,
		Format: string(b.Format),
		Canvas: &brain.Canvas{Width: cols, Height: rows},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("pseudo backend: %s", resp.Error)
	}
	if len(resp.Image) == 0 {
		return nil, fmt.Errorf("pseudo backend: empty image spec")
	}
	var spec pseudo.Spec
	if err := json.Unmarshal(resp.Image, &spec); err != nil {
		return nil, fmt.Errorf("pseudo backend: bad spec: %w", err)
	}
	if spec.Cols == 0 {
		spec.Cols = cols
	}
	if spec.Rows == 0 {
		spec.Rows = rows
	}
	return spec.Image(width, height)
}

var _ device.NeuralBackend = (*PseudoBackend)(nil)
