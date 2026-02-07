package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/pkg/color"
)

func runSend() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tvcp send <host:port> [pattern]")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  tvcp send localhost:5000")
		fmt.Fprintln(os.Stderr, "  tvcp send 192.168.1.100:5000 bounce")
		fmt.Fprintln(os.Stderr, "  tvcp send [::1]:5000 gradient")
		os.Exit(1)
	}

	remoteAddr := os.Args[2]
	pattern := "bounce"
	if len(os.Args) >= 4 {
		pattern = os.Args[3]
	}

	fmt.Println("🚀 TVCP Send Mode")
	fmt.Printf("Remote: %s\n", remoteAddr)
	fmt.Printf("Pattern: %s\n", pattern)

	// Resolve remote address
	udpAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving address: %v\n", err)
		os.Exit(1)
	}

	// Create transport (listen on random port)
	transport, err := network.NewTransport(":0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating transport: %v\n", err)
		os.Exit(1)
	}
	defer transport.Close()

	fmt.Printf("Local: %s\n", transport.LocalAddr())
	fmt.Println()

	// Create camera
	fps := 15.0
	camera := device.NewTestCamera(640, 480, fps, pattern)
	if err := camera.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening camera: %v\n", err)
		os.Exit(1)
	}
	defer camera.Close()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Frame timing
	frameDuration := time.Duration(1000.0/fps) * time.Millisecond
	ticker := time.NewTicker(frameDuration)
	defer ticker.Stop()

	frameCount := 0
	startTime := time.Now()
	lastStatsTime := startTime

	width := 80
	height := 24

	fmt.Println("Streaming... Press Ctrl+C to stop\n")

	for {
		select {
		case <-sigChan:
			elapsed := time.Since(startTime)
			actualFPS := float64(frameCount) / elapsed.Seconds()
			fmt.Printf("\n\n✓ Stream stopped\n")
			fmt.Printf("Frames sent: %d\n", frameCount)
			fmt.Printf("Duration: %.1fs\n", elapsed.Seconds())
			fmt.Printf("Average FPS: %.1f\n", actualFPS)
			return

		case <-ticker.C:
			// Capture frame
			img, err := camera.Read()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading frame: %v\n", err)
				continue
			}

				// Encode to terminal frame
			frame := babe.ImageToFrame(img, width, height)

			// Fragment frame
			fragments, err := network.FragmentFrame(frame, uint32(frameCount))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fragmenting frame: %v\n", err)
				continue
			}

			// Send each fragment as a packet
			timestamp := uint64(time.Now().UnixMilli())
			for _, fragData := range fragments {
				packet := &network.Packet{
					Type:      network.PacketTypeFrame,
					Sequence:  transport.NextSequence(),
					Timestamp: timestamp,
					Payload:   fragData,
				}

				err = transport.SendPacket(packet, udpAddr)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error sending packet: %v\n", err)
					break
				}
			}

			frameCount++

			// Show stats every second
			now := time.Now()
			if now.Sub(lastStatsTime) >= time.Second {
				elapsed := now.Sub(startTime)
				currentFPS := float64(frameCount) / elapsed.Seconds()
				fmt.Printf("%s[Send] Frame: %d | FPS: %.1f | Fragments: %d%s\n",
					color.Reset, frameCount, currentFPS, len(fragments), color.Reset)
				lastStatsTime = now
			}
		}
	}
}
