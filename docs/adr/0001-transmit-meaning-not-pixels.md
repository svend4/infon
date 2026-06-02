# ADR 0001 — Transmit meaning, not pixels

- **Status:** Accepted
- **Date:** 2026-06
- **Context tags:** tvcp-ai, protocol, scope

## Context

The project began as a terminal video-call platform: capture a webcam, encode it,
ship frames to a peer, decode and display. That path is bandwidth-bound, depends
on platform media stacks (WASAPI/CoreAudio/ALSA, AVFoundation/Media Foundation,
Opus, NAT traversal, GPU encode), and produces output that only a human can
consume — a wall of pixels with no structure a program can reason about.

Separately, the repository accumulated a symbolic substrate: glyph sets, a
pseudo-image spec, tangram figures, a draw DSL, mark-diff video. These let a
*decision* be expressed as a few hundred bytes of structured JSON that a receiver
renders locally. The question was whether to treat this substrate as a sideshow
to the video call, or as the actual product.

## Decision

We treat the symbolic layer as the product and name it **tvcp-ai**: an open,
transport-agnostic JSON contract in which participants exchange **meaning, not
pixels**. A participant sends intent — a move, a scene, a symbolic image spec, a
reaction — and the receiver renders it. The canonical contract is a `Request`/
`Response` pair over HTTP `POST /v1/decide`, with capabilities advertised at
`GET /v1/capabilities` (see `docs/TVCP_AI_PROTOCOL.md`).

Three corollaries follow:

1. **A brain is a URL.** Any backend that answers the contract — this process, a
   local model, a cloud model, a script — is interchangeable. Swapping partners is
   a config change, not a code change.
2. **The validator is the trust boundary.** `brain.CheckResponse` and the
   `ConformanceBattery` define correctness independently of any one brain, so a
   third party can claim conformance and be checked.
3. **Real-time media is out of scope for this protocol.** Microphone/camera
   capture, Opus, NAT traversal, mobile clients and GPU encode belong to the base
   platform. tvcp-ai never carries a media buffer.

## Consequences

**Positive.** Bytes-on-the-wire drop by orders of magnitude; the same message
renders to a terminal image, audio (`pkg/sona`) or braille (`pkg/braille`);
agents become composable across a registry (`pkg/registry`), discoverable
(`pkg/discovery`), verifiable (`pkg/sign`), and able to negotiate work
(`pkg/acl`, Contract Net); and the contract is testable in isolation.

**Negative / costs.** Fidelity is bounded by the receiver's renderer and the
vocabulary of the spec, not by the sender. Meaning that has no symbol cannot be
sent until the vocabulary is extended (`pkg/vocab`). And there is a standing risk
of building many disconnected demos rather than one coherent application — tracked
deliberately, and answered by unifying scenarios such as `cmd/watch`.

## Alternatives considered

- **Keep the video call as the product, symbols as a feature.** Rejected: it
  re-centers the bandwidth/hardware problems this decision exists to escape.
- **A binary protocol instead of JSON.** Deferred: JSON keeps brains trivially
  implementable in any language; a binary profile can be added later without
  changing the message shapes.
- **One monolithic "do everything" endpoint.** Rejected in favor of `kind` +
  `game` dispatch with capability negotiation, so brains can support a subset and
  clients can route around gaps.
