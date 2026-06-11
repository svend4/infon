// Command raymetaverse is a real networked meeting INSIDE a shared world (Block F:
// metaverse over the wire). Several peers "call in" over UDP; each sends only its
// PRESENCE — a 40-byte pose plus compact face keypoints — never pixels. One peer is
// the hub: it merges everyone's presence into a set and relays it to all, and it
// pushes world CHANGES as a compact scene-delta (not a re-send). Every peer rebuilds
// the SAME hexagram world locally and ray-traces it from its own seat, with everyone
// else's face drawn in. It runs the whole exchange over loopback sockets, renders
// each peer's point of view, and reports how little crosses the wire.
//
//	go run ./cmd/raymetaverse -n 3 -hexagram 110010
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net"
	"os"
	"time"

	"github.com/svend4/infon/internal/avatar"
	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

type peer struct {
	id     uint32
	t      *network.Transport
	addr   *net.UDPAddr
	pose   raydir.Pose
	face   avatar.Keypoints
	roster raydir.PresenceSet
	world  brain.SceneSpec
}

func main() {
	var (
		out  = flag.String("out", "metaverse", "output basename")
		n    = flag.Int("n", 3, "participants")
		hexS = flag.String("hexagram", "110010", "hexagram world to meet in")
		w    = flag.Int("w", 320, "panel width")
		h    = flag.Int("h", 230, "panel height")
		spp  = flag.Int("spp", 48, "samples per pixel")
	)
	flag.Parse()
	if *n < 2 {
		*n = 2
	}

	hex, ok := raydir.ParseHexagram(*hexS)
	if !ok {
		hex = raydir.HexagramFromNumber(0b110010)
	}
	// the shared world (everyone derives it identically) and a world CHANGE the hub
	// will push as a delta: a golden centrepiece the group gathers around.
	_, base, err := raydir.AuthorScene(brain.Local{}, hex.Prompt())
	if err != nil {
		fmt.Fprintln(os.Stderr, "author:", err)
		os.Exit(1)
	}
	changed := base
	changed.Objects = append(append([]brain.ObjSpec{}, base.Objects...), brain.ObjSpec{
		Kind: "sphere", X: 0, Y: 1.5, Z: 6, R: 0.45, Rough: 0.6,
		Color: [3]float64{0.95, 0.78, 0.3}, Emit: [3]float64{0.5, 0.38, 0.12},
	})
	changed.Name = base.Name + " + focus"
	delta, _ := raydir.DiffScene(base, changed)
	deltaBytes, _ := json.Marshal(delta)

	// seats in a circle, each facing the centre (the focus).
	center := raytrace.Vec3{X: 0, Y: 1.6, Z: 6}
	const radius = 2.7
	peers := make([]*peer, *n)
	for i := 0; i < *n; i++ {
		th := -math.Pi/2 + float64(i)/float64(*n)*2*math.Pi
		pos := raytrace.Vec3{X: center.X + math.Cos(th)*radius, Y: 1.6, Z: center.Z + math.Sin(th)*radius}
		tr, e := network.NewTransport("127.0.0.1:0")
		if e != nil {
			fmt.Fprintln(os.Stderr, "transport:", e)
			os.Exit(1)
		}
		defer func(t *network.Transport) { _ = t.Close() }(tr)
		peers[i] = &peer{
			id:     uint32(i + 1),
			t:      tr,
			addr:   tr.LocalAddr(),
			pose:   raydir.Pose{Pos: pos, Yaw: math.Atan2(center.X-pos.X, center.Z-pos.Z)},
			face:   avatar.Keypoints{Frame: 1, Points: raydir.DemoFace(i)},
			roster: raydir.PresenceSet{},
			world:  base,
		}
	}
	hub := peers[0]
	hub.world = changed // the hub holds the changed world directly

	send := func(t *network.Transport, to *net.UDPAddr, typ uint8, payload []byte) {
		_ = t.SendPacket(&network.Packet{Type: typ, Sequence: t.NextSequence(), Timestamp: uint64(time.Now().UnixMilli()), Payload: payload}, to)
	}
	drain := func(t *network.Transport, dur time.Duration, fn func(*network.Packet, *net.UDPAddr)) {
		conn := t.GetConn()
		deadline := time.Now().Add(dur)
		buf := make([]byte, network.MaxPacketSize)
		for {
			if time.Now().After(deadline) {
				return
			}
			_ = conn.SetReadDeadline(deadline)
			nb, from, e := conn.ReadFromUDP(buf)
			if e != nil {
				return
			}
			if pkt, de := network.Decode(buf[:nb]); de == nil {
				fn(pkt, from)
			}
		}
	}

	guestAddr := map[uint32]*net.UDPAddr{}
	const rounds = 4
	for r := 0; r < rounds; r++ {
		// guests announce their presence to the hub
		for i := 1; i < *n; i++ {
			pr := raydir.Presence{ID: peers[i].id, Pose: peers[i].pose, Face: peers[i].face}
			send(peers[i].t, hub.addr, network.PacketTypeAvatar, pr.Encode())
		}
		// hub gathers presences (learning each guest's address), adds itself
		drain(hub.t, 40*time.Millisecond, func(pkt *network.Packet, from *net.UDPAddr) {
			if pkt.Type == network.PacketTypeAvatar {
				if pr, e := raydir.DecodePresence(pkt.Payload); e == nil {
					hub.roster[pr.ID] = pr
					guestAddr[pr.ID] = from
				}
			}
		})
		hub.roster[hub.id] = raydir.Presence{ID: hub.id, Pose: hub.pose, Face: hub.face}
		// hub relays the whole set, and pushes the world change as a delta
		setBytes := hub.roster.Encode()
		for id, addr := range guestAddr {
			_ = id
			send(hub.t, addr, network.PacketTypeAvatar, setBytes)
			send(hub.t, addr, network.PacketTypeScreen, deltaBytes)
		}
		// guests receive the set and apply the delta to their local base world
		for i := 1; i < *n; i++ {
			p := peers[i]
			drain(p.t, 40*time.Millisecond, func(pkt *network.Packet, _ *net.UDPAddr) {
				switch pkt.Type {
				case network.PacketTypeAvatar:
					if set, e := raydir.DecodePresenceSet(pkt.Payload); e == nil {
						p.roster = set
					}
				case network.PacketTypeScreen:
					var d raydir.SceneDelta
					if json.Unmarshal(pkt.Payload, &d) == nil {
						p.world = raydir.ApplyScene(base, d)
					}
				}
			})
		}
	}

	// report what crossed the wire
	prBytes := len(raydir.Presence{ID: 1, Pose: peers[0].pose, Face: peers[0].face}.Encode())
	setBytes := len(hub.roster.Encode())
	fullSpec := raydir.SceneBytes(changed)
	pixels := *w * *h * 3
	fmt.Printf("metaverse in %q: %d peers over loopback UDP\n", hex.Name(), *n)
	converged := 0
	for _, p := range peers {
		if len(p.roster) == *n {
			converged++
		}
	}
	fmt.Printf("  converged: %d/%d peers hold the full roster of %d\n", converged, *n, *n)
	fmt.Printf("  presence up (guest->hub): %d B/peer/frame (40-byte pose + face keypoints)\n", prBytes)
	fmt.Printf("  roster down (hub->peer):  %d B/frame for the whole room\n", setBytes)
	fmt.Printf("  world change: scene-delta %d B vs a full keyframe %d B (%.0f%% smaller)\n",
		len(deltaBytes), fullSpec, 100*(1-float64(len(deltaBytes))/float64(fullSpec)))
	fmt.Printf("  a single rendered frame would be %d B of pixels — never sent\n", pixels)

	// render each peer's point of view of the shared, face-populated, delta-updated world
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 5, Seed: 6, NEE: true, MIS: true, Sobol: true}
	var panels []panel
	for _, p := range peers {
		scene := raydir.BuildScene(p.world)
		scene.Objects = append(scene.Objects, p.roster.Objects(p.id)...)
		scene.BuildBVH()
		cam := raytrace.Camera{Pos: p.pose.Pos.Add(raytrace.Vec3{Y: 0.1}), Yaw: p.pose.Yaw, Pitch: -0.04, FOV: math.Pi / 3}
		img := raytrace.PostProcess(raytrace.PathRender(scene, cam, *w, *h, opt), 1.0, 0.85, 0.5)
		panels = append(panels, panel{img, fmt.Sprintf("peer %d sees %d others", p.id, len(p.roster)-1)})
	}
	writePNG(*out+".png", montage(panels))
	fmt.Printf("wrote %s.png\n", *out)
}

type panel struct {
	img   image.Image
	label string
}

func montage(panels []panel) image.Image {
	pw := panels[0].img.Bounds().Dx()
	ph := panels[0].img.Bounds().Dy()
	const gap, labelH = 8, 16
	W := len(panels)*pw + (len(panels)-1)*gap
	out := image.NewRGBA(image.Rect(0, 0, W, ph+labelH))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	for i, p := range panels {
		x := i * (pw + gap)
		draw.Draw(out, image.Rect(x, labelH, x+pw, labelH+ph), p.img, p.img.Bounds().Min, draw.Src)
		microfont.Draw(out, x+4, 3, 1, p.label, color.RGBA{R: 230, G: 230, B: 235, A: 255})
	}
	return out
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
