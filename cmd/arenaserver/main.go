// Command arenaserver is a persistent tick-server with multiple networked
// spectators: it runs an arena match and broadcasts each tick as an
// RS-protected packet over UDP (real localhost loopback) to several spectator
// sockets, dropping ~15% of sends per viewer. Each spectator reconstructs the
// world independently (freeze-on-loss, keyframe resync) - a mini-MMO you watch
// over a lossy channel. One spectator's view is rendered to a GIF.
package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/fold"
)

func main() {
	out := "_arena"
	os.MkdirAll(out, 0o755)
	const MW, MH, W, H = 10, 8, 660, 430
	const gop, ecc = 4, 10

	a := &arena.Arena{W: MW, H: MH, Terrain: make([]uint8, MW*MH)}
	for ty := 0; ty < MH; ty++ {
		for tx := 0; tx < MW; tx++ {
			a.Terrain[ty*MW+tx] = uint8((tx*2 + ty*3 + (tx*ty)%5) % 4)
		}
	}
	add := func(k, x, y, f uint8) {
		a.Units = append(a.Units, arena.Unit{Kind: k, X: x, Y: y, HP: arena.Bestiary[k].HP, Faction: f, Alive: true})
	}
	add(0, 1, 1, 0)
	add(1, 1, 3, 0)
	add(2, 1, 5, 0)
	add(1, 1, 6, 0)
	add(4, 8, 1, 1)
	add(5, 8, 3, 1)
	add(3, 8, 5, 1)
	add(4, 8, 6, 1)

	var frames [][]byte
	c := arena.RefCommander{}
	for f := 0; f < 26; f++ {
		frames = append(frames, a.Snapshot())
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		a.Step(c, c)
	}
	wires := arena.EncodeMatch(frames, gop, ecc)
	width, nT := len(frames[0]), len(frames)

	ports := []int{9101, 9102, 9103}
	recvd := make([]map[int][]byte, len(ports))
	var wg sync.WaitGroup
	conns := make([]*net.UDPConn, len(ports))
	for si, port := range ports {
		recvd[si] = map[int][]byte{}
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
		if err != nil {
			fmt.Println("listen:", err)
			return
		}
		conns[si] = conn
		wg.Add(1)
		go func(si int, conn *net.UDPConn) {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				conn.SetReadDeadline(time.Now().Add(700 * time.Millisecond))
				n, _, err := conn.ReadFromUDP(buf)
				if err != nil {
					return
				}
				w := make([]byte, n)
				copy(w, buf[:n])
				recvd[si][arena.PacketTick(w, ecc)] = w
			}
		}(si, conn)
	}

	time.Sleep(120 * time.Millisecond)
	sender, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	rng := rand.New(rand.NewSource(13))
	sent := make([]int, len(ports))
	for _, wire := range wires {
		for si, port := range ports {
			if rng.Float64() < 0.15 {
				continue
			}
			sender.WriteToUDP(wire, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
			sent[si]++
		}
		time.Sleep(25 * time.Millisecond)
	}
	sender.Close()
	wg.Wait()

	fmt.Printf("tick-server: %d ticks, %d packets/tick-stream, %d spectators\n", nT, len(wires), len(ports))
	for si, port := range ports {
		sp := arena.NewSpectator(ecc, width)
		var frs [][]byte
		exact, frozen := 0, 0
		for t := 0; t < nT; t++ {
			var fr []byte
			var live bool
			if w, ok := recvd[si][t]; ok {
				fr, live = sp.Recv(w)
			} else {
				fr, live = sp.Miss()
			}
			frs = append(frs, fr)
			if live && string(fr) == string(frames[t]) {
				exact++
			} else {
				frozen++
			}
		}
		fmt.Printf("  spectator :%d  got %d/%d packets, %d/%d ticks exact, %d frozen\n", port, sent[si], nT, exact, nT, frozen)
		if si == 0 {
			spec := &arena.Arena{W: MW, H: MH, Terrain: a.Terrain}
			g := &gif.GIF{}
			for _, fr := range frs {
				spec.LoadSnapshot(fr)
				frame := fold.Render(arena.Faces(spec), W, H)
				pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
				draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
				g.Image = append(g.Image, pi)
				g.Delay = append(g.Delay, 10)
			}
			fp, _ := os.Create(out + "/arenaserver.gif")
			gif.EncodeAll(fp, g)
			fp.Close()
		}
	}
	fmt.Printf("wrote %s/arenaserver.gif (spectator :9101 view)\n", out)
}
