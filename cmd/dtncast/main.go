// Command dtncast streams an arena match over a DELAY-TOLERANT (deep-space) link
// (pkg/dtn over pkg/semvideo): the world's frames become bundles forwarded only
// during a few short contact passes, with a time-to-live. Many bundles expire,
// yet the surviving keyframes still let the receiver reconstruct most of the
// meaning — "video for the Bundle Protocol", measured.
//
//	go run ./cmd/dtncast
//	go run ./cmd/dtncast -ttl 10 -per-pass 6
package main

import (
	"flag"
	"fmt"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/arenacast"
	"github.com/svend4/infon/pkg/dtn"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
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

func main() {
	ecc := flag.Int("ecc", 8, "RS parity")
	gop := flag.Int("gop", 4, "keyframe interval")
	ttl := flag.Int("ttl", 6, "bundle time-to-live (ticks)")
	perPass := flag.Int("per-pass", 4, "bundles forwarded per contact")
	ticks := flag.Int("ticks", 24, "max ticks")
	flag.Parse()

	a := board()
	var truth []pseudo.Spec
	for f := 0; f < *ticks; f++ {
		truth = append(truth, arenacast.Spec(a))
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		a.Step(arena.Planner{}, arena.RefCommander{})
	}
	pkts := semvideo.EncodeFrames(truth, *ecc, *gop)
	n := len(pkts)

	// a deep-space schedule: brief, widely spaced line-of-sight passes.
	contacts := []dtn.Contact{{Start: 3, End: 4}, {Start: 9, End: 10}, {Start: 15, End: 16}, {Start: 21, End: 22}}
	rep := dtn.Simulate(n, contacts, *ttl, *perPass)

	// reconstruct: apply delivered bundles in frame order, freeze on the rest.
	r := semvideo.NewReceiver(*ecc)
	iouSum := 0.0
	for i := 0; i < n; i++ {
		if rep.Mask[i] {
			r.Apply(pkts[i])
		} else {
			r.MarkLoss()
		}
		s, ok := r.Frame()
		if !ok || s.Marks == nil {
			s = arenacast.Blank(MW, MH)
		}
		iouSum += semvideo.MarksIoU(truth[i], s)
	}
	iou := iouSum / float64(n)

	keys := 0
	for _, p := range pkts {
		if p.Kind == semvideo.KeyFrame {
			keys++
		}
	}

	fmt.Printf("a world of %d frames over a delay-tolerant (deep-space) link:\n", n)
	fmt.Printf("  %d contact passes · ttl %d ticks · %d bundles/pass · %d keyframes (gop %d, ecc %d)\n",
		len(contacts), *ttl, *perPass, keys, *gop, *ecc)
	fmt.Printf("  delivered %d/%d bundles (%d expired) · avg latency %.1f ticks\n",
		rep.Delivered, rep.Total, rep.Expired, rep.AvgLatency)
	fmt.Printf("  meaning reconstructed at the receiver: %.3f IoU — surviving keyframes still carry the world\n", iou)
}
