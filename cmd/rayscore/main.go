// Command rayscore writes the world's adaptive soundtrack to a WAV — activating
// raydir.ScoreWAV, which shipped with tests but no command. The score follows the
// world's state the way -music does live (major and bright by day, minor and slow
// at night; mood bends tempo and scale), but here it is rendered to a file you can
// keep.
//
//	go run ./cmd/rayscore -mood calm -night 0.8 -out night.wav
//	go run ./cmd/rayscore -mood restless -night 0.1 -seconds 12 -out day.wav
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/svend4/infon/pkg/raydir"
)

func main() {
	var (
		out     = flag.String("out", "score.wav", "output WAV path")
		mood    = flag.String("mood", "calm", "world mood: calm | curious | restless")
		night   = flag.Float64("night", 0.7, "night-ness 0 (day) .. 1 (night)")
		lively  = flag.Float64("lively", 0.5, "liveliness 0 (still) .. 1 (busy)")
		seconds = flag.Float64("seconds", 8, "length in seconds")
		seed    = flag.Int64("seed", 1, "melody seed")
		rate    = flag.Int("rate", 44100, "sample rate")
	)
	flag.Parse()

	p := raydir.ScoreForNight(*night, *lively, *mood)
	wav := raydir.ScoreWAV(p, *rate, *seconds, *seed)
	if err := os.WriteFile(*out, wav, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("score: mood=%s night=%.2f lively=%.2f -> %d BPM, root %.1f Hz, bright %.2f\n",
		*mood, *night, *lively, p.BPM, p.Root, p.Bright)
	fmt.Printf("wrote %s (%.1fs, %d bytes)\n", *out, *seconds, len(wav))
}
