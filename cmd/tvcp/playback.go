package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/svend4/infon/internal/recorder"
)

func runPlayback() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tvcp playback <recording-file>")
		fmt.Fprintln(os.Stderr, "\nPlays back a recorded call.")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  tvcp playback recordings/call-2026-02-07.tvcp")
		os.Exit(1)
	}

	filename := os.Args[2]

	fmt.Println("🎬 TVCP Playback")
	fmt.Printf("File: %s\n\n", filename)

	// Create player
	player := recorder.NewPlayer()

	// Load recording
	if err := player.Load(filename); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading recording: %v\n", err)
		os.Exit(1)
	}

	// Show info
	info := player.GetInfo()
	fmt.Println("Recording Information:")
	fmt.Printf("  Started: %s\n", info.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Duration: %.1f seconds\n", float64(info.Duration)/1000.0)
	fmt.Printf("  Resolution: %dx%d\n", info.FrameWidth, info.FrameHeight)
	fmt.Printf("  Video frames: %d (%.1f FPS)\n", info.FrameCount,
		float64(info.FrameCount)/(float64(info.Duration)/1000.0))
	fmt.Printf("  Audio chunks: %d (%d Hz)\n", info.AudioChunks, info.AudioRate)
	fmt.Println()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\n⏹️  Stopping playback...")
		player.Stop()
	}()

	// Play recording
	if err := player.Play(); err != nil {
		fmt.Fprintf(os.Stderr, "Error during playback: %v\n", err)
		os.Exit(1)
	}
}
