# Pseudo-3D from a tangram, three ways (and their synthesis)

A tangram is flat, like an unfolded sheet of origami. These three paths lift the
same flat substrate into pseudo-3D - software-rendered, low-bandwidth, semantic
(the scene is described by a few bytes, the pixels are derived). All three ship
over the same Reed-Solomon byte channel a semantic video call uses, so an
animated 3-D view survives lost/corrupted bytes.

## The three paths (built separately)

| Path | Package | Idea | Demo | Animation budget |
|------|---------|------|------|------------------|
| 1. Origami body | `pkg/fold` | Extrude each piece to a prism, or **fold** it up about a hinge by a dihedral angle. The flat dissection (the net) becomes a standing relief (the body). | `cmd/folddemo`, `cmd/foldcall` | 16-frame unfold = **123 bytes** (255 with RS) |
| 2. 2.5D relief | `pkg/relief` | Add one **height byte per cell** to a marks grid; extrude each glyph. The diagonal glyphs become slanted facets that catch the light - a voxel / Q*bert view. | `cmd/reliefdemo` | height-spec = **72 bytes** (8x9) |
| 3. Scene graph | `pkg/scene3d` | A `Scene` is placed pieces: each `Item` is `{piece, x, y, z, yaw (45deg), colour}` - **6 bytes**. A real game state. | `cmd/scenecall` | 20-frame orbit = **995 bytes** (~50 b/frame) |

## The intersections (used together)

Built apart, the three paths turned out to share three seams:

1. **One renderer.** `fold.Face` (a colour polygon in 3-D with a light multiplier)
   and `fold.Render` (2:1 isometric projection, painter-sorted, top/side shading)
   render all three. `relief` and `scene3d` both import `fold` and emit
   `[]fold.Face`. One iso engine, three sources.

2. **One stream.** Path 1 (`FoldAnim`) and path 3 (`SceneAnim`) independently
   invented the same codec: serialise frame 0 in full, then for each later frame
   only the bytes that changed, and protect the whole stream with interleaved
   Reed-Solomon. `pkg/deltastream` is that pattern extracted generically - a
   sequence of fixed-width byte frames; both animations are instances of it.

3. **One philosophy.** In every path the *spec is the data* (a height byte, an
   angle byte, a 6-byte item) and the pixels are derived; the whole animated
   scene is generable by a model and ships in the same few-hundred-bytes-over-loss
   budget as the rest of the project.

## The combined engine

`cmd/pseudo3d` proves they compose. One **50-byte-per-frame** record packs all
three - 7 fold angles (a folding body), 1 relief height (a rising voxel animal)
and 7 scene items (orbiting slabs) - ships as ONE `deltastream` over RS, is
corrupted by a 42-byte burst, recovered exactly, and rendered by the SINGLE
`fold.Render` into one isometric world.

```
frame record = 50 bytes (7 fold + 1 relief + 42 scene)
14 frames = 1322 bytes raw / 1530 bytes RS
42-byte burst -> recovered exact -> one rendered world
```

## Where this can go

- **A real game.** The combined record is a game state; a neural brain emits the
  next state, the deltastream ships it, the receiver renders. Fold bodies as
  actors, a relief grid as terrain, scene items as props - one world, one stream.
- **The origami bridge.** Path 2 showed the figure's information lives in the
  diagonal glyphs (the hard-mode benchmark). Those same diagonals are the lines a
  sheet would crease along - so the glyph alphabet is one step from a fold/crease
  pattern, tying path 2 (facets) directly to path 1 (folds).
- **Camera and depth.** `fold.Render` is fixed-iso today; one more field (camera
  yaw) in the record turns the whole thing rotatable without touching the codec.
