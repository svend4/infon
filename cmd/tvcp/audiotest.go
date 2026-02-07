package main

import (
	"fmt"
	"os"
	"time"

	"github.com/svend4/infon/internal/audio"
)

func runAudioTest() {
	fmt.Println("🔊 TVCP Audio Test")
	fmt.Println()

	// Create test audio source
	format := audio.DefaultFormat()
	fmt.Printf("Audio Format:\n")
	fmt.Printf("  Sample Rate: %d Hz\n", format.SampleRate)
	fmt.Printf("  Channels: %d (%s)\n", format.Channels, channelName(format.Channels))
	fmt.Printf("  Bit Depth: %d bits\n", format.BitDepth)
	fmt.Printf("  Bandwidth: %d bytes/sec (%.1f KB/s)\n",
		format.BytesPerSecond(), float64(format.BytesPerSecond())/1024.0)
	fmt.Println()

	source := audio.NewTestAudioSource(format)

	if err := source.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening audio source: %v\n", err)
		os.Exit(1)
	}
	defer source.Close()

	// Test different tones
	tones := []struct {
		name     string
		duration time.Duration
	}{
		{"sine", 2 * time.Second},
		{"beep", 3 * time.Second},
		{"silence", 1 * time.Second},
	}

	fmt.Println("Testing audio generation...")
	fmt.Println()

	for _, tone := range tones {
		fmt.Printf("Playing %s for %v...\n", tone.name, tone.duration)
		source.SetTone(tone.name)

		// Read audio samples
		framesPerRead := format.SampleRate / 10 // 100ms chunks
		buffer := make([]int16, framesPerRead)

		startTime := time.Now()
		totalFrames := 0

		for time.Since(startTime) < tone.duration {
			n, err := source.Read(buffer)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading audio: %v\n", err)
				os.Exit(1)
			}
			totalFrames += n

			// In a real application, we would send these samples over the network
			// For now, just count them
		}

		duration := time.Since(startTime)
		fmt.Printf("  Generated %d frames in %v (%.1f FPS)\n",
			totalFrames, duration, float64(totalFrames)/duration.Seconds())
		fmt.Println()
	}

	fmt.Println("✓ Audio test completed successfully")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Test audio transmission: tvcp audio-send <host:port>")
	fmt.Println("  2. Test audio reception: tvcp audio-receive <port>")
	fmt.Println("  3. Full audio+video call: tvcp call <host:port>")
}

func runListAudio() {
	fmt.Println("🎤 Audio Devices")
	fmt.Println()

	// List capture devices
	captureDevices, err := audio.ListCaptureDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing capture devices: %v\n", err)
	} else if len(captureDevices) == 0 {
		fmt.Println("No capture devices found.")
		fmt.Println("(Real audio device detection not yet implemented)")
	} else {
		fmt.Printf("Capture Devices (%d):\n", len(captureDevices))
		for _, dev := range captureDevices {
			fmt.Printf("  [%d] %s\n", dev.ID, dev.Name)
			if dev.IsDefault {
				fmt.Println("      (default)")
			}
		}
		fmt.Println()
	}

	// List playback devices
	playbackDevices, err := audio.ListPlaybackDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing playback devices: %v\n", err)
	} else if len(playbackDevices) == 0 {
		fmt.Println("No playback devices found.")
		fmt.Println("(Real audio device detection not yet implemented)")
	} else {
		fmt.Printf("Playback Devices (%d):\n", len(playbackDevices))
		for _, dev := range playbackDevices {
			fmt.Printf("  [%d] %s\n", dev.ID, dev.Name)
			if dev.IsDefault {
				fmt.Println("      (default)")
			}
		}
		fmt.Println()
	}

	fmt.Println("For now, using test audio sources.")
	fmt.Println()
	fmt.Println("Test audio:")
	fmt.Println("  tvcp audio-test")
}

func channelName(channels int) string {
	if channels == 1 {
		return "mono"
	} else if channels == 2 {
		return "stereo"
	}
	return fmt.Sprintf("%d channels", channels)
}
