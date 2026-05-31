// Command tangrambench is a spatial-reasoning benchmark for the tvcp-ai protocol.
// For each catalog figure it builds a tangram puzzle ("assemble this silhouette
// from the fixed pieces"), asks a brain to solve it via game "tangram", and
// scores the returned placements by intersection-over-union against the target,
// alongside a naive solid-fill baseline. With no -brain it uses the built-in
// reference solver; point -brain at any tvcp-ai endpoint to benchmark a model.
//
//	go run ./cmd/tangrambench [-brain URL] [-out dir]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/tangram"
)

const scale = 22

func main() {
	brainURL := flag.String("brain", "", "tvcp-ai endpoint (default: built-in reference solver)")
	out := flag.String("out", "./_tangram_bench", "output directory for renders")
	flag.Parse()

	decide := func(r brain.Request) (brain.Response, error) { return brain.Reference(r), nil }
	if *brainURL != "" {
		hb := brain.HTTPBrain{URL: *brainURL}
		decide = hb.Decide
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}

	cat := tangram.Catalog()
	names := make([]string, 0, len(cat))
	for n := range cat {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Printf("game:tangram benchmark  (brain: %s)\n", brainOrRef(*brainURL))
	fmt.Printf("%-7s %-8s %6s %7s %9s  %s\n", "figure", "solver", "IoU", "recall", "overflow", "well-formed")
	var sumBrain, sumNaive float64
	for _, n := range names {
		p := tangram.PuzzleFromFigure(cat[n])
		state, _ := json.Marshal(p)
		resp, err := decide(brain.Request{Kind: "move", Game: "tangram", State: state})
		if err != nil {
			fail(err)
		}
		var sol tangram.Figure
		if len(resp.Tangram) == 0 || json.Unmarshal(resp.Tangram, &sol) != nil {
			fail(fmt.Errorf("%s: brain returned no tangram solution", n))
		}
		bs := tangram.ScorePuzzle(p, sol)
		ns := tangram.ScorePuzzle(p, tangram.NaiveSolve(p))
		sumBrain += bs.IoU
		sumNaive += ns.IoU
		fmt.Printf("%-7s %-8s %6.3f %7.3f %9.3f  %v\n", n, "brain", bs.IoU, bs.Recall, bs.Overflow, bs.WellFormed)
		fmt.Printf("%-7s %-8s %6.3f %7.3f %9.3f  %v\n", "", "naive", ns.IoU, ns.Recall, ns.Overflow, ns.WellFormed)
	}
	fmt.Printf("\nmean IoU  brain=%.3f  naive=%.3f  (over %d figures)\n", sumBrain/float64(len(names)), sumNaive/float64(len(names)), len(names))

	// Visual: target cat vs brain solution vs naive solution.
	p := tangram.PuzzleFromFigure(tangram.Cat())
	st, _ := json.Marshal(p)
	resp, _ := decide(brain.Request{Kind: "move", Game: "tangram", State: st})
	var sol tangram.Figure
	_ = json.Unmarshal(resp.Tangram, &sol)
	savePNG(filepath.Join(*out, "cat_target.png"), tangram.Cat().Image(tangram.Cat().W*scale, tangram.Cat().H*scale))
	savePNG(filepath.Join(*out, "cat_brain.png"), sol.Image(sol.W*scale, sol.H*scale))
	naive := tangram.NaiveSolve(p)
	savePNG(filepath.Join(*out, "cat_naive.png"), naive.Image(naive.W*scale, naive.H*scale))
	fmt.Printf("renders: %s (cat_target / cat_brain / cat_naive)\n", *out)
}

func brainOrRef(u string) string {
	if u == "" {
		return "built-in reference"
	}
	return u
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

func fail(err error) {
	fmt.Fprintln(os.Stderr, "tangrambench:", err)
	os.Exit(1)
}
