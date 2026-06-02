// Command arena runs an "origami Warcraft" skirmish: two factions of tangram
// units, each driven by the reference Commander, charge across an isometric
// terrain board. Every tick is a few bytes; the whole match ships as one
// deltastream over Reed-Solomon (a burst is injected and recovered) and renders
// to a GIF. Swap a Commander for an LLM agent and the sides are neural networks.
package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"os"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/deltastream"
	"github.com/svend4/infon/pkg/fold"
)

func main() {
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

	g := &gif.GIF{}
	anim := deltastream.Anim{Width: a.FrameWidth()}
	c0, c1 := arena.RefCommander{}, arena.RefCommander{}
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
	raw := deltastream.RawSize(anim)
	wire := deltastream.Encode(anim, ecc)
	nb := len(wire) / 255
	burst := nb * (ecc / 2)
	corrupt := make([]byte, len(wire))
	copy(corrupt, wire)
	for i := 0; i < burst && len(wire)/2+i < len(corrupt); i++ {
		corrupt[len(wire)/2+i] ^= 0xFF
	}
	got, err := deltastream.Decode(corrupt, ecc)
	exact := err == nil && len(got.Frames) == len(anim.Frames)
	for f := 0; exact && f < len(anim.Frames); f++ {
		for i := range anim.Frames[f] {
			if got.Frames[f][i] != anim.Frames[f][i] {
				exact = false
				break
			}
		}
	}

	winner := "draw"
	if a.AliveCount(0) > a.AliveCount(1) {
		winner = "blue"
	} else if a.AliveCount(1) > a.AliveCount(0) {
		winner = "red"
	}
	fpg, _ := os.Create(out + "/arena.gif")
	gif.EncodeAll(fpg, g)
	fpg.Close()
	hero := fold.Render(arena.Faces(a), W, H)
	hp, _ := os.Create(out + "/arena_hero.png")
	png.Encode(hp, hero)
	hp.Close()

	fmt.Printf("\nwinner: %s  (units %d v %d)\n", winner, a.AliveCount(0), a.AliveCount(1))
	fmt.Printf("match = %d ticks x %d units = %d-byte frames; stream raw=%d wire=%d (%.1f bytes/tick)\n",
		len(anim.Frames), len(a.Units), a.FrameWidth(), raw, len(wire), float64(raw)/float64(len(anim.Frames)))
	fmt.Printf("injected %d-byte burst -> recovered exact=%v\n", burst, exact)
	fmt.Printf("wrote %s/arena.gif and arena_hero.png\n", out)
}
