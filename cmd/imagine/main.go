// Command imagine is the "protocol of shared imagination": one agent-world is
// broadcast ONCE as a semantic-video stream (pkg/arenacast + pkg/semvideo), and
// TWO independent receivers — with their own packet loss — each reconstruct it
// LOCALLY. You transmit the world, not the pixels; each mind assembles the image
// itself. The GIF shows three panels: the world as sent, and as imagined by mind
// A and mind B; the console reports how much MEANING each recovered.
//
//	go run ./cmd/imagine
//	go run ./cmd/imagine -loss 0.25
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"math/rand"
	"os"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/arenacast"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
	"github.com/svend4/infon/pkg/tangram"
)

func main() {
	loss := flag.Float64("loss", 0.18, "per-receiver packet loss")
	ecc := flag.Int("ecc", 8, "Reed-Solomon parity bytes")
	gop := flag.Int("gop", 6, "keyframe interval")
	ticks := flag.Int("ticks", 26, "max ticks")
	out := flag.String("out", "_arena", "output directory")
	flag.Parse()
	os.MkdirAll(*out, 0o755)

	const MW, MH = 10, 8
	a := &arena.Arena{W: MW, H: MH, Terrain: make([]uint8, MW*MH)}
	for ty := 0; ty < MH; ty++ {
		for tx := 0; tx < MW; tx++ {
			a.Terrain[ty*MW+tx] = uint8((tx*2 + ty*3 + (tx*ty)%5) % 4)
		}
	}
	for _, r := range []struct {
		name    string
		x, y, f uint8
	}{{"cat", 1, 1, 0}, {"fox", 1, 4, 0}, {"turtle", 1, 6, 0}, {"owl", 8, 1, 1}, {"swan", 8, 4, 1}, {"crab", 8, 6, 1}} {
		if fig, ok := tangram.Named(r.name); ok {
			if u, _, ok := arena.Summon(fig, r.f, r.x, r.y); ok {
				a.Units = append(a.Units, u)
			}
		}
	}
	// one shared world; the planner drives both sides so it plays out richly.
	var truth []pseudo.Spec
	for f := 0; f < *ticks; f++ {
		truth = append(truth, arenacast.Spec(a))
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		a.Step(arena.Planner{}, arena.RefCommander{})
	}

	pkts := semvideo.EncodeFrames(truth, *ecc, *gop)
	recvA, lossA := receive(pkts, *ecc, MW, MH, 11, *loss)
	recvB, lossB := receive(pkts, *ecc, MW, MH, 29, *loss)

	iou := func(rs []pseudo.Spec) float64 {
		s := 0.0
		for i := range truth {
			s += semvideo.MarksIoU(truth[i], rs[i])
		}
		return s / float64(len(truth))
	}
	wire := 0
	for _, p := range pkts {
		wire += len(p.Wire())
	}
	fmt.Printf("one world, two minds — broadcast once (%d B), imagined locally:\n", wire)
	fmt.Printf("  mind A: %d dropped, meaning recovered %.3f\n", lossA, iou(recvA))
	fmt.Printf("  mind B: %d dropped, meaning recovered %.3f\n", lossB, iou(recvB))

	path := fmt.Sprintf("%s/imagine.gif", *out)
	if err := writeGIF(path, truth, recvA, recvB); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (sent | mind A | mind B)\n", path)
}

func receive(pkts []semvideo.Packet, ecc, w, h int, seed int64, loss float64) ([]pseudo.Spec, int) {
	r := semvideo.NewReceiver(ecc)
	rng := rand.New(rand.NewSource(seed))
	out := make([]pseudo.Spec, len(pkts))
	dropped := 0
	for i, p := range pkts {
		if rng.Float64() < loss {
			r.MarkLoss()
			dropped++
		} else {
			r.Apply(p)
		}
		s, ok := r.Frame()
		if !ok || s.Marks == nil {
			s = arenacast.Blank(w, h)
		}
		out[i] = s
	}
	return out, dropped
}

const cw, ch = 220, 176
const gW, gH = 60 + 3*cw + 2*24, 260

func writeGIF(path string, truth, a, b []pseudo.Spec) error {
	g := &gif.GIF{}
	n := len(truth)
	for i := 0; i < n+3; i++ {
		k := i
		if k >= n {
			k = n - 1
		}
		fr := compose(truth[k], a[k], b[k])
		pi := image.NewPaletted(fr.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, fr.Bounds(), fr, image.Point{})
		g.Image = append(g.Image, pi)
		d := 18
		if i >= n {
			d = 120
		}
		g.Delay = append(g.Delay, d)
	}
	fp, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fp.Close()
	return gif.EncodeAll(fp, g)
}

func compose(t, a, b pseudo.Spec) image.Image {
	c := image.NewRGBA(image.Rect(0, 0, gW, gH))
	fillRect(c, 0, 0, gW, gH, color.RGBA{14, 14, 22, 255})
	xs := []int{20, 20 + cw + 24, 20 + 2*(cw+24)}
	cols := []color.RGBA{{210, 210, 220, 255}, {120, 180, 240, 255}, {240, 170, 90, 255}}
	for i, s := range []pseudo.Spec{t, a, b} {
		drawSpec(c, s, xs[i], 48)
		outline(c, xs[i]-4, 44, cw+8, ch+8, cols[i])
	}
	return c
}

func drawSpec(dst *image.RGBA, s pseudo.Spec, x, y int) {
	img, err := s.Image(cw, ch)
	if err != nil || img == nil {
		fillRect(dst, x, y, cw, ch, color.RGBA{20, 20, 30, 255})
		return
	}
	draw.Draw(dst, image.Rect(x, y, x+cw, y+ch), img, img.Bounds().Min, draw.Over)
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{c}, image.Point{}, draw.Src)
}

func outline(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	fillRect(img, x, y, w, 2, c)
	fillRect(img, x, y+h-2, w, 2, c)
	fillRect(img, x, y, 2, h, c)
	fillRect(img, x+w-2, y, 2, h, c)
}
