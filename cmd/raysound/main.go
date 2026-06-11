// Command raysound gives the Q6 world a voice (Block J: sonification). It maps a
// world's six axes to music — the sun sets the register, warmth the major/minor
// colour, density the tempo, glow the timbre, fog a muffling, scale the octave — so
// every hexagram has its own piece. With -tour it renders the Gray grand tour as one
// evolving track through every world; with -life it sonifies a living world's
// population so you can HEAR the boom and the crash.
//
//	go run ./cmd/raysound -hexagram 110010 -out world.wav
//	go run ./cmd/raysound -tour -out tour.wav
//	go run ./cmd/raysound -life -out life.wav
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/svend4/infon/pkg/raydir"
)

func scaleName(s []int) string {
	switch len(s) {
	case 5:
		if s[1] == 2 {
			return "pentatonic major"
		}
		return "pentatonic minor"
	case 7:
		if s[2] == 4 {
			return "major"
		}
		return "minor"
	}
	return "scale"
}

func main() {
	var (
		out     = flag.String("out", "sound.wav", "output WAV path")
		hexS    = flag.String("hexagram", "110010", "hexagram world to voice")
		tour    = flag.Bool("tour", false, "render the Gray grand tour of all 64 worlds")
		life    = flag.Bool("life", false, "sonify a living world's population (boom/crash)")
		seconds = flag.Float64("seconds", 10, "length in seconds (world mode)")
		seed    = flag.Int64("seed", 1, "seed")
		rate    = flag.Int("rate", 44100, "sample rate")
		ticks   = flag.Int("ticks", 240, "ticks to simulate (life mode)")
	)
	flag.Parse()

	var wav []byte
	switch {
	case *tour:
		walk := raydir.GrayCode()
		secPer := *seconds / 64
		if secPer < 0.15 {
			secPer = 0.15
		}
		wav = raydir.TourWAV(walk, *rate, secPer, *seed)
		fmt.Printf("sound of the grand tour: 64 worlds, %.2fs each -> %.1fs total\n", secPer, secPer*64)
	case *life:
		clim := raydir.NewClimate(*seed)
		eco := raydir.NewEcosystem(12, 12, 28, clim, *seed)
		series := make([]float64, 0, *ticks+1)
		series = append(series, float64(eco.Population()))
		for t := 0; t < *ticks; t++ {
			eco.Step()
			series = append(series, float64(eco.Population()))
		}
		secPer := *seconds / float64(len(series))
		wav = raydir.SonifySeriesWAV(series, *rate, secPer, 180, 540)
		peak := 0.0
		for _, v := range series {
			if v > peak {
				peak = v
			}
		}
		fmt.Printf("sound of life: %d ticks, population 0..%.0f mapped to 180..720 Hz\n", *ticks, peak)
	default:
		hex, ok := raydir.ParseHexagram(*hexS)
		if !ok {
			hex = raydir.HexagramFromNumber(0b110010)
		}
		v := raydir.VectorFromHexagram(hex)
		p := raydir.ScoreForVector(v)
		wav = raydir.ScoreWAV(p, *rate, *seconds, *seed)
		fmt.Printf("voice of %q (%06b), mood %q:\n", hex.Name(), hex.Number(), v.Mood())
		fmt.Printf("  %d BPM, root %.1f Hz, %s, brightness %.2f\n", p.BPM, p.Root, scaleName(p.Scale), p.Bright)
	}

	if err := os.WriteFile(*out, wav, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes)\n", *out, len(wav))
}
