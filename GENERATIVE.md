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
> tvcp synth audioreactive   # plasma that pulses to live audio
> tvcp synth neural "a red sunset over mountains"   # Layer 2: neural hook
>
> TVCP_RENDER_MODE=octant tvcp synth ripple   # densest 2×4 glyphs
> TVCP_KITTY=1 tvcp synth neural "a calm bay" # true bitmap on capable terminals
> TVCP_LOCAL_BRAIN=1 tvcp synth neural "..."  # fully offline, no GPU/network
> tvcp avatar send <host:port>                # neural-avatar P2P (~35 kbps)
> ```
>
> Status: this pipeline is implemented and merged — see the status table in
> [`docs/NEURAL_GRAPHICS_ROADMAP.md`](docs/NEURAL_GRAPHICS_ROADMAP.md).

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
(`plasma`, `ripple`) and
[`generators_reactive.go`](internal/device/generators_reactive.go)
(`audioreactive` — pulses to live audio via the spectral features in
[`internal/audio/reactive`](internal/audio/reactive)). They run at full frame
rate on a Raspberry Pi or over SSH and are the dependable fallback. This is also
exactly how a **game renderer** integrates (see `tvcp game`): render the game's
framebuffer into an `image.Image` and feed it through the same `FrameGenerator`
seam.

## Render modes & pixel protocols

A frame can be turned into terminal cells several ways, selected with
`TVCP_RENDER_MODE` (or auto-detected from the terminal via
[`terminal.DetectCapability`](pkg/terminal/capability.go)):

| Mode | Sub-pixels/cell | Notes |
|---|---|---|
| `quadrant` | 2×2, 2 colors | original; luminance clustering |
| `perceptual` | 2×2, 2 colors | OKLab clustering (better color) |
| `optimal` | 2×2, 2 colors | searches all 16 partitions for min perceptual error |
| `halfblock` | 1×2, 2 exact colors | `▀`; 2× vertical, no clustering |
| `sextant` | 2×3 | Unicode 13 (U+1FB00) |
| `octant` | 2×4 | Unicode 16 (U+1CD00) — densest glyph mode |
| `braille` | 2×4 dots | per-cell color; line art / edges |

For true bitmaps on capable terminals, `TVCP_SIXEL=1` (Sixel) or `TVCP_KITTY=1`
(Kitty graphics) emit the actual image pixel-perfect instead of glyph
approximation. Perceptual color uses OKLab; dithering (Floyd–Steinberg / Bayer)
lives in [`pkg/color/dither.go`](pkg/color/dither.go). The
[`DiffRenderer`](pkg/terminal/diff.go) repaints only changed cells, and
[`device.Pacer`](internal/device/pacer.go) adapts FPS under load.

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

### 3. Offline procedural (`LocalBackend`)

`TVCP_LOCAL_BRAIN=1` — turns a prompt into a landscape sketch deterministically
(keywords + a stable hash), with **no model, GPU, or network**. Keeps "visual
chat" working on a Raspberry Pi or air-gapped host.

### Decorators (composed automatically by the env tier policy)

- `RestoreBackend` (`RESTORE_API_URL`) — post-processes each frame through a
  super-resolution/restoration model (Real-ESRGAN class).
- `StreamingBackend` (`TVCP_STREAM_COHERENCE`, 0..1) — OKLab cross-fade between
  frames so a model stream *evolves* instead of flickering.
- `DirectorPainter` (`DIRECTOR_URL`) — an LLM "director" plans a sketch, the
  chosen backend (or the procedural renderer) "paints" it.

`internal/aisource.NeuralBackendFromEnv` composes these, best-available first:

```
IMAGE_API_URL ─┐
BRAIN_URL ─────┼─► base ─► [DIRECTOR_URL] ─► [RESTORE_API_URL] ─► [TVCP_STREAM_COHERENCE]
TVCP_LOCAL_BRAIN┘  (or placeholder)
```

All run through the async `NeuralGenerator`, so a slow model never stalls the
terminal — verified: against a 2s-per-reply brain, `tvcp ai -brain` still renders
~15 fps. Full wiring (HTTP sidecars + cloud providers, with reference Python
adapters) is in [`docs/EXTERNAL_MODELS.md`](docs/EXTERNAL_MODELS.md).

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

## Roadmap & status

[`docs/NEURAL_GRAPHICS_ROADMAP.md`](docs/NEURAL_GRAPHICS_ROADMAP.md) is the design
doc and the authoritative **implementation-status table**. Nearly all of it is
now implemented and merged: higher glyph density (half-block / sextant / octant /
Braille / Sixel / Kitty), the "visual chat" steering protocol
([`pkg/visualchat`](pkg/visualchat)), a learned/optimal block encoder, the
semantic (send-meaning-not-pixels) codec, audio-reactive synthesis, vision
overlays ([`internal/vision`](internal/vision)), and neural avatars
([`internal/avatar`](internal/avatar), `tvcp avatar`). The only external piece
for the research-tier items (super-resolution, vision, avatars) is the trained
model, wired in over HTTP per [`docs/EXTERNAL_MODELS.md`](docs/EXTERNAL_MODELS.md).
