package device

import (
	"context"
	"image"
	"image/color"
	"math"
	"sync"
	"time"
)

// This file implements Layer 2 of the synthesis architecture: the neural hook.
//
// The expensive part of "talk to a model and see pictures" is *generation*, not
// rendering — the terminal only needs a tiny image (e.g. 256x256) which is cheap
// to downsample. So the design treats a model as a pluggable backend that may be
// slow (a remote GPU or an HTTP API) and decouples it from the render loop with
// asynchronous generation: frames are produced in the background and the latest
// available frame is shown immediately, so the terminal never stalls waiting for
// the model.

// NeuralBackend is the seam where a real image-generation model is plugged in.
//
// Implement this against whatever you have:
//   - a local fast model (e.g. SD-Turbo / LCM / StreamDiffusion) for near
//     real-time on a GPU, or
//   - a remote GPU / HTTP API when running on a Raspberry Pi or over SSH.
//
// The returned image may be any size; it is downsampled to the terminal grid by
// the existing BABE renderer.
type NeuralBackend interface {
	// Name identifies the backend (e.g. "sd-turbo", "replicate").
	Name() string

	// Generate produces an image for prompt at the requested size. It should
	// honor ctx cancellation/timeout.
	Generate(ctx context.Context, prompt string, width, height int) (image.Image, error)
}

// NeuralGenerator adapts a NeuralBackend to the FrameGenerator interface.
//
// If no backend is configured it renders an animated placeholder so the pipeline
// is runnable and demoable without a model. With a backend, generation runs in a
// background goroutine (rate-limited by Interval) and Generate returns the most
// recent frame without blocking — this is what makes a slow model usable for a
// live, interactive ("visual chat") session.
type NeuralGenerator struct {
	mu         sync.Mutex
	backend    NeuralBackend
	prompt     string
	latest     image.Image
	lastErr    error
	generating bool
	lastGen    time.Time

	// Interval is the minimum time between background generations. Zero means
	// "regenerate as soon as the previous generation finished".
	Interval time.Duration
	// Timeout bounds a single backend call.
	Timeout time.Duration
}

// NewNeuralGenerator creates a neural-backed generator. backend may be nil, in
// which case an animated placeholder is shown until one is attached.
func NewNeuralGenerator(backend NeuralBackend, prompt string) *NeuralGenerator {
	return &NeuralGenerator{
		backend:  backend,
		prompt:   prompt,
		Interval: 1 * time.Second,
		Timeout:  30 * time.Second,
	}
}

// Name implements FrameGenerator.
func (n *NeuralGenerator) Name() string { return "neural" }

// SetBackend attaches (or replaces) the model backend at runtime.
func (n *NeuralGenerator) SetBackend(b NeuralBackend) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.backend = b
}

// SetPrompt updates the prompt; the next background generation uses it. This is
// the "you say something, the picture changes" control surface for an agent.
func (n *NeuralGenerator) SetPrompt(prompt string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.prompt = prompt
}

// LastError returns the most recent backend error, if any.
func (n *NeuralGenerator) LastError() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.lastErr
}

// HasBackend reports whether a model backend is attached.
func (n *NeuralGenerator) HasBackend() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.backend != nil
}

// Generate implements FrameGenerator. It never blocks on the backend.
func (n *NeuralGenerator) Generate(width, height, frameNum int, elapsed time.Duration) (image.Image, error) {
	n.mu.Lock()

	if n.backend == nil {
		n.mu.Unlock()
		return drawPlaceholder(width, height, elapsed, false), nil
	}

	latest := n.latest
	needKick := !n.generating && (n.lastGen.IsZero() || time.Since(n.lastGen) >= n.Interval)
	if needKick {
		n.generating = true
		backend := n.backend
		prompt := n.prompt
		timeout := n.Timeout
		n.mu.Unlock()
		go n.runGeneration(backend, prompt, width, height, timeout)
	} else {
		n.mu.Unlock()
	}

	if latest != nil {
		return latest, nil
	}
	// No frame yet: show a "warming up" placeholder while the first frame renders.
	return drawPlaceholder(width, height, elapsed, true), nil
}

func (n *NeuralGenerator) runGeneration(b NeuralBackend, prompt string, width, height int, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	img, err := b.Generate(ctx, prompt, width, height)

	n.mu.Lock()
	defer n.mu.Unlock()
	n.generating = false
	n.lastGen = time.Now()
	if err != nil {
		n.lastErr = err
		return
	}
	if img != nil {
		n.latest = img
		n.lastErr = nil
	}
}

// drawPlaceholder renders a distinctive animated standby pattern shown when no
// model is attached (warming==false) or before the first frame arrives
// (warming==true). It uses no font so it pulls in no extra dependencies; the
// CLI prints the human-readable status text alongside it.
func drawPlaceholder(width, height int, elapsed time.Duration, warming bool) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	t := elapsed.Seconds()

	// Base tint: cool blue for "no backend", warm amber while "warming up".
	var baseR, baseG, baseB float64
	if warming {
		baseR, baseG, baseB = 60, 45, 10
	} else {
		baseR, baseG, baseB = 10, 20, 45
	}

	// Moving diagonal scan stripes signal "no live signal".
	stripe := t * 40.0
	cx, cy := float64(width)/2.0, float64(height)/2.0
	pulse := 0.5 + 0.5*math.Sin(t*3.0)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			s := math.Sin((float64(x+y)+stripe)*0.3)*0.5 + 0.5

			// Pulsing center marker.
			d := math.Hypot(float64(x)-cx, float64(y)-cy)
			marker := 0.0
			if d < float64(height)/6.0 {
				marker = (1.0 - d/(float64(height)/6.0)) * pulse * 120.0
			}

			r := clampByte(baseR + s*40.0 + marker)
			g := clampByte(baseG + s*40.0 + marker*0.8)
			b := clampByte(baseB + s*60.0 + marker)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
