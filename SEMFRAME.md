# Semantic P-frames — inter-frame compression at the level of meaning

Diffusion and pixel codecs send frames; `pkg/semframe` sends **what changed in the
spec**. A keyframe is a full pseudo-image; every following frame is a tiny delta —
only the grid cells (and palette) that differ. When little moves, a "video" frame
costs tens of bytes, *independent of frame size*.

```go
delta, ok := semframe.Diff(prev, next)   // ok=false -> send a keyframe instead
next2     := semframe.Apply(prev, delta)  // exact reconstruction
```

```text
$ go run ./cmd/semdemo          # a sun drifting across a sky grid
keyframe: 855 bytes (full spec)
frame sun@3   P-frame 161 B   (full would be 855 B)   6 cells changed
...
total: 1458 B as P-frames vs 7695 B as full frames  (5.3x smaller)
reconstruction exact: true
```

The sun moving touches ~6 cells (~160 B) **no matter how big the frame is**, while
a full frame grows with size — so on a 40x24 scene the ratio is 20-50x. This is
the project's Tier-2 idea ("semantic codecs"): the natural medium for bandwidth-
poor and high-latency links (deep space / DTN, mesh, satellite) and for "visual
chat" that streams meaning instead of data.

Current prototype covers `grid`/`pixels`. Element-level deltas for `sigils`/`mixed`
("sigil 0 moved", "added a cloud") and an over-the-wire framing (keyframe interval,
NACK on loss) are the natural next steps.
