// Command worldloop runs the closed loop: a Brain emits the next 51-byte world
// state each tick, the states ship as one deltastream over Reed-Solomon, a burst
// is injected and recovered, and every state is rendered into a 3-D frame. Swap
// RefBrain for an LLM brain and this is "neural -> bytes -> 3-D frame" live.
package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"os"

	"github.com/svend4/infon/pkg/deltastream"
	"github.com/svend4/infon/pkg/world"
)

func main() {
	out := "_world"
	os.MkdirAll(out, 0o755)
	const W, H, N = 560, 420, 24

	states := world.Loop(world.RefBrain{}, world.State{}, N)

	anim := deltastream.Anim{Width: world.StateBytes}
	for _, s := range states {
		anim.Frames = append(anim.Frames, s.Bytes())
	}
	const ecc = 14
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

	fmt.Printf("game loop: RefBrain emitted %d world states\n", N)
	fmt.Printf("state = %d bytes; stream raw = %d bytes (%.1f bytes/frame)\n", world.StateBytes, raw, float64(raw)/N)
	fmt.Printf("RS wire = %d bytes (%d block(s), ecc=%d); %d-byte burst -> recovered exact=%v\n", len(wire), nb, ecc, burst, exact)

	g := &gif.GIF{}
	for _, fr := range got.Frames {
		frame := world.FromBytes(fr).Render(W, H)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 8)
	}
	fp, _ := os.Create(out + "/worldloop.gif")
	gif.EncodeAll(fp, g)
	fp.Close()
	hp, _ := os.Create(out + "/world_hero.png")
	png.Encode(hp, world.FromBytes(got.Frames[len(got.Frames)/2]).Render(W, H))
	hp.Close()
	fmt.Printf("wrote %s/worldloop.gif and world_hero.png\n", out)
}
