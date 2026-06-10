// Command raygallery browses a directory of saved worlds (.rwld) and recordings
// (.rrec), printing a summary of each and how to open it — a little gallery of
// shareable worlds and walks, each a few KB of meaning (not pixels).
//
//	go run ./cmd/raygallery worlds/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/svend4/infon/pkg/raydir"
)

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read dir:", err)
		os.Exit(1)
	}
	worlds, recs := 0, 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".rwld":
			if w, err := raydir.SummarizeWorld(path); err == nil {
				worlds++
				places := strings.Join(w.Places, ", ")
				if len(places) > 60 {
					places = places[:57] + "..."
				}
				fmt.Printf("🌍 %-24s %2d regions  %5d B  [%s]\n", e.Name(), w.Regions, w.Bytes, places)
				fmt.Printf("    open: go run ./cmd/raymeet -host -world %s\n", path)
			}
		case ".rrec":
			if r, err := raydir.SummarizeRecording(path); err == nil {
				recs++
				fmt.Printf("🎞  %-24s %2d events  %5.1fs  %5d B\n", e.Name(), r.Events, float64(r.DurationMs)/1000, r.Bytes)
				fmt.Printf("    open: go run ./cmd/rayplay %s\n", path)
			}
		}
	}
	fmt.Printf("\n%d world(s), %d recording(s) in %s\n", worlds, recs, dir)
}
