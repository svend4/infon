# AI integration for TVCP — what was implemented

A complete, working bridge that lets **any neural network** act as a partner for
the TVCP terminal — drawing, playing games, reacting, and acting as a video
source — behind one open format, **`tvcp-ai/1`** (JSON over HTTP). Everything
here was compiled with Go 1.24 and run for real; the preview images (on the
[`assets`](https://github.com/svend4/infon/tree/assets) branch) are
faithful renders of the actual terminal output.

## New packages
| package | status | what it does |
|---|---|---|
| `pkg/scene` | done | renders the draw-DSL (JSON drawing language) to `terminal.Frame`; size-clamped |
| `pkg/sketch` | done | high-level "sketch" (named colors + shapes) → draw-DSL; easy for small models |
| `pkg/brain` | done | the `tvcp-ai/1` protocol types, an HTTP client, and a self-contained reference brain (tic-tac-toe minimax, Wordle solver, UNO policy, draw, sketch, react) |
| `internal/aisource` | done | `AICamera` (implements `device.Camera`, compile-time checked) + `BrainFrame` (model-painted frames) |

## New commands (`cmd/`)
| command | status | what it does | build |
|---|---|---|---|
| `aidraw` | done | draw-DSL from file/stdin, or a scene/`-sketch` from any brain | `go build ./cmd/aidraw` |
| `aiimg` | done | raster image → terminal via `internal/codec/babe.ImageToFrame` | `go build ./cmd/aiimg` |
| `aiplay` | done | tic-tac-toe: scripted X vs repo `minimax` O, rendered | `-tags experimental` |
| `aiturn` | done | one-move-per-run tic-tac-toe over a shared folder; O = random or `-brain` (play vs a model) | `-tags experimental` |
| `aicards` | done | glyph "cards" + truecolor effect; `-brain -msg` → model `react` cards | `go build ./cmd/aicards` |
| `aicam` | done | AI video: procedural plasma (device.Camera→babe) or `-brain` (model paints) | `go build ./cmd/aicam` |
| `aiwordle` | done | Wordle: a brain guesses, repo `words` engine validates/feeds back | `-tags experimental` |
| `aiuno` | done | UNO: a brain plays vs a bot on the repo `cards` engine | `-tags experimental` |
| `braindemo` | done | full game + draw over real localhost HTTP using the protocol | `go build ./cmd/braindemo` |
| `brainserver` | done | the reference `tvcp-ai/1` HTTP server | `go build ./cmd/brainserver` |

## Integrated into the main binary
- `cmd/tvcp/ai.go` + a `case "ai"` in `cmd/tvcp/main.go`: **`tvcp ai`** streams the
  AI video source; **`tvcp ai -brain URL`** lets a model paint the feed. The full
  `tvcp.exe` builds with this (5.5 MB) and `tvcp help` lists `ai`.

## The open format `tvcp-ai/1` (see `ai/BRAIN_PROTOCOL.md`)
`POST /v1/decide` with `{protocol,kind,game,state,prompt,canvas}` → a `Response`.
Kinds: **move** (games `tictactoe` / `wordle` / `uno`), **draw** (full DSL),
**sketch** (simple), **react** (symbol cards). `Move` carries `row/col`,
`word`, or `card_index/draw/color` depending on the game.

## Plug in any model (`ai/adapters/`)
- `anthropic_brain.py` — Claude (e.g. Haiku); key read from env or
  `~/.tvcp_anthropic.key` (never in the repo).
- `ollama_brain.py` — local models via Ollama.
- `openai_brain.py` — any OpenAI-compatible endpoint (OpenAI, llama.cpp, vLLM).

Each is a tiny HTTP server speaking `tvcp-ai/1`. **Swap brains by changing a URL.**
Adapters retry, validate moves per-game, and fall back safely.

## Verified live (see the [`assets`](https://github.com/svend4/infon/tree/assets) branch and `ai/showcase.html`)
- Real binaries built and run on Windows for every command above.
- Reference brain solved Wordle (`GHOST` in 3) and played a full UNO game.
- **Claude Haiku, live via the user's API key**: played tic-tac-toe (you won),
  drew a scene via `sketch`, reacted with symbol cards, painted video frames,
  and played real Wordle (guesses `SLATE`, `CORNY`, `OPIUM`).

## Honest status / limitations
- Reference policies are intentionally simple: UNO plays the first legal card
  (it lost to the bot); the Wordle solver filters a small built-in word list.
- Small models are weak at the verbose raw draw-DSL — use `sketch` instead
  (the clients enforce this path with `-sketch`); Haiku did not always solve
  Wordle (LLM narrowing is imperfect).
- This work is the AI layer only; the base repo's media codecs (H264/Opus/
  WASAPI audio, etc.) are unchanged and unrelated.

## Quick start
```
go run ./cmd/aidraw scenes/sunset.json
go run -tags experimental ./cmd/aiwordle -word GHOST
go run ./cmd/brainserver 127.0.0.1:8088   # then point clients at it with -brain
```
