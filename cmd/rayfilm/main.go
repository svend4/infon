// Command rayfilm flies a cinematic camera through a brain-authored world and
// prints a contact sheet of the shots. It grows a world the way rayexplore does,
// threads a smooth Catmull-Rom spline through the world's landmarks (see
// raydir.Tour), walks the camera along it at a steady pace looking the way it
// travels, and renders evenly-spaced frames into one montage — a storyboard of the
// fly-through.
//
//	go run ./cmd/rayfilm -frames 9 -cols 3 -path -grade -out film.png
//	go run ./cmd/rayfilm -frames 6 -cols 3 -path -blur 0.06   # cinematic motion blur
//	BRAIN_URL=... go run ./cmd/rayfilm -prompt "a glass city"
package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	prompt := flag.String("prompt", "a calm world of spheres", "seed prompt for the AI director")
	frames := flag.Int("frames", 9, "number of shots along the fly-through")
	cols := flag.Int("cols", 3, "columns in the contact sheet")
	fw := flag.Int("fw", 320, "frame width in pixels")
	fh := flag.Int("fh", 200, "frame height in pixels")
	regions := flag.Int("regions", 5, "how many regions to grow (the tour waypoints)")
	pathT := flag.Bool("path", false, "path-trace each frame (prettier, slower)")
	blur := flag.Float64("blur", 0, "camera motion blur: shutter span as a fraction of the tour per shot (e.g. 0.06); needs -path")
	grade := flag.Bool("grade", false, "bloom + vignette + AgX tone map")
	out := flag.String("out", "film.png", "output PNG path")
	flag.Parse()

	var b brain.Brain = brain.Local{}
	if url := os.Getenv("BRAIN_URL"); url != "" {
		b = brain.HTTPBrain{URL: url}
	}
	prompts := []string{*prompt, "a forest with trees", "a crystal cave", "a village by water", "a surreal dreamscape", "a golden hall"}

	world := raydir.NewWorld()
	world.SetTime(0.4)
	for i := 0; i < *regions; i++ {
		jitter := float64((i%3)-1) * 4
		if _, err := world.Grow(b, prompts[i%len(prompts)], raytrace.Vec3{X: jitter, Z: float64(10 + i*14)}); err != nil {
			fmt.Fprintln(os.Stderr, "grow:", err)
		}
	}

	tour := raydir.TourFromLandmarks(world.Landmarks(), 2.6, 1.05)
	scene := world.Scene()
	opt := raytrace.PathOptions{Samples: 40, MaxDepth: 6, Seed: 1, NEE: true, MIS: true, Sobol: true}

	rows := (*frames + *cols - 1) / *cols
	sheet := image.NewRGBA(image.Rect(0, 0, *cols**fw, rows**fh))
	draw.Draw(sheet, sheet.Bounds(), image.Black, image.Point{}, draw.Src)
	for i := 0; i < *frames; i++ {
		u := 0.0
		if *frames > 1 {
			u = float64(i) / float64(*frames-1)
		}
		cam := tour.CameraAt(u)
		var im image.Image
		switch {
		case *pathT && *blur > 0: // cinematic camera motion blur over the shutter
			uc := u + *blur
			if uc > 1 {
				uc = 1
			}
			im = raytrace.PathRenderMotion(scene, cam, tour.CameraAt(uc), *fw, *fh, opt)
		case *pathT:
			im = raytrace.PathRender(scene, cam, *fw, *fh, opt)
		default:
			im = raytrace.Render(scene, cam, *fw, *fh, raytrace.Options{Samples: 2})
		}
		if *grade {
			im = raytrace.Grade(im, raytrace.GradeOptions{BloomThresh: 1.0, BloomStrength: 0.35, Vignette: 0.28, AgX: true})
		}
		x0, y0 := (i%*cols)**fw, (i / *cols)**fh
		draw.Draw(sheet, image.Rect(x0, y0, x0+*fw, y0+*fh), im, image.Point{}, draw.Src)
		fmt.Fprintf(os.Stderr, "shot %d/%d at u=%.2f\n", i+1, *frames, u)
	}
	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, sheet); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		return
	}
	fmt.Printf("wrote %s (%d shots, %dx%d)\n", *out, *frames, sheet.Bounds().Dx(), sheet.Bounds().Dy())
}
