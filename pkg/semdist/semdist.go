// Package semdist measures RATE vs DISTORTION OF MEANING — not of pixels. Given a
// short marks "video", it streams it through pkg/semvideo at different bitrates
// and channel conditions and reports, for each operating point, the wire bytes
// against how much MEANING survived (mean MarksIoU). It turns the project's
// "fidelity of meaning" thesis into a reproducible curve.
package semdist

import (
	"math/rand"

	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
)

// Point is one operating point on a rate-distortion-of-meaning curve.
type Point struct {
	Knob   int     // the varied parameter (gop, or ecc)
	Bytes  int     // total wire bytes
	IoU    float64 // mean MarksIoU of the received stream vs the truth
	Exact  int     // frames reconstructed exactly
	Frames int
}

func blank(w, h int) pseudo.Spec {
	rows := make([][]string, h)
	cols := make([][]string, h)
	for y := 0; y < h; y++ {
		rows[y] = make([]string, w)
		cols[y] = make([]string, w)
	}
	return pseudo.Spec{Format: pseudo.FormatMarks, Cols: w, Rows: h, Marks: &pseudo.MarkArt{Rows: rows, Colors: cols}}
}

// Scene builds an n-frame marks "video": a mark sweeping across the grid (so
// consecutive frames differ by a couple of cells — small P-frames).
func Scene(n, w, h int) []pseudo.Spec {
	specs := make([]pseudo.Spec, n)
	for i := 0; i < n; i++ {
		s := blank(w, h)
		x := i % w
		s.Marks.Rows[h/2][x], s.Marks.Colors[h/2][x] = "full", "gold"
		if x+1 < w {
			s.Marks.Rows[h/2][x+1], s.Marks.Colors[h/2][x+1] = "tri-ul", "skyblue"
		}
		specs[i] = s
	}
	return specs
}

func score(truth, recv []pseudo.Spec) (float64, int) {
	sum, exact := 0.0, 0
	for i := range truth {
		v := semvideo.MarksIoU(truth[i], recv[i])
		sum += v
		if v >= 0.999 {
			exact++
		}
	}
	if len(truth) == 0 {
		return 0, 0
	}
	return sum / float64(len(truth)), exact
}

func dims(specs []pseudo.Spec) (w, h int) {
	if len(specs) == 0 || specs[0].Marks == nil {
		return 1, 1
	}
	h = len(specs[0].Marks.Rows)
	if h > 0 {
		w = len(specs[0].Marks.Rows[0])
	}
	return w, h
}

// GopCurve varies the keyframe interval under a fixed packet-loss rate. Smaller
// gop costs more bytes (more keyframes) but resynchronizes faster after loss, so
// more meaning survives — a genuine bitrate-vs-meaning trade-off.
func GopCurve(specs []pseudo.Spec, gops []int, ecc int, loss float64, seed int64) []Point {
	w, h := dims(specs)
	var out []Point
	for _, gop := range gops {
		pkts := semvideo.EncodeFrames(specs, ecc, gop)
		r := semvideo.NewReceiver(ecc)
		rng := rand.New(rand.NewSource(seed))
		recv := make([]pseudo.Spec, len(pkts))
		bytes := 0
		for i, p := range pkts {
			bytes += len(p.Wire())
			if rng.Float64() < loss {
				r.MarkLoss()
			} else {
				r.Apply(p)
			}
			s, ok := r.Frame()
			if !ok || s.Marks == nil {
				s = blank(w, h)
			}
			recv[i] = s
		}
		iou, exact := score(specs, recv)
		out = append(out, Point{Knob: gop, Bytes: bytes, IoU: iou, Exact: exact, Frames: len(specs)})
	}
	return out
}

// EccCurve varies Reed-Solomon parity under a fixed per-packet byte corruption.
// More parity costs more bytes but heals more corruption, so meaning survives
// once ecc/2 >= the corrupted bytes.
func EccCurve(specs []pseudo.Spec, eccs []int, corruptBytes int, seed int64) []Point {
	w, h := dims(specs)
	var out []Point
	for _, ecc := range eccs {
		pkts := semvideo.EncodeFrames(specs, ecc, 4)
		rng := rand.New(rand.NewSource(seed))
		r := semvideo.NewReceiver(ecc)
		recv := make([]pseudo.Spec, len(pkts))
		bytes := 0
		for i, p := range pkts {
			// corrupt up to corruptBytes positions in the RS payload
			for k := 0; k < corruptBytes && len(p.Payload) > 0; k++ {
				j := rng.Intn(len(p.Payload))
				p.Payload[j] ^= byte(1 + rng.Intn(255))
			}
			bytes += len(p.Wire())
			r.Apply(p)
			s, ok := r.Frame()
			if !ok || s.Marks == nil {
				s = blank(w, h)
			}
			recv[i] = s
		}
		iou, exact := score(specs, recv)
		out = append(out, Point{Knob: ecc, Bytes: bytes, IoU: iou, Exact: exact, Frames: len(specs)})
	}
	return out
}
