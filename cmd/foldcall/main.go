// Command foldcall sends an origami unfold of a tangram figure as a stream of
// per-piece fold angles (keyframe + deltas) over the Reed-Solomon channel, then
// corrupts a burst of wire bytes, recovers, and renders the reconstructed
// animation to a GIF - a pseudo-3D fold that travels in a few dozen bytes and
// survives loss, exactly like a semantic video call.
package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/pkg/fold"
	t7 "github.com/svend4/infon/pkg/tangram7"
)

func main() {
	out := "_fold"
	_ = os.MkdirAll(out, 0o755)
	const W, H = 380, 380
	pieces := t7.Square()
	np := len(pieces)

	const N = 16
	fa := fold.FoldAnim{NPieces: np}
	for f := 0; f < N; f++ {
		fr := make([]byte, np)
		for i := 0; i < np; i++ {
			local := (float64(f) - float64(i)) / 7.0 // staggered start per piece
			if local < 0 {
				local = 0
			}
			if local > 1 {
				local = 1
			}
			local = (1 - math.Cos(math.Pi*local)) / 2 // ease
			target := 150.0 + 12.0*float64(i)
			if target > 240 {
				target = 240
			}
			fr[i] = byte(local * target)
		}
		fa.Frames = append(fa.Frames, fr)
	}

	const ecc = 12
	raw := fold.RawSize(fa)
	wire := fold.EncodeFoldStream(fa, ecc)
	burst := ecc / 2
	corrupt := make([]byte, len(wire))
	copy(corrupt, wire)
	start := len(wire) / 3
	for i := 0; i < burst; i++ {
		corrupt[start+i] ^= 0xFF
	}
	got, err := fold.DecodeFoldStream(corrupt, ecc)

	exact := err == nil && got.NPieces == fa.NPieces && len(got.Frames) == len(fa.Frames)
	for f := 0; exact && f < len(fa.Frames); f++ {
		for i := 0; i < np; i++ {
			if got.Frames[f][i] != fa.Frames[f][i] {
				exact = false
				break
			}
		}
	}

	fmt.Printf("origami unfold: %d frames x %d pieces\n", N, np)
	fmt.Printf("raw spec  = %d bytes  (keyframe + deltas)\n", raw)
	fmt.Printf("RS wire   = %d bytes  (%d block(s), ecc=%d, corrects %d/block)\n", len(wire), len(wire)/255, ecc, ecc/2)
	fmt.Printf("injected  = %d-byte burst at offset %d\n", burst, start)
	fmt.Printf("decode err=%v  recovered exact=%v\n", err, exact)

	// hero still of the fully folded (last) frame
	if len(got.Frames) > 0 {
		hero := fold.Render(fold.FacesFold(pieces, got.AnglesAt(len(got.Frames)-1)), W, H)
		hp, _ := os.Create(out + "/foldcall_solid.png")
		_ = png.Encode(hp, hero)
		hp.Close()
	}
	// render the RECONSTRUCTED animation (forward then back, for a loop)
	g := &gif.GIF{}
	order := []int{}
	for f := 0; f < len(got.Frames); f++ {
		order = append(order, f)
	}
	for f := len(got.Frames) - 1; f >= 0; f-- {
		order = append(order, f)
	}
	for _, f := range order {
		frame := fold.Render(fold.FacesFold(pieces, got.AnglesAt(f)), W, H)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 7)
	}
	fp, _ := os.Create(out + "/foldcall.gif")
	_ = gif.EncodeAll(fp, g)
	fp.Close()
	fmt.Printf("wrote %s/foldcall.gif\n", out)
}
