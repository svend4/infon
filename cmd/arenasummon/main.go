// Command arenasummon demonstrates crafting: a tangram figure is classified and,
// if it names a creature in the bestiary, a unit is summoned into the arena -
// "the address is the summoning spell". It prints the figure -> gate code ->
// classification -> summon, assembles the summoned creatures into two armies and
// renders their battle to a GIF.
package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/fold"
	"github.com/svend4/infon/pkg/tangram"
)

func main() {
	out := "_arena"
	_ = os.MkdirAll(out, 0o755)
	const MW, MH, W, H = 10, 8, 660, 430

	roster := []struct {
		name    string
		x, y, f uint8
	}{
		{"cat", 1, 1, 0}, {"fox", 1, 3, 0}, {"turtle", 1, 5, 0},
		{"owl", 8, 1, 1}, {"swan", 8, 3, 1}, {"crab", 8, 5, 1},
		{"house", 5, 7, 0}, {"boat", 4, 7, 1}, // not creatures
	}

	a := &arena.Arena{W: MW, H: MH, Terrain: make([]uint8, MW*MH)}
	for ty := 0; ty < MH; ty++ {
		for tx := 0; tx < MW; tx++ {
			a.Terrain[ty*MW+tx] = uint8((tx*2 + ty*3 + (tx*ty)%5) % 4)
		}
	}
	fmt.Printf("%-8s  %-10s  %-12s  summon\n", "figure", "gate", "classify")
	for _, r := range roster {
		f, _ := tangram.Named(r.name)
		u, cls, ok := arena.Summon(f, r.f, r.x, r.y)
		gate := f.GateCode(6)
		if ok {
			a.Units = append(a.Units, u)
			fmt.Printf("%-8s  %-10s  %-12s  -> %s (faction %d)\n", r.name, gate, cls, arena.Bestiary[u.Kind].Name, r.f)
		} else {
			fmt.Printf("%-8s  %-10s  %-12s  -> (no creature)\n", r.name, gate, cls)
		}
	}
	fmt.Printf("\nsummoned %d units; running their battle...\n", len(a.Units))

	g := &gif.GIF{}
	c := arena.RefCommander{}
	for f := 0; f < 24; f++ {
		frame := fold.Render(arena.Faces(a), W, H)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 10)
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		a.Step(c, c)
	}
	fp, _ := os.Create(out + "/arenasummon.gif")
	_ = gif.EncodeAll(fp, g)
	fp.Close()
	fmt.Printf("winner: blue=%d red=%d; wrote %s/arenasummon.gif\n", a.AliveCount(0), a.AliveCount(1), out)
}
