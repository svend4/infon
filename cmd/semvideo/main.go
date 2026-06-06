// Command semvideo is the "closed loop": a semantic video call. Instead of
// H.264, it sends a stream of marks specs (a keyframe + semframe P-frame deltas)
// wrapped in Reed-Solomon, renders the byte budget, tears a burst out of a
// mid-stream packet, and shows the call still decodes — then writes the original
// and the recovered streams as GIFs (they are identical).
//
//	go run ./cmd/semvideo [outdir]
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"
	"path/filepath"
	"reflect"

	"github.com/svend4/infon/pkg/glyphset"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
)

const (
	W, H, FPS, SCALE = 24, 12, 6, 18
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
	set(1, W-4, "tri-lr", "gold") // sun
	set(1, W-3, "full", "gold")
	set(2, W-4, "full", "gold")
	set(2, W-3, "tri-ul", "gold")
	for x := 0; x < W; x++ {
		set(H-1, x, "top", "teal") // ground band
	}
	x := 2 + step // bird flies right
	wing := "tri-ur"
	if step%2 == 0 {
		wing = "tri-lr"
	}
	set(5, x, wing, "white")     // flapping wing
	set(5, x+1, "full", "white") // body
	set(5, x+2, "tri-ul", "white")
	set(4, x+1, "tri-lr", "white") // head
	set(6, x, "tri-ur", "white")   // tail
	return pseudo.Spec{Format: pseudo.FormatMarks, Cols: W, Rows: H,
		Marks: &pseudo.MarkArt{Rows: rows, Colors: cols, Bg: "navy"}}
}

func main() {
	out := "./_semvideo"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	_ = os.MkdirAll(out, 0o755)

	const N = 16
	specs := make([]pseudo.Spec, N)
	for i := 0; i < N; i++ {
		specs[i] = frame(i)
	}
	ecc := 16

	pkts, err := semvideo.Encode(specs, ecc)
	if err != nil {
		fail(err)
	}
	stream := semvideo.Bytes(pkts)

	// all-keyframes baseline + raw (no ECC) payload.
	allKey, raw := 0, 0
	for _, s := range specs {
		kp, _ := semvideo.Encode([]pseudo.Spec{s}, ecc)
		allKey += semvideo.Bytes(kp)
		b, _ := json.Marshal(s)
		raw += len(b)
	}
	dur := float64(N) / float64(FPS)
	fmt.Println("=== semantic video call (marks + P-frames + Reed-Solomon) ===")
	fmt.Printf("%d frames, %dx%d, %d fps  (%.1fs)\n", N, W, H, FPS, dur)
	fmt.Printf("stream (1 keyframe + %d P-frames, RS ecc=%d): %d bytes  =>  %.0f bytes/sec\n", N-1, ecc, stream, float64(stream)/dur)
	fmt.Printf("all-keyframes would be: %d bytes   (P-frame stream = %.0f%% of that)\n", allKey, 100*float64(stream)/float64(allKey))
	fmt.Printf("raw spec JSON (no ECC): %d bytes; ECC overhead = %.1fx\n", raw, float64(stream)/float64(raw))
	fmt.Printf("the ENTIRE call is %d bytes — smaller than one H.264 keyframe.\n", stream)

	// Tear a burst out of a mid-stream P-frame (simulate packet damage).
	k := N / 2
	burst := ecc / 2
	for i := 4; i < 4+burst && i < len(pkts[k]); i++ {
		pkts[k][i] ^= 0xFF
	}
	fmt.Printf("\ndamaged %d bytes inside packet #%d (a P-frame)...\n", burst, k)

	got, err := semvideo.Decode(pkts, ecc)
	if err != nil {
		fmt.Printf("decode FAILED: %v\n", err)
	} else {
		exact := len(got) == N
		for i := 0; i < N && exact; i++ {
			if !reflect.DeepEqual(got[i].Marks.Rows, specs[i].Marks.Rows) {
				exact = false
			}
		}
		fmt.Printf("decode OK — recovered all %d frames exactly: %v\n", N, exact)
		saveGIF(filepath.Join(out, "recovered.gif"), got)
	}
	saveGIF(filepath.Join(out, "original.gif"), specs)
	fmt.Printf("\nGIFs in %s (original.gif, recovered.gif)\n", out)
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
func fail(err error) { fmt.Fprintln(os.Stderr, "semvideo:", err); os.Exit(1) }
