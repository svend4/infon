package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/audio"
	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/internal/group"
	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/internal/video"
)

func runGroupCall(peers []string, localPort string) error {
	fmt.Println("👥 TVCP Group Call")
	fmt.Printf("   Participants: %d\n", len(peers))
	fmt.Printf("   Local port: %s\n\n", localPort)

	// Create group call
	groupCall, err := group.NewGroupCall(localPort)
	if err != nil {
		return fmt.Errorf("failed to create group call: %w", err)
	}
	defer groupCall.Stop()

	// Add peers
	for _, peerAddr := range peers {
		addr, err := net.ResolveUDPAddr("udp", peerAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Invalid peer address %s: %v\n", peerAddr, err)
			continue
		}

		// Extract hostname/ID from address
		peerID := strings.Split(peerAddr, ":")[0]
		err = groupCall.AddPeer(peerID, addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to add peer %s: %v\n", peerID, err)
			continue
		}
		fmt.Printf("✅ Added peer: %s\n", peerID)
	}

	fmt.Println()

	// Initialize camera (device ID 0 = default)
	cam, err := device.NewCamera(0)
	if err != nil {
		return fmt.Errorf("failed to create camera: %w", err)
	}
	if err := cam.SetResolution(640, 480); err != nil {
		return fmt.Errorf("failed to set camera resolution: %w", err)
	}
	if err := cam.Open(); err != nil {
		return fmt.Errorf("failed to open camera: %w", err)
	}
	defer cam.Close()

	// Initialize audio using default capture/playback
	mic, err := audio.NewDefaultCapture()
	if err != nil {
		return fmt.Errorf("failed to initialize microphone: %w", err)
	}
	if err := mic.Open(); err != nil {
		return fmt.Errorf("failed to open microphone: %w", err)
	}
	defer mic.Close()

	speaker, err := audio.NewDefaultPlayback()
	if err != nil {
		return fmt.Errorf("failed to initialize speaker: %w", err)
	}
	if err := speaker.Open(); err != nil {
		return fmt.Errorf("failed to open speaker: %w", err)
	}
	defer speaker.Close()

	// Get audio format for mixer
	audioFormat := mic.GetFormat()

	// Initialize audio mixer
	audioMixer := group.NewAudioMixer(audioFormat.SampleRate, 320)

	// Initialize video grid
	videoGrid := group.NewVideoGrid(80, 30)

	// Start group call
	if err := groupCall.Start(); err != nil {
		return fmt.Errorf("failed to start group call: %w", err)
	}

	// P-frame encoder for sending
	pframeEncoder := video.NewPFrameEncoder(30)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Stats
	var sendCount, recvCount uint64

	// Fragment buffers for each peer
	peerFragmentBuffers := make(map[string]map[uint32][][]byte)

	// P-frame decoders for each peer
	peerDecoders := make(map[string]*video.PFrameDecoder)

	// Video sending goroutine
	go func() {
		ticker := time.NewTicker(time.Second / 15) // 15 FPS
		defer ticker.Stop()

		for range ticker.C {
			if !groupCall.IsRunning() {
				break
			}

			// Capture frame
			img, err := cam.Read()
			if err != nil {
				continue
			}

			// Encode to terminal frame
			frame := babe.ImageToFrame(img, 40, 30)

			// Encode with P-frames
			encodedFrame := pframeEncoder.Encode(frame)
			encodedData, err := network.EncodeEncodedFrame(encodedFrame)
			if err != nil {
				continue
			}

			// Fragment frame
			fragments, err := network.FragmentData(encodedData, uint32(sendCount))
			if err != nil {
				continue
			}

			// Broadcast to all peers
			timestamp := uint64(time.Now().UnixMilli())
			for _, fragData := range fragments {
				packet := &network.Packet{
					Type:      network.PacketTypeFrame,
					Sequence:  groupCall.GetTransport().NextSequence(),
					Timestamp: timestamp,
					Payload:   fragData,
				}
				groupCall.BroadcastPacket(packet)
			}

			sendCount++
		}
	}()

	// Audio sending goroutine
	go func() {
		// Audio buffer for reading (20ms at 16kHz = 320 samples)
		audioBuffer := make([]int16, 320)

		for {
			if !groupCall.IsRunning() {
				break
			}

			// Read from microphone
			n, err := mic.Read(audioBuffer)
			if err != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Encode as PCM (only the samples that were read)
			audioData := audio.EncodePCM(audioBuffer[:n])

			// Send to all peers
			packet := &network.Packet{
				Type:      network.PacketTypeAudio,
				Sequence:  groupCall.GetTransport().NextSequence(),
				Timestamp: uint64(time.Now().UnixMilli()),
				Payload:   audioData,
			}
			groupCall.BroadcastPacket(packet)
		}
	}()

	// Packet receiving goroutine
	go func() {
		buf := make([]byte, 2048)
		for {
			if !groupCall.IsRunning() {
				break
			}

			// Receive packet
			n, addr, err := groupCall.GetTransport().GetConn().ReadFromUDP(buf)
			if err != nil {
				continue
			}

			// Decode packet
			packet, err := network.Decode(buf[:n])
			if err != nil {
				continue
			}

			// Find peer by address
			var peerID string
			for _, peer := range groupCall.GetAllPeers() {
				if peer.Address.String() == addr.String() {
					peerID = peer.ID
					peer.UpdateLastSeen()
					break
				}
			}

			if peerID == "" {
				continue
			}

			switch packet.Type {
			case network.PacketTypeFrame:
				// Initialize buffers for this peer if needed
				if peerFragmentBuffers[peerID] == nil {
					peerFragmentBuffers[peerID] = make(map[uint32][][]byte)
				}
				if peerDecoders[peerID] == nil {
					peerDecoders[peerID] = video.NewPFrameDecoder()
				}

				// Parse fragment
				if len(packet.Payload) < 8 {
					continue
				}

				frameID := uint32(packet.Payload[0])<<24 | uint32(packet.Payload[1])<<16 |
					uint32(packet.Payload[2])<<8 | uint32(packet.Payload[3])
				fragID := uint16(packet.Payload[4])<<8 | uint16(packet.Payload[5])
				totalFrags := uint16(packet.Payload[6])<<8 | uint16(packet.Payload[7])

				// Store fragment
				if peerFragmentBuffers[peerID][frameID] == nil {
					peerFragmentBuffers[peerID][frameID] = make([][]byte, totalFrags)
				}
				peerFragmentBuffers[peerID][frameID][fragID] = packet.Payload

				// Check if complete
				allReceived := true
				for i := 0; i < int(totalFrags); i++ {
					if peerFragmentBuffers[peerID][frameID][i] == nil {
						allReceived = false
						break
					}
				}

				if allReceived {
					// Assemble and decode
					encodedData, err := network.AssembleData(peerFragmentBuffers[peerID][frameID])
					if err == nil {
						encodedFrame, err := network.DecodeEncodedFrame(encodedData)
						if err == nil {
							frame := peerDecoders[peerID].Decode(encodedFrame)
							videoGrid.SetFrame(peerID, frame)
							recvCount++
						}
					}
					delete(peerFragmentBuffers[peerID], frameID)
				}

			case network.PacketTypeAudio:
				// Decode audio
				samples, err := audio.DecodePCM(packet.Payload)
				if err == nil {
					// Add to mixer
					audioMixer.AddSource(peerID, samples)
				}
			}
		}
	}()

	// Audio playback goroutine
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond) // 20ms chunks
		defer ticker.Stop()

		for range ticker.C {
			if !groupCall.IsRunning() {
				break
			}

			// Mix audio from all peers
			mixed := audioMixer.Mix()

			// Play mixed audio
			if len(mixed) > 0 {
				speaker.Write(mixed)
			}
		}
	}()

	// Display goroutine
	go func() {
		ticker := time.NewTicker(time.Second / 15) // 15 FPS
		defer ticker.Stop()

		for range ticker.C {
			if !groupCall.IsRunning() {
				break
			}

			// Clear screen
			fmt.Print("\033[2J\033[H")

			// Render video grid
			gridFrame := videoGrid.Render()
			gridFrame.RenderToTerminal()

			// Display stats
			fmt.Println()
			fmt.Println(videoGrid.RenderStats(groupCall.GetActivePeers()))
			fmt.Printf("📊 Sent: %d frames | Received: %d frames\n", sendCount, recvCount)
		}
	}()

	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if !groupCall.IsRunning() {
				break
			}
			groupCall.CleanupInactivePeers()
		}
	}()

	// Wait for interrupt
	<-sigChan
	fmt.Println("\n\n👋 Leaving group call...")

	return nil
}

func handleGroupCallCommand(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: tvcp group <peer1> [peer2] [peer3] ...")
		fmt.Println()
		fmt.Println("Start a group video call with multiple participants")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --port <port>   Local port to listen on (default: 5000)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  tvcp group alice:5000 bob:5000")
		fmt.Println("  tvcp group [200:abc::1]:5000 [200:def::1]:5000")
		fmt.Println("  tvcp group --port 6000 peer1:5000 peer2:5000 peer3:5000")
		fmt.Println()
		fmt.Println("Features:")
		fmt.Println("  • Multi-party video conferencing")
		fmt.Println("  • Automatic video grid layout (1-9 participants)")
		fmt.Println("  • Audio mixing from all participants")
		fmt.Println("  • P-frame compression for bandwidth efficiency")
		fmt.Println("  • Automatic peer cleanup (inactive > 10s)")
		return nil
	}

	localPort := "5000"
	peers := []string{}

	for i := 0; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			localPort = args[i+1]
			i++
		} else {
			peers = append(peers, args[i])
		}
	}

	if len(peers) == 0 {
		return fmt.Errorf("no peers specified")
	}

	return runGroupCall(peers, ":"+localPort)
}
