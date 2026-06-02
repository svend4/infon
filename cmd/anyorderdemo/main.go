// Command anyorderdemo shows the MDLM "any-order with anchors" idea on the
// tangram address book: to identify a figure, reveal the most informative cell
// next (any-order) instead of scanning left-to-right. It prints, per figure, how
// many anchors each strategy needs, and the totals.
//
//	go run ./cmd/anyorderdemo
package main

import (
	"fmt"

	"github.com/svend4/infon/pkg/anyorder"
)

func main() {
	hyps, W, H := anyorder.Hypotheses()
	n := W * H
	fmt.Printf("hypotheses: %d figures on a %dx%d grid (%d cells)\n", len(hyps), W, H, n)
	fmt.Printf("%-10s %10s %10s\n", "figure", "any-order", "left2right")
	sumA, sumL := 0, 0
	for _, h := range hyps {
		a := anyorder.Reveals(h, hyps, n, "anyorder")
		l := anyorder.Reveals(h, hyps, n, "l2r")
		sumA += a
		sumL += l
		mark := ""
		if a < l {
			mark = "  <-"
		}
		fmt.Printf("%-10s %10d %10d%s\n", h.Name, a, l, mark)
	}
	fmt.Printf("%-10s %10.2f %10.2f\n", "AVG", float64(sumA)/float64(len(hyps)), float64(sumL)/float64(len(hyps)))
	fmt.Printf("anchors to identify a figure: any-order needs %.0f%% of the left-to-right total\n",
		100*float64(sumA)/float64(sumL))
}
