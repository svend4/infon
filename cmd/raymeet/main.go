// Command raymeet is the shared-world experience for a group: several people (and
// AIs) "call in" and walk the SAME 3-D world together, in the terminal, each
// seeing everyone else's avatar and able to talk. Pixels never cross the wire. One
// peer is the director host (the hub): it asks its brain to author each region and
// broadcasts the region's compact scene spec (game:rayscene); guests reconstruct
// each identical region locally. The hub also relays everyone's pose, chat and
// voice to everyone, so the wire carries ~100-byte region specs, 44-byte poses and
// short text/audio — meaning, not pixels — and the world stays in sync even with a
// live AI director (set BRAIN_URL on the host).
//
//	# host / director (the hub), listens on 5000
//	go run ./cmd/raymeet -host 5000
//	# each guest connects to the hub
//	go run ./cmd/raymeet localhost:5000 5001
//	go run ./cmd/raymeet localhost:5000 5002   # ...and a third, fourth, ...
//
// Controls (type then Enter): w/s walk, a/d strafe, q/e turn, r/f look,
// /message to chat, x quit. Add -voice to also carry audio (falls back to text).
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

	"github.com/svend4/infon/internal/audio"
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
	nameFlag := flag.String("name", "", "your chat name (default walker-<id>)")
	voice := flag.Bool("voice", false, "carry voice too (needs a mic/speaker; falls back to text-only)")
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
	name := *nameFlag
	if name == "" {
		name = fmt.Sprintf("walker-%04x", selfID&0xffff)
	}
	chat := raydir.NewChatLog(6)
	var chatMu sync.Mutex

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
	// sendGroup: a guest sends to the hub; the hub fans out to every peer. Chat and
	// voice both ride this path.
	sendGroup := func(typ uint8, payload []byte) {
		if *host {
			stateMu.Lock()
			for _, a := range peers {
				sendTo(a, typ, payload)
			}
			stateMu.Unlock()
		} else {
			sendTo(hubAddr, typ, payload)
		}
	}
	// relayToOthers (host only): forward a received packet to every peer but the
	// one it came from, so guests reach each other through the hub.
	relayToOthers := func(origin *net.UDPAddr, typ uint8, payload []byte) {
		if !*host {
			return
		}
		key := ""
		if origin != nil {
			key = origin.String()
		}
		stateMu.Lock()
		for k, a := range peers {
			if k != key {
				sendTo(a, typ, payload)
			}
		}
		stateMu.Unlock()
	}
	sendChat := func(msg string) {
		if payload, e := network.EncodeTextMessage(network.NewTextMessage(name, msg)); e == nil {
			sendGroup(network.PacketTypeTextChat, payload)
		}
		chatMu.Lock()
		chat.Add(name, msg)
		chatMu.Unlock()
	}
	if !*host {
		sendTo(hubAddr, network.PacketTypeHandshake, []byte("raymeet")) // announce to the hub
	}

	// Voice (optional): capture the mic in 20ms PCM chunks and ship them on the same
	// relay as chat; play received audio. Falls back to text-only if there is no
	// audio device, so the experience never depends on hardware.
	voiceOn := false
	var play audio.AudioPlayback
	if *voice {
		af := audio.DefaultFormat()
		capDev, e1 := audio.NewDefaultCapture()
		plDev, e2 := audio.NewDefaultPlayback()
		if e1 == nil && e2 == nil && capDev.Open() == nil {
			if plDev.Open() == nil {
				voiceOn, play = true, plDev
				defer func() { _ = capDev.Close() }()
				defer func() { _ = plDev.Close() }()
				go func() {
					buf := make([]int16, af.SampleRate/50) // 20ms
					t := time.NewTicker(20 * time.Millisecond)
					defer t.Stop()
					for range t.C {
						n, e := capDev.Read(buf)
						if e != nil || n == 0 {
							continue
						}
						ap := &network.AudioPacket{Timestamp: uint64(time.Now().UnixMilli()), SampleRate: uint16(af.SampleRate), Channels: uint8(af.Channels), Codec: network.AudioCodecPCM, Samples: append([]int16(nil), buf[:n]...)}
						if payload, e := network.EncodeAudioPacket(ap); e == nil {
							sendGroup(network.PacketTypeAudio, payload)
						}
					}
				}()
			} else {
				_ = capDev.Close()
			}
		}
		if !voiceOn {
			fmt.Fprintln(os.Stderr, "voice: no audio device available; continuing text-only")
		}
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

	// receive: poses, regions, chat and voice; peer discovery and relay on the host.
	go func() {
		for {
			p, addr, e := transport.ReceivePacket()
			if e != nil {
				if ne, ok := e.(net.Error); ok && ne.Timeout() {
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
				set, de := raydir.DecodePoseSet(p.Payload)
				if de != nil {
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
					if reg, de := raydir.DecodeRegion(p.Payload); de == nil {
						worldMu.Lock()
						world.AddRegion(reg)
						worldMu.Unlock()
					}
				}
			case network.PacketTypeTextChat:
				if tm, de := network.DecodeTextMessage(p.Payload); de == nil {
					chatMu.Lock()
					chat.Add(tm.Sender, tm.Message)
					chatMu.Unlock()
					relayToOthers(addr, network.PacketTypeTextChat, p.Payload)
				}
			case network.PacketTypeAudio:
				if voiceOn && play != nil {
					if ap, de := network.DecodeAudioPacket(p.Payload); de == nil && len(ap.Samples) > 0 {
						_, _ = play.Write(ap.Samples)
					}
				}
				relayToOthers(addr, network.PacketTypeAudio, p.Payload)
			}
		}
	}()

	lines := make(chan string, 64)
	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
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
	vtag := ""
	if voiceOn {
		vtag = " +voice"
	}
	fmt.Printf("raymeet [%s%s] — group shared world. Only poses, region specs and chat/voice cross the wire; pixels never do.\n", role, vtag)
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	tick := 0
	for range ticker.C {
		for drain := true; drain; {
			select {
			case ln, ok := <-lines:
				if !ok {
					return
				}
				t := strings.TrimSpace(ln)
				if strings.HasPrefix(t, "/") { // "/message" -> chat
					if msg := strings.TrimSpace(t[1:]); msg != "" {
						sendChat(msg)
					}
				} else {
					for _, ch := range strings.ToLower(t) {
						if !apply(ch) {
							return
						}
					}
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
		fmt.Printf("\n[%s as %s%s | chunks:%d | walkers:%d | you (%.1f,%.1f,%.1f) | w/s a/d q/e r/f, /msg, x=quit]",
			role, name, vtag, chunks, others+1, self.Pos.X, self.Pos.Y, self.Pos.Z)
		chatMu.Lock()
		for _, l := range chat.Lines() {
			fmt.Printf("\n  💬 %s", l)
		}
		chatMu.Unlock()
		fmt.Print(" ")
	}
}
