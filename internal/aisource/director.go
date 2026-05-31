// Director + painter composition (roadmap B5).
//
// Splits generation across two roles behind the same tvcp-ai/1 protocol:
//
//   - a "director" (a small LLM brain) turns a high-level intent into a
//     STRUCTURED scene plan (a tvcp-ai/1 sketch) — LLMs are good at planning a
//     scene but weak at emitting raw pixels, which the sketch abstraction
//     embraces;
//   - a "painter" executes the plan: either the procedural scene renderer
//     (no model) or a raster NeuralBackend handed an enriched prompt.
//
// DirectorPainter is itself a device.NeuralBackend, so it drops into the same
// async pipeline and tier policy as every other backend.
package aisource

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"net/http"
	"strings"
	"time"

	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/scene"
	"github.com/svend4/infon/pkg/sketch"
)

// DirectorPainter orchestrates a director brain and an optional raster painter.
type DirectorPainter struct {
	directorURL string               // tvcp-ai/1 brain that plans a sketch
	painter     device.NeuralBackend // optional raster painter; nil = procedural
	http        *http.Client
}

var _ device.NeuralBackend = (*DirectorPainter)(nil)

// NewDirectorPainter wires a director (brain endpoint) to a painter. If painter
// is nil, the planned sketch is rendered procedurally via the scene renderer.
func NewDirectorPainter(directorURL string, painter device.NeuralBackend) *DirectorPainter {
	return &DirectorPainter{
		directorURL: directorURL,
		painter:     painter,
		http:        &http.Client{Timeout: 60 * time.Second},
	}
}

// Name implements device.NeuralBackend.
func (d *DirectorPainter) Name() string {
	if d.painter != nil {
		return "director+" + d.painter.Name()
	}
	return "director+procedural"
}

// Generate implements device.NeuralBackend: the director plans a sketch from the
// prompt, then the painter executes it. Falls back to a plain procedural render
// of the prompt if the director is unavailable.
func (d *DirectorPainter) Generate(ctx context.Context, prompt string, width, height int) (image.Image, error) {
	sk, err := d.direct(prompt, width, height)
	if err != nil {
		// Director failed: degrade to a local procedural sketch of the prompt so
		// the stream never stops.
		return NewLocalBackend().Generate(ctx, prompt, width, height)
	}

	if d.painter != nil {
		// Hand the painter an enriched prompt derived from the plan (the director
		// chose palette/elements; describe them so the raster model honors them).
		return d.painter.Generate(ctx, enrichPrompt(prompt, sk), width, height)
	}

	// No raster painter: render the planned sketch procedurally.
	cols, rows := width/8, height/8
	if cols < 16 {
		cols = 16
	}
	if rows < 8 {
		rows = 8
	}
	sc := sketch.ToScene(sk, cols, rows)
	return frameToImage(scene.RenderScene(&sc), width, height), nil
}

// direct asks the director brain for a sketch plan.
func (d *DirectorPainter) direct(prompt string, width, height int) (sketch.Sketch, error) {
	cols, rows := width/8, height/8
	if cols < 16 {
		cols = 16
	}
	if rows < 8 {
		rows = 8
	}
	hb := brain.HTTPBrain{URL: d.directorURL, HTTP: d.http}
	resp, err := hb.Decide(brain.Request{
		Kind:   "sketch",
		Prompt: prompt,
		Canvas: &brain.Canvas{Width: cols, Height: rows},
	})
	if err != nil {
		return sketch.Sketch{}, err
	}
	var sk sketch.Sketch
	if resp.Error != "" || json.Unmarshal(resp.Sketch, &sk) != nil || len(sk.Shapes) == 0 {
		return sketch.Sketch{}, fmt.Errorf("director: no usable plan (%s)", resp.Error)
	}
	return sk, nil
}

// enrichPrompt turns a director's structured plan into extra natural-language
// guidance for a raster painter (so the two roles compose instead of ignoring
// each other).
func enrichPrompt(base string, sk sketch.Sketch) string {
	var b strings.Builder
	b.WriteString(base)
	if sk.Sky != "" {
		b.WriteString(", ")
		b.WriteString(sk.Sky)
		b.WriteString(" sky")
	}
	if sk.Ground != "" {
		b.WriteString(", ")
		b.WriteString(sk.Ground)
		b.WriteString(" ground")
	}
	// Summarize the planned elements (deduplicated kinds).
	seen := map[string]bool{}
	var kinds []string
	for _, s := range sk.Shapes {
		if s.Kind != "" && !seen[s.Kind] {
			seen[s.Kind] = true
			kinds = append(kinds, s.Kind)
		}
	}
	if len(kinds) > 0 {
		b.WriteString(", with ")
		b.WriteString(strings.Join(kinds, ", "))
	}
	return b.String()
}
