package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/audio"
	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/contacts"
	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/internal/raysource"
	"github.com/svend4/infon/internal/recorder"
	"github.com/svend4/infon/internal/video"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func runCall() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tvcp call [options] <name|host:port> [pattern] [local-port]")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		fmt.Fprintln(os.Stderr, "  --record              Record the call")
		fmt.Fprintln(os.Stderr, "  --output <file>       Output file for recording (default: auto-generated)")
		fmt.Fprintln(os.Stderr, "  --ray                 Send a live CPU ray-traced 3-D world instead of the camera")
		fmt.Fprintln(os.Stderr, "  --ray-spp <n>         Ray-tracer samples per pixel axis (default 1; higher = cleaner, slower)")
		fmt.Fprintln(os.Stderr, "  --no-audio            Video-only call (skip microphone and speaker)")
		fmt.Fprintln(os.Stderr, "\nThis starts a two-way video call with text chat.")
		fmt.Fprintln(os.Stderr, "Type messages and press Enter to send text during the call.")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  tvcp call alice                          # Call contact 'alice'")
		fmt.Fprintln(os.Stderr, "  tvcp call --record alice                 # Record the call")
		fmt.Fprintln(os.Stderr, "  tvcp call --record --output my-call.tvcp alice")
		fmt.Fprintln(os.Stderr, "  tvcp call --ray alice                    # Send a ray-traced world as your video")
		fmt.Fprintln(os.Stderr, "  tvcp call localhost:5001 bounce          # Local test call")
		fmt.Fprintln(os.Stderr, "  tvcp call [200:abc::1]:5001              # Yggdrasil call")
		os.Exit(1)
	}

	// Parse flags
	var enableRecording bool
	var outputFile string
	var useRay bool
	var noAudio bool
	raySPP := 1
	args := os.Args[2:]
	var positionalArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--record" {
			enableRecording = true
		} else if arg == "--output" {
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++ // Skip next arg
			} else {
				fmt.Fprintln(os.Stderr, "Error: --output requires a filename")
				os.Exit(1)
			}
		} else if arg == "--ray" {
			useRay = true
		} else if arg == "--no-audio" {
			noAudio = true
		} else if arg == "--ray-spp" {
			if i+1 < len(args) {
				n, err := strconv.Atoi(args[i+1])
				if err != nil || n < 1 {
					fmt.Fprintln(os.Stderr, "Error: --ray-spp requires a positive integer")
					os.Exit(1)
				}
				raySPP = n
				useRay = true // specifying samples implies --ray
				i++           // Skip next arg
			} else {
				fmt.Fprintln(os.Stderr, "Error: --ray-spp requires a number")
				os.Exit(1)
			}
		} else if !strings.HasPrefix(arg, "--") {
			positionalArgs = append(positionalArgs, arg)
		} else {
			// A mistyped option (say --recrod) used to be dropped silently, so
			// the call went ahead unrecorded while the user thought otherwise.
			fmt.Fprintf(os.Stderr, "Error: unknown option %s\n", arg)
			os.Exit(1)
		}
	}
	// --output names the recording, so it implies --record (as --ray-spp implies --ray).
	if outputFile != "" {
		enableRecording = true
	}

	if len(positionalArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing address argument")
		os.Exit(1)
	}

	nameOrAddr := positionalArgs[0]
	pattern := "bounce"
	localPort := "5000"

	if len(positionalArgs) >= 2 {
		pattern = positionalArgs[1]
	}
	if len(positionalArgs) >= 3 {
		localPort = positionalArgs[2]
	}

	// Generate default output file if recording but no output specified
	if enableRecording && outputFile == "" {
		// Create recordings directory
		recordingsDir := filepath.Join(os.Getenv("HOME"), ".tvcp", "recordings")
		_ = os.MkdirAll(recordingsDir, 0755)

		// Generate filename: call-YYYYMMDD-HHMMSS.tvcp
		timestamp := time.Now().Format("20060102-150405")
		outputFile = filepath.Join(recordingsDir, fmt.Sprintf("call-%s.tvcp", timestamp))
	}

	// Try to resolve from contacts if it's a name (no colons or brackets)
	remoteAddr := nameOrAddr
	resolvedFromContact := false
	if !strings.Contains(nameOrAddr, ":") && !strings.Contains(nameOrAddr, "[") {
		// Looks like a contact name, try to resolve
		cb, err := contacts.GetDefaultContactBook()
		if err == nil {
			if resolvedAddr, err := cb.Resolve(nameOrAddr); err == nil {
				// Contacts are stored as bare Yggdrasil IPv6 addresses. Use
				// net.JoinHostPort, which brackets IPv6 correctly: string-appending
				// ":5000" gave "200:abc::1:5000", which the normalisation below then
				// wrapped as "[200:abc::1:5000]:5000" — a syntactically valid but
				// different address (the "::" absorbs the extra group), so the call
				// silently dialled the wrong host. resolvedPort marks it already final.
				remoteAddr = net.JoinHostPort(resolvedAddr, "5000")
				resolvedFromContact = true
				fmt.Printf("📇 Resolved '%s' to %s\n", nameOrAddr, resolvedAddr)

				// Update last seen (ignore error as it's not critical)
				_ = cb.UpdateLastSeen(nameOrAddr)
			}
		}
	}

	// Add default port if not specified (skipped when the contact book already
	// produced a host:port via net.JoinHostPort).
	if resolvedFromContact {
		// already host:port, leave it
	} else if !strings.Contains(remoteAddr, "]:") && strings.Count(remoteAddr, ":") == 0 {
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
	defer func() { _ = transport.Close() }()

	fmt.Printf("Listening on: %s\n", transport.LocalAddr())
	fmt.Println("💬 Type messages and press Enter to send text during the call")
	fmt.Println()

	// Create camera for local video
	fps := 15.0
	var camera device.Camera

	if useRay {
		// Send a live CPU ray-traced 3-D world instead of a webcam. raysource is a
		// device.FrameGenerator; GenerativeSource adapts it to device.Camera, so it
		// is a drop-in source for the same capture/encode/send path.
		camera = device.NewGenerativeSource(640, 480, fps, raysource.New(nil, raySPP))
		fmt.Printf("🎥 Using ray-traced 3-D world as camera (spp=%d)\n", raySPP)
	}

	// Try to use real camera first (V4L2 on Linux)
	if camera == nil {
		cameras, err := device.ListCameras()
		if err == nil && len(cameras) > 0 {
			// Real camera available
			realCamera, err := device.NewCamera(cameras[0].ID) // Use first camera
			if err == nil {
				camera = realCamera
				fmt.Println("📹 Using real camera:", cameras[0].Name)
			}
		}
	}

	// Fall back to test camera if no real camera
	if camera == nil {
		camera = device.NewTestCamera(640, 480, fps, pattern)
		fmt.Println("📹 Using test camera (no real camera detected)")
	}

	if err = camera.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening camera: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = camera.Close() }()

	// Audio: microphone capture and speaker playback. Headless hosts (SSH
	// sessions, servers, containers) often have no sound card at all; rather than
	// refusing to start, the call then runs video-only, as it does with --no-audio.
	audioFormat := audio.DefaultFormat()
	var audioSource audio.AudioCapture
	var audioSink audio.AudioPlayback
	if !noAudio {
		src, err := audio.NewDefaultCapture()
		if err == nil {
			err = src.Open()
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Audio capture unavailable, continuing video-only: %v\n", err)
		} else {
			audioSource = src
			defer func() { _ = audioSource.Close() }()
		}
		sink, err := audio.NewDefaultPlayback()
		if err == nil {
			err = sink.Open()
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Audio playback unavailable, remote audio will be dropped: %v\n", err)
		} else {
			audioSink = sink
			defer func() { _ = audioSink.Close() }()
		}
	}
	if audioSource == nil && audioSink == nil {
		fmt.Println("Audio: disabled (video-only call)")
	} else {
		fmt.Printf("Audio: %d Hz, %d channels, %d-bit\n",
			audioFormat.SampleRate, audioFormat.Channels, audioFormat.BitDepth)
	}

	// Send handshake
	fmt.Println("Sending handshake...")
	handshake := &network.Packet{
		Type:      network.PacketTypeHandshake,
		Sequence:  transport.NextSequence(),
		Timestamp: uint64(time.Now().UnixMilli()),
		Payload:   []byte("TVCP/1.0"),
	}
	_ = transport.SendPacket(handshake, udpAddr)

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

	// P-frame decoder for receiving
	pframeDecoder := video.NewPFrameDecoder()

	// Loss recovery components
	lossDetector := network.NewLossDetector()
	retransmitter := network.NewRetransmissionManager()

	// Jitter buffer for smooth audio playback
	audioJitterBuffer := network.NewJitterBuffer(50) // 50 packets = ~1 second @ 50 chunks/s

	// Quality controller for adaptive bitrate
	qualityController := network.NewQualityController(15) // Start at 15 FPS
	qualityChanges := 0
	qualityController.SetQualityChangeCallback(func(fps, width, height int) {
		qualityChanges++
		fmt.Printf("\n📊 Quality adjusted: %d FPS (network conditions)\n", fps)
	})

	// Channel for audio packets from jitter buffer
	audioPlaybackChan := make(chan *network.AudioPacket, 10)

	// done stops the capture, encode and playback goroutines at hang-up; the main
	// loop waits for them (wg) before printing stats, stopping the recorder and
	// letting the deferred device Close calls run, so no goroutine is still
	// reading the camera or writing the speaker while it is being closed.
	done := make(chan struct{})
	var wg sync.WaitGroup

	// localPreview is the sender's latest local frame. The split-screen preview
	// renders it instead of reading the camera a second time from the receive
	// loop, which split the camera's frames between two readers and blocked the
	// receive loop on the capture device.
	var localPreview atomic.Pointer[terminal.Frame]

	// Terminal dimensions for split-screen
	// Top half: remote video (40 cols × 12 rows)
	// Bottom half: local video (40 cols × 12 rows)
	remoteWidth, remoteHeight := 40, 12
	localWidth, localHeight := 40, 12

	fmt.Println("Call connected! Press Ctrl+C to hang up")
	fmt.Print(color.ClearScreen)

	// Goroutine to pull packets from jitter buffer
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(audioPlaybackChan)                  // ends the playback goroutine
		ticker := time.NewTicker(10 * time.Millisecond) // Check every 10ms
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
			}
			// Try to get next packet from jitter buffer
			if packet := audioJitterBuffer.Get(); packet != nil {
				// Decode audio packet
				audioPacket, err := network.DecodeAudioPacket(packet.Payload)
				if err == nil {
					select {
					case audioPlaybackChan <- audioPacket:
					case <-done:
						return
					}
				}
			}
		}
	}()

	// Goroutine to play audio from jitter buffer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for audioPacket := range audioPlaybackChan {
			if len(audioPacket.Samples) > 0 && audioSink != nil {
				_, _ = audioSink.Write(audioPacket.Samples)

				mu.Lock()
				audioRecvCount++
				mu.Unlock()
			}
		}
	}()

	// Initialize recorder if enabled
	var rec *recorder.Recorder
	if enableRecording {
		rec = recorder.NewRecorder(localWidth, localHeight, audioFormat.SampleRate)
		if err := rec.Start(outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting recording: %v\n", err)
		} else {
			fmt.Printf("🔴 Recording to: %s\n", outputFile)
		}
	}

	// Initialize P-frame encoder (I-frame every 30 frames = ~2 seconds @ 15 FPS)
	pframeEncoder := video.NewPFrameEncoder(30)

	// Goroutine for sending (local video)
	wg.Add(1)
	go func() {
		defer wg.Done()
		lastFrameTime := time.Now()

		for {
			select {
			case <-done:
				return
			default:
			}
			// Get current FPS from quality controller
			currentFPS := qualityController.GetCurrentFPS()
			frameInterval := time.Duration(1000.0/float64(currentFPS)) * time.Millisecond

			// Wait for next frame time
			elapsed := time.Since(lastFrameTime)
			if elapsed < frameInterval {
				time.Sleep(frameInterval - elapsed)
			}
			lastFrameTime = time.Now()

			// Capture frame
			img, err := camera.Read()
			if err != nil {
				continue
			}

			// Encode to terminal frame
			frame := babe.ImageToFrame(img, localWidth, localHeight)
			localPreview.Store(frame)

			// Record frame if recording
			if rec != nil && rec.IsRecording() {
				_ = rec.RecordFrame(frame)
			}

			// Encode with P-frame compression
			encodedFrame := pframeEncoder.Encode(frame)

			// Serialize encoded frame
			encodedData, err := network.EncodeEncodedFrame(encodedFrame)
			if err != nil {
				continue
			}

			// Fragment encoded frame
			fragments, err := network.FragmentData(encodedData, uint32(sendCount))
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
				_ = transport.SendPacket(packet, udpAddr)

				// Store for potential retransmission
				retransmitter.OnPacketSent(packet, udpAddr)
			}

			mu.Lock()
			sendCount++
			mu.Unlock()
		}
	}()

	// Voice Activity Detection
	vad := audio.NewVAD(audioFormat.SampleRate)
	vad.SetSensitivity(0.7) // 0.7 = balanced sensitivity

	// Noise Suppression
	noiseSuppressor := audio.NewNoiseSuppressor(audioFormat.SampleRate, audioFormat.Channels)
	if err := noiseSuppressor.SetLevel(2); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to set noise suppression level: %v\n", err)
	}

	// Echo Cancellation
	// Note: Full echo cancellation requires speaker loopback capture
	// This is prepared for future integration when loopback is available
	echoCanceller := audio.NewEchoCanceller(audioFormat.SampleRate, audioFormat.SampleRate/50)
	echoCanceller.SetStepSize(0.01) // Conservative adaptation
	_ = echoCanceller               // Prepared for future use

	// Goroutine for sending audio (skipped when there is no capture device)
	if audioSource != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 20ms audio chunks (good balance between latency and efficiency)
			samplesPerChunk := audioFormat.SampleRate / 50 // 50 chunks per second = 20ms each
			buffer := make([]int16, samplesPerChunk)

			ticker := time.NewTicker(20 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-ticker.C:
				}
				// Read audio samples
				n, err := audioSource.Read(buffer)
				if err != nil || n == 0 {
					continue
				}

				// Apply noise suppression
				enhanced, _ := noiseSuppressor.Process(buffer[:n])

				// Record audio if recording (record original, not enhanced)
				if rec != nil && rec.IsRecording() {
					_ = rec.RecordAudio(buffer[:n])
				}

				// Voice Activity Detection - only send if speech detected
				isSpeaking := vad.Process(enhanced)
				if !isSpeaking {
					// Skip sending during silence to save bandwidth
					continue
				}

				// Create audio packet with enhanced audio
				audioPacket := &network.AudioPacket{
					Timestamp:  uint64(time.Now().UnixMilli()),
					SampleRate: uint16(audioFormat.SampleRate),
					Channels:   uint8(audioFormat.Channels),
					Codec:      network.AudioCodecPCM,
					Samples:    enhanced,
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

				_ = transport.SendPacket(packet, udpAddr)
				retransmitter.OnPacketSent(packet, udpAddr)

				mu.Lock()
				audioSendCount++
				mu.Unlock()
			}
		}()
	}

	// Channel for user text messages
	textMsgChan := make(chan string, 10)

	// Goroutine for sending text messages (user input)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			message := strings.TrimSpace(scanner.Text())
			if message != "" {
				textMsgChan <- message
			}
		}
	}()

	// Goroutine to process and send text messages
	go func() {
		username := "User"
		if hostname, err := os.Hostname(); err == nil {
			username = hostname
		}

		for message := range textMsgChan {
			// Create text message
			textMsg := network.NewTextMessage(username, message)
			payload, err := network.EncodeTextMessage(textMsg)
			if err != nil {
				continue
			}

			// Send packet
			packet := &network.Packet{
				Type:      network.PacketTypeTextChat,
				Sequence:  transport.NextSequence(),
				Timestamp: uint64(time.Now().UnixMilli()),
				Payload:   payload,
			}

			_ = transport.SendPacket(packet, udpAddr)
			retransmitter.OnPacketSent(packet, udpAddr)

			// Display sent message
			fmt.Printf("\n📤 You: %s\n", message)
		}
	}()

	// Main loop for receiving (remote video + audio)
	lastStatsTime := time.Now()

	for {
		select {
		case <-sigChan:
			// Stop capture/playback first so the stats are final, the recorder is
			// stopped after its last write, and the deferred Close calls run on idle
			// devices. A device read stuck in the kernel must not hang Ctrl+C, so
			// give up waiting after a short grace period.
			close(done)
			stopped := make(chan struct{})
			go func() { wg.Wait(); close(stopped) }()
			select {
			case <-stopped:
			case <-time.After(2 * time.Second):
			}

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

			// Show jitter buffer statistics
			jitterStats := audioJitterBuffer.GetStatistics()
			fmt.Printf("\nJitter Buffer:\n")
			fmt.Printf("  Buffer size: %d packets\n", jitterStats.BufferSize)
			fmt.Printf("  Total buffered: %d\n", jitterStats.TotalBuffered)
			fmt.Printf("  Total played: %d\n", jitterStats.TotalOutput)
			fmt.Printf("  Dropped: %d\n", jitterStats.Dropped)
			fmt.Printf("  Underruns: %d\n", jitterStats.Underruns)
			fmt.Printf("  Current delay: %dms\n", jitterStats.CurrentDelay.Milliseconds())

			// Show VAD statistics
			vadStats := vad.GetStatistics()
			fmt.Printf("\nVoice Activity Detection:\n")
			fmt.Printf("  Total frames: %d\n", vadStats.TotalFrames)
			fmt.Printf("  Speech frames: %d (%.1f%%)\n", vadStats.SpeechFrames, vadStats.ActivityRate)
			fmt.Printf("  Silence frames: %d (%.1f%%)\n", vadStats.SilenceFrames, 100.0-vadStats.ActivityRate)
			fmt.Printf("  Bandwidth saved: ~%.1f%%\n", 100.0-vadStats.ActivityRate)

			// Show noise suppression statistics
			// TODO: Re-enable when NoiseSuppressor has GetStatistics method
			// nsStats := noiseSuppressor.GetStatistics()
			// fmt.Printf("\nNoise Suppression:\n")
			// fmt.Printf("  Total frames: %d\n", nsStats.TotalFrames)
			// fmt.Printf("  Clean frames: %d (%.1f%%)\n", nsStats.CleanFrames, nsStats.CleanRatio)
			// fmt.Printf("  Noisy frames: %d (%.1f%%)\n", nsStats.NoisyFrames, 100.0-nsStats.CleanRatio)
			// fmt.Printf("  Calibrated: %v\n", nsStats.Calibrated)

			// Show echo cancellation statistics (when speaker loopback available)
			ecStats := echoCanceller.GetStatistics()
			if ecStats.TotalFrames > 0 {
				fmt.Printf("\nEcho Cancellation:\n")
				fmt.Printf("  Total frames: %d\n", ecStats.TotalFrames)
				fmt.Printf("  Echo detected: %d (%.1f%%)\n", ecStats.EchoDetected, ecStats.DetectionRate)
				fmt.Printf("  Echo suppressed: %d\n", ecStats.EchoSuppressed)
				fmt.Printf("  Avg reduction: %.1f%%\n", ecStats.AvgReduction)
				fmt.Printf("  Filter converged: %v\n", ecStats.FilterConverged)
			}

			// Stop and save recording
			if rec != nil && rec.IsRecording() {
				fmt.Printf("\n💾 Saving recording...\n")
				if err := rec.Stop(); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving recording: %v\n", err)
				} else {
					frameCount, audioCount, duration := rec.GetStats()
					fmt.Printf("✓ Recording saved: %s\n", outputFile)
					fmt.Printf("  Duration: %.1fs\n", duration.Seconds())
					fmt.Printf("  Video frames: %d\n", frameCount)
					fmt.Printf("  Audio chunks: %d\n", audioCount)
				}
			}

			// Show adaptive quality statistics
			if qualityChanges > 0 {
				qualityStats := qualityController.GetStatistics()
				fmt.Printf("\nAdaptive Quality:\n")
				fmt.Printf("  Quality changes: %d\n", qualityChanges)
				fmt.Printf("  Final FPS: %d\n", qualityStats.CurrentFPS)
				fmt.Printf("  Average loss: %.2f%%\n", qualityStats.AverageLoss)
			}

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

			// Track every packet for loss detection. The peer draws one sequence
			// counter for video, audio, text and NACK, so feeding the detector only
			// video sequences made every audio/text/NACK packet look like a gap: the
			// detector's expectedSeq then stuck, each later frame re-reported the whole
			// range as lost, and the call showed ~90% loss with a NACK storm that
			// throttled FPS to the floor. Retransmission requests stay keyed on the
			// shared sequence, which is what the sender stored.
			isNew := lossDetector.OnPacketReceived(packet.Sequence)
			if !isNew {
				// Duplicate packet, skip
				continue
			}
			lostPackets := lossDetector.GetLostPackets()
			if len(lostPackets) > 0 {
				nackPacket := network.CreateNACKPacket(lostPackets, transport.NextSequence())
				_ = transport.SendPacket(nackPacket, udpAddr)
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
								_ = transport.SendPacket(retransPacket, addr)
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

				frameID, fragID, totalFrags, ok := network.ParseFragmentHeader(packet.Payload, -1)
				if !ok {
					continue
				}
				if buf := fragmentBuffer[frameID]; buf != nil && len(buf) != int(totalFrags) {
					continue
				}

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
					// Assemble encoded frame data
					encodedData, err := network.AssembleData(fragmentBuffer[frameID])
					if err == nil {
						// Decode encoded frame (I-frame or P-frame)
						encodedFrame, err := network.DecodeEncodedFrame(encodedData)
						if err == nil {
							// Decode with P-frame decoder
							remoteFrame := pframeDecoder.Decode(encodedFrame)

							// Render split-screen: remote on top, local on bottom
							renderSplitScreen(localPreview.Load(), remoteFrame, localWidth, localHeight, remoteWidth, remoteHeight)

							mu.Lock()
							recvCount++
							mu.Unlock()
						}
					}

					delete(fragmentBuffer, frameID)
				}

			case network.PacketTypeAudio:
				// Add audio packet to jitter buffer for smooth playback
				audioJitterBuffer.Add(packet)

			case network.PacketTypeTextChat:
				// Decode text message
				textMsg, err := network.DecodeTextMessage(packet.Payload)
				if err != nil {
					continue
				}

				// Display message (will appear in terminal output)
				sender, message := chatLine(textMsg)
				fmt.Printf("\n💬 [%s] %s: %s\n", textMsg.FormatTime(), sender, message)
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
				jitterStats := audioJitterBuffer.GetStatistics()
				vadStats := vad.GetStatistics()

				// Update quality controller with network stats
				qualityController.UpdateNetworkStats(lossStats.LossRate, jitterStats.CurrentDelay)

				// Get current quality level
				quality := qualityController.GetCurrentQuality()

				// Determine VAD indicator
				vadIndicator := "🔇"
				if vadStats.IsSpeaking {
					vadIndicator = "🎤"
				}

				fmt.Printf("%s[Call] Video: %d/%d (%.1f/%.1f FPS → %d FPS) | Audio: %d/%d %s (VAD: %.1f%%) | Loss: %.1f%% | Time: %.0fs%s\n",
					color.Reset, sc, rc, float64(sc)/elapsed.Seconds(), float64(rc)/elapsed.Seconds(),
					quality.FPS, asc, arc, vadIndicator, vadStats.ActivityRate, lossStats.LossRate, elapsed.Seconds(), color.Reset)
				lastStatsTime = now
			}
		}
	}
}

// renderSplitScreen renders remote video on top, local video preview on bottom
// renderSplitScreen draws the remote frame above the local preview. localFrame is
// the sender's latest frame and is nil until the first one is captured.
func renderSplitScreen(localFrame *terminal.Frame, remoteFrame *terminal.Frame, localW, localH, remoteW, remoteH int) {
	// Clear screen
	fmt.Print(color.ClearScreen)

	// Render remote frame (top half)
	fmt.Println("Remote:")
	for y := 0; y < remoteFrame.Height; y++ {
		for x := 0; x < remoteFrame.Width; x++ {
			block := remoteFrame.Blocks[y][x]
			fmt.Print(block.Fg.FgString())
			fmt.Print(block.Bg.BgString())
			fmt.Printf("%c", terminal.SanitizeGlyph(block.Glyph))
		}
		fmt.Print(color.Reset)
		fmt.Println()
	}

	// Separator
	fmt.Println(color.Reset + "────────────────────────────────────────")

	// Render local preview (bottom half)
	fmt.Println("Local:")
	if localFrame != nil {
		for y := 0; y < localFrame.Height; y++ {
			for x := 0; x < localFrame.Width; x++ {
				block := localFrame.Blocks[y][x]
				fmt.Print(block.Fg.FgString())
				fmt.Print(block.Bg.BgString())
				fmt.Printf("%c", terminal.SanitizeGlyph(block.Glyph))
			}
			fmt.Print(color.Reset)
			fmt.Println()
		}
	}

	fmt.Print(color.Reset)
}
