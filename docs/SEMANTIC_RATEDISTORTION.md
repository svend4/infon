# Semantic rate–distortion — a reproducible benchmark

Classical rate–distortion trades bits against *pixel* error (PSNR/SSIM). TVCP asks
a different question: **how much MEANING survives a given bitrate or channel?**
`pkg/semdist` measures it — distortion is `1 − MarksIoU` (sub-cell coverage
intersection-over-union of the reconstructed vs. sent symbolic frame), so 1.000
means "all meaning preserved", 0.000 "none".

Reproduce: `go run ./cmd/semdistdemo` (deterministic; chart at `_bench/semdist.png`).

## A. Bitrate vs meaning, under 25% packet loss

The keyframe interval (`gop`) is the knob: smaller gop spends more bytes on
keyframes but resynchronises faster after a dropped packet, so more meaning
survives. A 24-frame marks scene, `ecc=6`:

| gop | wire bytes | meaning (MarksIoU) |
|----:|-----------:|-------------------:|
|   1 |     18 432 |              0.682 |
|   2 |     12 312 |              0.548 |
|   4 |      9 252 |              0.382 |
|   8 |      7 722 |              0.236 |
|  16 |      7 212 |              0.236 |

Reading: halving the bitrate (gop 1 → 8) keeps ~⅓ of the meaning under heavy loss;
keyframe-dense streams pay for resilience. This is the operating curve a deep-space
or mesh deployment would tune on (see also `cmd/dtncast`).

## B. Parity vs corruption, at ~constant frame size

Reed–Solomon parity (`ecc`) does not change the frame size here — it trades
payload capacity for resilience. Under 3 corrupted bytes per packet:

| ecc | wire bytes | meaning (MarksIoU) |
|----:|-----------:|-------------------:|
|   2 |      9 252 |              0.074 |
|   4 |      9 252 |              0.305 |
|   8 |      9 252 |              1.000 |
|  16 |      9 252 |              1.000 |

Reading: once `ecc/2 ≥` the per-packet corruption (ecc 8 ⇒ corrects 4 ≥ 3), meaning
is fully recovered at the same bitrate.

## Why this matters

"Fidelity of meaning" is a rate–distortion axis the industry barely measures.
Because the substrate is symbolic and verifiable (a marks frame has an exact
coverage), the distortion metric is principled and reproducible — not a perceptual
proxy. The same harness scores any codec/channel that emits `pseudo` marks specs.

## Related

- `pkg/semvideo` — the keyframe + semantic P-frame codec over Reed–Solomon.
- `pkg/dtn` + `cmd/dtncast` — the delay-tolerant (deep-space) channel profile.
- `pkg/semdist` + `cmd/semdistdemo` — this benchmark.
