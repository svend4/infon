// Package anyorder is the MDLM idea on the project's verifiable substrate: a
// figure is reconstructed by revealing ANCHOR cells in information order (the
// most discriminating cell next) rather than left-to-right. Over the tangram
// address book as the hypothesis set, any-order pins down the figure with fewer
// anchors than a raster scan — "think in any order, anchored", but provable
// because the grid is a real catalog with distinct ids.
package anyorder

import (
	"math/big"
	"sort"

	"github.com/svend4/infon/pkg/tangram"
)

const radix = 16 // tangram.AlphabetSize

// Hyp is a candidate figure as a row-major digit grid.
type Hyp struct {
	Name  string
	Cells []int
}

func figDigits(f tangram.Figure) []int {
	total := f.H * f.W
	ds := make([]int, total)
	base := big.NewInt(radix)
	q := new(big.Int).Set(f.Number())
	r := new(big.Int)
	for i := total - 1; i >= 0; i-- {
		q.QuoRem(q, base, r)
		ds[i] = int(r.Int64())
	}
	return ds
}

// Hypotheses returns the catalog figures that share the modal (most common) grid
// size — a clean same-shape hypothesis set with distinct ids — and that size.
func Hypotheses() (hyps []Hyp, W, H int) {
	cat := tangram.Catalog()
	type dim struct{ w, h int }
	count := map[dim]int{}
	for _, f := range cat {
		count[dim{f.W, f.H}]++
	}
	best, bestN := dim{}, -1
	for d, c := range count {
		if c > bestN || (c == bestN && (d.w*d.h) > best.w*best.h) {
			bestN, best = c, d
		}
	}
	W, H = best.w, best.h
	names := make([]string, 0, len(cat))
	for n, f := range cat {
		if f.W == W && f.H == H {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		hyps = append(hyps, Hyp{Name: n, Cells: figDigits(cat[n])})
	}
	return hyps, W, H
}

func consistent(h Hyp, revealed map[int]int) bool {
	for idx, v := range revealed {
		if idx < 0 || idx >= len(h.Cells) || h.Cells[idx] != v {
			return false
		}
	}
	return true
}

func candidates(hyps []Hyp, revealed map[int]int) []Hyp {
	var out []Hyp
	for _, h := range hyps {
		if consistent(h, revealed) {
			out = append(out, h)
		}
	}
	return out
}

// nextAnyOrder picks the unrevealed cell whose digit varies most across the
// current candidates (the most informative anchor); ties go to the lowest index.
func nextAnyOrder(cands []Hyp, revealed map[int]int, n int) int {
	best, bestScore := -1, -1
	for c := 0; c < n; c++ {
		if _, ok := revealed[c]; ok {
			continue
		}
		seen := map[int]bool{}
		for _, h := range cands {
			seen[h.Cells[c]] = true
		}
		if len(seen) > bestScore {
			bestScore, best = len(seen), c
		}
	}
	return best
}

func nextL2R(revealed map[int]int, n int) int {
	for i := 0; i < n; i++ {
		if _, ok := revealed[i]; !ok {
			return i
		}
	}
	return -1
}

// Reveals counts the anchors a strategy ("anyorder" or "l2r") needs to uniquely
// identify target among hyps.
func Reveals(target Hyp, hyps []Hyp, n int, strategy string) int {
	revealed := map[int]int{}
	for step := 0; step <= n; step++ {
		if len(candidates(hyps, revealed)) <= 1 {
			return len(revealed)
		}
		var c int
		if strategy == "anyorder" {
			c = nextAnyOrder(candidates(hyps, revealed), revealed, n)
		} else {
			c = nextL2R(revealed, n)
		}
		if c < 0 {
			break
		}
		revealed[c] = target.Cells[c]
	}
	return len(revealed)
}

// Identify reveals anchors in information order and returns the reconstructed
// hypothesis plus the anchors used.
func Identify(target Hyp, hyps []Hyp, n int) (Hyp, map[int]int) {
	revealed := map[int]int{}
	for step := 0; step <= n; step++ {
		cands := candidates(hyps, revealed)
		if len(cands) <= 1 {
			if len(cands) == 1 {
				return cands[0], revealed
			}
			break
		}
		c := nextAnyOrder(cands, revealed, n)
		if c < 0 {
			break
		}
		revealed[c] = target.Cells[c]
	}
	c := candidates(hyps, revealed)
	if len(c) >= 1 {
		return c[0], revealed
	}
	return Hyp{}, revealed
}
