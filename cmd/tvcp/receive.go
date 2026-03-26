package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/pkg/color"
)

func runReceive() {
	port := "5000" // default port
	if len(os.Args) >= 3 {
		port = os.Args[2]
	}

	listenAddr := ":" + port

	fmt.Println("📺 TVCP Receive Mode")
	fmt.Printf("Listening on: %s\n", listenAddr)

	// Create transport
	transport, err := network.NewTransport(listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating transport: %v\n", err)
		os.Exit(1)
	}
	defer transport.Close()

	fmt.Printf("Local: %s\n", transport.LocalAddr())
	fmt.Println("\nWaiting for video stream... Press Ctrl+C to stop\n")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	frameCount := 0
	startTime := time.Now()
	lastStatsTime := startTime
	var lastSeq uint32

	// Fragment reassembly buffer
	fragmentBuffer := make(map[uint32][][]byte) // frameID -> fragments

	// Clear screen initially
	fmt.Print(color.ClearScreen)

	// Main receive loop
	for {
		select {
		case <-sigChan:
			elapsed := time.Since(startTime)
			actualFPS := float64(frameCount) / elapsed.Seconds()
			fmt.Print(color.Reset)
			fmt.Print(color.ClearScreen)
			fmt.Printf("\n✓ Receive stopped\n")
			fmt.Printf("Frames received: %d\n", frameCount)
			fmt.Printf("Duration: %.1fs\n", elapsed.Seconds())
			fmt.Printf("Average FPS: %.1f\n", actualFPS)
			return

		default:
			// Receive packet (with timeout)
			packet, remoteAddr, err := transport.ReceivePacket()
			if err != nil {
				// Check if timeout (expected)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				fmt.Fprintf(os.Stderr, "\nError receiving packet: %v\n", err)
				continue
			}

				// Handle different packet types
			switch packet.Type {
			case network.PacketTypeFrame:
				// Extract fragment info from payload
				if len(packet.Payload) < 8 {
					fmt.Fprintf(os.Stderr, "\nInvalid fragment\n")
					continue
				}

				// Parse fragment header (first 8 bytes)
				frameID := uint32(packet.Payload[0])<<24 | uint32(packet.Payload[1])<<16 |
					uint32(packet.Payload[2])<<8 | uint32(packet.Payload[3])
				fragID := uint16(packet.Payload[4])<<8 | uint16(packet.Payload[5])
				totalFrags := uint16(packet.Payload[6])<<8 | uint16(packet.Payload[7])

				// Store fragment
				if fragmentBuffer[frameID] == nil {
					fragmentBuffer[frameID] = make([][]byte, totalFrags)
				}
				fragmentBuffer[frameID][fragID] = packet.Payload

				// Check if all fragments received
				allReceived := true
				for i := 0; i < int(totalFrags); i++ {
					if fragmentBuffer[frameID][i] == nil {
						allReceived = false
						break
					}
				}

				if allReceived {
					// Assemble frame
					frame, err := network.AssembleFrame(fragmentBuffer[frameID])
					if err != nil {
						fmt.Fprintf(os.Stderr, "\nError assembling frame: %v\n", err)
					} else {
						// Render frame to terminal
						frame.RenderToTerminal()
						frameCount++
					}

					// Clean up old fragments
					delete(fragmentBuffer, frameID)
				}

				lastSeq = packet.Sequence

				// Show stats every second
				now := time.Now()
				if now.Sub(lastStatsTime) >= time.Second {
					elapsed := now.Sub(startTime)
					currentFPS := float64(frameCount) / elapsed.Seconds()
					fmt.Printf("%s[Recv] Frame: %d | FPS: %.1f | From: %s | Seq: %d%s\n",
						color.Reset, frameCount, currentFPS, remoteAddr, lastSeq, color.Reset)
					lastStatsTime = now
				}

			case network.PacketTypeKeepAlive:
				// Ignore keep-alive for now
				continue

			default:
				fmt.Fprintf(os.Stderr, "\nUnknown packet type: %d\n", packet.Type)
			}
		}
	}
}
