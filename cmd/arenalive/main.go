// Command arenalive is the capstone: a LIVE war of two models, broadcast over a
// lossy channel to a spectator. Each side is commanded by a tvcp-ai brain
// (-brain0 / -brain1; point them at Haiku and OpenAI adapters). The match is
// encoded as keyframe+delta packets over Reed-Solomon, dropped and corrupted on
// a lossy channel, and the SPECTATOR's reconstructed view is rendered to a GIF -
// watch two neural networks fight over a few hundred bytes per second.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/fold"
)

func mkBrain(url string) brain.Brain {
	if url == "" {
		return brain.Local{}
	}
	return brain.HTTPBrain{URL: url, HTTP: &http.Client{Timeout: 60 * time.Second}}
}
func label(url string) string {
	if url == "" {
		return "reference"
	}
	return url
}

func main() {
	b0 := flag.String("brain0", "", "blue commander brain URL")
	b1 := flag.String("brain1", "", "red commander brain URL")
	flag.Parse()
	out := "_arena"
	_ = os.MkdirAll(out, 0o755)
	const W, H = 660, 430
	const MW, MH, N = 10, 8, 26

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

	c0 := arena.BrainCommander{B: mkBrain(*b0)}
	c1 := arena.BrainCommander{B: mkBrain(*b1)}
	fmt.Printf("LIVE war broadcast\n  blue: %s\n  red:  %s\n", label(*b0), label(*b1))

	var frames [][]byte
	for f := 0; f < N; f++ {
		frames = append(frames, a.Snapshot())
		fmt.Printf("tick %2d  blue=%d red=%d  hp=%d\n", f, a.AliveCount(0), a.AliveCount(1), a.TotalHP())
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		a.Step(c0, c1)
	}

	const gop, ecc = 4, 10
	wires := arena.EncodeMatch(frames, gop, ecc)
	total := 0
	for _, w := range wires {
		total += len(w)
	}
	rng := rand.New(rand.NewSource(11))
	lossy := make([][]byte, len(wires))
	dropped, corrupted := 0, 0
	for i, w := range wires {
		if i > 0 && rng.Float64() < 0.15 {
			dropped++
			continue
		}
		cc := make([]byte, len(w))
		copy(cc, w)
		if rng.Float64() < 0.30 {
			nb := len(w) / 255
			for k := 0; k < nb*(ecc/2) && len(w)/3+k < len(cc); k++ {
				cc[len(w)/3+k] ^= 0xFF
			}
			corrupted++
		}
		lossy[i] = cc
	}
	recon, live := arena.Spectate(lossy, ecc, len(frames[0]))
	exact, frozen := 0, 0
	for t := range frames {
		if live[t] && string(recon[t]) == string(frames[t]) {
			exact++
		} else {
			frozen++
		}
	}

	spec := &arena.Arena{W: MW, H: MH, Terrain: a.Terrain}
	g := &gif.GIF{}
	for _, fr := range recon {
		spec.LoadSnapshot(fr)
		frame := fold.Render(arena.Faces(spec), W, H)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 10)
	}
	fp, _ := os.Create(out + "/arenalive.gif")
	_ = gif.EncodeAll(fp, g)
	fp.Close()
	if len(recon) > 0 {
		spec.LoadSnapshot(recon[len(recon)-1])
		hp, _ := os.Create(out + "/arenalive_hero.png")
		_ = png.Encode(hp, fold.Render(arena.Faces(spec), W, H))
		hp.Close()
	}

	winner := "draw"
	if a.AliveCount(0) > a.AliveCount(1) {
		winner = "blue"
	} else if a.AliveCount(1) > a.AliveCount(0) {
		winner = "red"
	}
	fmt.Printf("\nwinner: %s (%d v %d)\n", winner, a.AliveCount(0), a.AliveCount(1))
	fmt.Printf("broadcast: %d ticks, %d packets, %d bytes (%.1f bytes/tick)\n", len(frames), len(wires), total, float64(total)/float64(len(frames)))
	fmt.Printf("lossy: %d dropped, %d corrupted; spectator %d/%d exact, %d frozen\n", dropped, corrupted, exact, len(frames), frozen)
	fmt.Printf("wrote %s/arenalive.gif\n", out)
}
