// Command arenammo is the capstone of the origami-MMORPG: armies are SUMMONED
// from tangram figures, each side is commanded by a LIVE tvcp-ai brain
// (-brain0/-brain1), the match is broadcast as RS packets over real UDP to
// several networked spectators (with loss), and one spectator's reconstructed
// view is rendered to a GIF. Units = address-book figures, commanders = neural
// networks, world = bytes over a lossy channel, audience = networked viewers.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/fold"
	"github.com/svend4/infon/pkg/tangram"
)

func mkBrain(url string) brain.Brain {
	if url == "" {
		return brain.Local{}
	}
	return brain.HTTPBrain{URL: url, HTTP: &http.Client{Timeout: 60 * time.Second}}
}
func label(u string) string {
	if u == "" {
		return "reference"
	}
	return u
}

func main() {
	b0 := flag.String("brain0", "", "blue commander brain URL")
	b1 := flag.String("brain1", "", "red commander brain URL")
	flag.Parse()
	out := "_arena"
	_ = os.MkdirAll(out, 0o755)
	const MW, MH, W, H, gop, ecc = 10, 8, 660, 430, 4, 10

	a := &arena.Arena{W: MW, H: MH, Terrain: make([]uint8, MW*MH)}
	for ty := 0; ty < MH; ty++ {
		for tx := 0; tx < MW; tx++ {
			a.Terrain[ty*MW+tx] = uint8((tx*2 + ty*3 + (tx*ty)%5) % 4)
		}
	}
	// summon the armies from tangram figures (the address is the spell)
	roster := []struct {
		name    string
		x, y, f uint8
	}{
		{"cat", 1, 1, 0}, {"fox", 1, 3, 0}, {"turtle", 1, 5, 0},
		{"owl", 8, 1, 1}, {"swan", 8, 3, 1}, {"crab", 8, 5, 1},
	}
	for _, r := range roster {
		fig, _ := tangram.Named(r.name)
		if u, _, ok := arena.Summon(fig, r.f, r.x, r.y); ok {
			a.Units = append(a.Units, u)
		}
	}
	fmt.Printf("summoned %d units; blue=%s red=%s\n", len(a.Units), label(*b0), label(*b1))

	c0 := arena.BrainCommander{B: mkBrain(*b0)}
	c1 := arena.BrainCommander{B: mkBrain(*b1)}
	var frames [][]byte
	for f := 0; f < 26; f++ {
		frames = append(frames, a.Snapshot())
		fmt.Printf("tick %2d  blue=%d red=%d hp=%d\n", f, a.AliveCount(0), a.AliveCount(1), a.TotalHP())
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		a.Step(c0, c1)
	}
	wires := arena.EncodeMatch(frames, gop, ecc)
	width, nT := len(frames[0]), len(frames)

	// broadcast over real UDP to networked spectators with loss
	ports := []int{9201, 9202, 9203}
	recvd := make([]map[int][]byte, len(ports))
	var wg sync.WaitGroup
	for si, port := range ports {
		recvd[si] = map[int][]byte{}
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
		if err != nil {
			fmt.Println("listen:", err)
			return
		}
		wg.Add(1)
		go func(si int, conn *net.UDPConn) {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				_ = conn.SetReadDeadline(time.Now().Add(700 * time.Millisecond))
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
	rng := rand.New(rand.NewSource(21))
	for _, wire := range wires {
		for _, port := range ports {
			if rng.Float64() < 0.15 {
				continue
			}
			_, _ = sender.WriteToUDP(wire, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
		}
		time.Sleep(25 * time.Millisecond)
	}
	_ = sender.Close()
	wg.Wait()

	winner := "draw"
	if a.AliveCount(0) > a.AliveCount(1) {
		winner = "blue"
	} else if a.AliveCount(1) > a.AliveCount(0) {
		winner = "red"
	}
	fmt.Printf("\nwinner: %s (%d v %d); %d ticks broadcast over UDP to %d spectators\n", winner, a.AliveCount(0), a.AliveCount(1), nT, len(ports))
	for si, port := range ports {
		sp := arena.NewSpectator(ecc, width)
		var frs [][]byte
		exact := 0
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
			}
		}
		fmt.Printf("  spectator :%d  %d/%d ticks exact\n", port, exact, nT)
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
			fp, _ := os.Create(out + "/arenammo.gif")
			_ = gif.EncodeAll(fp, g)
			fp.Close()
		}
	}
	fmt.Printf("wrote %s/arenammo.gif (spectator :9201 view)\n", out)
}
