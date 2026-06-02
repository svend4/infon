// Command orbit3d turns the pseudo-3D world into a turntable: it renders a relief
// voxel cat across a full 360-degree camera-yaw sweep (fold.RenderCam) and ships
// the camera path as a one-byte-per-frame deltastream - rotation costs nothing in
// the codec, it is just one more number in the record.
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
	"github.com/svend4/infon/pkg/tangram"
)

func main() {
	out := "_orbit"
	os.MkdirAll(out, 0o755)
	const W, H, N = 420, 420, 24

	cat := tangram.Cat()
	faces := relief.Faces(cat, relief.Uniform(cat, 120))

	track := deltastream.Anim{Width: 1}
	for f := 0; f < N; f++ {
		track.Frames = append(track.Frames, []byte{byte(f * 256 / N)})
	}
	wire := deltastream.Encode(track, 8)

	fmt.Printf("turntable: relief cat, %d frames over 360 degrees, %d faces\n", N, len(faces))
	fmt.Printf("camera track = %d bytes raw / %d bytes RS  (one yaw byte per frame)\n", deltastream.RawSize(track), len(wire))

	g := &gif.GIF{}
	for f := 0; f < N; f++ {
		yaw := 2 * math.Pi * float64(track.Frames[f][0]) / 256
		frame := fold.RenderCam(faces, W, H, yaw)
		pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 7)
	}
	fp, _ := os.Create(out + "/cat_turntable.gif")
	gif.EncodeAll(fp, g)
	fp.Close()
	hp, _ := os.Create(out + "/cat_turn45.png")
	png.Encode(hp, fold.RenderCam(faces, W, H, math.Pi/4))
	hp.Close()
	fmt.Printf("wrote %s/cat_turntable.gif and cat_turn45.png\n", out)
}
