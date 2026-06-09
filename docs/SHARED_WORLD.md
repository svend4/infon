# The Shared World — walking an AI-authored 3-D world in the terminal

This is the project's thesis made literal: **transmit meaning, not pixels.** An AI
director *describes* a 3-D world as a compact scene graph; each machine *renders*
it locally with a CPU path/ray tracer. Two or more people (and the AI) can walk
the **same** growing world together — in a plain terminal, for a handful of bytes
a second — because the world itself never crosses the wire. Only meaning does:
region descriptions (~100 bytes), poses (44 bytes), and short text/voice.

Everything is pure Go (+ stdlib); no GPU, no cloud required. `go build ./...` and
`go test ./...` are green.

## The three programs

| Command | What it is |
|---|---|
| `cmd/ray3d` | Render one demo scene with any of the engines (raster, path, BDPT, MLT, light tracer, PPM, ReSTIR); export PNG/GIF. |
| `cmd/rayexplore` | Walk solo through a world the AI director authors and **extends ahead of you** as you move. |
| `cmd/raymeet` | The shared world: several walkers (and the AI) explore one growing world together, seeing each other, with chat and optional voice. |

Controls in the walkers (type letters, then Enter): `w`/`s` walk, `a`/`d` strafe,
`q`/`e` turn, `r`/`f` look, `x` quit. In `rayexplore`, `g` grows a region now;
`p` toggles the path tracer. In `raymeet`, a line starting with `/` is a chat
message.

## Run the shared world (offline reference director)

The reference director runs in-process — no model needed — so you can try the full
experience immediately. One peer is the **hub** (`-host`); the others connect to
it.

```sh
# terminal A — the director hub, listening on UDP 5000
go run ./cmd/raymeet -host -name alice 5000

# terminal B — a guest
go run ./cmd/raymeet -name bob localhost:5000 5001

# terminal C — a third walker, fourth, ...
go run ./cmd/raymeet -name cara localhost:5000 5002
```

Walk forward (`w`) and the world unfolds ahead on every machine in lock-step. You
see the others as glowing coloured avatars. Type `/hello` to chat. Add `-voice`
on machines with a mic/speaker to also talk (it falls back to text-only where
there's no audio device).

## Run with a real AI director

Point the hub's brain at any tvcp-ai/1 endpoint with `BRAIN_URL`. Adapters for
Ollama (local), OpenAI-compatible and Anthropic live in `ai/adapters/`:

```sh
# 1) start a brain adapter (local Ollama example), serving /v1/decide on :8090
ollama serve                                  # in its own terminal
OLLAMA_MODEL=llama3.2 python ai/adapters/ollama_brain.py 8090

# 2) run the hub pointed at it; guests are unchanged
BRAIN_URL=http://127.0.0.1:8090/v1/decide go run ./cmd/raymeet -host 5000
```

(Cloud variants: `ai/adapters/openai_brain.py` on :8091, `ai/adapters/anthropic_brain.py`
on :8092 with `ANTHROPIC_API_KEY`.) Only the **host** needs a brain — it authors
each region and broadcasts the description; guests reconstruct it. The director
can build spheres, boxes, pyramids, cylinders, trees and houses with full
materials (diffuse, glass, metal, emissive); see `docs/TVCP_AI_PROTOCOL.md`,
game `rayscene`. A model's output is sanitised, so a bad response can't crash or
corrupt the world — it just falls back to a safe region.

## What crosses the wire (meaning, not pixels)

| Direction | Payload | Size |
|---|---|---|
| host → guests | a region's scene graph (`Region`) | ~100–300 bytes, once per region |
| any → all | a participant's pose (`Pose`) | 44 bytes per tick |
| any → all | chat (`ChatMsg`, id-tagged + deduped) / voice (`EncodeVoice`, origin-tagged) | a line / a 20 ms PCM chunk |
| guest → host | region ack (have-set) | tiny, ~1/s |

No geometry, no frames, no video. A walker who explores for an hour received a
trickle of descriptions and reconstructed every pixel locally.

## How it fits together

- `pkg/raytrace` — the renderer: five unbiased light-transport engines, full
  materials, BVH, procedural meshes.
- `pkg/raydir` — the bridge and the experience: `FlyCam` (free camera), `World`
  (a floor plus brain-authored regions), `Pose`/`PoseSet`, `Region` (the
  authored chunk on the wire), `ChatLog`, and the reliability/sanitisation
  helpers. This is where the testable core lives.
- `pkg/brain` — the tvcp-ai/1 protocol and the `rayscene` scene-graph types the
  director speaks.
- `cmd/raymeet` — the hub: learns peers, authors/broadcasts regions, relays
  poses/chat/voice, fills gaps on ack, prunes the disconnected.

The hub is a star: guests talk to it, it fans out. The world stays in sync because
the host is the single author and broadcasts each region's *meaning*; guests apply
it idempotently and ack what they have, so loss and late joins self-heal.

## Honest limits

- The reference director composes a coherent but repetitive world; the real
  novelty comes from a live `BRAIN_URL` model.
- Voice is wired and degrades gracefully but needs real audio hardware to hear.
  Concurrent speakers are now mixed at the listener (`VoiceMixer`: per-speaker
  jitter buffers summed each frame), not played back-to-back.
- Reliability is ack-based for regions (the state that must persist) and id-based
  for chat (`ChatSync`: unique ids, dedup, hub re-broadcast — so a dropped line
  self-heals without double-display); poses stay loss-tolerant (replaced next
  tick).
- Geometry is procedural primitives + composites *and* named meshes: `kind:mesh`
  instances a model from the renderer's library (built-in `crystal`/`rock`, plus
  any `.obj` loaded via `LoadMeshDir`), placed as a shared `Instance` — so a
  hundred copies cost one mesh, and still only a name crosses the wire. A `mesh`
  that sets material fields is tinted per-placement (one mesh, many finishes).
- A host can **persist and resume** its world (`-world file`): the world is just
  its region specs, so it saves/loads as a tiny file — restart where you left off,
  or copy a world to share it. (`SaveWorld`/`LoadWorldFile`.)
