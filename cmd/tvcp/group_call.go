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
	defer func() { _ = groupCall.Stop() }()

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

	// Camera, microphone and speaker. mic and speaker are nil when there is no
	// usable sound card; the call then runs without that direction of audio.
	cam, mic, speaker, err := openGroupDevices()
	if err != nil {
		return err
	}
	defer func() { _ = cam.Close() }()
	if mic != nil {
		defer func() { _ = mic.Close() }()
	}
	if speaker != nil {
		defer func() { _ = speaker.Close() }()
	}

	// Get audio format for mixer
	audioFormat := audio.DefaultFormat()
	if mic != nil {
		audioFormat = mic.GetFormat()
	}

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

	// Audio sending goroutine (skipped when there is no microphone)
	go func() {
		if mic == nil {
			return
		}
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

				frameID, fragID, totalFrags, ok := network.ParseFragmentHeader(packet.Payload, -1)
				if !ok {
					continue
				}
				if buf := peerFragmentBuffers[peerID][frameID]; buf != nil && len(buf) != int(totalFrags) {
					continue
				}

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

	// Audio playback goroutine (skipped when there is no speaker)
	go func() {
		if speaker == nil {
			return
		}
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
				_, _ = speaker.Write(mixed)
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

	// Liveness goroutine. Membership is fixed by the command line, so a quiet peer
	// is kept, not evicted: media goes only to active peers, and every quiet one
	// gets a keep-alive. Evicting quiet peers meant that a participant who started
	// more than 10 s after the others was dropped before its first packet arrived
	// and then ignored, so the call never connected; and after an outage longer
	// than 10 s both sides had dropped each other for good.
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if !groupCall.IsRunning() {
				break
			}
			groupCall.PingInactive()
		}
	}()

	// Wait for interrupt
	<-sigChan
	fmt.Println("\n\n👋 Leaving group call...")

	return nil
}

// The device constructors openGroupDevices uses; tests replace them so they never
// open the real camera or sound card of the machine they run on.
var (
	groupListCameras = device.ListCameras
	groupNewCamera   = device.NewCamera
	groupNewCapture  = audio.NewDefaultCapture
	groupNewPlayback = audio.NewDefaultPlayback
)

// openGroupDevices opens the camera, microphone and speaker for a group call.
// Like `tvcp call`, it falls back to a test camera when no real camera can be
// opened, and returns a nil mic or speaker (with a warning) when audio is
// unavailable, so the call also runs on headless hosts with no camera or sound
// card. The camera is opened first, so an error leaves nothing open.
func openGroupDevices() (cam device.Camera, mic audio.AudioCapture, speaker audio.AudioPlayback, err error) {
	if cameras, err := groupListCameras(); err == nil && len(cameras) > 0 {
		webcam, err := groupNewCamera(0) // device ID 0 = default
		if err == nil {
			err = webcam.SetResolution(640, 480)
		}
		if err == nil {
			err = webcam.Open()
		}
		if err == nil {
			cam = webcam
			fmt.Println("📹 Using real camera:", cameras[0].Name)
		} else {
			fmt.Fprintf(os.Stderr, "⚠️  Camera unavailable, using test camera: %v\n", err)
		}
	}
	if cam == nil {
		test := device.NewTestCamera(640, 480, 15, "bounce")
		if err := test.Open(); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to open camera: %w", err)
		}
		cam = test
		fmt.Println("📹 Using test camera (no real camera detected)")
	}

	if src, err := groupNewCapture(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Microphone unavailable, not sending audio: %v\n", err)
	} else if err := src.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Microphone unavailable, not sending audio: %v\n", err)
	} else {
		mic = src
	}

	if sink, err := groupNewPlayback(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Speaker unavailable, remote audio will be dropped: %v\n", err)
	} else if err := sink.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Speaker unavailable, remote audio will be dropped: %v\n", err)
	} else {
		speaker = sink
	}

	return cam, mic, speaker, nil
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
