// Command raymeet is the shared-world experience for a group: several people (and
// AIs) "call in" and walk the SAME 3-D world together, in the terminal, each
// seeing everyone else's avatar. Pixels never cross the wire. One peer is the
// director host (the hub): it asks its brain to author each region and broadcasts
// the region's compact scene spec (game:rayscene); guests reconstruct each
// identical region locally. The hub also relays everyone's pose to everyone, so
// the only things on the wire are ~100-byte region specs and 44-byte poses —
// meaning, not pixels — and the world stays in sync even with a live AI director
// (set BRAIN_URL on the host).
//
//	# host / director (the hub), listens on 5000
//	go run ./cmd/raymeet -host 5000
//	# each guest connects to the hub
//	go run ./cmd/raymeet localhost:5000 5001
//	go run ./cmd/raymeet localhost:5000 5002   # ...and a third, fourth, ...
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
	host := flag.Bool("host", false, "be the director hub: author/broadcast the world and relay poses")
	modeFlag := flag.String("mode", "auto", "terminal render mode")
	pathT := flag.Bool("path", false, "path-trace (prettier, slower)")
	cols := flag.Int("w", 80, "width in cells")
	rows := flag.Int("h", 36, "height in cells")
	flag.Parse()
	args := flag.Args()
	pxW, pxH := *cols*2, *rows*4

	var hubAddr *net.UDPAddr
	localPort := "5000"
	if *host {
		if len(args) >= 1 {
			localPort = args[0]
		}
	} else {
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "usage: raymeet -host [localport]   |   raymeet <hostaddr> [localport]")
			os.Exit(1)
		}
		a, err := net.ResolveUDPAddr("udp", args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "resolve:", err)
			os.Exit(1)
		}
		hubAddr = a
		localPort = "5001"
		if len(args) >= 2 {
			localPort = args[1]
		}
	}
	selfID := uint32(time.Now().UnixNano()) ^ uint32(os.Getpid())<<8

	// director's brain (host only): reference offline, or a live endpoint via BRAIN_URL.
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
	var regions []raydir.Region

	self := raydir.FlyCam{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: 0}, Pitch: -0.08, FOV: math.Pi / 3}

	var stateMu sync.Mutex
	poses := raydir.PoseSet{}          // host: everyone; guest: the hub's relayed table
	peers := map[string]*net.UDPAddr{} // host: connected guests
	lastSeen := map[uint32]time.Time{} // host: prune the disconnected

	transport, err := network.NewTransport(":" + localPort)
	if err != nil {
		fmt.Fprintln(os.Stderr, "transport:", err)
		os.Exit(1)
	}
	defer func() { _ = transport.Close() }()
	sendTo := func(addr *net.UDPAddr, typ uint8, payload []byte) {
		if addr != nil {
			_ = transport.SendPacket(&network.Packet{Type: typ, Sequence: transport.NextSequence(), Timestamp: uint64(time.Now().UnixMilli()), Payload: payload}, addr)
		}
	}
	if !*host {
		sendTo(hubAddr, network.PacketTypeHandshake, []byte("raymeet")) // announce to the hub
	}

	growHost := func(count int) {
		worldMu.Lock()
		stateMu.Lock()
		addrs := make([]*net.UDPAddr, 0, len(peers))
		for _, a := range peers {
			addrs = append(addrs, a)
		}
		stateMu.Unlock()
		for len(regions) < count {
			i := len(regions)
			reg, _, e := world.AuthorRegion(b, prompts[i%len(prompts)], i, regionAt(i))
			if e != nil {
				break
			}
			regions = append(regions, reg)
			for _, a := range addrs {
				sendTo(a, network.PacketTypeScreen, reg.Encode())
			}
		}
		worldMu.Unlock()
	}
	if *host {
		growHost(3)
	}

	// receive: poses (everyone), regions (guests), and peer discovery (host).
	go func() {
		for {
			p, addr, err := transport.ReceivePacket()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				return
			}
			switch p.Type {
			case network.PacketTypeHandshake:
				if *host && addr != nil {
					stateMu.Lock()
					peers[addr.String()] = addr
					stateMu.Unlock()
				}
			case network.PacketTypeAvatar:
				set, e := raydir.DecodePoseSet(p.Payload)
				if e != nil {
					continue
				}
				stateMu.Lock()
				if *host {
					if addr != nil {
						peers[addr.String()] = addr
					}
					for id, pose := range set { // merge each guest's own entry
						poses[id] = pose
						lastSeen[id] = time.Now()
					}
				} else {
					poses = set // the hub's full relayed table
				}
				stateMu.Unlock()
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
	fmt.Printf("raymeet [%s] — group shared world. Only poses and region specs cross the wire; pixels never do.\n", role)
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
		myPose := raydir.PoseOf(self)

		if *host {
			stateMu.Lock()
			poses[selfID] = myPose
			lastSeen[selfID] = time.Now()
			for id, ts := range lastSeen { // prune walkers we haven't heard from
				if id != selfID && time.Since(ts) > 5*time.Second {
					delete(poses, id)
					delete(lastSeen, id)
				}
			}
			front := self.Pos.Z
			for _, p := range poses {
				front = math.Max(front, p.Pos.Z)
			}
			setBytes := poses.Encode()
			addrs := make([]*net.UDPAddr, 0, len(peers))
			for _, a := range peers {
				addrs = append(addrs, a)
			}
			stateMu.Unlock()
			growHost(int((front-8)/12) + 3)
			for _, a := range addrs {
				sendTo(a, network.PacketTypeAvatar, setBytes) // relay everyone's poses
			}
			if tick%8 == 0 {
				worldMu.Lock()
				rs := append([]raydir.Region(nil), regions...)
				worldMu.Unlock()
				for _, a := range addrs {
					for _, r := range rs {
						sendTo(a, network.PacketTypeScreen, r.Encode())
					}
				}
			}
		} else {
			sendTo(hubAddr, network.PacketTypeAvatar, raydir.PoseSet{selfID: myPose}.Encode())
		}

		// draw everyone except yourself.
		stateMu.Lock()
		var extra []raytrace.Object
		others := 0
		for id, p := range poses {
			if id != selfID {
				extra = append(extra, raydir.AvatarSpheres(p, raydir.AvatarColor(id))...)
				others++
			}
		}
		stateMu.Unlock()
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
		fmt.Printf("\n[%s | chunks:%d | walkers:%d | you (%.1f,%.1f,%.1f) | w/s a/d q/e r/f x=quit] ",
			role, chunks, others+1, self.Pos.X, self.Pos.Y, self.Pos.Z)
	}
}
