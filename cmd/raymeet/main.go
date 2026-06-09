// Command raymeet is the shared-world experience: two people (or a person and an
// AI) "call in" and walk the SAME 3-D world together, in the terminal, seeing each
// other's avatar. The trick, true to the project, is that the world is never sent
// over the wire: both sides derive it identically from a common script (the AI
// director's region sequence), so the only thing exchanged is a 40-byte pose each
// tick — meaning, not pixels. Walk forward and the world unfolds ahead on both
// machines in lock-step.
//
//	# terminal A (listens on 5000, talks to B on 5001)
//	go run ./cmd/raymeet localhost:5001 5000
//	# terminal B
//	go run ./cmd/raymeet localhost:5000 5001
//
// Controls (type then Enter): w/s walk, a/d strafe, q/e turn, r/f look, x quit.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"math"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
	"github.com/svend4/infon/pkg/terminal"
)

func main() {
	modeFlag := flag.String("mode", "auto", "terminal render mode")
	pathT := flag.Bool("path", false, "path-trace (prettier, slower)")
	cols := flag.Int("w", 80, "width in cells")
	rows := flag.Int("h", 36, "height in cells")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: raymeet <host:port> [localport]")
		os.Exit(1)
	}
	remoteAddr := args[0]
	localPort := "5000"
	if len(args) >= 2 {
		localPort = args[1]
	}
	pxW, pxH := *cols*2, *rows*4

	// The shared script: region i is a pure function of i, so both peers derive an
	// identical world without exchanging any geometry. Swap brain.Local for an
	// HTTPBrain to have a real director (then broadcast specs to stay in sync).
	var b brain.Brain = brain.Local{}
	prompts := []string{"a calm world", "a gold sphere", "glass and metal", "a scene at night", "a calm world"}
	regionAt := func(i int) raytrace.Vec3 {
		return raytrace.Vec3{X: math.Sin(float64(i)*1.7) * 2.5, Y: 0, Z: 8 + float64(i)*12}
	}
	world := raydir.NewWorld()
	growTo := func(n int) {
		for world.Chunks() < n {
			i := world.Chunks()
			if _, err := world.Grow(b, prompts[i%len(prompts)], regionAt(i)); err != nil {
				break
			}
		}
	}
	growTo(3)

	self := raydir.FlyCam{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: 0}, Pitch: -0.08, FOV: math.Pi / 3}
	var mu sync.Mutex
	var remote raydir.Pose
	haveRemote := false

	transport, err := network.NewTransport(":" + localPort)
	if err != nil {
		fmt.Fprintln(os.Stderr, "transport:", err)
		os.Exit(1)
	}
	defer func() { _ = transport.Close() }()
	udpAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolve:", err)
		os.Exit(1)
	}
	_ = transport.SendPacket(&network.Packet{Type: network.PacketTypeHandshake, Sequence: transport.NextSequence(), Timestamp: uint64(time.Now().UnixMilli()), Payload: []byte("raymeet")}, udpAddr)

	// input: stdin -> command runes.
	cmds := make(chan rune, 128)
	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			for _, ch := range strings.ToLower(strings.TrimSpace(sc.Text())) {
				cmds <- ch
			}
		}
		close(cmds)
	}()

	// receive: peer poses (the only thing on the wire besides the handshake).
	go func() {
		for {
			p, _, err := transport.ReceivePacket()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				return
			}
			if p.Type == network.PacketTypeAvatar {
				if pose, e := raydir.DecodePose(p.Payload); e == nil {
					mu.Lock()
					remote, haveRemote = pose, true
					mu.Unlock()
				}
			}
		}
	}()

	mode := *modeFlag
	if mode == "" || mode == "auto" {
		mode = terminal.DetectCapability().BestBlitMode()
	}
	rm, _ := babe.ParseRenderMode(mode)
	dr := terminal.NewDiffRenderer()
	pathOpt := raytrace.PathOptions{Samples: 3, MaxDepth: 4, Seed: 1, NEE: true, MIS: true, Sobol: true}
	avatarColor := raytrace.Vec3{X: 0.25, Y: 0.85, Z: 1.0}

	apply := func(ch rune) bool {
		switch ch {
		case 'w':
			self.Walk(1.2, 0)
		case 's':
			self.Walk(-1.2, 0)
		case 'a':
			self.Walk(0, -1.2)
		case 'd':
			self.Walk(0, 1.2)
		case 'q':
			self.Turn(-0.2, 0)
		case 'e':
			self.Turn(0.2, 0)
		case 'r':
			self.Turn(0, 0.08)
		case 'f':
			self.Turn(0, -0.08)
		case 'p':
			*pathT = !*pathT
		case 'x':
			return false
		}
		return true
	}

	fmt.Printf("raymeet — a shared world with %s. Only poses cross the wire; the world is derived identically on both sides.\n", remoteAddr)
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		// drain all pending input.
		for drain := true; drain; {
			select {
			case ch, ok := <-cmds:
				if !ok || !apply(ch) {
					return
				}
			default:
				drain = false
			}
		}
		// keep the world grown a few regions ahead of wherever we are.
		if need := int((self.Pos.Z-8)/12) + 3; need > world.Chunks() {
			growTo(need)
		}
		// share our pose (40 bytes).
		_ = transport.SendPacket(&network.Packet{Type: network.PacketTypeAvatar, Sequence: transport.NextSequence(), Timestamp: uint64(time.Now().UnixMilli()), Payload: raydir.PoseOf(self).Encode()}, udpAddr)

		mu.Lock()
		rp, have := remote, haveRemote
		mu.Unlock()
		var extra []raytrace.Object
		peer := "waiting…"
		if have {
			extra = raydir.AvatarSpheres(rp, avatarColor)
			peer = fmt.Sprintf("(%.1f,%.1f,%.1f)", rp.Pos.X, rp.Pos.Y, rp.Pos.Z)
		}
		scene := world.SceneWith(extra)
		var im image.Image
		if *pathT {
			im = raytrace.PathRender(scene, self.Camera(), pxW, pxH, pathOpt)
		} else {
			im = raytrace.Render(scene, self.Camera(), pxW, pxH, raytrace.Options{Samples: 1})
		}
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, *cols, *rows, rm)))
		fmt.Printf("\n[shared world | chunks:%d | you (%.1f,%.1f,%.1f) | peer %s | w/s a/d q/e r/f x=quit] ",
			world.Chunks(), self.Pos.X, self.Pos.Y, self.Pos.Z, peer)
	}
}
