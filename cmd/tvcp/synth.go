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
		// Backend chosen by the tiered policy (roadmap B4):
		//   IMAGE_API_URL (raster model) > BRAIN_URL (tvcp-ai/1 sketch) >
		//   TVCP_LOCAL_BRAIN=1 (offline procedural) > placeholder.
		// All run through the async NeuralGenerator, so a slow model never stalls
		// the terminal.
		backend := aisource.NeuralBackendFromEnv()
		gen = device.NewNeuralGenerator(backend, prompt)
		if backend != nil {
			fmt.Printf("Generator: neural via %s\n", backend.Name())
		} else {
			fmt.Println("Generator: neural (Layer 2 hook) — placeholder")
			fmt.Println("Backend: none. Set IMAGE_API_URL (raster), BRAIN_URL (sketch),")
			fmt.Println("         or TVCP_LOCAL_BRAIN=1 (offline procedural) to render with a model.")
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
			fmt.Fprintf(os.Stderr, "Unknown TVCP_RENDER_MODE %q (use quadrant|perceptual|optimal|halfblock|sextant|braille|octant|triangle)\n", envMode)
			os.Exit(1)
		}
		mode = m
	} else {
		mode, _ = babe.ParseRenderMode(terminal.DetectCapability().BestBlitMode())
	}

	// Pixel-protocol paths (roadmap A5): true bitmaps instead of glyphs on
	// capable terminals. Explicit opt-in via TVCP_SIXEL=1 / TVCP_KITTY=1, or auto
	// when the terminal advertises support and no glyph mode was forced. Kitty is
	// preferred over Sixel when both are available.
	cap := terminal.DetectCapability()
	autoPixel := os.Getenv("TVCP_RENDER_MODE") == ""
	kitty := os.Getenv("TVCP_KITTY") == "1" || (autoPixel && cap.Kitty)
	sixel := os.Getenv("TVCP_SIXEL") == "1" || (autoPixel && cap.Sixel && !kitty)

	source := device.NewGenerativeSource(srcWidth, srcHeight, fps, gen)
	if err := source.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening source: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = source.Close() }()

	fmt.Printf("Source: %dx%d @ %.0f FPS\n", source.GetWidth(), source.GetHeight(), source.GetFPS())
	switch {
	case kitty:
		fmt.Println("Render mode: kitty (pixel protocol)")
	case sixel:
		fmt.Println("Render mode: sixel (pixel protocol)")
	default:
		fmt.Printf("Render mode: %s\n", mode)
	}
	fmt.Printf("Terminal: %dx%d characters\n", termWidth, termHeight)
	fmt.Println("\nPress Ctrl+C to stop")

	time.Sleep(500 * time.Millisecond)

	// Setup signal handling for graceful shutdown.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Poll faster than the target FPS; the pacer (roadmap D3) decides when a
	// frame is actually due and skips slots when producing/rendering falls
	// behind, so cadence stays smooth under load instead of drifting.
	ticker := time.NewTicker(time.Duration(1000.0/(fps*2)) * time.Millisecond)
	defer ticker.Stop()
	pacer := device.NewPacer(fps, 0)

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
			fmt.Printf("Frames rendered: %d (skipped %d under load)\n", frameCount, pacer.Skipped)
			fmt.Printf("Duration: %.1fs\n", elapsed.Seconds())
			fmt.Printf("Average FPS: %.1f\n", actualFPS)
			return

		case <-ticker.C:
			if !pacer.ShouldRender(time.Now()) {
				continue
			}
			workStart := time.Now()
			img, err := source.Read()
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nError reading frame: %v\n", err)
				continue
			}

			if kitty {
				// Pixel-perfect path (roadmap A5): Kitty graphics protocol.
				fmt.Print(terminal.MoveCursor(1, 1))
				fmt.Print(terminal.EncodeKitty(img))
			} else if sixel {
				// Pixel-perfect path (roadmap A5): Sixel bitmap.
				fmt.Print(terminal.MoveCursor(1, 1))
				fmt.Print(terminal.EncodeSixel(img, 64))
			} else {
				frame := babe.ImageToFrameMode(img, termWidth, termHeight, mode)
				frame.RenderToTerminal()
			}
			pacer.Observe(time.Since(workStart))

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
