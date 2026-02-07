package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/pkg/color"
)

func runPreview() {
	fmt.Println("🎥 TVCP Live Preview")
	fmt.Println("Starting camera...")

	// Parse options
	pattern := "bounce" // Default pattern
	fps := 15.0
	width := 80
	height := 24

	if len(os.Args) >= 3 {
		pattern = os.Args[2]
	}

	// Create test camera
	camera := device.NewTestCamera(640, 480, fps, pattern)

	// Open camera
	if err := camera.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening camera: %v\n", err)
		os.Exit(1)
	}
	defer camera.Close()

	fmt.Printf("Camera: %dx%d @ %.0f FPS\n", camera.GetWidth(), camera.GetHeight(), camera.GetFPS())
	fmt.Printf("Terminal: %dx%d characters\n", width, height)
	fmt.Printf("Pattern: %s\n", pattern)
	fmt.Println("\nPress Ctrl+C to stop")

	time.Sleep(500 * time.Millisecond)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Frame timing
	frameDuration := time.Duration(1000.0/fps) * time.Millisecond
	ticker := time.NewTicker(frameDuration)
	defer ticker.Stop()

	frameCount := 0
	startTime := time.Now()
	lastStatsTime := startTime

	// Clear screen
	fmt.Print(color.ClearScreen)

	for {
		select {
		case <-sigChan:
			// Graceful shutdown
			fmt.Print(color.Reset)
			fmt.Print(color.ClearScreen)
			elapsed := time.Since(startTime)
			actualFPS := float64(frameCount) / elapsed.Seconds()
			fmt.Printf("\n✓ Preview stopped\n")
			fmt.Printf("Frames rendered: %d\n", frameCount)
			fmt.Printf("Duration: %.1fs\n", elapsed.Seconds())
			fmt.Printf("Average FPS: %.1f\n", actualFPS)
			return

		case <-ticker.C:
			// Read frame from camera
			img, err := camera.Read()
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nError reading frame: %v\n", err)
				continue
			}

			// Convert to terminal frame
			frame := babe.ImageToFrame(img, width, height)

			// Render to terminal
			frame.RenderToTerminal()

			frameCount++

			// Show stats every second
			now := time.Now()
			if now.Sub(lastStatsTime) >= time.Second {
				elapsed := now.Sub(startTime)
				currentFPS := float64(frameCount) / elapsed.Seconds()
				fmt.Printf("%s[Stats] Frame: %d | FPS: %.1f | Time: %.1fs%s\n",
					color.Reset, frameCount, currentFPS, elapsed.Seconds(), color.Reset)
				lastStatsTime = now
			}
		}
	}
}

func runPreviewHelp() {
	fmt.Println("Usage: tvcp preview [pattern]")
	fmt.Println("\nAvailable patterns:")
	fmt.Println("  bounce      Animated bouncing ball (default)")
	fmt.Println("  gradient    Animated color gradient")
	fmt.Println("  noise       Random noise (like TV static)")
	fmt.Println("  colorbar    SMPTE color bars")
	fmt.Println("\nExamples:")
	fmt.Println("  tvcp preview")
	fmt.Println("  tvcp preview gradient")
	fmt.Println("  tvcp preview noise")
	fmt.Println("\nPress Ctrl+C to stop preview")
}
