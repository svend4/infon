// Command scenecall animates a 3-D scene of tangram pieces (orbit + spin + bob),
// ships it as a SceneAnim (keyframe + deltas) over the Reed-Solomon channel,
// corrupts a contiguous burst, recovers, and renders the reconstructed scene to
// a GIF - "a real game" that travels in a few dozen bytes per frame.
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

	"github.com/svend4/infon/pkg/scene3d"
)

func clamp(v float64) byte {
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return byte(v)
}

func main() {
	out := "_scene"
	os.MkdirAll(out, 0o755)
	const W, H = 440, 440
	const ni, N = 7, 20

	a := scene3d.SceneAnim{NItems: ni}
	for f := 0; f < N; f++ {
		items := make([]scene3d.Item, ni)
		for i := 0; i < ni; i++ {
			ang := float64(i)*(2*math.Pi/float64(ni)) + float64(f)*0.30
			items[i] = scene3d.Item{
				Piece: byte(i),
				X:     clamp(128 + 70*math.Cos(ang)),
				Y:     clamp(128 + 70*math.Sin(ang)),
				Z:     clamp(120 + 90*math.Sin(float64(f)*0.4+float64(i))),
				Yaw:   byte((i + f) % 8),
				Color: byte(i),
			}
		}
		a.Frames = append(a.Frames, items)
	}

	const ecc = 12
	raw := scene3d.RawSize(a)
	wire := scene3d.EncodeSceneStream(a, ecc)
	nb := len(wire) / 255
	burst := nb * (ecc / 2)
	corrupt := make([]byte, len(wire))
	copy(corrupt, wire)
	start := len(wire)/2 - burst/2
	for i := 0; i < burst && start+i < len(wire); i++ {
		corrupt[start+i] ^= 0xFF
	}
	got, err := scene3d.DecodeSceneStream(corrupt, ecc)

	exact := err == nil && got.NItems == a.NItems && len(got.Frames) == len(a.Frames)
	for f := 0; exact && f < len(a.Frames); f++ {
		for i := 0; i < ni; i++ {
			if got.Frames[f][i] != a.Frames[f][i] {
				exact = false
				break
			}
		}
	}

	fmt.Printf("3D scene: %d items x %d frames (orbit + spin + bob)\n", ni, N)
	fmt.Printf("raw spec = %d bytes  (%.1f bytes/frame)\n", raw, float64(raw)/float64(N))
	fmt.Printf("RS wire  = %d bytes  (%d block(s), ecc=%d)\n", len(wire), nb, ecc)
	fmt.Printf("injected = %d-byte contiguous burst at %d\n", burst, start)
	fmt.Printf("decode err=%v  recovered exact=%v\n", err, exact)

	g := &gif.GIF{}
	for f := 0; f < len(got.Frames); f++ {
		frame := scene3d.Render(got.Frames[f], W, H)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 7)
	}
	fp, _ := os.Create(out + "/scenecall.gif")
	gif.EncodeAll(fp, g)
	fp.Close()
	if len(got.Frames) > 0 {
		hp, _ := os.Create(out + "/scene_hero.png")
		png.Encode(hp, scene3d.Render(got.Frames[len(got.Frames)/3], W, H))
		hp.Close()
	}
	fmt.Printf("wrote %s/scenecall.gif and scene_hero.png\n", out)
}
