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
	"github.com/svend4/infon/pkg/terminal"
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
		// Backend priority:
		//   1. IMAGE_API_URL — a real text-to-image model (raster output): the
		//      true "describe it, the model paints it" path.
		//   2. BRAIN_URL — a tvcp-ai/1 brain returns a vector sketch we rasterize.
		//   3. neither — an animated placeholder, so the pipeline still runs.
		// All three run through the async NeuralGenerator, so a slow model never
		// stalls the terminal.
		if ib, ok := aisource.NewImageBackendFromEnv(); ok {
			gen = device.NewNeuralGenerator(ib, prompt)
			fmt.Println("Generator: neural via text-to-image API (raster)")
		} else if url := os.Getenv("BRAIN_URL"); url != "" {
			// Route generation through the open tvcp-ai/1 protocol: any model
			// (Ollama / OpenAI / Anthropic adapter) paints frames asynchronously.
			gen = device.NewNeuralGenerator(aisource.NewBrainBackend(url), prompt)
			fmt.Printf("Generator: neural via tvcp-ai/1 brain at %s\n", url)
		} else {
			gen = device.NewNeuralGenerator(nil, prompt)
			fmt.Println("Generator: neural (Layer 2 hook) — placeholder")
			fmt.Println("Backend: not connected. Set IMAGE_API_URL (raster model) or")
			fmt.Println("         BRAIN_URL (tvcp-ai/1 sketch) to render with a model.")
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

	// Render mode (roadmap A1-A4): explicit TVCP_RENDER_MODE wins; otherwise
	// auto-pick the best glyph mode the terminal can show (roadmap D2).
	var mode babe.RenderMode
	if envMode := os.Getenv("TVCP_RENDER_MODE"); envMode != "" {
		m, ok := babe.ParseRenderMode(envMode)
		if !ok {
			fmt.Fprintf(os.Stderr, "Unknown TVCP_RENDER_MODE %q (use quadrant|perceptual|halfblock|sextant|braille)\n", envMode)
			os.Exit(1)
		}
		mode = m
	} else {
		mode, _ = babe.ParseRenderMode(terminal.DetectCapability().BestBlitMode())
	}

	source := device.NewGenerativeSource(srcWidth, srcHeight, fps, gen)
	if err := source.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening source: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = source.Close() }()

	fmt.Printf("Source: %dx%d @ %.0f FPS\n", source.GetWidth(), source.GetHeight(), source.GetFPS())
	fmt.Printf("Render mode: %s\n", mode)
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

			frame := babe.ImageToFrameMode(img, termWidth, termHeight, mode)
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
