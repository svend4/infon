// Streaming backend (roadmap B1, buildable part).
//
// Real streaming diffusion (SD-Turbo / LCM / StreamDiffusion) needs a GPU; what
// we CAN build here, model-agnostically, is the temporal-coherence layer that
// makes a frame stream look live instead of flickering between unrelated
// samples. StreamingBackend wraps any NeuralBackend and cross-fades from the
// previously shown frame toward each new generation in OKLab space — the
// perceptual effect of low-denoising img2img — so the picture *evolves*.
//
// It composes with the async NeuralGenerator: the wrapped backend may be slow,
// while StreamingBackend returns intermediate blended frames cheaply.
package aisource

import (
	"context"
	"image"
	gocolor "image/color"
	"sync"

	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/color"
)

// StreamingBackend adds frame-to-frame coherence around an inner backend.
type StreamingBackend struct {
	inner device.NeuralBackend

	// Coherence is the blend toward the newest generation, 0..1: 1 = show the new
	// frame outright (no smoothing), lower = slower, smoother evolution.
	Coherence float64

	mu   sync.Mutex
	prev *image.RGBA // last emitted frame (for cross-fade)
}

var _ device.NeuralBackend = (*StreamingBackend)(nil)

// NewStreamingBackend wraps inner with temporal coherence. coherence is clamped
// to (0,1]; 0 is treated as a sensible default.
func NewStreamingBackend(inner device.NeuralBackend, coherence float64) *StreamingBackend {
	if coherence <= 0 || coherence > 1 {
		coherence = 0.5
	}
	return &StreamingBackend{inner: inner, Coherence: coherence}
}

// Name implements device.NeuralBackend.
func (s *StreamingBackend) Name() string {
	if s.inner == nil {
		return "streaming"
	}
	return "streaming/" + s.inner.Name()
}

// Generate produces the next frame: it asks the inner backend for a target, then
// returns a frame cross-faded from the previous output toward that target. The
// first frame (no history) is returned as-is.
func (s *StreamingBackend) Generate(ctx context.Context, prompt string, width, height int) (image.Image, error) {
	target, err := s.inner.Generate(ctx, prompt, width, height)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	prev := s.prev
	out := blendImages(prev, target, s.Coherence)
	s.prev = out
	return out, nil
}

// Reset drops the coherence history (next frame is shown outright).
func (s *StreamingBackend) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prev = nil
}

// blendImages returns prev*(1-a) + target*a per pixel in OKLab space. If prev is
// nil or sizes differ, target is returned as a fresh RGBA copy.
func blendImages(prev *image.RGBA, target image.Image, a float64) *image.RGBA {
	tb := target.Bounds()
	w, h := tb.Dx(), tb.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))

	if prev == nil || prev.Bounds().Dx() != w || prev.Bounds().Dy() != h {
		// No usable history: copy target through.
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, _ := target.At(tb.Min.X+x, tb.Min.Y+y).RGBA()
				out.SetRGBA(x, y, gocolor.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255})
			}
		}
		return out
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pr, pg, pb, _ := prev.At(x, y).RGBA()
			tr, tg, tb2, _ := target.At(tb.Min.X+x, tb.Min.Y+y).RGBA()
			p := color.RGB{R: uint8(pr >> 8), G: uint8(pg >> 8), B: uint8(pb >> 8)}
			t := color.RGB{R: uint8(tr >> 8), G: uint8(tg >> 8), B: uint8(tb2 >> 8)}
			blended := p.BlendOKLab(t, a)
			out.SetRGBA(x, y, gocolor.RGBA{R: blended.R, G: blended.G, B: blended.B, A: 255})
		}
	}
	return out
}
