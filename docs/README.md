# Documentation

Technical documentation. The index below lists what actually lives in `docs/`.

## Start here

- **[CONCEPT_MAP.md](CONCEPT_MAP.md)** — the **current development stage**: a
  conceptual map of what the ray/Q6/alife platform can do and how the capabilities
  compose and get reused (the A–L feature series + `raypipe`). Read this first to
  see where the project is now.

## The shared world & the meaning bridge

- [SHARED_WORLD.md](SHARED_WORLD.md) — walking an AI-authored 3-D world together in
  the terminal (group mode, voice, driving it with a real model).
- [adr/0001-transmit-meaning-not-pixels.md](adr/0001-transmit-meaning-not-pixels.md)
  — the founding decision: send meaning, not pixels.
- [SEMANTIC_RATEDISTORTION.md](SEMANTIC_RATEDISTORTION.md) — rate–distortion of
  semantic (scene-spec) transport vs pixels.

## AI / brains (tvcp-ai/1)

- [TVCP_AI_PROTOCOL.md](TVCP_AI_PROTOCOL.md) — the `tvcp-ai/1` director protocol
  (`POST /v1/decide`).
- [EXTERNAL_MODELS.md](EXTERNAL_MODELS.md) — wiring real models via HTTP sidecars or
  cloud APIs (the seams, env vars `BRAIN_URL`/`VISION_API_URL`/…, reference Python
  adapters).
- [Q6_INTEROP.md](Q6_INTEROP.md) — the Q6 hypercube as a shared coordinate across
  sibling repos: the three interchange formats (a 6-bit hexagram string, the
  continuous 6-float Q6 vector byte-identical in Go and Python, and the JSON
  director protocol).
- [AI_NEXT.md](AI_NEXT.md) — open directions for the AI layer.

## Graphics

- [NEURAL_GRAPHICS_ROADMAP.md](NEURAL_GRAPHICS_ROADMAP.md) — design doc +
  implementation-status table for graphics × neural networks.
- [PSEUDO3D.md](PSEUDO3D.md) — the pseudo-3D renderer notes.

## Networking

- [dns-peer-discovery.md](dns-peer-discovery.md) — DNS-based peer discovery
  (human-readable `user@domain` addressing via DNS TXT records).

## Related (repo root)

- [../README.md](../README.md) — project overview & quickstart.
- [../CHANGELOG.md](../CHANGELOG.md) — running history of notable changes.
- [../tvcp-business-plan.md](../tvcp-business-plan.md),
  [../tvcp-appendix.md](../tvcp-appendix.md) — the TVCP platform background.
- TVCP-core subsystem docs (audio, cameras, screen sharing, recording, codecs,
  Yggdrasil, …) live at the repo root as `*.md` and describe their own subsystems.
