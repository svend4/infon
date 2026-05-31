# Design: Graphics × Neural Networks in the Terminal — Future Directions

> A technical & design document exploring how far the TVCP graphics pipeline and
> the `tvcp-ai/1` neural layer can be pushed when they interact. It catalogs
> concrete, buildable directions, the methodologies behind them, and the
> trade-offs — so we can pick deliberately rather than ad hoc.
>
> Status legend: ✅ done · 🟡 partial / scaffolded · ⬜ proposed (not started)

## Implementation status (this work)

Most of Phases 1–3 are now built, tested, and on the branch:

| Item | What shipped | Status |
|---|---|---|
| A1 half-block | `babe.ImageToFrameHalfBlock` + `RenderMode` | ✅ |
| A2 sextant | verified U+1FB00 table + `ImageToFrameSextant` | ✅ |
| A3 Braille | `babe.ImageToFrameBraille` | ✅ |
| A4 perceptual color + dithering | `pkg/color` OKLab + Floyd–Steinberg/Bayer; `EncodeBlockPerceptual` | ✅ |
| A5 Sixel | `terminal.EncodeSixel` + `TVCP_SIXEL` | ✅ |
| B1 streaming coherence | `aisource.StreamingBackend` (OKLab cross-fade) | ✅ (GPU model still external) |
| B2 visual-chat steering | `pkg/visualchat` (Steer + Controller) | ✅ |
| B3 semantic codec | `aisource` SemanticFrame (≈68× smaller) | ✅ |
| B4 local/offline brain | `aisource.LocalBackend` + 4-tier policy | ✅ |
| C2 learned/optimal encoder | `babe.EncodeBlockOptimal` (exhaustive perceptual) | ✅ |
| C4 audio-reactive | `internal/audio/reactive` + `AudioReactiveGenerator` | ✅ |
| D1 P-frame metering | `video.StreamStats` (≈28× on static scenes) | ✅ |
| D2 capability probe | `terminal.DetectCapability` + auto mode | ✅ |
| D5 golden render tests | `pkg/terminal/golden_test.go` | ✅ |
| C1 super-resolution | `aisource.RestoreBackend` + `restore_sidecar.py` | ✅ (model external) |
| C3 vision overlays | `internal/vision` (Sobel/edges + `FrameAnalyzer`) + `vision_sidecar.py` | ✅ (model external) |
| C5 neural avatars | `internal/avatar` (keypoint codec ≈65× smaller) + sender/receiver sidecars | ✅ (model external) |

The research-tier items (C1/C3/C5) ship as **complete Go pipelines with seams,
codecs, overlays, tests, and reference sidecars** — the only external piece is
the trained model itself, which plugs in via HTTP (local sidecar or cloud). See
[`EXTERNAL_MODELS.md`](EXTERNAL_MODELS.md) for wiring real models in.

Render modes are selectable via `TVCP_RENDER_MODE`
(`quadrant|perceptual|optimal|halfblock|sextant|braille`) and auto-detected from
the terminal otherwise. Neural backends are chosen by env
(`IMAGE_API_URL` > `BRAIN_URL` > `TVCP_LOCAL_BRAIN` > placeholder), optionally
wrapped for streaming (`TVCP_STREAM_COHERENCE`).

---

## 0. Where we are today (the substrate)

The pieces these proposals build on already exist in the repo:

| Capability | Where | Status |
|---|---|---|
| Image → terminal blocks (2×2 quadrants, 24-bit) | `internal/codec/babe`, `pkg/terminal` | ✅ |
| Source-agnostic frame seam (`device.Camera`) | `internal/device/camera.go` | ✅ |
| Procedural generators (plasma, ripple) | `internal/device/generators_procedural.go` | ✅ |
| Async neural generator + backend seam | `internal/device/generators_neural.go` | ✅ |
| Neural backends: sketch (`tvcp-ai/1`) + raster (image API) | `internal/aisource/*backend.go` | ✅ |
| Flicker-free diff renderer | `pkg/terminal/diff.go` | ✅ |
| Synthesized/AI video over P2P (send) | `cmd/tvcp/send.go`, `internal/aisource/source.go` | ✅ |
| Real-time interactive game (Snake) | `experimental/games/snake` | ✅ |
| P-frames (inter-frame delta) | `internal/video/pframe.go` | 🟡 exists, not in synth path |
| Half-block / Braille glyph helpers | `internal/codec/glyphs/glyphs.go` | 🟡 defined, **not** used by encoder |
| Audio: Opus, PCM, **FFT**, VAD | `internal/audio`, `internal/codec` | ✅ (unused by graphics) |

**Key insight that unlocks most of this document:** the encoder currently maps
each character cell to one of **16 quadrant glyphs** (a 2×2 grid). The terminal
can represent far more detail per cell. Raising the *glyph density* is the single
highest-leverage graphics improvement, and it composes with everything else.

---

## Part A — Graphics fidelity (make the picture better)

### A1. Half-block mode — 2× vertical resolution, full color ⬜
**Methodology.** Render each cell as the upper-half-block `▀` with the foreground
color = top pixel and background color = bottom pixel. One cell then encodes **2
vertically-stacked true-color pixels** instead of a 2-color 2×2 approximation.
This is the de-facto standard in `chafa`, `viu`, `timg`, and most "images in the
terminal" tools because it maximizes color fidelity.

**Why it matters.** The current 2×2/2-color cell must *quantize* 4 pixels into 2
colors (k-means by luminance). Half-block keeps both colors exact. Effective
resolution becomes `cols × (rows·2)` with zero color loss.

**Cost / risk.** Low. The `▀` glyph and `HalfBlockGlyphs` already exist; it is a
new `EncodeBlock` mode. Trade-off: loses horizontal sub-cell detail (1 wide × 2
tall instead of 2×2), so it should be a *selectable* mode, not a replacement.

### A2. Sextant (2×3) and octant (2×4) modes — up to 4× density ⬜
**Methodology.** Unicode 13+ "Symbols for Legacy Computing" (U+1FB00…) provides
**sextants** (2×3 = 64 combinations) and Unicode 16 adds **octants** (2×4 = 256
combinations). Map each cell's sub-grid to the nearest legacy-computing glyph,
exactly as we do for quadrants today.

**Why it matters.** Sextants give 6 sub-pixels/cell (vs 4), octants give 8 — a
1.5–2× density jump on top of half-block's color fidelity. `chafa` uses these as
its highest-quality mode on capable terminals.

**Cost / risk.** Medium. Needs a glyph table + nearest-pattern matcher and a
**capability probe** (older terminals/fonts lack these glyphs → must fall back).

### A3. Braille mode — 2×4 monochrome-ish, ubiquitous ⬜
**Methodology.** Braille patterns (U+2800…) pack a 2×4 dot matrix into one cell
(8 dots = 256 patterns). `BraillePattern()` already exists. Color is per-cell
(one fg), so it suits line art, plots, edges, and "ASCII-art" aesthetics.

**Why it matters.** Highest *spatial* resolution (8 sub-cells) with the widest
font support of all these schemes. Great for the **edge/feature overlays** in
Part C and for data-viz.

**Cost / risk.** Low–medium. Helper exists; needs an encoder mode + dithering
(below) to look good.

### A4. Perceptual color + dithering ⬜
**Methodology.** Two upgrades to color selection:
- Quantize in a **perceptual space** (OKLab/CIELAB) instead of raw RGB, so the
  k-means split matches human vision (current `EncodeBlock` splits by luminance).
- Add **ordered (Bayes) or Floyd–Steinberg dithering** when reducing to the
  cell's available colors, to suppress banding in gradients.

**Why it matters.** At terminal resolution, *perceived* quality is dominated by
color choice, not pixel count. This is cheap and dramatically improves gradients
and skin tones.

**Cost / risk.** Low. Pure functions over the existing pixel block; unit-testable.

### A5. Pixel-protocol fast path: Sixel / Kitty / iTerm2 ⬜
**Methodology.** Detect terminals that support **true bitmap protocols** (Sixel,
Kitty graphics, iTerm2 inline images) and, when present, send the actual image
instead of glyph approximation — while keeping the Unicode-block path as the
universal fallback.

**Why it matters.** This is the ceiling: real photographic/diffusion output at
true pixel resolution. The neural raster backend (`ImageBackend`) already
produces a real `image.Image` — on a Kitty/Sixel terminal we could show it
*pixel-perfect*.

**Cost / risk.** Medium–high (protocol encoders + capability negotiation). High
payoff specifically for the neural-image use case.

> **Recommended graphics order:** A4 (dithering/OKLab, cheap, universal) →
> A1 (half-block) → A2 (sextant/octant w/ probe) → A3 (Braille) → A5 (Sixel/Kitty).

---

## Part B — The neural layer (smarter models, better interaction)

### B1. Streaming / latent-consistency backend ⬜
**Methodology.** Add a `NeuralBackend` for **real-time** image models —
SD-Turbo, SDXL-Turbo, LCM, or **StreamDiffusion** — that keep a warm pipeline and
emit a frame every 30–150 ms. Use **img2img with low denoising** seeded by the
previous frame for temporal coherence (so the picture *evolves* instead of
flickering between unrelated samples).

**Why it matters.** This turns "type a prompt, wait" into genuinely *live*
visual generation — the real answer to "visual conversation with a model." The
async `NeuralGenerator` is already built for it; only the backend is missing.

**Cost / risk.** Backend code is small; the model needs a GPU. Degrades
gracefully (async keeps last frame).

### B2. Prompt-control channel & "visual chat" protocol ⬜
**Methodology.** Extend `tvcp-ai/1` with a bidirectional control message so a
peer (or an agent) can `SetPrompt`, adjust style, or send a *reference image*
mid-stream. Define `kind:"steer"` carrying `{prompt, strength, seed, ref}`.

**Why it matters.** Makes the model an *interactive medium*: two people on a P2P
call could co-paint a scene by talking to the same model. This is the concrete
"общение картинами" (communicating in pictures) idea.

**Cost / risk.** Low protocol work; reuses the existing HTTP/JSON contract and
`SetPrompt`. Needs a small UI for live prompt entry.

### B3. Semantic frame codec ("send meaning, not pixels") ⬜
**Methodology.** Instead of transmitting encoded blocks over the network,
transmit the **sketch/scene description** (or a tiny latent) and re-render it on
the receiver with *their* model/quality. The sketch is ~hundreds of bytes vs
~hundreds of KB for frames.

**Why it matters.** Aligns perfectly with TVCP's "9× less bandwidth" thesis: a
scene graph is radically smaller than pixels, and each side renders at its own
resolution. This is a research-flavored but very on-brand direction.

**Cost / risk.** Medium. `pkg/scene`/`pkg/sketch` already are the wire format;
needs negotiation + a fallback to pixel frames when the model is absent.

### B4. Local, offline brains (no cloud, no GPU) ⬜
**Methodology.** First-class adapters for **Ollama** (already a Python adapter)
and **llama.cpp/whisper.cpp**-class local models, plus a pure-Go procedural
"neural-ish" fallback. Document a tiered policy: local-fast → local-LLM-sketch →
cloud-raster → procedural.

**Why it matters.** TVCP's identity is headless, offline, Raspberry-Pi-friendly,
P2P. A fully **local** visual-AI path preserves that (no accounts, no servers).

**Cost / risk.** Low (adapters + docs). Mostly integration & testing.

### B5. Multi-agent / "director + painter" composition ⬜
**Methodology.** Split roles across models: a small **LLM "director"** turns a
high-level intent into a structured scene plan (`tvcp-ai/1` sketch), and a
**"painter"** (raster model or procedural renderer) executes it. Agents can be
different backends behind the same protocol.

**Why it matters.** Plays to the protocol's strength (swap brains by URL) and
gives reliable structure (LLMs are good at *planning* a scene, weak at emitting
raw pixels — which the `sketch` abstraction already acknowledges).

**Cost / risk.** Low–medium; orchestration logic only.

---

## Part C — Where graphics and neural networks *interact* (the core ask)

### C1. Neural super-resolution / restoration for tiny frames ⬜
**Methodology.** Run a fast model **specifically tuned for the 160×48-ish
domain**: upscale/denoise/sharpen the small frame *before* glyph encoding, or
learn a glyph-aware encoder that picks colors to look best after block
quantization.

**Why it matters.** The terminal's tiny canvas is a *fixed* target — a model can
be trained once for it. This is a novel niche: "neural codec for terminal video."

**Cost / risk.** High (training/data) but unique and publishable.

### C2. Learned block encoder (replace k-means) ⬜
**Methodology.** Replace the hand-written `EncodeBlock` (luminance k-means) with
a tiny learned function: given a 2×N pixel block, predict the glyph + 2 colors
that minimize *perceptual* reconstruction error. Could be a lookup table
distilled from an offline optimizer — **no GPU at runtime.**

**Why it matters.** Directly improves every frame from every source (camera,
game, model) with zero runtime cost once distilled. Very high leverage.

**Cost / risk.** Medium. Offline optimization + a baked table; runtime stays pure
Go. Strong fit for the existing `glyphs`/`babe` code.

### C3. Edge/feature overlays from a vision model ⬜
**Methodology.** Run a light vision model (edge detection, segmentation, face
landmarks) on the source frame and **composite** its output as a Braille/line
overlay (Part A3) — e.g. outline the speaker, label objects, draw motion
vectors.

**Why it matters.** Mixes *analysis* AI with *generative* graphics; useful for
monitoring/IoT camera use cases already in the README.

**Cost / risk.** Medium; needs a vision backend + compositor.

### C4. Audio-reactive & cross-modal synthesis ⬜
**Methodology.** The repo already has **FFT** (`internal/audio/fft_filter.go`)
and VAD. Feed spectral features into the procedural generators (plasma that
pulses to voice) and/or into the model prompt ("calm" vs "energetic" from audio
energy). On a call, the *other* person's voice could drive *your* avatar's
visuals.

**Why it matters.** Reuses existing audio DSP, creates a striking demo, and ties
the audio and video subsystems together for the first time.

**Cost / risk.** Low–medium; FFT exists, generators exist — needs a feature bus.

### C5. Neural avatars for ultra-low-bandwidth calls ⬜
**Methodology.** Send only **facial keypoints / expression coefficients** over
the wire; reconstruct the face with a generative model on the receiver
(first-order-motion / talking-head methodology). Falls back to block video when
no model is present.

**Why it matters.** The logical extreme of B3 for *calls*: kilobits/sec for a
talking face. Extremely on-brand for the satellite/tactical use cases.

**Cost / risk.** High (model + landmark tracking), highest "wow", clearly
research-tier.

---

## Part D — Performance, quality, and engineering enablers

| Item | Methodology | Why | Cost |
|---|---|---|---|
| **D1. P-frames in the synth/AI path** ⬜ | Reuse `internal/video/pframe.go` so generated/AI video sends only changed blocks | Big bandwidth win for slowly-changing AI scenes | Low (code exists) |
| **D2. Terminal capability probe** ⬜ | Query truecolor/Sixel/Kitty/Unicode-13 support at startup; pick the best A1–A5 mode | Prerequisite for sextant/Sixel without breaking old terminals | Medium |
| **D3. Frame pacing / adaptive FPS** ⬜ | Drop/skip frames under load; decouple model FPS from render FPS | Smoothness when the model or link is slow | Low |
| **D4. GPU/remote backend pooling** ⬜ | Connection pool + request coalescing for the raster backend | Cost/latency control for cloud models | Medium |
| **D5. Golden-image render tests** ⬜ | Snapshot the ANSI output of known frames; diff in CI | Prevents silent rendering regressions as modes grow | Low |
| **D6. Benchmarks** ⬜ | `testing.B` for encode + diff render | Quantify A-series gains; guard perf | Low |

---

## Part E — Recommended sequencing (impact ÷ effort)

**Phase 1 — cheap, universal quality wins (no new deps, no GPU):**
1. A4 dithering + OKLab quantization
2. A1 half-block mode (selectable)
3. D1 P-frames in the synth path · D5 golden-image tests

**Phase 2 — density & interaction:**
4. A2 sextant/octant + D2 capability probe
5. B2 prompt-control "visual chat" + B4 local brains
6. C4 audio-reactive generators (reuses existing FFT)

**Phase 3 — the differentiators (research-tier):**
7. A5 Sixel/Kitty fast path (pairs with the raster backend)
8. C2 learned block encoder · C1 terminal-domain super-resolution
9. B1 streaming diffusion · B3 semantic codec · C5 neural avatars

---

## Part F — Methodology notes & prior art

- **Block/Unicode rendering:** `chafa` (symbol selection, sextants), `viu`/`timg`
  (half-block), `notcurses` (capability detection, blitters), `libsixel`.
- **Real-time diffusion:** Latent Consistency Models (LCM), SD/SDXL-Turbo,
  StreamDiffusion (pipelined img2img for video-rate generation).
- **Low-bitrate neural video:** first-order motion model, face-vid2vid,
  keypoint-based talking heads (the basis for C5).
- **Perceptual color:** OKLab (Björn Ottosson), CIEDE2000; dithering: Floyd–
  Steinberg, ordered/Bayer.
- **On-brand constraint:** every path must keep a **non-AI fallback** so TVCP
  stays headless, offline, and Raspberry-Pi-viable — AI is an *enhancement*,
  never a hard dependency.

---

## Appendix — Architectural invariants to preserve

1. **The `device.Camera` seam stays the universal source contract** — every new
   source (model, game, vision pipeline) implements it, so the network/record/
   export paths never change.
2. **`NeuralBackend` stays the model seam** — new models are new backends, async
   by default; the render loop never blocks.
3. **Graphics modes are negotiated, never assumed** — always degrade to the
   16-glyph truecolor (or half-block) path on limited terminals.
4. **Protocol over engine lock-in** — `tvcp-ai/1` (swap brains by URL) remains
   the integration boundary; no model SDK leaks into the core.
