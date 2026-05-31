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

# C5 — neural avatars over the network (talking face for kilobits/sec)
python ai/adapters/avatar_landmark_sidecar.py 8097   # sender: extracts keypoints
python ai/adapters/avatar_generate_sidecar.py 8098   # receiver: reconstructs face

# receiver:                         sender:
AVATAR_GEN_URL=http://127.0.0.1:8098/ ./bin/tvcp avatar receive 5000
AVATAR_LM_URL=http://127.0.0.1:8097/  ./bin/tvcp avatar send localhost:5000
# (measured ~35 kbps for a 15 fps talking face vs ~350 KB/s for block video)
```

Installing the real models upgrades each sidecar automatically:

```bash
pip install -r ai/adapters/requirements.txt   # all, or pick per feature:
pip install pillow realesrgan torch            # C1 restoration
pip install pillow ultralytics                 # C3 object detection
pip install pillow mediapipe numpy             # C3 faces + C5 landmarks
```

## Cloud generation with no local GPU (proxy sidecar)

`ai/adapters/cloud_image_sidecar.py` forwards the raster contract to a hosted
provider, so `tvcp synth neural` works with zero local model:

```bash
# Replicate (fast FLUX-schnell by default):
CLOUD_PROVIDER=replicate REPLICATE_API_TOKEN=r8_... \
  python ai/adapters/cloud_image_sidecar.py 8099
IMAGE_API_URL=http://127.0.0.1:8099/ ./bin/tvcp synth neural "a calm bay"

# Or fal.ai / OpenAI images:
CLOUD_PROVIDER=fal    FAL_KEY=...          python ai/adapters/cloud_image_sidecar.py 8099
CLOUD_PROVIDER=openai OPENAI_API_KEY=sk-... python ai/adapters/cloud_image_sidecar.py 8099
```

It uses only the Python standard library (no pip installs) — just provider keys
in the environment, never logged. A failed call returns a clean 502 so the Go
side keeps showing the last frame.

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
