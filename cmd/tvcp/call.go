package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/audio"
	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/contacts"
	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func runCall() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tvcp call <name|host:port> [pattern] [local-port]")
		fmt.Fprintln(os.Stderr, "\nThis starts a two-way video call.")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  tvcp call alice                   # Call contact 'alice'")
		fmt.Fprintln(os.Stderr, "  tvcp call localhost:5001 bounce   # Local test call")
		fmt.Fprintln(os.Stderr, "  tvcp call [200:abc::1]:5001       # Yggdrasil call")
		os.Exit(1)
	}

	nameOrAddr := os.Args[2]
	pattern := "bounce"
	localPort := "5000"

	if len(os.Args) >= 4 {
		pattern = os.Args[3]
	}
	if len(os.Args) >= 5 {
		localPort = os.Args[4]
	}

	// Try to resolve from contacts if it's a name (no colons or brackets)
	remoteAddr := nameOrAddr
	if !strings.Contains(nameOrAddr, ":") && !strings.Contains(nameOrAddr, "[") {
		// Looks like a contact name, try to resolve
		cb, err := contacts.GetDefaultContactBook()
		if err == nil {
			if resolvedAddr, err := cb.Resolve(nameOrAddr); err == nil {
				remoteAddr = resolvedAddr + ":5000" // Default port
				fmt.Printf("📇 Resolved '%s' to %s\n", nameOrAddr, resolvedAddr)

				// Update last seen
				cb.UpdateLastSeen(nameOrAddr)
			}
		}
	}

	// Add default port if not specified
	if !strings.Contains(remoteAddr, "]:") && strings.Count(remoteAddr, ":") == 0 {
		// No port specified for IPv4/hostname
		remoteAddr = remoteAddr + ":5000"
	} else if strings.HasPrefix(remoteAddr, "[") && !strings.Contains(remoteAddr, "]:") {
		// IPv6 in brackets but no port
		remoteAddr = remoteAddr + ":5000"
	} else if strings.Count(remoteAddr, ":") > 1 && !strings.Contains(remoteAddr, "[") {
		// IPv6 without brackets, add brackets and port
		remoteAddr = "[" + remoteAddr + "]:5000"
	}

	fmt.Println("📞 TVCP Call Mode")
	fmt.Printf("Remote: %s\n", remoteAddr)
	fmt.Printf("Local port: %s\n", localPort)
	fmt.Printf("Pattern: %s\n", pattern)

	// Resolve remote address
	udpAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving address: %v\n", err)
		os.Exit(1)
	}

	// Create transport
	transport, err := network.NewTransport(":" + localPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating transport: %v\n", err)
		os.Exit(1)
	}
	defer transport.Close()

	fmt.Printf("Listening on: %s\n", transport.LocalAddr())
	fmt.Println()

	// Create camera for local video
	fps := 15.0
	camera := device.NewTestCamera(640, 480, fps, pattern)
	if err := camera.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening camera: %v\n", err)
		os.Exit(1)
	}
	defer camera.Close()

	// Create audio source for local microphone
	audioFormat := audio.DefaultFormat()
	audioSource, err := audio.NewDefaultCapture()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating audio source: %v\n", err)
		os.Exit(1)
	}
	if err := audioSource.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening audio source: %v\n", err)
		os.Exit(1)
	}
	defer audioSource.Close()

	// Create audio sink for remote audio playback
	audioSink, err := audio.NewDefaultPlayback()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating audio sink: %v\n", err)
		os.Exit(1)
	}
	if err := audioSink.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening audio sink: %v\n", err)
		os.Exit(1)
	}
	defer audioSink.Close()

	fmt.Printf("Audio: %d Hz, %d channels, %d-bit\n",
		audioFormat.SampleRate, audioFormat.Channels, audioFormat.BitDepth)

	// Send handshake
	fmt.Println("Sending handshake...")
	handshake := &network.Packet{
		Type:      network.PacketTypeHandshake,
		Sequence:  transport.NextSequence(),
		Timestamp: uint64(time.Now().UnixMilli()),
		Payload:   []byte("TVCP/1.0"),
	}
	transport.SendPacket(handshake, udpAddr)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Stats
	var mu sync.Mutex
	sendCount := 0
	recvCount := 0
	audioSendCount := 0
	audioRecvCount := 0
	startTime := time.Now()

	// Fragment reassembly buffer
	fragmentBuffer := make(map[uint32][][]byte)

	// Loss recovery components
	lossDetector := network.NewLossDetector()
	retransmitter := network.NewRetransmissionManager()
	// TODO: Integrate jitter buffer for smoother playback
	_ = network.NewJitterBuffer(100)

	// Terminal dimensions for split-screen
	// Top half: remote video (40 cols × 12 rows)
	// Bottom half: local video (40 cols × 12 rows)
	remoteWidth, remoteHeight := 40, 12
	localWidth, localHeight := 40, 12

	fmt.Println("Call connected! Press Ctrl+C to hang up")
	fmt.Print(color.ClearScreen)

	// Goroutine for sending (local video)
	go func() {
		ticker := time.NewTicker(time.Duration(1000.0/fps) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Capture frame
				img, err := camera.Read()
				if err != nil {
					continue
				}

				// Encode to terminal frame
				frame := babe.ImageToFrame(img, localWidth, localHeight)

				// Fragment frame
				fragments, err := network.FragmentFrame(frame, uint32(sendCount))
				if err != nil {
					continue
				}

				// Send each fragment
				timestamp := uint64(time.Now().UnixMilli())
				for _, fragData := range fragments {
					packet := &network.Packet{
						Type:      network.PacketTypeFrame,
						Sequence:  transport.NextSequence(),
						Timestamp: timestamp,
						Payload:   fragData,
					}
					transport.SendPacket(packet, udpAddr)

					// Store for potential retransmission
					retransmitter.OnPacketSent(packet, udpAddr)
				}

				mu.Lock()
				sendCount++
				mu.Unlock()
			}
		}
	}()

	// Goroutine for sending audio
	go func() {
		// 20ms audio chunks (good balance between latency and efficiency)
		samplesPerChunk := audioFormat.SampleRate / 50 // 50 chunks per second = 20ms each
		buffer := make([]int16, samplesPerChunk)

		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Read audio samples
				n, err := audioSource.Read(buffer)
				if err != nil || n == 0 {
					continue
				}

				// Create audio packet
				audioPacket := &network.AudioPacket{
					Timestamp:  uint64(time.Now().UnixMilli()),
					SampleRate: uint16(audioFormat.SampleRate),
					Channels:   uint8(audioFormat.Channels),
					Codec:      network.AudioCodecPCM,
					Samples:    buffer[:n],
				}

				// Encode audio packet
				audioData, err := network.EncodeAudioPacket(audioPacket)
				if err != nil {
					continue
				}

				// Send as network packet
				packet := &network.Packet{
					Type:      network.PacketTypeAudio,
					Sequence:  transport.NextSequence(),
					Timestamp: audioPacket.Timestamp,
					Payload:   audioData,
				}

				transport.SendPacket(packet, udpAddr)
				retransmitter.OnPacketSent(packet, udpAddr)

				mu.Lock()
				audioSendCount++
				mu.Unlock()
			}
		}
	}()

	// Main loop for receiving (remote video + audio)
	lastStatsTime := time.Now()

	for {
		select {
		case <-sigChan:
			elapsed := time.Since(startTime)
			mu.Lock()
			sc, rc := sendCount, recvCount
			asc, arc := audioSendCount, audioRecvCount
			mu.Unlock()

			lossStats := lossDetector.GetStatistics()
			retransStats := retransmitter.GetStatistics()

			fmt.Print(color.Reset)
			fmt.Print(color.ClearScreen)
			fmt.Printf("\n✓ Call ended\n")
			fmt.Printf("Duration: %.1fs\n", elapsed.Seconds())
			fmt.Printf("\nVideo:\n")
			fmt.Printf("  Sent: %d frames (%.1f FPS)\n", sc, float64(sc)/elapsed.Seconds())
			fmt.Printf("  Received: %d frames (%.1f FPS)\n", rc, float64(rc)/elapsed.Seconds())
			fmt.Printf("\nAudio:\n")
			fmt.Printf("  Sent: %d chunks (%.1f chunks/s)\n", asc, float64(asc)/elapsed.Seconds())
			fmt.Printf("  Received: %d chunks (%.1f chunks/s)\n", arc, float64(arc)/elapsed.Seconds())
			fmt.Printf("\nNetwork Quality:\n")
			fmt.Printf("  Packets received: %d\n", lossStats.TotalReceived)
			fmt.Printf("  Packets lost: %d (%.2f%%)\n", lossStats.TotalLost, lossStats.LossRate)
			fmt.Printf("  Out of order: %d\n", lossStats.OutOfOrder)
			fmt.Printf("  Duplicates: %d\n", lossStats.Duplicate)
			fmt.Printf("  Retransmissions: %d\n", retransStats.TotalRetransmits)
			fmt.Printf("  NACKs sent: %d\n", retransStats.TotalNACKs)
			return

		default:
			// Receive packet
			packet, _, err := transport.ReceivePacket()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				continue
			}

			// Track packet for loss detection
			if packet.Type == network.PacketTypeFrame {
				isNew := lossDetector.OnPacketReceived(packet.Sequence)
				if !isNew {
					// Duplicate packet, skip
					continue
				}

				// Check for lost packets and send NACK
				lostPackets := lossDetector.GetLostPackets()
				if len(lostPackets) > 0 {
					nackPacket := network.CreateNACKPacket(lostPackets, transport.NextSequence())
					transport.SendPacket(nackPacket, udpAddr)
				}
			}

			switch packet.Type {
			case network.PacketTypeHandshake:
				// Handshake received, connection established
				continue

			case network.PacketTypeControl:
				// Handle NACK - retransmit requested packets
				if len(packet.Payload) > 0 {
					lostSeqs, err := network.ParseNACKPayload(packet.Payload)
					if err == nil {
						for _, seq := range lostSeqs {
							retransPacket, addr, ok := retransmitter.ProcessNACK(seq)
							if ok {
								transport.SendPacket(retransPacket, addr)
							}
						}
					}
				}
				continue

			case network.PacketTypeFrame:
				// Extract fragment info
				if len(packet.Payload) < 8 {
					continue
				}

				frameID := uint32(packet.Payload[0])<<24 | uint32(packet.Payload[1])<<16 |
					uint32(packet.Payload[2])<<8 | uint32(packet.Payload[3])
				fragID := uint16(packet.Payload[4])<<8 | uint16(packet.Payload[5])
				totalFrags := uint16(packet.Payload[6])<<8 | uint16(packet.Payload[7])

				// Store fragment
				if fragmentBuffer[frameID] == nil {
					fragmentBuffer[frameID] = make([][]byte, totalFrags)
				}
				fragmentBuffer[frameID][fragID] = packet.Payload

				// Check if complete
				allReceived := true
				for i := 0; i < int(totalFrags); i++ {
					if fragmentBuffer[frameID][i] == nil {
						allReceived = false
						break
					}
				}

				if allReceived {
					// Assemble remote frame
					remoteFrame, err := network.AssembleFrame(fragmentBuffer[frameID])
					if err == nil {
						// Render split-screen: remote on top, local on bottom
						renderSplitScreen(camera, remoteFrame, localWidth, localHeight, remoteWidth, remoteHeight)

						mu.Lock()
						recvCount++
						mu.Unlock()
					}

					delete(fragmentBuffer, frameID)
				}

			case network.PacketTypeAudio:
				// Decode audio packet
				audioPacket, err := network.DecodeAudioPacket(packet.Payload)
				if err != nil {
					continue
				}

				// Play audio samples
				if len(audioPacket.Samples) > 0 {
					audioSink.Write(audioPacket.Samples)

					mu.Lock()
					audioRecvCount++
					mu.Unlock()
				}

			case network.PacketTypeTextChat:
				// Decode text message
				textMsg, err := network.DecodeTextMessage(packet.Payload)
				if err != nil {
					continue
				}

				// Display message (will appear in terminal output)
				sender := textMsg.Sender
				if sender == "" {
					sender = "Remote"
				}
				fmt.Printf("\n💬 [%s] %s: %s\n", textMsg.FormatTime(), sender, textMsg.Message)
			}

			// Show stats
			now := time.Now()
			if now.Sub(lastStatsTime) >= 2*time.Second {
				mu.Lock()
				sc, rc := sendCount, recvCount
				asc, arc := audioSendCount, audioRecvCount
				mu.Unlock()

				elapsed := now.Sub(startTime)
				lossStats := lossDetector.GetStatistics()

				fmt.Printf("%s[Call] Video: %d/%d (%.1f/%.1f FPS) | Audio: %d/%d | Loss: %.1f%% | Time: %.0fs%s\n",
					color.Reset, sc, rc, float64(sc)/elapsed.Seconds(), float64(rc)/elapsed.Seconds(),
					asc, arc, lossStats.LossRate, elapsed.Seconds(), color.Reset)
				lastStatsTime = now
			}
		}
	}
}

// renderSplitScreen renders remote video on top, local video preview on bottom
func renderSplitScreen(camera *device.TestCamera, remoteFrame *terminal.Frame, localW, localH, remoteW, remoteH int) {
	// Clear screen
	fmt.Print(color.ClearScreen)

	// Render remote frame (top half)
	fmt.Println("Remote:")
	for y := 0; y < remoteFrame.Height; y++ {
		for x := 0; x < remoteFrame.Width; x++ {
			block := remoteFrame.Blocks[y][x]
			fmt.Print(block.Fg.FgString())
			fmt.Print(block.Bg.BgString())
			fmt.Printf("%c", block.Glyph)
		}
		fmt.Print(color.Reset)
		fmt.Println()
	}

	// Separator
	fmt.Println(color.Reset + "────────────────────────────────────────")

	// Render local preview (bottom half)
	fmt.Println("Local:")
	img, _ := camera.Read()
	if img != nil {
		localFrame := babe.ImageToFrame(img, localW, localH)
		for y := 0; y < localFrame.Height; y++ {
			for x := 0; x < localFrame.Width; x++ {
				block := localFrame.Blocks[y][x]
				fmt.Print(block.Fg.FgString())
				fmt.Print(block.Bg.BgString())
				fmt.Printf("%c", block.Glyph)
			}
			fmt.Print(color.Reset)
			fmt.Println()
		}
	}

	fmt.Print(color.Reset)
}
