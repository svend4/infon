# Connecting External Models (sidecar & cloud)

This guide shows how to plug real neural networks into TVCP for the research-tier
features (C1 restoration, C3 vision overlays, C5 neural avatars), using the two
integration methods that already work in this repo:

1. **HTTP sidecar** ⭐ — a local process (usually Python) holds the model and
   listens on `localhost:PORT`; Go sends it POST requests. This is the proven
   path: Claude Haiku was driven this way via `ai/adapters/anthropic_brain.py`.
2. **Cloud API** — Go sends requests to a hosted endpoint (Replicate, fal.ai,
   Together, OpenAI images, an Anthropic-compatible gateway, …).

Both use the **same Go seams**, so switching is just a URL/env change. Every
feature keeps a **non-AI fallback**, so TVCP stays headless/offline-viable.

## The seams (where models attach)

| Seam | Method | Used by |
|------|--------|---------|
| `device.NeuralBackend` | `Generate(prompt,w,h) → image` | raster generation, restoration, streaming |
| `vision.FrameAnalyzer` | `Analyze(img) → []Detection` | object/face overlays (C3) |
| `avatar.Reconstructor` | `Reconstruct(keypoints) → image` | neural avatars (C5, receiver) |

## Quick start — sidecars (zero heavy deps to try)

Every sidecar runs with a model-free fallback, so you can exercise the whole
pipeline before installing anything heavy.

```bash
# C1 — image restoration / super-resolution
python ai/adapters/restore_sidecar.py 8094
RESTORE_API_URL=http://127.0.0.1:8094/  TVCP_LOCAL_BRAIN=1 \
  ./bin/tvcp synth neural "a calm bay at sunset"

# C3 — vision overlays (object/face boxes)
python ai/adapters/vision_sidecar.py 8096
# set VISION_API_URL=http://127.0.0.1:8096/ in the consumer (overlay path)

# C5 — neural avatars (sender + receiver)
python ai/adapters/avatar_landmark_sidecar.py 8097   # extracts keypoints
python ai/adapters/avatar_generate_sidecar.py 8098   # reconstructs the face
```

Installing the real models upgrades each sidecar automatically:
`pip install realesrgan` (C1), `pip install ultralytics` or `mediapipe` (C3),
`pip install mediapipe` + a talking-head model (C5).

## Cloud API instead of a sidecar

The same env vars accept a hosted URL. Point them at a provider and add the key:

```bash
# raster generation (B-tier) — e.g. a Replicate/fal proxy that returns base64
IMAGE_API_URL=https://your-proxy/v1/images
IMAGE_API_KEY=sk-...
IMAGE_B64_PATH=data.0.b64_json      # where the base64 image sits in the JSON

# restoration via a cloud SR endpoint
RESTORE_API_URL=https://your-proxy/restore
RESTORE_API_KEY=sk-...
```

Because cloud calls cost latency, wrap them with temporal coherence so the
terminal stays live between responses:

```bash
TVCP_STREAM_COHERENCE=0.4   # cross-fade frames (roadmap B1)
```

## The Haiku pattern (reference)

`ai/adapters/anthropic_brain.py` is the template: read the key from the
environment (never hard-code it), translate the tvcp-ai/1 request into the
provider's API, validate/repair the reply, and fall back safely. The new
sidecars (`restore_`, `vision_`, `avatar_*`) follow the same shape.

## Tier policy (what runs when)

`internal/aisource.NeuralBackendFromEnv` composes the layers, best-available
first, each optional:

```
IMAGE_API_URL ─┐
BRAIN_URL ─────┼─► base backend ─► [RESTORE_API_URL] ─► [TVCP_STREAM_COHERENCE]
TVCP_LOCAL_BRAIN┘   (or placeholder)     restoration         streaming coherence
```

So a fully local, offline run is `TVCP_LOCAL_BRAIN=1`; a cloud, high-fidelity,
smooth run is `IMAGE_API_URL=… RESTORE_API_URL=… TVCP_STREAM_COHERENCE=0.4`.

## Honest constraints

- **Latency**: a cloud call per frame is seconds, not real-time. Use streaming
  coherence and/or a local fast model for live use.
- **GPU**: C5 reconstruction and real diffusion need one; the sidecars degrade
  to model-free stand-ins without it.
- **Keys**: never commit API keys. The adapters read them from env or a file
  outside the repo, mirroring `anthropic_brain.py`.
