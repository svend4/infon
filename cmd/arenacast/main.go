// Command arenacast broadcasts an arena match as a loss-tolerant semantic stream
// and renders the SPECTATOR's view: keyframe+delta packets over Reed-Solomon are
// dropped and corrupted on a lossy channel, the spectator freezes on loss and
// resyncs at the next keyframe. The agent-world rides the same channel as a
// semantic video call - watch the AI war over a few hundred bytes per second.
package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"math/rand"
	"os"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/fold"
)

func main() {
	out := "_arena"
	_ = os.MkdirAll(out, 0o755)
	const W, H = 660, 430
	const MW, MH = 10, 8

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

	const gop, ecc = 4, 10
	wires := arena.EncodeMatch(frames, gop, ecc)
	total := 0
	for _, w := range wires {
		total += len(w)
	}

	// lossy broadcast: drop ~15% of packets (keep the first keyframe), corrupt a
	// within-budget burst in ~30% of the rest.
	rng := rand.New(rand.NewSource(7))
	lossy := make([][]byte, len(wires))
	dropped, corrupted := 0, 0
	for i, w := range wires {
		if i > 0 && rng.Float64() < 0.15 {
			lossy[i] = nil
			dropped++
			continue
		}
		c := make([]byte, len(w))
		copy(c, w)
		if rng.Float64() < 0.30 {
			nb := len(w) / 255
			burst := nb * (ecc / 2)
			start := len(w) / 3
			for k := 0; k < burst && start+k < len(c); k++ {
				c[start+k] ^= 0xFF
			}
			corrupted++
		}
		lossy[i] = c
	}

	recon, live := arena.Spectate(lossy, ecc, len(frames[0]))
	exact, frozen := 0, 0
	for t := range frames {
		if live[t] && t < len(recon) && string(recon[t]) == string(frames[t]) {
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
	fp, _ := os.Create(out + "/arenacast.gif")
	_ = gif.EncodeAll(fp, g)
	fp.Close()

	fmt.Printf("broadcast: %d ticks, %d packets (gop=%d, ecc=%d), %d bytes total (%.1f bytes/tick)\n",
		len(frames), len(wires), gop, ecc, total, float64(total)/float64(len(frames)))
	fmt.Printf("lossy channel: %d dropped, %d corrupted\n", dropped, corrupted)
	fmt.Printf("spectator: %d/%d ticks exact, %d frozen (resynced at keyframes)\n", exact, len(frames), frozen)
	fmt.Printf("wrote %s/arenacast.gif\n", out)
}
