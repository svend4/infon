# feat/ai-next — overview

This branch grows the terminal video platform a second nervous system: an open
protocol that lets any neural network *be* the video source, a symbolic
substrate (a glyph alphabet that scales up into figures, numbers, error
corrected codes and semantic video), and a small pseudo-3D engine driven by the
same few-hundred-bytes-over-loss budget. Everything is plain Go (+ stdlib) with
three optional Python brain adapters. All packages build and test green.

## The tvcp-ai/1 protocol (`pkg/brain`)

A brain answers `Decide(Request) -> Response`. `Reference` is a self-contained
brain; `Local` wraps it; `HTTPBrain{URL}` speaks to any `/v1/decide` server, so
a live model is a drop-in. `kind` is `move | draw | sketch | image | react`;
`move` carries a `game`:

| game | state in | answer out |
|------|----------|-----------|
| tictactoe / wordle / uno | board / guesses / hand | `move` |
| tangram | `PuzzleState` (glyph grid) | `tangram` figure (scored by IoU) |
| world | `Brief` (fold%, relief%, camera) | `world` directives `{fold,rise,spin,camera}` |

`pkg/brain/conformance.go` validates every shape. `cmd/conform` and
`cmd/brainserver` run the battery and a server.

### Live model adapters (`ai/adapters/`)

`anthropic_brain.py`, `ollama_brain.py`, `openai_brain.py` are tvcp-ai/1 servers
backed by Haiku, a local Ollama model, or any OpenAI-compatible endpoint. All
three handle every kind plus `game:tangram` and `game:world`. Keys are read from
the environment / a file and never printed. Point any client at the URL:

    python ai/adapters/anthropic_brain.py 8092
    go run ./cmd/tangrambench -brain http://127.0.0.1:8092/v1/decide
    go run ./cmd/worldai      -brain http://127.0.0.1:8092/v1/decide

## The symbolic substrate

- `pkg/glyphset` — a diagonal-triangle glyph alphabet with sub-cell coverage
  masks; a tiny font; the renderer's `triangle` mode.
- `pkg/pseudo` — pseudo-image formats (grid/pixels/glyphs/sigils/vector/sketch/
  mixed/marks) so a text model can "paint"; `pkg/blazon` compiles heraldry to it.
- `pkg/tangram` — grid-glyph figures: a 16-entry **address book** (`Catalog`),
  `Validate` (tiling rule), `Creature` grammar, `Number`/`CanonicalNumber`
  (an I-Ching-style positional id), `Classify`/`Address`, and the `PuzzleState`
  spatial-reasoning task with `Solve`/`ScorePuzzle` (IoU) plus the silhouette
  **hard mode** (`cmd/tangramhard`: easy 1.00 / hard 0.998 / coarse 0.893).
- `pkg/tangram7` — the *authentic* 7-piece geometric tangram (continuous
  polygons, verified dissection) with authored figures (square/cat/swan).
- `pkg/glyphqr` — GF(256) Reed-Solomon + a glyph-QR codec + an interleaved
  byte-stream FEC that survives bursts.
- `pkg/semframe` / `pkg/semvideo` — semantic P-frames (spec deltas) and a
  keyframe+delta video codec over RS; `cmd/semcall` is a lossy UDP "semantic
  video call", `cmd/semcodec` a rate-distortion-of-meaning benchmark.

## The pseudo-3D program (see `docs/PSEUDO3D.md`)

Three ways to lift the flat tangram into pseudo-3D, then their synthesis:

1. `pkg/fold` — origami: extrude or fold each piece up by a dihedral angle;
   `cmd/folddemo`, `cmd/foldcall` (a 16-frame unfold = 123 bytes over RS).
2. `pkg/relief` — 2.5D: one height byte per marks cell, diagonal glyphs become
   lit facets (`cmd/reliefdemo`, a voxel/Q*bert view).
3. `pkg/scene3d` — a `SceneSpec` of placed pieces `{piece,x,y,z,yaw,colour}`;
   `cmd/scenecall` orbits 7 pieces in ~50 bytes/frame over RS.

Synthesis: `pkg/deltastream` is the generic keyframe+byte-delta codec all three
share; `fold.RenderCam` is the one isometric renderer (camera-yaw turntable,
`cmd/orbit3d`); `cmd/pseudo3d` packs all three into one 50-byte/frame stream and
one world. `pkg/world` closes the loop: a 51-byte `State`, a `Brain`/`Controller`
that emits the next state, and `cmd/worldloop` / `cmd/worldai` — a brain (the
reference director, or a **live** model via `game:world`) directs the 3-D world
frame by frame: neural -> bytes -> 3-D frame.

## Running things

    go build ./...            # everything builds
    go test ./...             # green
    go run ./cmd/tvcp ...     # the terminal video platform
    go run ./cmd/tangramzoo   # render the 16-figure address book
    go run ./cmd/pseudo3d     # the unified pseudo-3D scene + stream
    go run ./cmd/worldai      # a brain directs the 3-D world (-brain URL for live)

Binaries, the API key, and scratch dirs are git-ignored and never committed.
