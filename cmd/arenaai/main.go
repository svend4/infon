// Command arenaai runs the origami-Warcraft skirmish with each side commanded by
// a tvcp-ai brain (game "rpg"): by default the reference commander on both, or a
// live model per side via -brain0 / -brain1. Two models can fight: point -brain0
// at one adapter (e.g. Haiku) and -brain1 at another (e.g. OpenAI). The match
// ships as one deltastream over Reed-Solomon and renders to a GIF.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"net/http"
	"os"
	"time"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/deltastream"
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
	b0 := flag.String("brain0", "", "tvcp-ai brain URL commanding blue (default: reference)")
	b1 := flag.String("brain1", "", "tvcp-ai brain URL commanding red (default: reference)")
	flag.Parse()
	out := "_arena"
	os.MkdirAll(out, 0o755)
	const W, H = 660, 430
	const MW, MH, N = 10, 8, 28

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
	add(3, 1, 6, 0)
	add(3, 8, 1, 1)
	add(2, 8, 3, 1)
	add(1, 8, 5, 1)
	add(0, 8, 6, 1)

	c0 := arena.BrainCommander{B: mkBrain(*b0)}
	c1 := arena.BrainCommander{B: mkBrain(*b1)}
	fmt.Printf("blue commanded by: %s\nred  commanded by: %s\n", label(*b0), label(*b1))

	g := &gif.GIF{}
	anim := deltastream.Anim{Width: a.FrameWidth()}
	snap := func() {
		frame := fold.Render(arena.Faces(a), W, H)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 9)
		anim.Frames = append(anim.Frames, a.Snapshot())
	}
	for f := 0; f < N; f++ {
		snap()
		fmt.Printf("tick %2d  blue=%d red=%d  totalHP=%d\n", f, a.AliveCount(0), a.AliveCount(1), a.TotalHP())
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			snap()
			snap()
			break
		}
		a.Step(c0, c1)
	}

	const ecc = 12
	wire := deltastream.Encode(anim, ecc)
	winner := "draw"
	if a.AliveCount(0) > a.AliveCount(1) {
		winner = "blue"
	} else if a.AliveCount(1) > a.AliveCount(0) {
		winner = "red"
	}
	fpg, _ := os.Create(out + "/arenaai.gif")
	gif.EncodeAll(fpg, g)
	fpg.Close()
	hp, _ := os.Create(out + "/arenaai_hero.png")
	png.Encode(hp, fold.Render(arena.Faces(a), W, H))
	hp.Close()
	fmt.Printf("\nwinner: %s (units %d v %d); match raw=%d wire=%d bytes\n",
		winner, a.AliveCount(0), a.AliveCount(1), deltastream.RawSize(anim), len(wire))
	fmt.Printf("wrote %s/arenaai.gif\n", out)
}
