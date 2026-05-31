# tvcp-ai — protocol specification (v2)

`tvcp-ai` is an OPEN, transport-agnostic format that lets ANY neural network (or
any program) be a partner for a terminal: playing games, drawing, painting
pseudo-images, and reacting. **The protocol is the product; the model is a
swappable part.** You change brains by changing a URL.

- Version string: `tvcp-ai/<major>`. This document specifies **v2**, a
  backward-compatible superset of v1: a v2 brain answers every v1 request, and a
  v1 client ignores v2-only fields. Unknown fields MUST be ignored.
- Wire form (v1/v2): **HTTP `POST /v1/decide`**, `Content-Type: application/json`,
  one `Request` in, one `Response` out. (A streaming `WebSocket` binding is
  reserved for a future minor.)

## Envelope

**Request**

| field | type | notes |
|---|---|---|
| `protocol` | string | `"tvcp-ai/1"` or `"tvcp-ai/2"` (optional; server may default) |
| `kind` | string | `move` \| `draw` \| `sketch` \| `image` \| `react` |
| `game` | string | for `move`: `tictactoe` \| `wordle` \| `uno` |
| `state` | object | game-specific state |
| `prompt` | string | free text for `draw`/`sketch`/`image`/`react` |
| `canvas` | object | `{width,height}` target size in terminal cells |
| `format` | string | for `image`: `grid\|pixels\|glyphs\|sigils\|vector\|sketch\|mixed` |
| `palette` | string | for `image`: mood preset `dawn\|dusk\|neon\|mono\|forest\|ocean` |

**Response**

| field | type | notes |
|---|---|---|
| `protocol`,`kind` | string | echo |
| `move` | object | `{row,col}` / `{word}` / `{card_index,color,draw}` |
| `scene` | object | a draw-DSL document (`pkg/scene`) |
| `sketch` | object | a high-level sketch (`pkg/sketch`) |
| `image` | object | a pseudo-image spec (`pkg/pseudo`) |
| `cards` | string[] | glyph "cards" for `react` |
| `reasoning` | string | human-readable rationale / **alt-text** (see below) |
| `error` | string | non-empty signals failure; clients fall back |

## Kinds (normative requirements)

- **move / tictactoe** → `move.row`,`move.col` ∈ 0..2 on an empty legal cell.
- **move / wordle** → `move.word`: one real 5-letter word, consistent with
  `state.guesses` (per-letter marks `correct|present|absent`).
- **move / uno** → either `move.card_index` (a playable card) or `move.draw=true`;
  for a wild, set `move.color`.
- **draw** → `scene` with ≥1 op.
- **sketch** → `sketch` with ≥1 shape.
- **image** → `image` is a valid pseudo-image spec that renders to a non-empty
  frame. Honor `format` and `palette` when given.
- **react** → `cards` with ≥1 entry.

## Principles

- **Alt-text duality.** Every visual response SHOULD be self-describing: a
  pseudo-image spec already *is* a compact, human-readable description, and
  `reasoning`/`title` SHOULD carry a one-line caption. This makes the medium
  natively screen-reader friendly — the picture and its description are one.
- **Determinism & provenance.** A spec is tiny, human-readable, and renders the
  same picture every time: it can be versioned, diffed, signed, and replayed.
- **Graceful degradation.** A brain MAY downgrade fidelity (image → sketch →
  cards) under constraints; a client MUST fall back on `error`.

## Capability negotiation (optional, v2)

A brain MAY answer `GET /v1/capabilities` with
`{"protocol":"tvcp-ai/2","kinds":[...],"games":[...],"formats":[...]}` so clients
can discover what it supports. Absence means "assume v1 core".

## Security (recommended for production)

Brains SHOULD require a bearer token over TLS; responses MAY be detached-signed.
Never embed API keys in specs or logs.

## Conformance

An implementation is **conformant** iff it passes the standard battery in
`pkg/brain` (`ConformanceBattery` / `CheckResponse`). Run it against any endpoint:

```bash
go run ./cmd/conform                                       # reference brain
go run ./cmd/conform -url http://127.0.0.1:8092/v1/decide  # a live model
```

Machine-readable schemas: `ai/schema/tvcp-ai.request.schema.json`,
`ai/schema/tvcp-ai.response.schema.json` (JSON Schema 2020-12).
