# glyphset — a diagonal/triangle sub-cell codec (from a hand-drawn alphabet)

A paper sheet enumerating sub-cell marks (dots → diagonals → triangles → the
crossed box `⊠`) became `pkg/glyphset`: a richer terminal-graphics primitive than
the 2×2 quadrant blocks the `babe` codec uses.

- **RenderTriangles(img, cols, rows)** maps an image to a `terminal.Frame`,
  choosing for each cell the glyph whose fill mask best matches the bright region.
  Slanted edges (mountain slopes, sun discs, sails, heraldic charges) come out as
  crisp diagonals (`◤◥◣◢`) instead of the quadrant codec's stair-steps.
- **Marks** is the digitized catalog of the sheet (blanks, halves, quadrants incl.
  the diagonal pair `▞▚`, triangles, diagonal lines `╱╲╳`, box `□⊠`); **Chart**
  lays it out as the clean, code-backed version of the drawing.
- **Rasterize(frame, cell)** paints each cell's glyph MASK (not just its color),
  so triangle/diagonal shapes are reproduced faithfully in PNG previews.

```bash
go run ./cmd/glyphdemo ./_glyph   # a_quadrant.png vs b_triangles.png + c_alphabet.png
```

`coverage()` is the single source of truth for every glyph's S×S mask — it drives
both the best-match renderer and the faithful rasterizer.

This is a new render primitive: a candidate `tvcp` render mode beside quadrant /
sextant / braille, strongest exactly where the others are weakest — diagonal and
curved edges.

## The myFront4Solan4 font (© svend)

Rasterize's **primary text font** is the user's own hand-designed bitmap font
`myFront4Solan4`: 8×8 geometric/diagonal letters and digits, plus 16×16 pattern
tiles — the grid-paper alphabet formalized. It is generated into
`pkg/glyphset/solan.go` from the JSON source `ai/fonts/myFront4Solan4.json`; the
5×7 micro-font covers any glyph it doesn't define. `cmd/glyphdemo` writes a
specimen `f_solan.png`. So captions and labels across the project now render in
the author's own typeface — no system font required.
