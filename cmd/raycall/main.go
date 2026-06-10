// Command raycall is the semantic video call, end to end, over a real (loopback)
// UDP socket: it joins the pieces this project built separately —
//
//	camera -> vision detections -> rayscene (SceneFromDetections)
//	       -> scene deltas (DiffScene/ApplyScene) -> the wire (reliable, acked)
//	       -> reconstruct -> ray-trace at the receiver
//
// The sender turns each camera frame into a tiny rayscene P-frame and ships it with
// stop-and-wait acks (the reliable "region" transport raymeet uses); the receiver
// rebuilds the whole scene from the deltas and renders it locally. Detections are
// synthesized here (a subject moving across frame); set VISION_API_URL to drive the
// scene from a real vision model instead. -drop simulates packet loss to show the
// acks recover it.
//
//	go run ./cmd/raycall -frames 48
//	go run ./cmd/raycall -frames 48 -drop 0.3
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/internal/vision"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out    = flag.String("out", "call", "output basename")
		frames = flag.Int("frames", 48, "frames in the call")
		w      = flag.Int("w", 360, "render width")
		h      = flag.Int("h", 240, "render height")
		spp    = flag.Int("spp", 56, "samples per pixel")
		drop   = flag.Float64("drop", 0, "simulate packet loss 0..1 (acks/retransmit recover it)")
	)
	flag.Parse()

	recv, err := network.NewTransport("127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "recv:", err)
		os.Exit(1)
	}
	defer func() { _ = recv.Close() }()
	send, err := network.NewTransport("127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "send:", err)
		os.Exit(1)
	}
	defer func() { _ = send.Close() }()
	recvAddr := recv.LocalAddr()

	// receiver: rebuild the scene from keyframe + deltas, acking each packet.
	var cur brain.SceneSpec
	got := make([]brain.SceneSpec, 0, *frames)
	seen := map[uint32]bool{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for len(got) < *frames {
			_ = recv.GetConn().SetReadDeadline(time.Now().Add(5 * time.Second))
			pkt, from, err := recv.ReceivePacket()
			if err != nil {
				return // timed out — the call ended or stalled
			}
			if pkt.Type != network.PacketTypeScreen || len(pkt.Payload) < 1 {
				continue
			}
			if !seen[pkt.Sequence] {
				seen[pkt.Sequence] = true
				body := pkt.Payload[1:]
				if pkt.Payload[0] == 'K' {
					_ = json.Unmarshal(body, &cur)
				} else {
					var d raydir.SceneDelta
					if json.Unmarshal(body, &d) == nil {
						cur = raydir.ApplyScene(cur, d)
					}
				}
				got = append(got, cur)
			}
			_ = recv.SendPacket(&network.Packet{Type: network.PacketTypeControl, Sequence: pkt.Sequence}, from) // ack
		}
	}()

	// sender: camera -> detections -> rayscene -> delta -> reliable send.
	an, hasVision := vision.NewHTTPAnalyzerFromEnv()
	rng := rand.New(rand.NewSource(1))
	var prev brain.SceneSpec
	wire, keys := 0, 0
	for f := 0; f < *frames; f++ {
		spec := frameScene(f, *frames, an, hasVision)
		var payload []byte
		if f == 0 {
			b, _ := json.Marshal(spec)
			payload, keys = append([]byte{'K'}, b...), keys+1
		} else if d, ok := raydir.DiffScene(prev, spec); ok {
			b, _ := json.Marshal(d)
			payload = append([]byte{'D'}, b...)
		} else {
			b, _ := json.Marshal(spec)
			payload, keys = append([]byte{'K'}, b...), keys+1
		}
		wire += len(payload)
		sendReliable(send, recvAddr, payload, *drop, rng)
		prev = spec
	}
	select {
	case <-done:
	case <-time.After(8 * time.Second):
	}
	if len(got) < *frames {
		fmt.Fprintf(os.Stderr, "warning: receiver got %d/%d frames\n", len(got), *frames)
		os.Exit(1)
	}

	// render the receiver's reconstruction of the first and a mid frame (fidelity).
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.4, Z: -3.5}, Pitch: -0.12, FOV: math.Pi / 3}
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true}
	imgA := raytrace.PathRender(raydir.BuildScene(got[0]), cam, *w, *h, opt)
	imgB := raytrace.PathRender(raydir.BuildScene(got[len(got)/2]), cam, *w, *h, opt)

	pixelEach := pngBytes(imgA)
	fmt.Printf("semantic call: %d frames %s -> %s (UDP)\n", *frames, send.LocalAddr(), recvAddr)
	fmt.Printf("  on the wire: %d B of scene deltas (%d keyframe, %d deltas)\n", wire, keys, *frames-keys)
	fmt.Printf("  vs pixels (~%d B/frame): %d B -> %.0fx smaller\n", pixelEach, pixelEach**frames, float64(pixelEach**frames)/float64(maxi(wire, 1)))
	if *drop > 0 {
		fmt.Printf("  with %.0f%% simulated loss: acks + retransmit delivered all %d frames\n", *drop*100, len(got))
	}

	writePNG(*out+".png", montage(imgA, "received frame 0", imgB, fmt.Sprintf("received frame %d", len(got)/2)))
	fmt.Printf("wrote %s.png\n", *out)
}

// frameScene turns frame f's detections into a rayscene; a subject moves across the
// frame. Real detections via VISION_API_URL, else synthesized.
func frameScene(f, frames int, an *vision.HTTPAnalyzer, hasVision bool) brain.SceneSpec {
	if hasVision {
		if ds, err := an.Analyze(context.Background(), cameraFrame(f)); err == nil && len(ds) > 0 {
			return raydir.SceneFromDetections(ds)
		}
	}
	t := float64(f) / float64(frames)
	dets := []vision.Detection{
		{Label: "tree", Confidence: 0.92, X: 0.04, Y: 0.18, W: 0.16, H: 0.6},
		{Label: "person", Confidence: 0.95, X: 0.42, Y: 0.46, W: 0.10, H: 0.40},
		{Label: "car", Confidence: 0.90, X: 0.04 + t*0.78, Y: 0.6, W: 0.18, H: 0.16}, // drives across
	}
	return raydir.SceneFromDetections(dets)
}

// sendReliable ships a payload stop-and-wait: send, await the ack for this sequence,
// retransmit on timeout (up to a few tries). With drop>0 it randomly "loses" a
// transmission to exercise the recovery.
func sendReliable(send *network.Transport, addr *net.UDPAddr, payload []byte, drop float64, rng *rand.Rand) {
	seq := send.NextSequence()
	pkt := &network.Packet{Type: network.PacketTypeScreen, Sequence: seq, Timestamp: uint64(time.Now().UnixMilli()), Payload: payload}
	conn := send.GetConn()
	buf := make([]byte, network.MaxPacketSize)
	// Read acks straight off the conn so our short retransmit timeout applies
	// (Transport.ReceivePacket would reset the deadline to 5s).
	for try := 0; try < 30; try++ {
		if drop <= 0 || rng.Float64() >= drop { // maybe "lose" this transmission
			_ = send.SendPacket(pkt, addr)
		}
		_ = conn.SetReadDeadline(time.Now().Add(120 * time.Millisecond))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue // timeout: retransmit
		}
		if ack, derr := network.Decode(buf[:n]); derr == nil && ack.Type == network.PacketTypeControl && ack.Sequence == seq {
			return
		}
	}
}

// cameraFrame is a small synthetic frame (used only when a real vision model is
// attached): a moving bright patch on grey.
func cameraFrame(f int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 160, 120))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 90, G: 92, B: 96, A: 255}}, image.Point{}, draw.Src)
	x := 10 + (f*4)%130
	for y := 50; y < 80; y++ {
		for dx := 0; dx < 24; dx++ {
			img.SetRGBA(x+dx, y, color.RGBA{R: 220, G: 200, B: 160, A: 255})
		}
	}
	return img
}

func montage(a image.Image, la string, b image.Image, lb string) image.Image {
	pw, ph := a.Bounds().Dx(), a.Bounds().Dy()
	const gap, labelH = 8, 16
	out := image.NewRGBA(image.Rect(0, 0, 2*pw+gap, ph+labelH))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(out, image.Rect(0, labelH, pw, labelH+ph), a, a.Bounds().Min, draw.Src)
	draw.Draw(out, image.Rect(pw+gap, labelH, 2*pw+gap, labelH+ph), b, b.Bounds().Min, draw.Src)
	white := color.RGBA{R: 230, G: 230, B: 235, A: 255}
	microfont.Draw(out, 4, 3, 1, la, white)
	microfont.Draw(out, pw+gap+4, 3, 1, lb, white)
	return out
}

func pngBytes(img image.Image) int {
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Len()
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
