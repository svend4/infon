// Command folddemo folds flat tangram7 figures into a pseudo-3D isometric relief:
// it writes a hero still and an animated GIF of the figure rising out of the
// plane, and prints the tiny FoldSpec that describes the whole animation.
package main

import (
	"encoding/json"
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

func savePNG(name string, img image.Image) {
	fp, _ := os.Create(name)
	defer fp.Close()
	png.Encode(fp, img)
}

func ease(i, n int) float64 { return (1 - math.Cos(math.Pi*float64(i)/float64(n))) / 2 }

func main() {
	out := "_fold"
	os.MkdirAll(out, 0o755)
	const W, H = 380, 380

	figs := map[string][]t7.Piece{"square": t7.Square(), "cat": t7.Figures()["cat"]}
	for name, pieces := range figs {
		savePNG(fmt.Sprintf("%s/%s_flat.png", out, name), fold.Render(fold.Frame(pieces, 0), W, H))
		savePNG(fmt.Sprintf("%s/%s_solid.png", out, name), fold.Render(fold.Frame(pieces, 1), W, H))

		g := &gif.GIF{}
		const N = 12
		ts := []float64{}
		for i := 0; i <= N; i++ {
			ts = append(ts, ease(i, N)) // 0 -> 1
		}
		for i := N; i >= 0; i-- {
			ts = append(ts, ease(i, N)) // 1 -> 0 (loop)
		}
		for _, t := range ts {
			frame := fold.Render(fold.Frame(pieces, t), W, H)
			pi := image.NewPaletted(frame.Bounds(), palette.Plan9)
			draw.FloydSteinberg.Draw(pi, frame.Bounds(), frame, image.Point{})
			g.Image = append(g.Image, pi)
			g.Delay = append(g.Delay, 6)
		}
		fp, _ := os.Create(fmt.Sprintf("%s/%s_fold.gif", out, name))
		gif.EncodeAll(fp, g)
		fp.Close()

		spec := fold.SpecFor(pieces)
		js, _ := json.Marshal(spec)
		fmt.Printf("%-7s  pieces=%d  FoldSpec=%d bytes  heights=%v\n", name, len(pieces), len(js), spec.Heights)
	}
	fmt.Printf("wrote flats, solids and fold GIFs to %s/\n", out)
}
