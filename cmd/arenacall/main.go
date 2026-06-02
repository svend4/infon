// Command arenacall is the project's "three bodies into one core" demo: a live
// arena match (the agent-world) is rendered as marks pseudo-images and streamed
// through pkg/semvideo — the very codec TVCP uses for a semantic video call — so
// a spectator WATCHES THE WAR OF TWO NEURAL NETS AS A FEW-HUNDRED-BYTES/S CALL
// that survives packet loss. It prints the call's byte budget and how much
// MEANING (MarksIoU) survived, and renders a GIF: left = the world as sent,
// right = the world as received after loss (freeze on drop, resync on keyframe).
//
//	go run ./cmd/arenacall                 # reference brains, 15% loss
//	go run ./cmd/arenacall -loss 0.3
//	go run ./cmd/arenacall -brain0 URL -brain1 URL   # live LLM commanders
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
	"net/http"
	"os"
	"time"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/arenacast"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
	"github.com/svend4/infon/pkg/tangram"
)

func mkBrain(url string) brain.Brain {
	if url == "" {
		return brain.Local{}
	}
	return brain.HTTPBrain{URL: url, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

func main() {
	b0 := flag.String("brain0", "", "blue commander brain URL (default: reference)")
	b1 := flag.String("brain1", "", "red commander brain URL (default: reference)")
	loss := flag.Float64("loss", 0.15, "packet loss probability")
	ecc := flag.Int("ecc", 8, "Reed-Solomon parity bytes per packet")
	gop := flag.Int("gop", 6, "keyframe interval in frames")
	maxTicks := flag.Int("ticks", 28, "max match ticks")
	fps := flag.Int("fps", 6, "ticks per second (for the bytes/second figure)")
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
	c0 := arena.BrainCommander{B: mkBrain(*b0)}
	c1 := arena.BrainCommander{B: mkBrain(*b1)}

	// 1) run the match, capturing each tick as a marks pseudo-image.
	var truth []pseudo.Spec
	for f := 0; f < *maxTicks; f++ {
		truth = append(truth, arenacast.Spec(a))
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		a.Step(c0, c1)
	}

	// 2) encode as a semantic-video stream (keyframe + P-frame deltas over RS).
	pkts := semvideo.EncodeFrames(truth, *ecc, *gop)

	// 3) push it through a lossy channel into a freeze/resync receiver.
	r := semvideo.NewReceiver(*ecc)
	rng := rand.New(rand.NewSource(7))
	recv := make([]pseudo.Spec, len(pkts))
	var wire, dropped, keyN, pfN int
	for i, p := range pkts {
		wire += len(p.Wire())
		if p.Kind == semvideo.KeyFrame {
			keyN++
		} else {
			pfN++
		}
		if rng.Float64() < *loss {
			r.MarkLoss()
			dropped++
		} else {
			r.Apply(p)
		}
		s, ok := r.Frame()
		if !ok || s.Marks == nil {
			s = arenacast.Blank(MW, MH)
		}
		recv[i] = s
	}

	// 4) how much MEANING survived?
	iouSum, exact := 0.0, 0
	exactFrame := make([]bool, len(truth))
	for i := range truth {
		iou := semvideo.MarksIoU(truth[i], recv[i])
		iouSum += iou
		if iou >= 0.999 {
			exact++
			exactFrame[i] = true
		}
	}
	avg := iouSum / float64(len(truth))
	secs := float64(len(truth)) / float64(*fps)

	label := func(u string) string {
		if u == "" {
			return "reference"
		}
		return u
	}
	fmt.Printf("arena as a tvcp semantic call — blue=%s red=%s\n", label(*b0), label(*b1))
	fmt.Printf("  %d frames, %d packets (%d key / %d P), %d dropped (%.0f%% loss)\n",
		len(truth), len(pkts), keyN, pfN, dropped, *loss*100)
	fmt.Printf("  wire: %d B total · %.1f B/frame · ~%.0f B/s @ %d fps (ecc=%d, gop=%d)\n",
		wire, float64(wire)/float64(len(truth)), float64(wire)/secs, *fps, *ecc, *gop)
	fmt.Printf("  meaning: avg MarksIoU %.3f · %d/%d frames exact (freeze+resync on loss)\n",
		avg, exact, len(truth))

	gifPath := fmt.Sprintf("%s/arenacall.gif", *out)
	if err := writeGIF(gifPath, truth, recv, exactFrame); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (left = world as sent · right = world as received)\n", gifPath)
}

const (
	cw, ch = 300, 240
	gW, gH = 656, 320
)

func writeGIF(path string, truth, recv []pseudo.Spec, exact []bool) error {
	g := &gif.GIF{}
	n := len(truth)
	for i := 0; i < n+3; i++ {
		k := i
		if k >= n {
			k = n - 1
		}
		fr := compose(truth[k], recv[k], exact[k])
		pi := image.NewPaletted(fr.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, fr.Bounds(), fr, image.Point{})
		g.Image = append(g.Image, pi)
		d := 16
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

func compose(sent, got pseudo.Spec, ok bool) image.Image {
	c := image.NewRGBA(image.Rect(0, 0, gW, gH))
	fillRect(c, 0, 0, gW, gH, color.RGBA{14, 14, 22, 255})

	lx, rx, y := 16, 340, 48
	drawSpec(c, sent, lx, y)
	drawSpec(c, got, rx, y)

	// sent panel: steady white frame.
	outline(c, lx-4, y-4, cw+8, ch+8, color.RGBA{210, 210, 220, 255})
	// received panel: green when exact, amber when frozen/degraded.
	rb := color.RGBA{90, 220, 120, 255}
	if !ok {
		rb = color.RGBA{240, 180, 60, 255}
	}
	outline(c, rx-4, y-4, cw+8, ch+8, rb)
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
	fillRect(img, x, y, w, 3, c)
	fillRect(img, x, y+h-3, w, 3, c)
	fillRect(img, x, y, 3, h, c)
	fillRect(img, x+w-3, y, 3, h, c)
}
