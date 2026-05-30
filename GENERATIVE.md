# Generative Graphics (Synthesis Pipeline)

TVCP renders any image into the terminal. That image does not have to come from
a camera — it can be **synthesized**. This document describes the generative
pipeline: procedural graphics today, with a clean hook for plugging in a neural
network for "visual communication" (you describe something, the model draws it,
and it streams into the terminal like video).

> TL;DR
> ```bash
> tvcp synth plasma          # Layer 1: procedural, real-time, runs anywhere
> tvcp synth ripple
> tvcp synth neural "a red sunset over mountains"   # Layer 2: neural hook
> ```

## Why this works at all

The renderer is **source-agnostic**. The whole stack downstream of a frame —
[`babe.ImageToFrame`](internal/codec/babe/converter.go) → ANSI rendering
([`pkg/terminal/renderer.go`](pkg/terminal/renderer.go)) → UDP/Yggdrasil
transport — only ever sees a Go `image.Image`. It does not care whether those
pixels came from a webcam, a pattern, a game, or a diffusion model.

The seam is the [`device.Camera`](internal/device/camera.go) interface, whose
only producer method is `Read() (image.Image, error)`. A webcam
(`V4L2Camera`), a test pattern (`TestCamera`) and now a synthesizer
(`GenerativeSource`) all implement it, so a synthesizer is a **drop-in video
source** for preview, `send`, `call`, group calls and recording.

```
FrameGenerator  ──►  GenerativeSource (implements device.Camera)
   │                        │ Read() image.Image
   │                        ▼
   │                 babe.ImageToFrame ──► ANSI renderer ──► terminal / network
   ▼
 plasma / ripple (Layer 1)   |   neural (Layer 2)
```

## Two layers

### Layer 1 — procedural (always real-time)

Cheap, deterministic, GPU-free generators in
[`generators_procedural.go`](internal/device/generators_procedural.go)
(`plasma`, `ripple`). They run at full frame rate on a Raspberry Pi or over SSH
and are the dependable fallback. This is also exactly how a **game renderer**
would integrate: render the game's framebuffer into an `image.Image` and feed it
through the same `FrameGenerator` seam.

### Layer 2 — neural hook (optional, heavier)

[`generators_neural.go`](internal/device/generators_neural.go) defines:

- **`NeuralBackend`** — the one interface you implement to attach a real model.
- **`NeuralGenerator`** — adapts a backend to `FrameGenerator`. With no backend
  it shows an animated placeholder so the pipeline is runnable without a model.

The important design choice: **generation is asynchronous**. The model may be
slow (a remote GPU, an HTTP API), so `NeuralGenerator.Generate` never blocks —
it kicks generation in the background and returns the most recent finished
frame. That is what makes a slow model usable for a live, interactive session
instead of freezing the terminal.

## Plugging in a real model

Implement `device.NeuralBackend` and attach it:

```go
type MyBackend struct{ /* http client, model handle, ... */ }

func (MyBackend) Name() string { return "my-model" }

func (b MyBackend) Generate(ctx context.Context, prompt string, w, h int) (image.Image, error) {
    // Call a local fast model (SD-Turbo / LCM / StreamDiffusion) or a remote
    // GPU/API. Decode the result (PNG/JPEG) into an image.Image and return it.
    // Honor ctx for timeout/cancel.
    return img, nil
}

gen := device.NewNeuralGenerator(MyBackend{}, "a calm ocean at dawn")
src := device.NewGenerativeSource(256, 256, 15, gen)
// ... drive src.Read() through babe.ImageToFrame like any camera.

// An agent can change what is drawn at any time:
gen.SetPrompt("now a storm rolls in")
```

## Practical notes (honest constraints)

- **Low terminal resolution is an advantage here.** The terminal shows ~160×48
  subpixels, so the model only needs to output a small image (128×128 / 256×256),
  which downsamples cheaply. The cost is *generation*, not rendering.
- **Where the model runs decides "real-time."** A local fast model on a GPU can
  approach real-time; a cloud API per frame is seconds of latency. The async
  design degrades gracefully: the last frame keeps showing until the next lands.
  Use `NeuralGenerator.Interval` to rate-limit calls (and cost).
- **Targets like Raspberry Pi / 256 MB RAM cannot run a model locally** — point
  the backend at a remote GPU/API instead.

## Files

| File | Role |
|------|------|
| [`internal/device/generative.go`](internal/device/generative.go) | `FrameGenerator` interface + `GenerativeSource` (the `device.Camera` adapter) |
| [`internal/device/generators_procedural.go`](internal/device/generators_procedural.go) | Layer 1: `plasma`, `ripple` |
| [`internal/device/generators_neural.go`](internal/device/generators_neural.go) | Layer 2: `NeuralBackend` + `NeuralGenerator` (async) |
| [`cmd/tvcp/synth.go`](cmd/tvcp/synth.go) | `tvcp synth` live demo command |
