# Q6 interop — one 6-bit coordinate across the projects

A recurring "house style" runs through several sibling repositories by the same
author: **the 6-dimensional hypercube Q6 — 64 vertices, each a 6-bit number, edges
one bit apart — used as a shared coordinate.** This note records how `infon`
connects to the others **by data and protocol, not by code**: each project stays
independent (different languages, different domains), and they interoperate through
two tiny, stable interchange formats.

## The projects on the Q6 axis

| Repo | What Q6 is there |
|---|---|
| [`meta`](https://github.com/svend4/meta) | the mathematics of Q6 (the `hexcore` graph library: neighbours, Gray code, antipode) — adapted here as `Hexagram.Neighbors/Antipode/GrayWalk` and the bubble-graph layout |
| [`pro2`](https://github.com/svend4/pro2) | a transformer whose inductive bias is Q6 (64 hexagrams) — wired here as the `ai/adapters/yijing_brain.py` director |
| [`info150`](https://github.com/svend4/info150) | its `portal` places domains on a 6-bit coordinate and bridges by Hamming distance — adapted here as `HexBridges` |
| `infon` (this repo) | hexagram **worlds**: a hexagram casts a 3-D world (`-hexagram`), a Gray walk tours all 64 (`-q6walk`), worlds bridge by Q6 proximity |

## Interchange format 1 — a hexagram (6 bits)

A hexagram is a number `0..63` (bit 0 = bottom line), exchanged as a **six-character
string of `1`/`0`, bottom line first** (also accepts `y`/`n`):

```
"101010"   # canonical (Hexagram.String)
"yynnyn"   # accepted alias
```

- `infon`: `raydir.ParseHexagram(s)` / `Hexagram.String()` round-trip all 64
  (tested); `rayexplore -hexagram 101010` casts that world.
- A reading produced by `meta`/`pro2`/`info150` in this form drops straight in — no
  shared code, just the six bits.

## Interchange format 2 — the director protocol (`tvcp-ai/1`)

Any brain that speaks `tvcp-ai/1` (`POST /v1/decide`, see
[`TVCP_AI_PROTOCOL.md`](TVCP_AI_PROTOCOL.md)) can direct the world; its `rayscene`
output is validated against [`../ai/schema/rayscene.schema.json`](../ai/schema/rayscene.schema.json).
`pro2` plugs in exactly this way via `ai/adapters/yijing_brain.py`:

```
python3 ai/adapters/yijing_brain.py 8095
BRAIN_URL=http://127.0.0.1:8095/v1/decide go run ./cmd/rayexplore -hexagram 011010
```

## The principle

Keep the engines apart; let them meet at the data. `infon` is the Go renderer and
shared world; `meta` is the maths; `pro2` is the model; `info150` is the applied
systems layer. The only things that cross the boundary are **six bits** and a small
**JSON protocol** — which is exactly why each can evolve on its own.

## The manifest — `nautilus.json`

This interop surface is declared, machine-readable, at the repo root in
[`nautilus.json`](../nautilus.json): infon's id and role on Q6, the protocols and
schema it speaks, the brain adapters it ships, and its links to the sibling nodes.
A test (`nautilus_test.go`) checks every file the manifest points at actually
exists, so the declaration can't drift from the repo. (Modeled on this doc; the
fields are infon's own — reconcile with the wider federation schema as it settles.)
