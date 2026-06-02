// Command pseudo3d is the synthesis of the three pseudo-3D paths in one scene
// and one stream. A single 50-byte-per-frame record packs: 7 fold angles (path
// 1, a folding tangram body), 1 relief height (path 2, a rising voxel animal)
// and 7 scene items x 6 bytes (path 3, orbiting slabs). The whole animation is
// shipped as ONE deltastream over Reed-Solomon, corrupted by a burst, recovered,
// and rendered by the SINGLE isometric renderer (fold.Render) into one world.
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

	"github.com/svend4/infon/pkg/deltastream"
	"github.com/svend4/infon/pkg/fold"
	"github.com/svend4/infon/pkg/relief"
	"github.com/svend4/infon/pkg/scene3d"
	"github.com/svend4/infon/pkg/tangram"
	t7 "github.com/svend4/infon/pkg/tangram7"
)

const (
	W, H  = 560, 420
	NF    = 14
	width = 7 + 1 + 7*6 // fold angles + relief height + scene items
)

func ease(x float64) float64 { return (1 - math.Cos(math.Pi*x)) / 2 }
func clamp(v float64) byte {
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return byte(v)
}

func shift(fs []fold.Face, dx, dy, dz float64) []fold.Face {
	out := make([]fold.Face, len(fs))
	for i, f := range fs {
		pts := make([]fold.Pt3, len(f.Pts))
		for j, p := range f.Pts {
			pts[j] = fold.Pt3{X: p.X + dx, Y: p.Y + dy, Z: p.Z + dz}
		}
		out[i] = fold.Face{Pts: pts, Color: f.Color, Shade: f.Shade}
	}
	return out
}

func packFrame(f int) []byte {
	fr := make([]byte, width)
	// path 1: fold angles (staggered ramp)
	for i := 0; i < 7; i++ {
		local := (float64(f) - float64(i)) / 6.0
		if local < 0 {
			local = 0
		}
		if local > 1 {
			local = 1
		}
		target := 150.0 + 12.0*float64(i)
		if target > 240 {
			target = 240
		}
		fr[i] = byte(ease(local) * target)
	}
	// path 2: relief height (rise)
	fr[7] = byte(8 + ease(float64(f)/float64(NF-1))*120)
	// path 3: scene items (orbit + spin + bob)
	for i := 0; i < 7; i++ {
		ang := float64(i)*(2*math.Pi/7) + float64(f)*0.30
		off := 8 + i*6
		fr[off] = byte(i)
		fr[off+1] = clamp(128 + 70*math.Cos(ang))
		fr[off+2] = clamp(128 + 70*math.Sin(ang))
		fr[off+3] = clamp(120 + 90*math.Sin(float64(f)*0.4+float64(i)))
		fr[off+4] = byte((i + f) % 8)
		fr[off+5] = byte(i)
	}
	return fr
}

func renderFrame(fr []byte) image.Image {
	angles := make([]float64, 7)
	for i := 0; i < 7; i++ {
		angles[i] = float64(fr[i]) / 255 * fold.FoldMaxAngle
	}
	cat := tangram.Cat()
	items := make([]scene3d.Item, 7)
	for i := 0; i < 7; i++ {
		off := 8 + i*6
		items[i] = scene3d.Item{Piece: fr[off], X: fr[off+1], Y: fr[off+2], Z: fr[off+3], Yaw: fr[off+4], Color: fr[off+5]}
	}
	var all []fold.Face
	all = append(all, shift(relief.Faces(cat, relief.Uniform(cat, fr[7])), 0, 0, 0)...)
	all = append(all, shift(fold.FacesFold(t7.Square(), angles), 13, 2, 0)...)
	all = append(all, shift(scene3d.SceneFaces(items), 23, 0, 0)...)
	return fold.Render(all, W, H)
}

func main() {
	out := "_pseudo3d"
	os.MkdirAll(out, 0o755)

	anim := deltastream.Anim{Width: width}
	for f := 0; f < NF; f++ {
		anim.Frames = append(anim.Frames, packFrame(f))
	}

	const ecc = 14
	raw := deltastream.RawSize(anim)
	wire := deltastream.Encode(anim, ecc)
	nb := len(wire) / 255
	burst := nb * (ecc / 2)
	corrupt := make([]byte, len(wire))
	copy(corrupt, wire)
	for i := 0; i < burst && len(wire)/2+i < len(corrupt); i++ {
		corrupt[len(wire)/2+i] ^= 0xFF
	}
	got, err := deltastream.Decode(corrupt, ecc)
	exact := err == nil && len(got.Frames) == len(anim.Frames)
	for f := 0; exact && f < len(anim.Frames); f++ {
		for i := 0; i < width; i++ {
			if got.Frames[f][i] != anim.Frames[f][i] {
				exact = false
				break
			}
		}
	}

	fmt.Printf("unified scene: folding body + rising relief animal + orbiting slabs\n")
	fmt.Printf("frame record = %d bytes (7 fold + 1 relief + 42 scene)\n", width)
	fmt.Printf("raw stream = %d bytes (%d frames, %.1f bytes/frame)\n", raw, NF, float64(raw)/NF)
	fmt.Printf("RS wire    = %d bytes (%d block(s), ecc=%d)\n", len(wire), nb, ecc)
	fmt.Printf("injected   = %d-byte burst; decode err=%v; recovered exact=%v\n", burst, err, exact)

	g := &gif.GIF{}
	for f := 0; f < len(got.Frames); f++ {
		frame := renderFrame(got.Frames[f])
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 8)
	}
	fp, _ := os.Create(out + "/pseudo3d.gif")
	gif.EncodeAll(fp, g)
	fp.Close()
	hp, _ := os.Create(out + "/pseudo3d_hero.png")
	png.Encode(hp, renderFrame(got.Frames[len(got.Frames)-1]))
	hp.Close()
	fmt.Printf("wrote %s/pseudo3d.gif and pseudo3d_hero.png\n", out)
}
