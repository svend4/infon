// Command watch is the project's headline scenario, fused into ONE experience:
// two neural-or-reference commanders fight an arena match; the match is broadcast
// as a lossy SEMANTIC VIDEO CALL (pkg/arenacast + pkg/semvideo); a spectator
// reconstructs it; the events become a byte CHRONICLE (pkg/chronicle) shown as a
// live caption; and the match's energy is SONIFIED (pkg/sona) to a melody. You
// WATCH and HEAR the war of two AIs over a few hundred bytes per second — one
// command that ties the symbolic substrate, the agent-world, and the call into a
// single product.
//
//	go run ./cmd/watch
//	go run ./cmd/watch -loss 0.3
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
	"github.com/svend4/infon/pkg/chronicle"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
	"github.com/svend4/infon/pkg/sona"
	"github.com/svend4/infon/pkg/tangram"
)

const MW, MH = 10, 8

func board() *arena.Arena {
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
	return a
}

// commander returns the blue/red commander: the planner / reference bot by
// default, or a live tvcp-ai brain when a URL is given.
func commander(url string, blue bool) arena.Commander {
	if url == "" {
		if blue {
			return arena.Planner{}
		}
		return arena.RefCommander{}
	}
	return arena.BrainCommander{B: brain.HTTPBrain{URL: url, HTTP: &http.Client{Timeout: 60 * time.Second}}}
}

func main() {
	loss := flag.Float64("loss", 0.15, "packet loss")
	ecc := flag.Int("ecc", 8, "RS parity")
	gop := flag.Int("gop", 6, "keyframe interval")
	ticks := flag.Int("ticks", 26, "max ticks")
	b0 := flag.String("brain0", "", "blue commander brain URL (default: planner)")
	b1 := flag.String("brain1", "", "red commander brain URL (default: reference)")
	out := flag.String("out", "_watch", "output directory")
	flag.Parse()
	os.MkdirAll(*out, 0o755)

	c0 := commander(*b0, true)
	c1 := commander(*b1, false)

	// 1) run the match ONCE, capturing the world, its energy, and unit snapshots
	//    (so the chronicle comes from the same match — works with live brains too).
	a := board()
	var truth []pseudo.Spec
	var hp, a0, a1 []int
	frames := [][]arena.Unit{append([]arena.Unit(nil), a.Units...)}
	for f := 0; f < *ticks; f++ {
		truth = append(truth, arenacast.Spec(a))
		hp = append(hp, a.TotalHP())
		a0 = append(a0, a.AliveCount(0))
		a1 = append(a1, a.AliveCount(1))
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		a.Step(c0, c1)
		frames = append(frames, append([]arena.Unit(nil), a.Units...))
	}
	n := len(truth)

	// 2) the chronicle, derived from the very frames we just played.
	log := chronicle.FromFrames(frames)
	caption := captions(log, n)

	// 3) sonify the match's energy and write the WAV.
	notes := sona.SeriesMelody(hp, 170*time.Millisecond)
	wav := sona.WAV(sona.Synth(notes, sona.DefaultRate), sona.DefaultRate)
	wavPath := fmt.Sprintf("%s/watch.wav", *out)
	_ = os.WriteFile(wavPath, wav, 0o644)

	// 4) broadcast the world as a lossy semantic call; a spectator reconstructs it.
	pkts := semvideo.EncodeFrames(truth, *ecc, *gop)
	r := semvideo.NewReceiver(*ecc)
	rng := rand.New(rand.NewSource(7))
	recv := make([]pseudo.Spec, n)
	wire, dropped := 0, 0
	for i := 0; i < n && i < len(pkts); i++ {
		wire += len(pkts[i].Wire())
		if rng.Float64() < *loss {
			r.MarkLoss()
			dropped++
		} else {
			r.Apply(pkts[i])
		}
		s, ok := r.Frame()
		if !ok || s.Marks == nil {
			s = arenacast.Blank(MW, MH)
		}
		recv[i] = s
	}
	iouSum := 0.0
	for i := 0; i < n; i++ {
		iouSum += semvideo.MarksIoU(truth[i], recv[i])
	}
	avgIoU := iouSum / float64(n)

	winner := "draw"
	if a0[n-1] > a1[n-1] {
		winner = "blue"
	} else if a1[n-1] > a0[n-1] {
		winner = "red"
	}

	fmt.Printf("TVCP WATCH — two AIs, one channel:\n")
	fmt.Printf("  %d-tick match, winner %s (%d v %d)\n", n, winner, a0[n-1], a1[n-1])
	fmt.Printf("  call: %d B over RS (~%.0f B/s @ 6fps), %d dropped (%.0f%% loss), meaning %.3f IoU\n",
		wire, float64(wire)/(float64(n)/6.0), dropped, *loss*100, avgIoU)
	fmt.Printf("  chronicle: %d events in %d bytes\n", len(log.Events), len(log.Bytes()))
	fmt.Printf("  song: %d notes\n", len(notes))

	gifPath := fmt.Sprintf("%s/watch.gif", *out)
	if err := writeGIF(gifPath, recv, a0, a1, caption, notes, winner); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s + %s (watch the call, read the chronicle, hear the war)\n", gifPath, wavPath)
}

func host(f int) string {
	if f == 0 {
		return "blue"
	}
	return "red"
}

// captions builds a sticky per-frame caption from the chronicle's events.
func captions(log chronicle.Log, n int) []string {
	name := func(i int) string {
		if i < 0 || i >= len(log.Roster) {
			return "?"
		}
		return host(log.Roster[i].Faction) + " " + log.Roster[i].Name
	}
	byTick := make([]string, n+2)
	for _, e := range log.Events {
		if e.Tick < 0 || e.Tick >= len(byTick) {
			continue
		}
		switch e.Kind {
		case chronicle.Strike:
			byTick[e.Tick] = name(e.Actor) + " hits " + name(e.Target)
		case chronicle.Fall:
			byTick[e.Tick] = name(e.Target) + " falls"
		}
	}
	out := make([]string, n)
	last := "the hosts meet"
	for i := 0; i < n; i++ {
		if i < len(byTick) && byTick[i] != "" {
			last = byTick[i]
		}
		out[i] = last
	}
	return out
}

const gW, gH = 760, 430

func writeGIF(path string, recv []pseudo.Spec, a0, a1 []int, caption []string, notes []sona.Note, winner string) error {
	g := &gif.GIF{}
	n := len(recv)
	for i := 0; i < n+4; i++ {
		k := i
		if k >= n {
			k = n - 1
		}
		fr := compose(recv[k], a0[k], a1[k], caption[k], notes, k, i >= n, winner)
		pi := image.NewPaletted(fr.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, fr.Bounds(), fr, image.Point{})
		g.Image = append(g.Image, pi)
		d := 17
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

func compose(world pseudo.Spec, a0, a1 int, cap string, notes []sona.Note, cur int, done bool, winner string) image.Image {
	c := image.NewRGBA(image.Rect(0, 0, gW, gH))
	fillRect(c, 0, 0, gW, gH, color.RGBA{14, 14, 22, 255})
	white := color.RGBA{232, 236, 245, 255}
	blue := color.RGBA{90, 150, 235, 255}
	orange := color.RGBA{240, 150, 60, 255}

	microfont.Draw(c, 24, 14, 2, "TVCP WATCH  TWO AIS ONE CHANNEL", color.RGBA{240, 200, 90, 255})

	// survivor bar
	const bx, by, bw, bh = 24, 40, 380, 14
	tot := a0 + a1
	if tot == 0 {
		tot = 1
	}
	wb := bw * a0 / tot
	fillRect(c, bx, by, wb, bh, blue)
	fillRect(c, bx+wb, by, bw-wb, bh, orange)

	// the received world (the "call")
	if img, err := world.Image(380, 250); err == nil && img != nil {
		draw.Draw(c, image.Rect(24, 64, 24+380, 64+250), img, img.Bounds().Min, draw.Over)
	}
	outline(c, 20, 60, 388, 258, color.RGBA{40, 40, 56, 255})

	// chronicle caption (live)
	microfont.Draw(c, 24, 332, 2, upper(cap), white)

	// melody piano-roll (right): notes so far, height by pitch
	rx, ry, rw, rh := 430, 64, 306, 250
	fillRect(c, rx, ry, rw, rh, color.RGBA{22, 22, 32, 255})
	microfont.Draw(c, rx, ry-16, 1, "SONG (TOTAL HP)", white)
	if len(notes) > 0 {
		bwn := rw / len(notes)
		if bwn < 2 {
			bwn = 2
		}
		const lo, hi = 57, 92
		for j := 0; j <= cur && j < len(notes); j++ {
			frac := float64(notes[j].Midi-lo) / float64(hi-lo)
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			h := 4 + int(frac*float64(rh-8))
			col := color.RGBA{120, 200, 200, 255}
			if j == cur {
				col = white
			}
			fillRect(c, rx+j*bwn, ry+rh-h, bwn-1, h, col)
		}
	}

	if done {
		msg := "DRAW"
		if winner == "blue" {
			msg = "BLUE (PLANNER) HOLDS THE FIELD"
		} else if winner == "red" {
			msg = "RED HOLDS THE FIELD"
		}
		microfont.Draw(c, 24, 360, 2, msg, color.RGBA{90, 220, 120, 255})
	}
	return c
}

func upper(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 32
		}
	}
	return string(b)
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
