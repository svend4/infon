package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/aisource"
	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/color"
)

// runSynth renders synthesized graphics live in the terminal.
//
// It demonstrates the generative pipeline: a FrameGenerator (procedural Layer 1,
// or the neural Layer 2 hook) feeds a GenerativeSource, which is rendered with
// the exact same BABE -> ANSI path used for real camera video.
//
// Usage:
//
//	tvcp synth                 # default: plasma
//	tvcp synth plasma
//	tvcp synth ripple
//	tvcp synth neural [prompt] # neural hook (placeholder until a model is wired)
func runSynth() {
	fmt.Println("🎨 TVCP Generative Synthesis")

	// Parse options.
	name := "plasma"
	if len(os.Args) >= 3 {
		name = os.Args[2]
	}

	fps := 15.0
	termWidth := 80
	termHeight := 24
	// Synthesis "sensor" resolution. Small is fine: the renderer downsamples to
	// the terminal grid, so most of the cost is the generator, not the render.
	srcWidth := 320
	srcHeight := 240

	var gen device.FrameGenerator
	if name == "neural" {
		prompt := strings.Join(os.Args[3:], " ")
		if prompt == "" {
			prompt = "(no prompt)"
		}
		fmt.Printf("Prompt: %q\n", prompt)
		if url := os.Getenv("BRAIN_URL"); url != "" {
			// Route generation through the open tvcp-ai/1 protocol: any model
			// (Ollama / OpenAI / Anthropic adapter) paints frames asynchronously.
			gen = device.NewNeuralGenerator(aisource.NewBrainBackend(url), prompt)
			fmt.Printf("Generator: neural via tvcp-ai/1 brain at %s\n", url)
		} else {
			gen = device.NewNeuralGenerator(nil, prompt)
			fmt.Println("Generator: neural (Layer 2 hook) — placeholder")
			fmt.Println("Backend: not connected. Set BRAIN_URL to a tvcp-ai/1 endpoint to render with a model.")
		}
	} else {
		g, err := device.NewProceduralGenerator(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		gen = g
		fmt.Printf("Generator: %s (Layer 1 procedural)\n", g.Name())
	}

	source := device.NewGenerativeSource(srcWidth, srcHeight, fps, gen)
	if err := source.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening source: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = source.Close() }()

	fmt.Printf("Source: %dx%d @ %.0f FPS\n", source.GetWidth(), source.GetHeight(), source.GetFPS())
	fmt.Printf("Terminal: %dx%d characters\n", termWidth, termHeight)
	fmt.Println("\nPress Ctrl+C to stop")

	time.Sleep(500 * time.Millisecond)

	// Setup signal handling for graceful shutdown.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	frameDuration := time.Duration(1000.0/fps) * time.Millisecond
	ticker := time.NewTicker(frameDuration)
	defer ticker.Stop()

	frameCount := 0
	startTime := time.Now()
	lastStatsTime := startTime

	fmt.Print(color.ClearScreen)

	for {
		select {
		case <-sigChan:
			fmt.Print(color.Reset)
			fmt.Print(color.ClearScreen)
			elapsed := time.Since(startTime)
			actualFPS := float64(frameCount) / elapsed.Seconds()
			fmt.Printf("\n✓ Synthesis stopped\n")
			fmt.Printf("Frames rendered: %d\n", frameCount)
			fmt.Printf("Duration: %.1fs\n", elapsed.Seconds())
			fmt.Printf("Average FPS: %.1f\n", actualFPS)
			return

		case <-ticker.C:
			img, err := source.Read()
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nError reading frame: %v\n", err)
				continue
			}

			frame := babe.ImageToFrame(img, termWidth, termHeight)
			frame.RenderToTerminal()

			frameCount++

			now := time.Now()
			if now.Sub(lastStatsTime) >= time.Second {
				elapsed := now.Sub(startTime)
				currentFPS := float64(frameCount) / elapsed.Seconds()
				fmt.Printf("%s[Stats] Generator: %s | Frame: %d | FPS: %.1f | Time: %.1fs%s\n",
					color.Reset, gen.Name(), frameCount, currentFPS, elapsed.Seconds(), color.Reset)
				lastStatsTime = now
			}
		}
	}
}
