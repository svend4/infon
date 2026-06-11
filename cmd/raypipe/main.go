// Command raypipe shows the closed triangle end to end, offline and without a server:
// an AI brain authors a world from a prompt (2 — tvcp-ai/1), the world crosses the
// "wire" as MEANING — a compact SceneSpec, exactly the bytes raymeet ships over UDP
// (1 — the network), and is rebuilt and ray-traced locally (3 — the renderer). It
// verifies the trip is lossless, reports how little crosses the wire versus a frame of
// pixels, reads the delivered world's hexagram and mood back out of it, and renders
// each world side by side.
//
//	go run ./cmd/raypipe "a calm dawn" "a vast cold desert"
//	BRAIN_URL=http://host:port go run ./cmd/raypipe   # a live external model authors
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"reflect"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out = flag.String("out", "pipe", "output basename")
		w   = flag.Int("w", 300, "panel width")
		h   = flag.Int("h", 220, "panel height")
		spp = flag.Int("spp", 56, "samples per pixel")
	)
	flag.Parse()
	prompts := flag.Args()
	if len(prompts) == 0 {
		prompts = []string{"a calm dawn", "a vast cold desert", "a glowing crystal forest", "a quiet rainy dusk"}
	}

	// (2) the brain: a live external model if BRAIN_URL is set, else the offline Q6
	// author (the pro2 director's deterministic fallback — VectorFromPrompt).
	var b brain.Brain
	who := "offline Q6 author (hash->vector, no server)"
	live := false
	if url := os.Getenv("BRAIN_URL"); url != "" {
		b, who, live = brain.HTTPBrain{URL: url}, url, true
	}
	author := func(prompt string) (brain.SceneSpec, error) {
		if live {
			_, spec, err := raydir.AuthorScene(b, prompt)
			return spec, err
		}
		return raydir.VectorFromPrompt(prompt).SceneSpec(), nil
	}

	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true}
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: -1}, Pitch: -0.07, FOV: math.Pi / 3}
	frameBytes := *w * *h * 3

	fmt.Printf("pipe: prompt -> brain(2) -> SceneSpec -> wire(1) -> render(3)   author: %s\n", who)
	fmt.Printf("  %-26s %5s %7s %9s %7s  %s\n", "prompt", "objs", "wire", "frame", "ratio", "delivered (read back)")
	var panels []panel
	totalWire, allLossless := 0, true
	for i, p := range prompts {
		spec, err := author(p) // (2) author the world from the prompt
		if err != nil {
			fmt.Fprintln(os.Stderr, "author:", err)
			os.Exit(1)
		}
		wire := raydir.Region{Index: i, Spec: spec}.Encode() // (1) the exact bytes raymeet sends over UDP
		got, err := raydir.DecodeRegion(wire)                // ...received on the other side
		if err != nil {
			fmt.Fprintln(os.Stderr, "decode:", err)
			os.Exit(1)
		}
		lossless := reflect.DeepEqual(got.Spec, spec)
		allLossless = allLossless && lossless
		totalWire += len(wire)

		scene := raydir.BuildScene(got.Spec) // (3) rebuild + ray-trace locally from the wire bytes
		img := raytrace.PostProcess(raytrace.PathRender(scene, cam, *w, *h, opt), 1.0, 0.9, 0.4)

		read := raydir.ReadScene(got.Spec) // read the delivered world's meaning back out (the reader)
		mark := "✓"
		if !lossless {
			mark = "✗"
		}
		fmt.Printf("  %-26.26s %5d %6dB %8dB %6.0fx  %06b %s %s\n",
			p, len(spec.Objects), len(wire), frameBytes, float64(frameBytes)/float64(len(wire)),
			read.Hexagram().Number(), read.Mood(), mark)
		panels = append(panels, panel{img, fmt.Sprintf("%.20s | %dB %s", p, len(wire), read.Mood())})
	}

	fmt.Printf("  total: %d worlds in %d B of meaning vs %d B of pixels (%.0fx less on the wire); lossless round trip: %v\n",
		len(prompts), totalWire, frameBytes*len(prompts), float64(frameBytes*len(prompts))/float64(totalWire), allLossless)
	writePNG(*out+".png", montage(panels))
	fmt.Printf("wrote %s.png\n", *out)
}

type panel struct {
	img   image.Image
	label string
}

func montage(panels []panel) image.Image {
	if len(panels) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	pw, ph := panels[0].img.Bounds().Dx(), panels[0].img.Bounds().Dy()
	const gap, labelH = 8, 16
	cols := len(panels)
	if cols > 4 {
		cols = (len(panels) + 1) / 2
	}
	rows := (len(panels) + cols - 1) / cols
	W := cols*pw + (cols-1)*gap
	H := rows*(ph+labelH) + (rows-1)*gap
	out := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	for i, p := range panels {
		r, c := i/cols, i%cols
		x, y := c*(pw+gap), r*(ph+labelH+gap)
		draw.Draw(out, image.Rect(x, y+labelH, x+pw, y+labelH+ph), p.img, p.img.Bounds().Min, draw.Src)
		microfont.Draw(out, x+4, y+3, 1, p.label, color.RGBA{R: 230, G: 230, B: 235, A: 255})
	}
	return out
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
