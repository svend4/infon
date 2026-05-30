# tvcp-ai — connecting a neural net to terminal graphics

A staged bridge between an AI/LLM and the `svend4/infon` (TVCP) terminal
renderer. The model decides *what* to draw / play / show; the repo's own code
turns it into 24-bit colored Unicode blocks.

    prompt / state  ->  (neural net = Claude)  ->  data  ->  repo code  ->  terminal

Everything below was compiled with the real Go toolchain (Go 1.24) on the PC and
run for real; the PNG/GIF previews are produced by `render_preview.py`, a twin of
`pkg/terminal/renderer.go` that was verified pixel-identical to the Go output.

## Programs (all import the repo's packages)

| program        | path                | uses repo code                         | run |
|----------------|---------------------|----------------------------------------|-----|
| aidraw (L1a)   | cmd/aidraw          | pkg/terminal, pkg/color                | `go run ./cmd/aidraw scenes/sunset.json` |
| aiimg  (L1b)   | cmd/aiimg           | internal/codec/babe.ImageToFrame       | `go run ./cmd/aiimg assets/aurora.png 80 50` |
| aiplay (L2)    | cmd/aiplay          | experimental/games/board (minimax)     | `go run -tags experimental ./cmd/aiplay` |
| aicards (L3)   | cmd/aicards         | pkg/terminal, pkg/color                | `go run ./cmd/aicards` |

Prebuilt Windows binaries are in `infon/bin/` (aidraw.exe, aiimg.exe,
aiplay.exe, aicards.exe). Run any in a TrueColor terminal, e.g.:

    & "C:\Users\stefan\Documents\claudeaidaten\cowork\infon\bin\aidraw.exe" `
      "C:\Users\stefan\Documents\claudeaidaten\cowork\infon\scenes\sunset.json"

## draw-DSL schema (aidraw)

Canvas: `{ "width": int, "height": int, "bg": [r,g,b]?, "ops": [ ... ] }`

| op          | fields                                              |
|-------------|-----------------------------------------------------|
| fill        | glyph, fg, bg                                       |
| rect        | x, y, w, h, glyph, fg, bg?                           |
| vgradient   | x, y, w, h, glyph, from, to   (bg per row)          |
| hgradient   | x, y, w, h, glyph, from, to   (fg per column)       |
| text        | x, y, s, fg, bg?                                     |
| box         | x, y, w, h, fg, bg            (uses the repo's box)  |
| quad        | x, y, tl, tr, bl, br, fg, bg  (2x2 sub-cell glyph)  |
| disc        | cx, cy, r, glyph, fg, bg?     (aspect-corrected)    |

`bg` omitted = keep the existing cell background (composite over a gradient).

## Roadmap

- [x] Phase 0  — build/verify the renderer
- [x] Phase 1a — draw-DSL -> Go interpreter -> Frame            (aidraw)
- [x] Phase 1b — image -> babe.ImageToFrame -> terminal         (aiimg)
- [x] Phase 2  — AI player over experimental/games/board        (aiplay)
- [x] Phase 3  — visual comms: cards + truecolor effect         (aicards)
- [ ] Phase 4  — AIFrameSource + `tvcp ai` subcommand (AI as a video source,
                 so the existing call / record / export carry AI-generated video)

## previews/
sunset.png, sunset_real.png (real Go output), aurora_terminal.png, ttt.gif,
ttt_sheet.png, cards.png, effect.gif

## Phase 4 — AI as a video source (done)

`internal/aisource/camera.go`: `AICamera` implements `device.Camera`, checked at
compile time by `var _ device.Camera = (*AICamera)(nil)`. `cmd/aicam` runs it
through the REAL pipeline `device.Camera -> babe.ImageToFrame -> terminal.Frame`
(see previews/aicam.gif). Drop-in: copy `integration/ai.go.txt` to
`cmd/tvcp/ai.go` and add `case "ai": runAI()` to main.go — then the same source
flows through send / call / record / export unchanged.

    go run ./cmd/aicam        # or bin\aicam.exe
