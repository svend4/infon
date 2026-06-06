# Governance — keeping tvcp-ai a commons

`tvcp-ai/1` is meant to be infrastructure, not a product: **the protocol is the
commons; a model is a swappable part.** This document states how it stays that
way, so the open format is not quietly privatised.

## Principles

1. **Open by default.** The wire format, the reference brain, and the conformance
   suite are public and permissively licensed. Anyone may implement a brain or a
   client without permission.
2. **Rough consensus + running code.** Changes land as a reference
   implementation plus tests, not as a spec edict. A change that no one can run
   is not a change.
3. **No engine lock-in.** "Bring your own brain": a model is reached by URL and
   negotiated by capability (see `pkg/registry`). The protocol must never require
   a specific vendor, model, or endpoint.
4. **Plural vocabulary.** The symbol/palette/sigil vocabulary encodes a worldview
   ("what is sky, what is home"). It must be extensible and localizable; no single
   cultural default is privileged. New vocabularies are additive, namespaced, and
   opt-in.
5. **Accessibility is a feature, not a footnote.** Because a pseudo-image is its
   own description, every visual is also its alt-text. Changes must preserve this
   isomorphism.

## Versioning & compatibility

- The version string (`tvcp-ai/1`) is carried on every message and authenticated
  (`pkg/sign`). Breaking changes bump the major version; receivers reject versions
  they do not implement rather than guessing.
- Reference implementations and conformance cases define behaviour; ambiguity is
  resolved by what the reference does.

## Decision process

- Proposals are issues/PRs with a reference implementation and tests.
- Anyone may propose; maintainers merge by rough consensus.
- Security-relevant changes follow `SECURITY.md`.

## Stewardship

The goal is a long-lived commons usable on salvaged hardware, low bandwidth, and
across cultures. Stewardship decisions are judged against that goal, not against
any one deployment's convenience.
