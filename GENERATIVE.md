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

## Built-in backends

Two ready-made `NeuralBackend` implementations ship in
[`internal/aisource`](internal/aisource), both driven by `tvcp synth neural`:

### 1. Raster — a real text-to-image model (`ImageBackend`)

The genuine "describe it, the model paints it" path: POST a prompt to an
image-generation HTTP API and decode the returned **picture** (PNG/JPEG).
Provider-agnostic and stdlib-only; configured via env vars (takes priority when
`IMAGE_API_URL` is set):

```bash
export IMAGE_API_URL=https://your-endpoint/v1/images
export IMAGE_API_KEY=sk-...                 # optional bearer token
export IMAGE_B64_PATH=data.0.b64_json       # optional: dot-path to base64 in a
                                            # JSON response (OpenAI-style). Omit
                                            # if the body is raw image bytes.
tvcp synth neural "a calm harbor at dawn"
```

Responses are handled three ways: an `image/*` Content-Type (raw bytes), a
configured base64 JSON path (incl. `data:` URLs), or a raw-body fallback.
`IMAGE_PROMPT_FIELD` / `IMAGE_WIDTH_FIELD` / `IMAGE_HEIGHT_FIELD` override the
request field names for non-standard APIs.

### 2. Vector — a tvcp-ai/1 brain (`BrainBackend`)

Asks any [`tvcp-ai/1`](ai/BRAIN_PROTOCOL.md) brain for a high-level *sketch* and
rasterizes it locally. Good for small/local LLMs (Ollama) that can emit simple
JSON but not pixels:

```bash
export BRAIN_URL=http://127.0.0.1:8088/v1/decide
tvcp synth neural "a calm harbor at dawn"
```

Both run through the async `NeuralGenerator`, so a slow model never stalls the
terminal — verified: against a 2s-per-reply brain, `tvcp ai -brain` still renders
~15 fps.

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

## Where this can go next

See [`docs/NEURAL_GRAPHICS_ROADMAP.md`](docs/NEURAL_GRAPHICS_ROADMAP.md) for a
detailed design doc on future directions where graphics and neural networks
interact: higher glyph density (half-block / sextant / Braille / Sixel),
streaming diffusion backends, a "visual chat" steering protocol, a learned block
encoder, semantic (send-meaning-not-pixels) transport, audio-reactive synthesis,
and neural avatars — with a recommended impact-vs-effort sequencing.
