// Command tangramnum gives every tangram figure a unique NUMBER, the way the
// I Ching numbers its 64 hexagrams: read the figure's glyph grid row-major as a
// base-16 integer. It prints each catalog figure's number, its pose-invariant
// canonical id, a round-trip check (number -> figure -> number), and the trigram
// reading of the cat (each cell = a 3-bit I Ching line-stack).
//
//	go run ./cmd/tangramnum
package main

import (
	"fmt"
	"sort"

	"github.com/svend4/infon/pkg/tangram"
)

func short(s string) string {
	if len(s) <= 28 {
		return s
	}
	return s[:12] + "..." + s[len(s)-12:]
}

func main() {
	cat := tangram.Catalog()
	names := make([]string, 0, len(cat))
	for n := range cat {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Println("every figure -> one unique number (base-16 over the piece alphabet):")
	fmt.Printf("  %-6s %-5s %-7s %-30s %s\n", "figure", "cells", "trigram", "number", "canonical-id")
	for _, n := range names {
		f := cat[n]
		num := f.Number()
		can := f.CanonicalNumber()
		rt := tangram.FromNumber(num, f.H, f.W).Number().Cmp(num) == 0
		fmt.Printf("  %-6s %-5d %-7v %-30s %s  rt=%v\n",
			n, f.H*f.W, f.TrigramClean(), short(num.String()), short(can.String()), rt)
	}

	// A silhouette and its mirror share a canonical id.
	c := tangram.Cat()
	fmt.Printf("\ncat == mirror(cat) under canonical id: %v\n",
		c.CanonicalNumber().Cmp(c.MirrorX().CanonicalNumber()) == 0)

	// Trigram reading of the cat (digits 0..7 -> 3 bits per cell).
	fmt.Println("\ncat as I Ching trigrams (3 bits per cell, '.'=blank rows hidden):")
	g := tangram.Cat().Marks("").Marks.Rows
	_ = g
	for _, row := range catDigits() {
		any := false
		for _, d := range row {
			if d != 0 {
				any = true
			}
		}
		if !any {
			continue
		}
		line := "   "
		for _, d := range row {
			line += fmt.Sprintf(" %03b", d)
		}
		fmt.Println(line)
	}
}

// catDigits mirrors the demo grid used in chat (shape only).
func catDigits() [][]int {
	f := tangram.Cat()
	num := f.Number()
	rt := tangram.FromNumber(num, f.H, f.W)
	// recover digits from the reconstructed figure
	out := make([][]int, f.H)
	for y := 0; y < f.H; y++ {
		out[y] = make([]int, f.W)
	}
	for _, p := range rt.Pieces {
		out[p.Y][p.X] = tangram.DigitForGlyph(p.Glyph)
	}
	return out
}
