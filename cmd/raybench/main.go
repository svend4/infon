// Command raybench evaluates a rayscene director (the reference brain, or a live
// model via -url): it asks for many scenes and reports how renderable, rich and
// varied they are — a quick objective read before a live session.
//
//	go run ./cmd/raybench                                  # the reference author
//	go run ./cmd/raybench -url http://127.0.0.1:8090/v1/decide   # a live model
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
)

func main() {
	url := flag.String("url", "", "tvcp-ai/1 endpoint; empty = in-process reference brain")
	flag.Parse()

	var b brain.Brain = brain.Local{}
	who := "reference (offline)"
	if *url != "" {
		b = brain.HTTPBrain{URL: *url, HTTP: &http.Client{Timeout: 60 * time.Second}}
		who = *url
	}
	prompts := []string{
		"a calm world", "a gold sphere", "a scene at night", "glass and metal",
		"a quiet forest with a house", "a crystal cave", "a field of boulders",
		"a marble statue", "a surreal dream of a mandelbulb", "a calm lake at the shore",
		"a flock of birds over a meadow", "a pulsing beacon at dusk",
	}
	r := raydir.BenchDirector(b, prompts)

	fmt.Printf("director: %s\n", who)
	fmt.Printf("  prompts:      %d\n", r.N)
	fmt.Printf("  renderable:   %d/%d (%.0f%%)\n", r.Renderable, r.N, 100*float64(r.Renderable)/float64(max1(r.N)))
	fmt.Printf("  avg objects:  %.1f\n", r.AvgObjects())
	fmt.Printf("  variety:      %d kinds\n", r.Variety())
	fmt.Printf("  transport errs: %d\n", r.Errors)
	kinds := make([]string, 0, len(r.Kinds))
	for k := range r.Kinds {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool { return r.Kinds[kinds[i]] > r.Kinds[kinds[j]] })
	for _, k := range kinds {
		fmt.Printf("    %-10s %d\n", k, r.Kinds[k])
	}
	if r.Renderable < r.N { // a healthy director renders every prompt
		os.Exit(1)
	}
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
