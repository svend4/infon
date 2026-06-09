// Command raymeet is the shared-world experience: two people (or a person and an
// AI) "call in" and walk the SAME 3-D world together, in the terminal, seeing each
// other's avatar. The world is never sent as pixels. One peer is the director host:
// it asks its brain to author each region and broadcasts the region's compact scene
// description (game:rayscene); the guest reconstructs each identical region locally.
// So the only things on the wire are 40-byte poses and ~100-byte region specs —
// meaning, not pixels — and the world stays in sync even with a live, non-
// deterministic AI director (set BRAIN_URL on the host).
//
//	# host (the director) listens on 5000, talks to the guest on 5001
//	go run ./cmd/raymeet -host localhost:5001 5000
//	# guest
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
	host := flag.Bool("host", false, "be the director: author and broadcast the world (otherwise receive it)")
	modeFlag := flag.String("mode", "auto", "terminal render mode")
	pathT := flag.Bool("path", false, "path-trace (prettier, slower)")
	cols := flag.Int("w", 80, "width in cells")
	rows := flag.Int("h", 36, "height in cells")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: raymeet [-host] <host:port> [localport]")
		os.Exit(1)
	}
	remoteAddr := args[0]
	localPort := "5000"
	if len(args) >= 2 {
		localPort = args[1]
	}
	pxW, pxH := *cols*2, *rows*4

	// The director's brain (only the host authors): reference offline, or a live
	// tvcp-ai/1 endpoint via BRAIN_URL.
	var b brain.Brain = brain.Local{}
	who := "reference (offline)"
	if url := os.Getenv("BRAIN_URL"); url != "" {
		b = brain.HTTPBrain{URL: url}
		who = url
	}
	prompts := []string{"a calm world", "a gold sphere", "glass and metal", "a scene at night", "a calm world"}
	regionAt := func(i int) raytrace.Vec3 {
		return raytrace.Vec3{X: math.Sin(float64(i)*1.7) * 2.5, Y: 0, Z: 8 + float64(i)*12}
	}

	world := raydir.NewWorld()
	var worldMu sync.Mutex
	var regions []raydir.Region // host: authored regions, kept for re-broadcast

	self := raydir.FlyCam{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: 0}, Pitch: -0.08, FOV: math.Pi / 3}
	var poseMu sync.Mutex
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
	send := func(typ uint8, payload []byte) {
		_ = transport.SendPacket(&network.Packet{Type: typ, Sequence: transport.NextSequence(), Timestamp: uint64(time.Now().UnixMilli()), Payload: payload}, udpAddr)
	}
	send(network.PacketTypeHandshake, []byte("raymeet"))

	// host: author regions up to `count` and broadcast each new one.
	growHost := func(count int) {
		worldMu.Lock()
		for len(regions) < count {
			i := len(regions)
			reg, _, err := world.AuthorRegion(b, prompts[i%len(prompts)], i, regionAt(i))
			if err != nil {
				break
			}
			regions = append(regions, reg)
			send(network.PacketTypeScreen, reg.Encode()) // broadcast the region spec
		}
		worldMu.Unlock()
	}
	if *host {
		growHost(3)
	}

	// receive: peer poses, and (for the guest) authored regions.
	go func() {
		for {
			p, _, err := transport.ReceivePacket()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				return
			}
			switch p.Type {
			case network.PacketTypeAvatar:
				if pose, e := raydir.DecodePose(p.Payload); e == nil {
					poseMu.Lock()
					remote, haveRemote = pose, true
					poseMu.Unlock()
				}
			case network.PacketTypeScreen:
				if !*host {
					if reg, e := raydir.DecodeRegion(p.Payload); e == nil {
						worldMu.Lock()
						world.AddRegion(reg)
						worldMu.Unlock()
					}
				}
			}
		}
	}()

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

	role := "guest"
	if *host {
		role = "host/director (" + who + ")"
	}
	fmt.Printf("raymeet [%s] — shared world with %s. Only poses and region specs cross the wire; pixels never do.\n", role, remoteAddr)
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	tick := 0
	for range ticker.C {
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
		tick++
		if *host {
			// keep the world a few regions ahead of BOTH walkers.
			poseMu.Lock()
			gz := 0.0
			if haveRemote {
				gz = remote.Pos.Z
			}
			poseMu.Unlock()
			front := math.Max(self.Pos.Z, gz)
			growHost(int((front-8)/12) + 3)
			// re-broadcast everything periodically so a lossy/late guest catches up.
			if tick%8 == 0 {
				worldMu.Lock()
				for _, r := range regions {
					send(network.PacketTypeScreen, r.Encode())
				}
				worldMu.Unlock()
			}
		}
		send(network.PacketTypeAvatar, raydir.PoseOf(self).Encode())

		poseMu.Lock()
		rp, have := remote, haveRemote
		poseMu.Unlock()
		var extra []raytrace.Object
		peer := "waiting…"
		if have {
			extra = raydir.AvatarSpheres(rp, avatarColor)
			peer = fmt.Sprintf("(%.1f,%.1f,%.1f)", rp.Pos.X, rp.Pos.Y, rp.Pos.Z)
		}
		worldMu.Lock()
		scene := world.SceneWith(extra)
		chunks := world.Chunks()
		worldMu.Unlock()
		var im image.Image
		if *pathT {
			im = raytrace.PathRender(scene, self.Camera(), pxW, pxH, pathOpt)
		} else {
			im = raytrace.Render(scene, self.Camera(), pxW, pxH, raytrace.Options{Samples: 1})
		}
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, *cols, *rows, rm)))
		fmt.Printf("\n[%s | chunks:%d | you (%.1f,%.1f,%.1f) | peer %s | w/s a/d q/e r/f x=quit] ",
			role, chunks, self.Pos.X, self.Pos.Y, self.Pos.Z, peer)
	}
}
