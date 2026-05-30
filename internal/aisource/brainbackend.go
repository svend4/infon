// BrainBackend bridges the open tvcp-ai/1 brain protocol (pkg/brain) to the
// device.NeuralBackend seam added by the generative pipeline. It lets
// `tvcp synth neural` paint frames with any model (Ollama / OpenAI / Anthropic)
// behind the protocol, using the async generative pipeline so a slow model
// never stalls the terminal.
package aisource

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"net/http"
	"time"

	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/scene"
	"github.com/svend4/infon/pkg/sketch"
	"github.com/svend4/infon/pkg/terminal"
)

// BrainBackend asks a tvcp-ai/1 brain for a sketch and rasterizes it to an
// image, so it can be attached to device.NeuralGenerator as the model backend.
type BrainBackend struct {
	URL  string
	HTTP *http.Client
}

// NewBrainBackend creates a backend that talks to a tvcp-ai/1 endpoint.
func NewBrainBackend(url string) *BrainBackend {
	return &BrainBackend{URL: url, HTTP: &http.Client{Timeout: 120 * time.Second}}
}

// Name implements device.NeuralBackend.
func (b *BrainBackend) Name() string { return "tvcp-ai" }

// Generate implements device.NeuralBackend: it requests a sketch for the prompt
// and renders it to a raster image (downsampled later by the BABE renderer).
func (b *BrainBackend) Generate(ctx context.Context, prompt string, width, height int) (image.Image, error) {
	_ = ctx // the HTTP client carries its own timeout
	cols, rows := width/8, height/8
	if cols < 16 {
		cols = 16
	}
	if rows < 8 {
		rows = 8
	}
	hb := brain.HTTPBrain{URL: b.URL, HTTP: b.HTTP}
	resp, err := hb.Decide(brain.Request{Kind: "sketch", Prompt: prompt, Canvas: &brain.Canvas{Width: cols, Height: rows}})
	if err != nil {
		return nil, err
	}
	var sk sketch.Sketch
	if resp.Error != "" || json.Unmarshal(resp.Sketch, &sk) != nil || len(sk.Shapes) == 0 {
		return nil, fmt.Errorf("brain backend: no usable sketch (%s)", resp.Error)
	}
	sc := sketch.ToScene(sk, cols, rows)
	return frameToImage(scene.RenderScene(&sc), width, height), nil
}

// frameToImage rasterizes a terminal.Frame (a grid of colored cells) into a
// width x height RGBA image: each pixel takes its cell's foreground color for a
// drawn glyph, otherwise the background color.
func frameToImage(f *terminal.Frame, w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if f.Width == 0 || f.Height == 0 {
		return img
	}
	for y := 0; y < h; y++ {
		cy := y * f.Height / h
		for x := 0; x < w; x++ {
			cx := x * f.Width / w
			blk := f.Blocks[cy][cx]
			c := blk.Bg
			if blk.Glyph != ' ' {
				c = blk.Fg
			}
			img.SetRGBA(x, y, color.RGBA{R: c.R, G: c.G, B: c.B, A: 255})
		}
	}
	return img
}

var _ device.NeuralBackend = (*BrainBackend)(nil)
