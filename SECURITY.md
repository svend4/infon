# Security & threat model — tvcp-ai/1

The protocol connects untrusted "brains" over a network and renders what they
return. That surface deserves an explicit threat model. This is a starting
model, not a guarantee.

## Assets
- The integrity of messages between client and brain (no tampering, no replay of
  the wrong version).
- The renderer and host: a malicious brain must not be able to exhaust resources
  or escape the rendering sandbox.
- The user's attention and trust (no silent impersonation of a brain).

## Threats & mitigations

| Threat | Mitigation |
|---|---|
| **Tampered / forged messages** on the wire | `pkg/sign`: HMAC-SHA256 stamp over (version ‖ payload) under a shared key; receivers `Verify` before acting. |
| **Version confusion** (a v1 client acting on a v2 message) | The version is signed and checked; mismatches are rejected, not coerced. |
| **Oversized / malformed render payloads** (a brain returns a 10000×10000 canvas, ragged grids, bad JSON) | Clients clamp canvas size (`pseudo` min/max cols/rows), tolerate ragged grids, and reject unknown formats; the draw renderer hard-clamps dimensions. |
| **Unbounded compute** from adversarial specs | Deterministic, bounded renderers (no diffusion, no scripting); fixed alphabets; size clamps. |
| **Impersonation of a brain** in discovery | `pkg/registry` entries can be signed; clients should pin or verify keys for sensitive partners. |
| **Untrusted code execution** | There is none by design: a brain returns DATA (a spec / a move), never code; the renderer interprets a fixed, total grammar. |
| **Key compromise** | Keys are read from the environment or a file outside the repo and never logged or committed (`.gitignore`); rotate by re-keying both ends. |

## Non-goals / known gaps
- No confidentiality on the wire yet (sign provides integrity, not encryption);
  run over TLS or a secure channel for secrecy.
- No rate limiting or authn/z framework in the reference transport; deployments
  must add their own for public endpoints.
- Replay protection beyond version checking (nonces/timestamps) is not yet
  specified.

## Reporting
Report vulnerabilities privately to the maintainers before public disclosure.
