// Command raydebate stages adversarial co-direction (the "debate × director"
// combination): two brains each author a region for a prompt, the rayscene
// validator referees, and the richer valid region wins — plus a MERGED region that
// unions both directors' forms. With BRAIN_URL_A / BRAIN_URL_B set, two models
// compete; otherwise the reference brain plays both sides with divergent prompts,
// so it runs offline. It renders the winner beside the merge and prints the verdict.
//
//	go run ./cmd/raydebate "a temple by water"
//	BRAIN_URL_A=http://a/v1/decide BRAIN_URL_B=http://b/v1/decide go run ./cmd/raydebate "a city"
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
	"strings"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out = flag.String("out", "debate", "output basename")
		w   = flag.Int("w", 380, "panel width")
		h   = flag.Int("h", 260, "panel height")
		spp = flag.Int("spp", 64, "samples per pixel")
	)
	flag.Parse()
	prompt := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if prompt == "" {
		prompt = "a quiet place of stone and water"
	}

	brainA := brainFor("BRAIN_URL_A")
	brainB := brainFor("BRAIN_URL_B")
	promptA, promptB := prompt, prompt
	if os.Getenv("BRAIN_URL_A") == "" && os.Getenv("BRAIN_URL_B") == "" {
		// diverge the two reference directors so the merge visibly unites two styles
		promptA = prompt + " calm jade pond"
		promptB = prompt + " gold fire at night"
	}

	winner, merged, rep := raydir.CoDirect(brainA, promptA, brainB, promptB)
	fmt.Printf("co-direction of %q:\n", prompt)
	fmt.Printf("  A: %d forms (score %d)   B: %d forms (score %d)   winner: %s\n", rep.ObjsA, rep.ScoreA, rep.ObjsB, rep.ScoreB, rep.Winner)
	fmt.Printf("  merged region: %d forms (both directors)\n", rep.Merged)

	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.4, Z: -2}, Pitch: -0.12, FOV: math.Pi / 3}
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true}
	winImg := raytrace.PathRender(raydir.BuildScene(winner), cam, *w, *h, opt)
	mrgImg := raytrace.PathRender(raydir.BuildScene(merged), cam, *w, *h, opt)

	sheet := montage([]panel{{winImg, "winner: director " + rep.Winner}, {mrgImg, "merged (both directors)"}})
	writePNG(*out+".png", sheet)
	fmt.Printf("wrote %s.png\n", *out)
}

func brainFor(env string) brain.Brain {
	if url := os.Getenv(env); url != "" {
		return brain.HTTPBrain{URL: url}
	}
	return brain.Local{}
}

type panel struct {
	img   image.Image
	label string
}

func montage(panels []panel) image.Image {
	pw := panels[0].img.Bounds().Dx()
	ph := panels[0].img.Bounds().Dy()
	const gap, labelH = 8, 16
	W := len(panels)*pw + (len(panels)-1)*gap
	out := image.NewRGBA(image.Rect(0, 0, W, ph+labelH))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	for i, p := range panels {
		x := i * (pw + gap)
		draw.Draw(out, image.Rect(x, labelH, x+pw, labelH+ph), p.img, p.img.Bounds().Min, draw.Src)
		microfont.Draw(out, x+4, 3, 1, p.label, color.RGBA{R: 230, G: 230, B: 235, A: 255})
	}
	return out
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
