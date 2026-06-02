// Command semcall is a REAL semantic video call: it sends a marks-spec stream
// (keyframes + semframe P-frames, each Reed-Solomon protected) over an actual
// localhost UDP socket under lossy conditions — packets are dropped and bytes
// are corrupted in flight. The receiver RS-corrects damaged packets and freezes
// /re-syncs across dropped ones (periodic keyframes), then renders the received
// timeline to a GIF. A few hundred bytes/sec of meaning that survives a bad link.
//
//	go run ./cmd/semcall [outdir]
package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
)

const (
	W, H, SCALE = 24, 12, 18
	N, GOP, ECC = 24, 6, 16
)

func frame(step int) pseudo.Spec {
	rows := make([][]string, H)
	cols := make([][]string, H)
	for y := 0; y < H; y++ {
		rows[y] = make([]string, W)
		cols[y] = make([]string, W)
	}
	set := func(y, x int, g, c string) {
		if y >= 0 && y < H && x >= 0 && x < W {
			rows[y][x], cols[y][x] = g, c
		}
	}
	set(1, W-4, "tri-lr", "gold")
	set(1, W-3, "full", "gold")
	set(2, W-4, "full", "gold")
	set(2, W-3, "tri-ul", "gold")
	for x := 0; x < W; x++ {
		set(H-1, x, "top", "teal")
	}
	x := 1 + step%(W-5)
	wing := "tri-ur"
	if step%2 == 0 {
		wing = "tri-lr"
	}
	set(5, x, wing, "white")
	set(5, x+1, "full", "white")
	set(5, x+2, "tri-ul", "white")
	set(4, x+1, "tri-lr", "white")
	set(6, x, "tri-ur", "white")
	return pseudo.Spec{Format: pseudo.FormatMarks, Cols: W, Rows: H,
		Marks: &pseudo.MarkArt{Rows: rows, Colors: cols, Bg: "navy"}}
}

func main() {
	out := "./_semcall"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	_ = os.MkdirAll(out, 0o755)

	specs := make([]pseudo.Spec, N)
	for i := range specs {
		specs[i] = frame(i)
	}
	pkts := semvideo.EncodeFrames(specs, ECC, GOP)

	// Real UDP loopback.
	rx, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		fail(err)
	}
	defer rx.Close()
	tx, err := net.DialUDP("udp", nil, rx.LocalAddr().(*net.UDPAddr))
	if err != nil {
		fail(err)
	}
	defer tx.Close()

	arrived := make(chan semvideo.Packet, 256)
	go func() {
		buf := make([]byte, 65535)
		for {
			_ = rx.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
			n, _, err := rx.ReadFromUDP(buf)
			if err != nil {
				close(arrived)
				return
			}
			if p, ok := semvideo.ParseWire(buf[:n]); ok {
				cp := make([]byte, len(p.Payload))
				copy(cp, p.Payload)
				arrived <- semvideo.Packet{Idx: p.Idx, Kind: p.Kind, Payload: cp}
			}
		}
	}()

	// Lossy sender: drop ~18%, corrupt a within-budget burst in ~28% (idx 0 safe).
	rng := rand.New(rand.NewSource(7))
	sent, dropped, corrupted := 0, 0, 0
	corruptedIdx := map[uint16]bool{}
	for _, p := range pkts {
		if p.Idx != 0 && rng.Float64() < 0.18 {
			dropped++
			continue
		}
		w := p.Wire()
		if p.Idx != 0 && rng.Float64() < 0.28 {
			nb := len(p.Payload) / 255
			if nb < 1 {
				nb = 1
			}
			burst := nb * (ECC/2 - 1) // stays within Reed-Solomon budget
			off := 3 + 8
			for i := off; i < off+burst && i < len(w); i++ {
				w[i] ^= 0xFF
			}
			corrupted++
			corruptedIdx[p.Idx] = true
		}
		_, _ = tx.Write(w)
		sent++
		time.Sleep(2 * time.Millisecond)
	}
	time.Sleep(60 * time.Millisecond)
	_ = rx.SetReadDeadline(time.Now().Add(1 * time.Millisecond)) // unblock the reader

	got := map[uint16]semvideo.Packet{}
	for p := range arrived {
		got[p.Idx] = p
	}

	// Reconstruct the timeline with freeze-on-loss + keyframe re-sync.
	r := semvideo.NewReceiver(ECC)
	outSpecs := make([]pseudo.Spec, N)
	frozen, recovered := 0, 0
	for i := 0; i < N; i++ {
		if p, ok := got[uint16(i)]; ok {
			adv := r.Apply(p)
			if corruptedIdx[uint16(i)] && adv {
				recovered++
			}
		} else {
			r.MarkLoss()
			frozen++
		}
		f, have := r.Frame()
		if have {
			outSpecs[i] = f
		} else {
			outSpecs[i] = specs[0] // before first keyframe
		}
	}

	exact := 0
	for i := 0; i < N; i++ {
		if outSpecs[i].Marks != nil && reflect.DeepEqual(outSpecs[i].Marks.Rows, specs[i].Marks.Rows) {
			exact++
		}
	}

	fmt.Println("=== semantic video call over a REAL lossy UDP link ===")
	fmt.Printf("%d frames %dx%d, GOP=%d, RS ecc=%d\n", N, W, H, GOP, ECC)
	fmt.Printf("packets: %d sent, %d dropped(lost), %d corrupted-in-flight\n", sent, dropped, corrupted)
	fmt.Printf("receiver: %d corrupted packets Reed-Solomon-RECOVERED, %d frames frozen (held over loss)\n", recovered, frozen)
	fmt.Printf("result: %d/%d frames exact; the rest are clean freezes that re-sync at the next keyframe\n", exact, N)
	saveGIF(filepath.Join(out, "received.gif"), outSpecs)
	saveGIF(filepath.Join(out, "sent.gif"), specs)
	fmt.Printf("GIFs in %s (sent.gif, received.gif)\n", out)
}

func render(s pseudo.Spec) image.Image {
	fr, err := s.Frame()
	if err != nil {
		fail(err)
	}
	return glyphset.RasterizeSize(fr, W*SCALE, H*SCALE)
}
func saveGIF(path string, specs []pseudo.Spec) {
	g := &gif.GIF{}
	for _, s := range specs {
		p := image.NewPaletted(image.Rect(0, 0, W*SCALE, H*SCALE), palette.Plan9)
		draw.Draw(p, p.Bounds(), render(s), image.Point{}, draw.Src)
		g.Image = append(g.Image, p)
		g.Delay = append(g.Delay, 16)
	}
	f, err := os.Create(path)
	if err != nil {
		fail(err)
	}
	defer func() { _ = f.Close() }()
	if err := gif.EncodeAll(f, g); err != nil {
		fail(err)
	}
}
func fail(err error) { fmt.Fprintln(os.Stderr, "semcall:", err); os.Exit(1) }
