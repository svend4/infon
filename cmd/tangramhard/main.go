// Command tangramhard benchmarks silhouette-only ("hard") tangram solving: given
// just the figure's outline (no per-cell glyphs), recover the tiling. It scores
// three solvers against every catalog figure - easy (glyphs given), hard
// (silhouette, nearest-glyph per cell) and coarse (cell occupancy -> solids).
package main

import (
	"fmt"
	"sort"

	"github.com/svend4/infon/pkg/tangram"
)

type acc struct{ iou, overflow, recall float64 }

func main() {
	cat := tangram.Catalog()
	names := make([]string, 0, len(cat))
	for k := range cat {
		names = append(names, k)
	}
	sort.Strings(names)

	var e, h, c acc
	fmt.Printf("%-10s  %6s  %6s  %6s   %s\n", "figure", "easy", "hard", "coarse", "coarse-overflow")
	for _, name := range names {
		f := cat[name]
		ps := tangram.PuzzleFromFigure(f)
		hp := tangram.HardFromFigure(f)
		easy := tangram.ScorePuzzle(ps, tangram.Solve(ps))
		hard := tangram.ScorePuzzle(ps, tangram.SolveHard(hp))
		coarse := tangram.ScorePuzzle(ps, tangram.SolveCoarse(hp))
		fmt.Printf("%-10s  %6.3f  %6.3f  %6.3f   %.2f\n", name, easy.IoU, hard.IoU, coarse.IoU, coarse.Overflow)
		e.iou += easy.IoU
		h.iou += hard.IoU
		h.overflow += hard.Overflow
		c.iou += coarse.IoU
		c.overflow += coarse.Overflow
	}
	n := float64(len(names))
	fmt.Printf("\nmean IoU   easy=%.3f  hard=%.3f  coarse=%.3f\n", e.iou/n, h.iou/n, c.iou/n)
	fmt.Printf("mean overflow   hard=%.3f  coarse=%.3f\n", h.overflow/n, c.overflow/n)
	fmt.Printf("\nexample silhouette (swan, half-res) - this is all the hard solver sees:\n\n%s", cat["swan"].SilhouetteString(2))
}
