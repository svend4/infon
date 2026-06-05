// Command worldai drives the 3-D world through the tvcp-ai protocol: each tick a
// brain receives a Brief of the world (game "world") and returns directives
// {fold,rise,spin,camera}; the world physics applies them and renders the frame.
// By default the reference brain (brain.Local) directs; pass -brain URL to let a
// live LLM (e.g. Haiku via the adapter) drive the world - neural -> bytes -> 3-D.
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

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/world"
)

func main() {
	url := flag.String("brain", "", "tvcp-ai/1 brain URL (default: reference brain.Local)")
	n := flag.Int("n", 24, "ticks")
	flag.Parse()
	out := "_worldai"
	_ = os.MkdirAll(out, 0o755)
	const W, H = 560, 420

	var b brain.Brain = brain.Local{}
	driver := "reference (brain.Local)"
	if *url != "" {
		b = brain.HTTPBrain{URL: *url, HTTP: &http.Client{Timeout: 60 * time.Second}}
		driver = *url
	}
	ctrl := &world.BrainController{B: b}

	fmt.Printf("world directed by: %s\n", driver)
	g := &gif.GIF{}
	s := world.Init()
	var track []byte
	for f := 0; f < *n; f++ {
		d := ctrl.Decide(s)
		s = world.Apply(s, d)
		track = append(track, world.DirByte(d))
		br := world.MakeBrief(s)
		fmt.Printf("tick %2d  dir{fold:%+d rise:%+d spin:%+d cam:%+d}  fold%%=%d relief%%=%d cam=%d\n",
			f, d.Fold, d.Rise, d.Spin, d.Camera, br.FoldPct, br.ReliefPct, s.Cam)
		frame := s.Render(W, H)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 8)
	}
	fp, _ := os.Create(out + "/worldai.gif")
	_ = gif.EncodeAll(fp, g)
	fp.Close()
	hp, _ := os.Create(out + "/worldai_hero.png")
	_ = png.Encode(hp, world.Init().Render(W, H))
	hp.Close()
	fmt.Printf("brain emitted %d decisions = %d-byte control track (1 byte/tick)\n", *n, len(track))
	fmt.Printf("wrote %s/worldai.gif\n", out)
}
