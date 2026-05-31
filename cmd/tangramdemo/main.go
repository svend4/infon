// Command tangramdemo proves the tangram idea end to end: ONE small set of
// sub-cell pieces (◤◥◣◢ ▀▄▌▐ ▞▚ █) composes into many figures, every figure is
// checked against the tangram rule (pieces meet edge to edge, never stacking),
// procedural creatures are grown from a seed, and one figure MORPHS into another
// as a stream of semantic P-frames (cat -> swan), saved as a GIF.
//
//	go run ./cmd/tangramdemo [outdir]
package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"github.com/svend4/infon/pkg/semframe"
	"github.com/svend4/infon/pkg/tangram"
)

const scale = 22 // pixels per cell

func main() {
	out := "./_tangram_go"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fail(err)
	}

	// 1) Catalog: render each figure and print its tiling report.
	cat := tangram.Catalog()
	names := make([]string, 0, len(cat))
	for n := range cat {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Println("catalog (one piece vocabulary, many figures):")
	for _, n := range names {
		f := cat[n]
		rep := f.Validate()
		savePNG(filepath.Join(out, "fig_"+n+".png"), f.Image(f.W*scale, f.H*scale))
		fmt.Printf("  %-6s %d×%d  filled=%2d complete=%2d coverage=%4.0f%%  well-formed=%v\n",
			n, f.W, f.H, rep.FilledCells, rep.CompleteCells, rep.Coverage*100, rep.WellFormed)
		printGlyphMap(f)
	}

	// 2) Procedural bestiary: a few creatures from seeds, each validated.
	fmt.Println("\nprocedural creatures (deterministic from seed):")
	for _, seed := range []int64{1, 2, 3, 7, 11} {
		c := tangram.Creature(seed)
		rep := c.Validate()
		savePNG(filepath.Join(out, fmt.Sprintf("creature_%d.png", seed)), c.Image(c.W*scale, c.H*scale))
		fmt.Printf("  seed %2d -> %d×%d  pieces=%2d  well-formed=%v\n", seed, c.W, c.H, len(c.Pieces), rep.WellFormed)
	}

	// 3) Morph: cat -> swan as a dissolve driven by semframe P-frames.
	from, to := tangram.Cat(), tangram.Swan()
	base := from.Marks("")
	delta, ok := semframe.DiffMarks(base, to.Marks(""))
	if !ok {
		fail(fmt.Errorf("cat and swan do not share a grid shape"))
	}
	W, H := from.W*scale, from.H*scale
	steps := 18
	var frames []image.Image
	hold := func(img image.Image, n int) {
		for i := 0; i < n; i++ {
			frames = append(frames, img)
		}
	}
	for s := 0; s <= steps; s++ {
		k := s * len(delta.Cells) / steps
		partial := semframe.MarksDelta{Cells: append([]semframe.MarkCellChange(nil), delta.Cells[:k]...)}
		spec := semframe.ApplyMarks(base, partial)
		img, err := spec.Image(W, H)
		if err != nil {
			fail(err)
		}
		if s == 0 || s == steps {
			hold(img, 8) // pause on the cat and the swan
		} else {
			frames = append(frames, img)
		}
	}
	saveGIF(filepath.Join(out, "morph_cat_to_swan.gif"), frames)
	fmt.Printf("\nmorph cat -> swan: %d changed cells over %d frames -> morph_cat_to_swan.gif\n", len(delta.Cells), len(frames))
	fmt.Printf("done: %s\n", out)
}

// printGlyphMap dumps the figure's glyph grid (no colour) — a quick textual proof.
func printGlyphMap(f tangram.Figure) {
	rows := f.Marks("").Marks.Rows
	for _, r := range rows {
		line := "    "
		blank := true
		for _, g := range r {
			if g == "" {
				line += " "
			} else {
				line += g
				blank = false
			}
		}
		if !blank {
			fmt.Println(line)
		}
	}
}

func savePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fail(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		fail(err)
	}
}

func saveGIF(path string, frames []image.Image) {
	g := &gif.GIF{}
	for _, fr := range frames {
		g.Image = append(g.Image, toPaletted(fr))
		g.Delay = append(g.Delay, 8)
	}
	f, err := os.Create(path)
	if err != nil {
		fail(err)
	}
	defer func() { _ = f.Close() }()
	if err := gif.EncodeAll(f, g); err != nil {
		fail(err)
	}
}

func toPaletted(img image.Image) *image.Paletted {
	p := image.NewPaletted(img.Bounds(), palette.Plan9)
	draw.Draw(p, p.Bounds(), img, img.Bounds().Min, draw.Src)
	return p
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "tangramdemo:", err)
	os.Exit(1)
}
