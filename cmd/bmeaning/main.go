// Command bmeaning demonstrates bidirectional B-frames of meaning (pkg/semvideo):
// a constant-shape marks "video" is coded as anchors (keyframe + P-frames) with
// B-frames in between, each predicted from the nearer anchor. It shows that
// coding order differs from display order (DTS vs PTS), that the stream
// round-trips exactly through Reed-Solomon, and that dropping every B-frame still
// decodes a valid lower-frame-rate video — B-frames are disposable.
//
//	go run ./cmd/bmeaning
package main

import (
	"fmt"

	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
)

const (
	nFrames = 9
	width   = 8
	gop     = 4
	ecc     = 12
)

// frame draws a "full" dot walking the middle row and a faster "tri-ul" mark on
// the top row — a tiny moving scene on a fixed grid.
func frame(step int) pseudo.Spec {
	rows := make([][]string, 3)
	for y := range rows {
		rows[y] = make([]string, width)
		for x := range rows[y] {
			rows[y][x] = ""
		}
	}
	rows[1][step%width] = "full"
	rows[0][(step*2)%width] = "tri-ul"
	return pseudo.Spec{Format: pseudo.FormatMarks, Cols: width, Rows: 3,
		Marks: &pseudo.MarkArt{Rows: rows, Bg: "navy"}}
}

func render(s pseudo.Spec, prefix string) string {
	gl := map[string]rune{"": '.', "full": '#', "tri-ul": '/'}
	out := ""
	for _, row := range s.Marks.Rows {
		out += prefix
		for _, t := range row {
			r := gl[t]
			if r == 0 {
				r = '?'
			}
			out += string(r)
		}
		out += "\n"
	}
	return out
}

func kinds(pk [][]byte) (k, p, b int) {
	for _, x := range pk {
		switch x[0] {
		case semvideo.KeyFrame:
			k++
		case semvideo.PFrame:
			p++
		case semvideo.BFrame:
			b++
		}
	}
	return
}

func main() {
	src := make([]pseudo.Spec, nFrames)
	for i := range src {
		src[i] = frame(i)
	}

	pk, disp, err := semvideo.BiEncode(src, gop, ecc)
	if err != nil {
		fmt.Println("encode:", err)
		return
	}
	k, p, b := kinds(pk)
	fmt.Printf("%d frames -> %d packets: %d key, %d P (anchors), %d B; %d bytes (ecc=%d)\n",
		nFrames, len(pk), k, p, b, semvideo.Bytes(pk), ecc)
	fmt.Printf("display index of each coded packet (DTS -> PTS): %v\n", disp)
	fmt.Println("  (coding order puts each closing anchor BEFORE the B-frames that predict from it)")

	got, err := semvideo.BiDecode(pk, disp, ecc)
	if err != nil {
		fmt.Println("decode:", err)
		return
	}
	exact := true
	for i := range src {
		if render(got[i], "") != render(src[i], "") {
			exact = false
		}
	}
	fmt.Printf("full round-trip through Reed-Solomon: exact=%v\n\n", exact)

	ap, ad := semvideo.DropBFrames(pk, disp)
	sub, err := semvideo.BiDecode(ap, ad, ecc)
	if err != nil {
		fmt.Println("decode anchors:", err)
		return
	}
	fmt.Printf("temporal scalability: drop all B-frames -> %d packets (display %v)\n", len(ap), ad)
	fmt.Println("still a valid video, just a coarser frame rate:")
	for _, i := range ad {
		fmt.Printf("  frame %d\n", i)
		fmt.Print(render(sub[i], "    "))
	}
	fmt.Println("a lost B-frame costs ONE frame, never the stream — bidirectional frames are disposable.")
}
