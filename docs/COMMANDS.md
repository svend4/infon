# Commands — how to run the ray/Q6/alife platform

A user-friendly run guide for the ray/Q6/alife commands. For *what they are and how
they compose*, see [CONCEPT_MAP.md](CONCEPT_MAP.md); this page is *how to launch them*.

## How to run anything

Everything is a plain Go command — **no build or install needed** (Go 1.24+):

```bash
go run ./cmd/<name> [flags]      # run it
go run ./cmd/<name> -h           # list its flags
```

- Most commands **write a PNG** (default `<name>.png`, change with `-out`); the
  sound ones write a **WAV**; the meeting/walker ones run **live in the terminal**.
- Render size/quality knobs are usually `-w -h -spp` (width, height, samples/pixel).
- A live AI model is optional: set `BRAIN_URL` (director) and/or `VISION_API_URL`
  (vision); without them, deterministic offline reference authors are used. See
  [EXTERNAL_MODELS.md](EXTERNAL_MODELS.md).
- `scripts/qa.sh` runs a fast smoke of all of these (tested invocations).

## Quick start — try these five first

```bash
go run ./cmd/raypipe "a calm dawn" "a vast cold desert"  # prompt -> world; prints bytes saved
go run ./cmd/rayask "a forest at night"                  # a brain authors a world from a prompt
go run ./cmd/raytour                                     # an 8x8 atlas of all 64 hexagram worlds
go run ./cmd/raylife -seed 7 -ticks 240                  # a self-running ecosystem (boom/bust)
go run ./cmd/raysound -hexagram 110010 -out world.wav    # hear a world as music
```

## Meaning bridge — prompt / image ⇄ world

| Run | What it does |
|---|---|
| `go run ./cmd/raypipe "a calm dawn"` | prompt → brain → wire → render, end to end; prints meaning-vs-pixels (~40×) + reads the world's hexagram/mood back |
| `go run ./cmd/rayask "a forest at night"` | a brain authors a world from a prompt and path-traces it |
| `go run ./cmd/raydirect -seed 3 -rounds 250` | a learning director (engagement → taste) + the inverse reader |
| `go run ./cmd/rayread` | image/photo → Q6 hexagram + mood (the inverse reader; `-img file.png` for a real image) |
| `go run ./cmd/rayimagine photo.png` | read a picture, render the 3-D world it implies |
| `go run ./cmd/raydebate "a temple by water"` | adversarial co-direction (debate × director) |

## The Q6 hypercube as a place

| Run | What it does |
|---|---|
| `go run ./cmd/raytour` | the Gray-code grand tour: an 8×8 atlas of all 64 worlds (`-morph` for a morph film) |
| `go run ./cmd/rayquest -seed 7 -walls 0.5` | a roguelike maze on the 64-hexagram cube (changing lines = moves) |
| `go run ./cmd/rayquestai -seed 16 -walls 0.6 -episodes 12` | an agent that learns the maze under fog of war |
| `go run ./cmd/raycurator -seed 16 -walls 0.5` | finds the world a viewer loves most (engagement heatmap) |
| `go run ./cmd/rayreading -seed 7` | cast an I-Ching reading and render it as a world |
| `go run ./cmd/rayclimate -seed 42` | the hexagram-cellular-automaton climate (weather zones) |

## Living worlds (alife)

| Run | What it does |
|---|---|
| `go run ./cmd/raylife -seed 7 -ticks 240` | a self-running foraging ecosystem (boom/bust, climate-fed) |
| `go run ./cmd/raydungeon -seed 16 -walls 0.6 -ticks 200` | an ecosystem at each maze room, coupled by migration (source–sink) |
| `go run ./cmd/rayseasons -years 4 -year 120` | a living world through the seasons (a moving rain front; migration) |

## Sound

| Run | What it does |
|---|---|
| `go run ./cmd/raysound -hexagram 110010 -out world.wav` | a hexagram world as music (`-tour` for the grand tour, `-life` for a sonified ecosystem) |
| `go run ./cmd/rayscore -mood calm -night 0.8 -out night.wav` | the world's adaptive soundtrack to a WAV |

## Meeting & shared world (networked, in the terminal)

| Run | What it does |
|---|---|
| `go run ./cmd/raymeet -host 5000` then `go run ./cmd/raymeet localhost:5000 5001` | a group walks the SAME AI-grown world over UDP (`/msg` chat, `-voice` audio) |
| `go run ./cmd/rayexplore` | walk a 3-D AI-grown world in the terminal (single player) |
| `go run ./cmd/raygather -n 5 -hexagram 110010` | a meeting inside a hexagram world (N avatar faces) |
| `go run ./cmd/raymetaverse -n 3 -hexagram 110010` | a networked meeting: presence (pose + face) over UDP, world changes as deltas |
| `go run ./cmd/raycall -frames 48` | the semantic video call, end to end over (loopback) UDP |
| `go run ./cmd/raystream -frames 60` | a call where the wire carries scene DELTAS, not pixels |
| `go run ./cmd/rayspectate` | the shared world populated like an MMORPG (spectator + player view) |
| `go run ./cmd/rayface` | a meeting inside the world: avatar faces from keypoints |
| `go run ./cmd/rayvoice` | voice prosody steering the world's mood |
| `go run ./cmd/rayplay walk.rrec` | replay a recorded shared-world session |

## Renderer, scenes & terrain

| Run | What it does |
|---|---|
| `go run ./cmd/ray3d` | a CPU ray-traced scene to the terminal (`-renderer bdpt -spp 32 -png out.png` for 5 renderers) |
| `go run ./cmd/rayview` | an interactive terminal navigator for the ray tracer |
| `go run ./cmd/rayfilm -frames 9 -cols 3 -path -grade -out film.png` | a cinematic camera flying through a brain-authored world |
| `go run ./cmd/rayworld` | fly a ray-traced camera driven by a tvcp-ai brain |
| `go run ./cmd/rayarena` | a tvcp-ai brain authors a moving 3-D "arena" |
| `go run ./cmd/rayfx -mode caustics` | the caustics / adaptive / temporal renderers |
| `go run ./cmd/rayhard` | the integrator "renderer brain" — auto-picks BDPT/MLT for hard scenes |
| `go run ./cmd/rayterrain -seed 3 -amp 9` | Delaunay-meshed terrain |
| `go run ./cmd/rayvoxel -seed 7 -out voxel.png` | a voxel-space landscape |
| `go run ./cmd/raymandala -fold 8 -mirror` | a kaleidoscopic mandala (dihedral symmetry) |
| `go run ./cmd/raygallery worlds/` | browse a directory of saved worlds |

## Fleet & robots (the monitoring layer)

| Run | What it does |
|---|---|
| `go run ./cmd/rayfleet` | watch a fleet of machines (camera → signatures → assess → VFX) |
| `go run ./cmd/raywatch` | a camera observes a robot with a hot motor and reacts |
| `go run ./cmd/raydetect` | a vision model's detections → a world |
| `go run ./cmd/raycamp` | a robot logistics yard (writes `camp*.png`) |
| `go run ./cmd/raygates` | the sim-to-real conformance gates (writes `gates.png`) |
| `go run ./cmd/rayagent` | robots under a tvcp-ai/1 brain |
| `go run ./cmd/rayyard` | the shared world as a robot yard (a contact sheet over time) |
| `go run ./cmd/raybench` | benchmark a rayscene director (reference or a live `BRAIN_URL`) |

## The full set

There are ~120 commands in `cmd/` (this page covers the ray/Q6/alife platform). To
discover the rest:

```bash
ls ./cmd                          # every command
go doc ./cmd/<name>               # its package doc — purpose + usage example
```

For the original terminal-video platform (`tvcp`, `connect`, camera/audio/screen),
see the repo-root subsystem docs and [../README.md](../README.md).
