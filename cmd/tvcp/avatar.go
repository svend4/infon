package main

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg" // register JPEG decoder for AVATAR_REF
	_ "image/png"  // register PNG decoder for AVATAR_REF
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/avatar"
	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/device"
	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/pkg/color"
)

// runAvatar drives the neural-avatar transport (roadmap C5 over the network):
// the sender extracts face keypoints from each camera frame and ships only those
// (~hundreds of bytes), while the receiver reconstructs the face from keypoints
// + a one-time reference image. This is "a talking face for kilobits/sec".
//
// Usage:
//
//	tvcp avatar send <host:port>     # camera -> landmarks -> wire
//	tvcp avatar receive [port]       # wire -> reconstructed face -> terminal
//
// Required env:
//
//	AVATAR_LM_URL    landmark sidecar (sender)   e.g. http://127.0.0.1:8097/
//	AVATAR_GEN_URL   generator sidecar (receiver) e.g. http://127.0.0.1:8098/
//	AVATAR_REF       optional path to a reference face image (receiver)
func runAvatar() {
	if len(os.Args) < 3 {
		printAvatarUsage()
		os.Exit(1)
	}
	switch os.Args[2] {
	case "send":
		avatarSend()
	case "receive", "recv":
		avatarReceive()
	default:
		printAvatarUsage()
		os.Exit(1)
	}
}

func printAvatarUsage() {
	fmt.Fprintln(os.Stderr, "Usage: tvcp avatar <send|receive> [args]")
	fmt.Fprintln(os.Stderr, "\nNeural-avatar transport (ultra-low bandwidth talking face):")
	fmt.Fprintln(os.Stderr, "  tvcp avatar send <host:port>   # AVATAR_LM_URL must be set")
	fmt.Fprintln(os.Stderr, "  tvcp avatar receive [port]     # AVATAR_GEN_URL must be set")
	fmt.Fprintln(os.Stderr, "\nStart the sidecars first:")
	fmt.Fprintln(os.Stderr, "  python ai/adapters/avatar_landmark_sidecar.py 8097")
	fmt.Fprintln(os.Stderr, "  python ai/adapters/avatar_generate_sidecar.py 8098")
}

func avatarSend() {
	if len(os.Args) < 4 {
		printAvatarUsage()
		os.Exit(1)
	}
	remoteAddr := os.Args[3]
	lmURL := os.Getenv("AVATAR_LM_URL")
	if lmURL == "" {
		fmt.Fprintln(os.Stderr, "AVATAR_LM_URL is required (landmark sidecar).")
		os.Exit(1)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving address: %v\n", err)
		os.Exit(1)
	}
	transport, err := network.NewTransport(":0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating transport: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = transport.Close() }()

	fps := 15.0
	cam := device.NewTestCamera(640, 480, fps, "bounce") // any device.Camera
	if err := cam.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening camera: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = cam.Close() }()

	ext := avatar.NewExtractor(lmURL)
	fmt.Println("🧑 TVCP Avatar Send")
	fmt.Printf("Remote: %s  | landmarks: %s\n", remoteAddr, lmURL)
	fmt.Println("Streaming keypoints... Press Ctrl+C to stop")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	ticker := time.NewTicker(time.Duration(1000.0/fps) * time.Millisecond)
	defer ticker.Stop()

	var frameNum uint32
	var totalBytes int
	start := time.Now()
	for {
		select {
		case <-sig:
			elapsed := time.Since(start).Seconds()
			fmt.Printf("\n✓ Avatar send stopped. Frames: %d, %d bytes total (%.2f kbps)\n",
				frameNum, totalBytes, float64(totalBytes*8)/1000.0/elapsed)
			return
		case <-ticker.C:
			img, err := cam.Read()
			if err != nil {
				continue
			}
			kp, err := ext.Extract(context.Background(), img, frameNum)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\rextract error: %v", err)
				continue
			}
			payload := kp.Encode()
			pkt := &network.Packet{
				Type:      network.PacketTypeAvatar,
				Sequence:  transport.NextSequence(),
				Timestamp: uint64(time.Now().UnixMilli()),
				Payload:   payload,
			}
			if err := transport.SendPacket(pkt, udpAddr); err != nil {
				fmt.Fprintf(os.Stderr, "\rsend error: %v", err)
				continue
			}
			totalBytes += len(payload)
			frameNum++
		}
	}
}

func avatarReceive() {
	port := "5000"
	if len(os.Args) >= 4 {
		port = os.Args[3]
	}
	genURL := os.Getenv("AVATAR_GEN_URL")
	if genURL == "" {
		fmt.Fprintln(os.Stderr, "AVATAR_GEN_URL is required (generator sidecar).")
		os.Exit(1)
	}

	// Optional reference face for the reconstructor.
	var ref image.Image
	if p := os.Getenv("AVATAR_REF"); p != "" {
		if f, err := os.Open(p); err == nil {
			ref, _, _ = image.Decode(f)
			_ = f.Close()
		}
	}
	rc, err := avatar.NewReconstructor(genURL, ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating reconstructor: %v\n", err)
		os.Exit(1)
	}

	transport, err := network.NewTransport(":" + port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating transport: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = transport.Close() }()

	fmt.Println("🧑 TVCP Avatar Receive")
	fmt.Printf("Listening on :%s | generator: %s\n", port, genURL)
	fmt.Println("Waiting for keypoints... Press Ctrl+C to stop")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// Render received reconstructions in a background goroutine fed by a channel,
	// so a slow generator never blocks packet reception.
	frames := make(chan avatar.Keypoints, 4)
	go func() {
		termW, termH := 80, 24
		fmt.Print(color.ClearScreen)
		for kp := range frames {
			face, err := rc.Reconstruct(context.Background(), kp)
			if err != nil {
				continue
			}
			babe.ImageToFrame(face, termW, termH).RenderToTerminal()
		}
	}()

	count := 0
	for {
		select {
		case <-sig:
			close(frames)
			fmt.Printf("\n✓ Avatar receive stopped. Reconstructed %d frames.\n", count)
			return
		default:
			pkt, _, err := transport.ReceivePacket()
			if err != nil {
				continue
			}
			if pkt.Type != network.PacketTypeAvatar {
				continue
			}
			kp, err := avatar.DecodeKeypoints(pkt.Payload)
			if err != nil {
				continue
			}
			select {
			case frames <- kp: // hand off to the renderer
				count++
			default: // renderer busy: drop this frame (favor freshness)
			}
		}
	}
}
