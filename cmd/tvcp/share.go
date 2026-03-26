package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/internal/screen"
	"github.com/svend4/infon/pkg/terminal"
)

func shareScreen(address string, command string) error {
	// Parse address
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	fmt.Printf("📺 Screen Sharing Session\n")
	fmt.Printf("   Remote: %s\n", address)
	fmt.Printf("   Command: %s\n", command)
	fmt.Printf("   Terminal: 40x24\n\n")

	// Create transport
	transport, err := network.NewTransport(":0") // Use any available local port
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}
	defer transport.Close()

	// Create screen share
	screenShare := screen.NewScreenShare(command, 40, 24)

	// Send handshake
	handshake := &network.Packet{
		Type:      network.PacketTypeHandshake,
		Sequence:  transport.NextSequence(),
		Timestamp: uint64(time.Now().UnixMilli()),
		Payload:   []byte("TVCP-SCREEN/1.0"),
	}
	if err := transport.SendPacket(handshake, udpAddr); err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	// Start screen sharing
	if err := screenShare.Start(); err != nil {
		return fmt.Errorf("failed to start screen sharing: %w", err)
	}
	defer screenShare.Stop()

	fmt.Println("✅ Screen sharing started")
	fmt.Println("   Press Ctrl+C to stop")
	fmt.Println()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Stats
	var framesSent uint64
	var bytesSent uint64
	startTime := time.Now()

	// Frame sending goroutine
	frameChan := screenShare.GetFrameChannel()
	doneChan := make(chan bool)

	go func() {
		for frame := range frameChan {
			// Fragment frame directly
			fragments, err := network.FragmentFrame(frame, uint32(framesSent))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fragmenting frame: %v\n", err)
				continue
			}

			// Send each fragment
			timestamp := uint64(time.Now().UnixMilli())
			for _, fragData := range fragments {
				packet := &network.Packet{
					Type:      network.PacketTypeScreen,
					Sequence:  transport.NextSequence(),
					Timestamp: timestamp,
					Payload:   fragData,
				}

				if err := transport.SendPacket(packet, udpAddr); err != nil {
					fmt.Fprintf(os.Stderr, "Error sending packet: %v\n", err)
					continue
				}

				bytesSent += uint64(len(fragData))
			}

			framesSent++

			// Display stats periodically
			if framesSent%15 == 0 { // Every second at 15 FPS
				duration := time.Since(startTime).Seconds()
				fps := float64(framesSent) / duration
				kbps := float64(bytesSent*8) / duration / 1000

				fmt.Printf("\r📊 Stats: %d frames | %.1f FPS | %.1f kbps | %s",
					framesSent, fps, kbps, formatDuration(time.Since(startTime)))
			}
		}
		doneChan <- true
	}()

	// Wait for interrupt
	<-sigChan
	fmt.Println("\n\n🛑 Stopping screen share...")

	screenShare.Stop()
	<-doneChan

	// Final stats
	duration := time.Since(startTime).Seconds()
	fmt.Printf("\n📊 Final Stats:\n")
	fmt.Printf("   Frames sent: %d\n", framesSent)
	fmt.Printf("   Duration: %s\n", formatDuration(time.Since(startTime)))
	fmt.Printf("   Average FPS: %.1f\n", float64(framesSent)/duration)
	fmt.Printf("   Total data: %.2f MB\n", float64(bytesSent)/1024/1024)
	fmt.Printf("   Average bitrate: %.1f kbps\n", float64(bytesSent*8)/duration/1000)

	return nil
}

func receiveScreen(port int) error {
	// Create UDP listener
	addr := fmt.Sprintf(":%d", port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer conn.Close()

	fmt.Printf("📺 Waiting for screen share on port %d...\n", port)
	fmt.Println("   Press Ctrl+C to stop")
	fmt.Println()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Frame assembly
	fragmentBuffer := make(map[uint32][][]byte)
	var framesReceived uint64
	var bytesReceived uint64
	startTime := time.Now()

	// Clear screen
	fmt.Print("\033[2J\033[H")

	// Receive loop
	buf := make([]byte, 2048)
	go func() {
		for {
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}

			bytesReceived += uint64(n)

			// Decode packet
			packet, err := network.Decode(buf[:n])
			if err != nil {
				continue
			}

			switch packet.Type {
			case network.PacketTypeHandshake:
				fmt.Printf("✅ Connected to: %s\n\n", string(packet.Payload))

			case network.PacketTypeScreen:
				// Parse fragment header
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
					// Assemble frame
					frame, err := network.AssembleFrame(fragmentBuffer[frameID])
					if err == nil {
						framesReceived++

						// Display frame
						displayScreenFrame(frame)

						// Show stats below frame
						duration := time.Since(startTime).Seconds()
						fps := float64(framesReceived) / duration
						kbps := float64(bytesReceived*8) / duration / 1000

						fmt.Printf("\n📊 %d frames | %.1f FPS | %.1f kbps\n",
							framesReceived, fps, kbps)
					}

					delete(fragmentBuffer, frameID)
				}
			}
		}
	}()

	// Wait for interrupt
	<-sigChan
	fmt.Println("\n\n🛑 Stopped")

	return nil
}

func displayScreenFrame(frame *terminal.Frame) {
	// Move cursor to top
	fmt.Print("\033[H")

	// Render frame
	for row := 0; row < frame.Height; row++ {
		for col := 0; col < frame.Width; col++ {
			block := frame.Blocks[row][col]

			// Set colors
			fmt.Printf("\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm",
				block.Fg.R, block.Fg.G, block.Fg.B,
				block.Bg.R, block.Bg.G, block.Bg.B)

			// Print character
			fmt.Print(string(block.Glyph))
		}
		fmt.Print("\033[0m\n")
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func handleShareCommand(args []string) error {
	if len(args) < 2 {
		fmt.Println("Usage: tvcp share <address> <command>")
		fmt.Println()
		fmt.Println("Share terminal output with a remote peer")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  tvcp share localhost:5000 \"tail -f /var/log/syslog\"")
		fmt.Println("  tvcp share [200:abc::1]:5000 \"htop\"")
		fmt.Println("  tvcp share 192.168.1.100:5000 \"npm run build\"")
		fmt.Println("  tvcp share localhost:5000 \"docker logs -f myapp\"")
		return nil
	}

	address := args[0]
	command := strings.Join(args[1:], " ")

	return shareScreen(address, command)
}

func handleReceiveScreenCommand(args []string) error {
	port := 5000
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &port)
	}

	return receiveScreen(port)
}
