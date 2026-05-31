# Pseudo-Images — let an ordinary text model paint (no diffusion model)

Diffusion models are not the only way to make a neural net draw. A plain chat
model — one that can only emit text/JSON, with **no image head** — can paint for
the terminal if we let it speak a tiny, structured "pseudo-image" and expand it
deterministically into pixels. That is what `pkg/pseudo` does.

The model emits a few tokens of JSON (a coarse color grid, a glyph map, a list of
named symbols, or vector shapes). The renderer turns that into a `terminal.Frame`
(for the live `.babe` → ANSI path) or an `image.Image` (for the video pipeline).
Swap the brain (Ollama / OpenAI / Anthropic / the built-in reference) without
changing anything else — it is the same open `tvcp-ai/1` protocol.

## The formats

| `format` | What the model emits | Style | Best for |
|----------|----------------------|-------|----------|
| `grid`   | coarse grid of named colors | smooth, painterly (**pseudo-diffusion**) | scenes, moods, gradients |
| `pixels` | the same grid | crisp mosaic / pixel-art | logos, icons, retro art |
| `glyphs` | a character grid + a char→color map | colored ASCII/Unicode art | text-shaped pictures |
| `sigils` | named pictographs placed on a canvas | iconographic | symbolic / hieroglyph scenes |
| `vector` | the draw-DSL (`pkg/scene`) | precise vector shapes | diagrams, exact layouts |
| `sketch` | named scene shapes (`pkg/sketch`) | weak-model friendly | quick landscapes |

All six describe the SAME subject in `cmd/pseudodemo` so you can compare them.

## Pseudo-diffusion (the `grid` format)

A 12×8-ish grid of named colors is the smallest "image" a model can produce. The
renderer:

1. **upscales** it (bilinear) to the target size, and
2. **blurs** it (separable box blur, 3 passes).

The result is a soft, painterly raster that *looks* diffusion-made — but there is
no diffusion model, no GPU, and it is fully deterministic. `Grid.DiffuseFrames`
additionally produces a "denoising" animation (seeded RGB noise resolving into
the target as the blur fades out) for transitions and demos.

```json
{
  "format": "grid",
  "cols": 80, "rows": 36,
  "grid": { "rows": [
    ["navy","navy","purple","amber","gold","gold","amber","purple","navy","navy","navy","navy"],
    ["navy","purple","amber","gold","white","gold","gold","amber","purple","navy","navy","navy"],
    ["slate","amber","gold","gold","gold","amber","teal","teal","slate","slate","slate","slate"],
    ["teal","teal","skyblue","teal","teal","skyblue","teal","teal","teal","skyblue","teal","teal"]
  ] }
}
```

Color tokens are palette names (`navy`, `gold`, `teal`, `slate`, …, shared with
`pkg/sketch`), `#rrggbb` / `#rgb`, or `r,g,b`.

## The protocol: `kind: "image"`

Request (a field `format` selects the encoding; empty = `grid`):

```json
{ "protocol": "tvcp-ai/1", "kind": "image", "prompt": "a calm harbor at dawn",
  "format": "grid", "canvas": { "width": 64, "height": 28 } }
```

Response carries a pseudo-image spec in `image`:

```json
{ "protocol": "tvcp-ai/1", "kind": "image", "image": { "format": "grid", "grid": { "rows": [ ... ] } } }
```

The **reference brain** (`brain.Reference`, served by `brain.Handler`) implements
`kind:"image"` for every format — so the protocol can paint with **no model at
all**. Any external model just has to emit the same shapes (see `adapters/`).

## Glyphs, sigils, vector, sketch

```json
{ "format":"glyphs", "glyphs": {
    "bg":"navy", "palette": {"*":"gold","^":"slate","~":"skyblue","=":"teal"},
    "rows": ["      *      ", "   ^^^^^^^   ", " ==~~==~~==~ "] } }

{ "format":"sigils", "sigils": {
    "sky":"navy", "ground":"teal",
    "items": [ {"name":"sun","x":0.7,"y":0.25,"color":"gold"},
               {"name":"mountain","x":0.3,"y":0.6,"color":"slate"} ] } }

{ "format":"vector", "vector": { "width":80, "height":28, "bg":[28,38,88],
    "ops": [ {"op":"disc","cx":56,"cy":8,"r":4,"glyph":"█","fg":[245,200,90]} ] } }

{ "format":"sketch", "sketch": { "sky":"navy", "ground":"teal",
    "shapes": [ {"kind":"sun","x":56,"y":6,"color":"gold"},
                {"kind":"mountain","x":24,"h":8,"color":"slate"} ] } }
```

`sigils` understands names like `sun ☀ · moon ☾ · star ★ · mountain ▲ · cloud ☁ ·
wave ∿ · anchor ⚓ · house ⌂ · heart ♥ · bolt ⚡` (default `◆`).

## Mixed format + palette presets (v2)

`mixed` layers crisp symbols and captions over a soft pseudo-diffusion grid — a
painterly background the model describes as a color grid, with sharp sigils and
labels placed on top (best of both worlds):

```json
{ "format":"mixed", "palette":"dusk", "mixed": {
    "grid":   { "rows": [["navy","amber","gold","amber","navy"], ["teal","skyblue","teal","blue","teal"]] },
    "sigils": [ {"name":"sun","x":0.7,"y":0.2,"color":"gold"}, {"name":"anchor","x":0.6,"y":0.8,"color":"white"} ],
    "labels": [ {"text":"tvcp-ai","x":0.03,"y":0.92,"color":"white"} ] } }
```

A `palette` preset remaps the named colors so the model can pick a mood without
re-specifying every color — `dawn`, `dusk`, `neon`, `mono`, `forest`, `ocean`.
It applies to every format, and over the protocol via `Request.Palette` (or the
`palette` field of the spec):

```bash
TVCP_PSEUDO=mixed BRAIN_URL=http://localhost:8088 tvcp synth neural "a calm harbor at dusk"
```

New sigils since v1: flag, skull, yinyang, phone, pencil, scissors, plane, crown,
king, pawn, boat, comet, atom, peace.

## Using it

Render a spec directly:

```go
frame, _ := pseudo.Render(specJSON)     // -> *terminal.Frame
img, _   := spec.Image(480, 216)        // -> image.Image (for the video path)
```

Generate previews of all formats:

```bash
go run ./cmd/pseudodemo ./_pseudo_go     # writes 1_grid.png … 6_sketch.png + pseudo_diffusion.gif
```

Paint the live synth video source with any text model behind the protocol:

```bash
# terminal 1: a tvcp-ai/1 endpoint (reference brain, Ollama, or an adapter)
tvcp braindemo            # or: python adapters/ollama_brain.py

# terminal 2: pseudo-image neural source. TVCP_PSEUDO picks the format
#             ("1" or "grid" = pseudo-diffusion; or glyphs/sigils/vector/sketch)
set BRAIN_URL=http://localhost:8088
set TVCP_PSEUDO=grid
tvcp synth neural "a calm harbor at dawn"
```

The slow model never stalls the terminal: generation runs through the async
`device.NeuralGenerator`, exactly like the sketch backend.

## Ideas this opens up

- **Denoise transitions** — use the `DiffuseFrames` animation as a scene wipe.
- **Mixed mode** — a soft `grid` background with a few crisp `sigils`/labels on top.
- **Temporal coherence** — keep a latent grid and let the model nudge it per frame
  (live `SetPrompt`) for smoothly evolving video from a plain text model.

## Code

- `pkg/pseudo` — formats, renderers, pseudo-diffusion, tests.
- `pkg/brain` — `kind:"image"` protocol + reference painter (`refImage`).
- `internal/aisource/pseudobackend.go` — `device.NeuralBackend` bridge.
- `cmd/pseudodemo` — preview generator.
