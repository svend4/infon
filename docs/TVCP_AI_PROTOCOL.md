# tvcp-ai — protocol specification

**Wire version:** `tvcp-ai/1` (the string carried in every message)
**Document revision:** v2 (consolidates capability negotiation, conformance,
discovery, trust and multiagent negotiation into one normative spec)
**Status:** stable for the kinds, games and formats listed here; extensible.

This document is the normative reference for the format. The authority is the Go
code in `pkg/brain`; where prose and code disagree, the code wins, and the
drift-guard test `pkg/brain/speccoverage_test.go` exists to keep them aligned.

---

## 1. Design principle

tvcp-ai carries **meaning, not pixels**. A participant never ships a rendered
image, an audio buffer or a video frame across the wire. It ships a small piece
of structured JSON — a move, a scene description, a symbolic image spec, a set of
reaction glyphs — and the *receiver* renders it locally. A 30×20 painting is a
few hundred bytes of intent; the recipient's renderer turns those bytes into
something to look at, hear (`pkg/sona`), or feel (`pkg/braille`).

A consequence of this principle bounds the scope of this document: tvcp-ai
describes only the **symbolic** layer. Real-time media transports (microphone
capture, camera capture, Opus, NAT traversal, GPU encode) are deliberately *out
of scope* here and live in the base platform, not in this protocol.

A *brain* is anything that can answer a `Request` with a `Response`. The backend
may be this process, a local model (Ollama), a cloud model (OpenAI, Anthropic),
or a shell script. You swap brains by changing a URL.

---

## 2. Transport

The default transport is **JSON over HTTP**.

| Endpoint | Method | Purpose |
|---|---|---|
| `/v1/decide` | POST | Send a `Request`, receive a `Response`. |
| `/v1/capabilities` | GET | Receive the brain's advertised `Capabilities`. |

`pkg/brain.Mux(caps)` returns an `http.Handler` that serves both: the reference
brain at `/v1/decide` and the given capabilities at `/v1/capabilities`. A client
uses `HTTPBrain{URL: ".../v1/decide"}` to call a brain and
`FetchCapabilities(".../v1/capabilities")` to read its capabilities.

A **streaming profile** (`pkg/stream`) exposes the same decisions as
Server-Sent Events for spectators who want to watch a match unfold rather than
request a single decision. Streaming is an additive profile; the request/response
contract below is unchanged.

Every message SHOULD carry `"protocol": "tvcp-ai/1"`. A client that omits it has
it filled in by `HTTPBrain.Decide`; a server always stamps it on the way out.

---

## 3. Request

```json
{
  "protocol": "tvcp-ai/1",
  "kind":     "move|draw|sketch|image|react",
  "game":     "tictactoe",          // move only; omitted otherwise
  "state":    { ... },              // game-specific, opaque JSON
  "prompt":   "a calm harbor at dawn",
  "canvas":   { "width": 48, "height": 20 },
  "format":   "grid",               // image only
  "palette":  "dusk"                // image only, optional mood preset
}
```

| Field | Type | Meaning |
|---|---|---|
| `protocol` | string | Format version; `tvcp-ai/1`. |
| `kind` | string | One of the five kinds (§5). Required. |
| `game` | string | Game identifier; present only when `kind = "move"`. |
| `state` | object | Game-specific state; opaque to the transport (§6). |
| `prompt` | string | Free text for `draw`, `sketch`, `image`, `react`. |
| `canvas` | object | Target size in terminal cells (`width`, `height`). |
| `format` | string | Image encoding (§7); `image` only. |
| `palette` | string | Optional mood preset; `image` only. |

---

## 4. Response

```json
{
  "protocol":  "tvcp-ai/1",
  "kind":      "move",
  "move":      { "row": 1, "col": 1 },
  "scene":     { ... },   // draw
  "sketch":    { ... },   // sketch
  "image":     { ... },   // image: a pkg/pseudo Spec
  "tangram":   { ... },   // move/tangram: a solution figure
  "world":     { ... },   // move/world: next-tick directives
  "rpg":       [ ... ],    // move/rpg: per-unit moves
  "ray":       [ ... ],    // move/ray: per-sphere 3-D scene moves
                           // move/rayscene: a full authored scene graph (objects+materials+light+sky)
  "cards":     ["★","♥"],  // react
  "reasoning": "optional human-readable rationale",
  "error":     ""          // non-empty iff the brain failed the request
}
```

Exactly one payload field is populated per response, selected by `kind` (and by
`game` for moves). A non-empty `error` means the request was not satisfied and
all payload fields MUST be ignored.

The `move` object covers every board/word/card game:

| Field | Used by | Meaning |
|---|---|---|
| `row`, `col` | tic-tac-toe | cell coordinates, each in `0..2` |
| `word` | wordle | a 5-letter guess (`^[A-Za-z]{5}$`) |
| `card_index` | uno | index into the hand; `null` if drawing |
| `draw` | uno | `true` to draw instead of playing |
| `color` | uno | chosen color for a wild |

---

## 5. Kinds

| Kind | Response payload | Validated by §8 as |
|---|---|---|
| `move` | `move` / `tangram` / `world` / `rpg` / `ray` | legal for the named game |
| `draw` | `scene` | a `scene.Scene` with ≥1 op |
| `sketch` | `sketch` | a `sketch.Sketch` with ≥1 shape |
| `image` | `image` | a `pseudo.Spec` that renders to a non-empty frame |
| `react` | `cards` | ≥1 glyph card |

---

## 6. Games (for `kind = "move"`)

| Game | State shape (abridged) | Response field |
|---|---|---|
| `tictactoe` | `board[3][3]`, `turn`, `you`, `legal[]` | `move.row/col` |
| `wordle` | `length`, `guesses[]{word,marks[]}` | `move.word` |
| `uno` | `hand[]`, `color`, `playable[]` | `move.card_index`/`draw` |
| `tangram` | a `tangram.PuzzleState` | `tangram` (placements) |
| `world` | scene brief (`fold_pct`, `relief_pct`, …) | `world` (4 directives ∈ `{-1,0,1}`) |
| `rpg` | `you`, `w`, `h`, `units[]`, `enemies[]` | `rpg` (`[]{id,dx,dy}`) |
| `ray` | `spheres[]{id,x,y,z}` | `ray` (`[]{id,dx,dy,dz}`, each ∈ `{-1,0,1}`) |
| `rayscene` | `{prompt}` | `ray` (a scene graph: `{objects[]{kind:sphere\|box\|pyramid\|cylinder\|tree\|house\|mesh\|fractal\|plane,name,tex,x,y,z,r,s:[hx,hy,hz],color,emit,glass,metal,reflect,rough}, light, skyTop, skyBottom}`; `kind:mesh` instances a named model — `name` ∈ the renderer's mesh library, e.g. `crystal`, `rock`, plus any `.obj` loaded; `kind:fractal` (alias `sdf`) ray-marches a named signed-distance form — `name` ∈ `mandelbulb`, `menger`, `sierpinski`, `mandala`, `melt`, `escher`; a mesh/fractal that sets material fields is tinted per placement; `tex` names a surface texture — `checker`, `marble`, `wood`, `stone`, `clouds`, or any image loaded; `bump` names a normal/bump map — `ripple`, `waves`, `bumps`) |

`state` is opaque to the transport: a brain parses only the games it advertises.

---

## 7. Image formats (for `kind = "image"`)

`grid` (pseudo-diffusion seed), `pixels`, `glyphs`, `sigils`, `vector`, `sketch`,
`mixed`, `marks`. Each maps to a field of the `pseudo.Spec`. The receiver renders
the spec; the validator (§8) requires only that the spec parses and produces a
non-empty frame, so new renderers can improve fidelity without breaking the wire.

---

## 8. Conformance

A brain is **tvcp-ai conformant** iff it correctly answers every case in the
standard battery (`brain.ConformanceBattery`, 14 cases spanning all five kinds
and all seven games). `brain.CheckResponse` is the normative validator; it checks,
per kind/game: a move is legal, a wordle guess is 5 letters, a tangram solution
is overlap-free with intersection-over-union ≥ 0.5, a draw/sketch carries shapes,
an image spec renders. Run the battery against any endpoint with `cmd/conform`;
run it against the reference brain in-process with `RunConformance`.

The advertised `Capabilities` and the conformance battery are bound by
`pkg/brain/speccoverage_test.go`: every advertised kind and game must be
exercised by the battery, and the battery may not test a kind the brain does not
advertise. This is what keeps this document honest.

---

## 9. Capability negotiation

```json
GET /v1/capabilities
{
  "protocol": "tvcp-ai/1",
  "kinds":    ["move","draw","sketch","image","react"],
  "games":    ["tictactoe","wordle","uno","tangram","world","rpg","ray","rayscene"],
  "formats":  ["grid","pixels","glyphs","sigils","vector","sketch","mixed","marks"]
}
```

`Capabilities.Supports(kind, game)` reports whether a brain covers a request
before you send it. A client SHOULD fetch capabilities and check `Supports`
rather than discovering unsupported kinds by trial. A registry or discovery layer
(§10) routes by the same capability set.

---

## 10. Discovery, registry and trust

These layers are normative companions, not part of the request/response hot path.

- **Discovery** (`pkg/discovery`): brains publish their endpoint and capabilities
  as DNS-TXT records, so a client can find a brain that supports `move/rpg` the
  same way it resolves a hostname.
- **Registry** (`pkg/registry`): a list of known brains with their capabilities.
- **Trust** (`pkg/sign`): registry entries can be signed with HMAC so a client can
  verify an entry came from a trusted curator before routing work to it. A signed
  entry binds `(name, url, capabilities)` to a signature; an unverified entry is
  treated as untrusted.

---

## 11. Multiagent negotiation (Contract Net)

Beyond one client calling one brain, agents can **negotiate** who does a task via
a FIPA-style Contract Net (`pkg/acl`). This is the first layer where agents
cooperate by bargaining rather than only competing on a board.

| Performative | Direction | Meaning |
|---|---|---|
| `cfp` | initiator → bidders | call for proposals on a task |
| `propose` | bidder → initiator | I can do it at this cost |
| `refuse` | bidder → initiator | I decline |
| `accept-proposal` | initiator → winner | you are awarded the task |
| `reject-proposal` | initiator → loser | a lower bid won |
| `inform` | winner → initiator | done |

`ContractNet(initiator, task, bidders)` broadcasts a `cfp`, collects
`propose`/`refuse`, awards the lowest bid (ties broken by name), and returns the
`Outcome` plus the full performative-by-performative conversation log. A live
tvcp-ai brain bids through the same `Bidder` interface that an in-process
`FuncBidder` uses, so negotiation composes with §9 capabilities: route a `cfp`
only to brains whose `Supports` covers the task.

---

## 12. Versioning and extension

The wire string stays `tvcp-ai/1` while the message *shapes* in §3–§4 remain
backward compatible. New kinds, games or image formats are added by extending
`ReferenceCapabilities` and the conformance battery together (the §8 test
enforces this). A brain that does not recognize an advertised game simply omits
it from its own capabilities; clients negotiate around the gap via §9. A future
incompatible change to the message shapes — not anticipated here — would bump the
wire string to `tvcp-ai/2`.

---

## 13. Errors

A brain reports failure by returning a `Response` with a non-empty `error` and no
payload. The HTTP handler returns `400` with an `error` response for a malformed
request body. Transport failures (timeouts, refused connections) surface as Go
errors from `HTTPBrain.Decide` and are reported by the conformance runner as
`transport: …`.

---

## 14. Security considerations

- A brain endpoint is untrusted code answering on the network: validate every
  response with `CheckResponse` before acting on it. The validator is the trust
  boundary, not the brain.
- Route work only to brains whose registry entry verifies under §10 signing.
- `state` and `prompt` are attacker-influenced; renderers MUST treat a
  `pseudo.Spec` as data and bound its size (the validator already rejects specs
  that fail to render).
- Capability advertisements are claims, not proofs: a brain may advertise a kind
  it answers badly. Conformance (§8) is how a claim is checked.
