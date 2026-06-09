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
//	# host that resumes / persists its authored world to a tiny file
//	go run ./cmd/raymeet -host -world myworld.rwld 5000
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

// clockOf formats a time of day in [0,1) as a wall clock for the status line.
func clockOf(t float64) string {
	t -= math.Floor(t)
	mins := int(t * 24 * 60)
	return fmt.Sprintf("🕓 %02d:%02d", mins/60, mins%60)
}

func main() {
	host := flag.Bool("host", false, "be the director hub: author/broadcast the world and relay poses")
	modeFlag := flag.String("mode", "auto", "terminal render mode")
	nameFlag := flag.String("name", "", "your chat name (default walker-<id>)")
	voice := flag.Bool("voice", false, "carry voice too (needs a mic/speaker; falls back to text-only)")
	pathT := flag.Bool("path", false, "path-trace (prettier, slower)")
	cols := flag.Int("w", 80, "width in cells")
	rows := flag.Int("h", 36, "height in cells")
	worldFile := flag.String("world", "", "host: persist the authored world to this file (load on start, save as it grows)")
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
	csync := raydir.NewChatSync(selfID, *host) // reliable chat: ids + dedup + re-broadcast

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
	// persistence: a host resumes a saved world (a tiny file of region specs —
	// meaning, not pixels) instead of re-authoring it from scratch.
	if *host && *worldFile != "" {
		if loaded, e := raydir.LoadWorldFile(*worldFile); e == nil && len(loaded) > 0 {
			for _, r := range loaded {
				world.AddRegion(r)
				regions = append(regions, r)
			}
			fmt.Printf("resumed %d regions from %s\n", len(loaded), *worldFile)
		}
	}

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
		m := csync.Compose(name, msg)
		sendGroup(network.PacketTypeTextChat, raydir.EncodeChatMsgs([]raydir.ChatMsg{m}))
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
	var mixer *raydir.VoiceMixer // sums concurrent speakers into one output stream
	if *voice {
		af := audio.DefaultFormat()
		frame := af.SampleRate / 50 // 20ms
		capDev, e1 := audio.NewDefaultCapture()
		plDev, e2 := audio.NewDefaultPlayback()
		if e1 == nil && e2 == nil && capDev.Open() == nil {
			if plDev.Open() == nil {
				voiceOn, play, mixer = true, plDev, raydir.NewVoiceMixer(frame)
				defer func() { _ = capDev.Close() }()
				defer func() { _ = plDev.Close() }()
				go func() { // capture: ship 20ms PCM chunks tagged with our id
					buf := make([]int16, frame)
					t := time.NewTicker(20 * time.Millisecond)
					defer t.Stop()
					for range t.C {
						n, e := capDev.Read(buf)
						if e != nil || n == 0 {
							continue
						}
						sendGroup(network.PacketTypeAudio, raydir.EncodeVoice(selfID, buf[:n]))
					}
				}()
				go func() { // playback: pull mixed frames at a steady rate
					t := time.NewTicker(20 * time.Millisecond)
					defer t.Stop()
					for range t.C {
						if f := mixer.Mix(); f != nil {
							_, _ = play.Write(f)
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
		before := len(regions)
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
		if *worldFile != "" && len(regions) > before { // persist only when it grew
			_ = raydir.SaveWorld(*worldFile, regions)
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
				// reliable chat: dedup by message id, show only the new ones, and (on
				// the hub) relay just those onward. Re-broadcasts of seen ids are dropped.
				if msgs, de := raydir.DecodeChatMsgs(p.Payload); de == nil {
					var fresh []raydir.ChatMsg
					for _, m := range msgs {
						if csync.Observe(m) {
							chatMu.Lock()
							chat.Add(m.Sender, m.Text)
							chatMu.Unlock()
							fresh = append(fresh, m)
						}
					}
					if len(fresh) > 0 {
						relayToOthers(addr, network.PacketTypeTextChat, raydir.EncodeChatMsgs(fresh))
					}
				}
			case network.PacketTypeAudio:
				// feed each speaker's frame into the mixer (kept separate, summed at
				// playback) rather than writing straight to the device, so simultaneous
				// talkers blend instead of serialising.
				if origin, samples, de := raydir.DecodeVoice(p.Payload); de == nil {
					if voiceOn && mixer != nil && origin != selfID {
						mixer.Add(origin, samples)
					}
					relayToOthers(addr, network.PacketTypeAudio, p.Payload)
				}
			case network.PacketTypeControl:
				if *host && addr != nil {
					// a guest's region ack: re-send only what it is missing, to it alone.
					if have, de := raydir.DecodeAck(p.Payload); de == nil {
						worldMu.Lock()
						miss := raydir.MissingRegions(have, regions)
						worldMu.Unlock()
						for _, r := range miss {
							sendTo(addr, network.PacketTypeScreen, r.Encode())
						}
					}
				} else if !*host {
					// the host's time-of-day broadcast: keep the whole group's sky in sync.
					if t, de := raydir.DecodeEnv(p.Payload); de == nil {
						worldMu.Lock()
						world.SetTime(t)
						worldMu.Unlock()
					}
				}
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
	dayTime := 0.30              // host's time of day; advances toward dusk/night
	const dayStep = 0.12 / 150.0 // a full day in ~150s at the 120ms tick
	if *host {
		worldMu.Lock()
		world.SetTime(dayTime)
		worldMu.Unlock()
	}
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
			dayTime += dayStep // advance and broadcast the time of day to the group
			worldMu.Lock()
			world.SetTime(dayTime)
			worldMu.Unlock()
			env := raydir.EncodeEnv(dayTime)
			for _, a := range addrs {
				sendTo(a, network.PacketTypeAvatar, setBytes) // relay everyone's poses
				sendTo(a, network.PacketTypeControl, env)     // keep the group's sky in sync
			}
			// new regions are pushed immediately in growHost; gaps are filled on
			// demand when a guest acks (no blind re-broadcast of everything).
		} else {
			sendTo(hubAddr, network.PacketTypeAvatar, raydir.PoseSet{selfID: myPose}.Encode())
			if tick%8 == 0 { // tell the hub which regions we have so it fills the gaps
				worldMu.Lock()
				known := world.Known()
				worldMu.Unlock()
				sendTo(hubAddr, network.PacketTypeControl, raydir.EncodeAck(known))
			}
		}

		// reliable chat retransmit: every ~1s re-send the recent ring so a dropped
		// message self-heals (the hub re-broadcasts everyone's; a guest re-sends its
		// own to the hub). Receivers dedup by id, so this never double-shows.
		if tick%8 == 4 {
			if recent := csync.Recent(); len(recent) > 0 {
				sendGroup(network.PacketTypeTextChat, raydir.EncodeChatMsgs(recent))
			}
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
		tod := world.Time
		worldMu.Unlock()

		var im image.Image
		if *pathT {
			im = raytrace.PathRender(scene, self.Camera(), pxW, pxH, pathOpt)
		} else {
			im = raytrace.Render(scene, self.Camera(), pxW, pxH, raytrace.Options{Samples: 1})
		}
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, *cols, *rows, rm)))
		fmt.Printf("\n[%s as %s%s | chunks:%d | walkers:%d | %s | you (%.1f,%.1f,%.1f) | w/s a/d q/e r/f, /msg, x=quit]",
			role, name, vtag, chunks, others+1, clockOf(tod), self.Pos.X, self.Pos.Y, self.Pos.Z)
		chatMu.Lock()
		for _, l := range chat.Lines() {
			fmt.Printf("\n  💬 %s", l)
		}
		chatMu.Unlock()
		fmt.Print(" ")
	}
}
