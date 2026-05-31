package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/aisource"
	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/color"
)

// runAI streams an AI-generated video source into the terminal.
//
// Both modes are a device.Camera driven by a single paced render loop, so the
// terminal renders at a steady FPS and never busy-spins:
//
//   - default:  AICamera (a procedural plasma field), and
//   - -brain U: a GenerativeSource backed by the async NeuralGenerator +
//     tvcp-ai/1 BrainBackend (the same path as `tvcp synth neural`).
//
// Routing -brain through NeuralGenerator means a slow model is fetched in the
// background while the loop keeps showing the most recent frame — it never
// blocks the terminal the way a synchronous per-frame HTTP request would.
func runAI() {
	fs := flag.NewFlagSet("ai", flag.ContinueOnError)
	brainURL := fs.String("brain", "", "paint frames with a tvcp-ai/1 brain (async)")
	promptFlag := fs.String("prompt", "a calm landscape", "scene prompt (with -brain)")
	fps := fs.Float64("fps", 15, "render frame rate")
	_ = fs.Parse(os.Args[2:])

	cols, rows := 80, 24

	// Both branches yield a device.Camera so the loop below is identical.
	var cam device.Camera
	var neural *device.NeuralGenerator // non-nil only in brain mode (for prompt cycling)

	if *brainURL != "" {
		neural = device.NewNeuralGenerator(aisource.NewBrainBackend(*brainURL), *promptFlag)
		// A painterly model is slow; refresh roughly once per second and let the
		// loop reuse the latest frame in between.
		neural.Interval = time.Second
		cam = device.NewGenerativeSource(cols*2, rows*2, *fps, neural)
		fmt.Printf("AI video via tvcp-ai/1 brain at %s\n", *brainURL)
		fmt.Printf("Prompt: %q (cycles through times of day)\n", *promptFlag)
	} else {
		cam = aisource.NewAICamera(cols*2, rows*2, *fps)
		fmt.Println("AI video: procedural plasma (set -brain URL to let a model paint)")
	}

	if err := cam.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening AI source: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = cam.Close() }()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	tick := time.NewTicker(time.Duration(1000.0/cam.GetFPS()) * time.Millisecond)
	defer tick.Stop()

	// In brain mode, slowly cycle the prompt through times of day for variety.
	phases := []string{"at dawn", "in the morning", "at noon", "at sunset", "at night"}
	phaseTick := time.NewTicker(3 * time.Second)
	defer phaseTick.Stop()
	phase := 0

	fmt.Print(color.ClearScreen)
	for {
		select {
		case <-sig:
			fmt.Print(color.Reset, color.ClearScreen)
			return
		case <-phaseTick.C:
			if neural != nil {
				phase++
				neural.SetPrompt(*promptFlag + " " + phases[phase%len(phases)])
			}
		case <-tick.C:
			img, err := cam.Read()
			if err != nil {
				continue
			}
			babe.ImageToFrame(img, cols, rows).RenderToTerminal()
		}
	}
}
