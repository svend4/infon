// Command rayimagine reads a picture and renders the 3-D world it implies — the
// project's idea in reverse: meaning extracted FROM pixels. It downsamples the
// image to a rayscene (sky/ground from the bands, a sun from the brightest patch,
// coloured forms from the dominant colours), then path-traces that scene.
//
//	go run ./cmd/rayimagine photo.png            # writes photo.scene.png
//	go run ./cmd/rayimagine photo.jpg out.png
package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"strings"

	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: rayimagine <image> [out.png]")
		os.Exit(1)
	}
	in := os.Args[1]
	out := in
	if dot := strings.LastIndex(out, "."); dot >= 0 {
		out = out[:dot]
	}
	out += ".scene.png"
	if len(os.Args) > 2 {
		out = os.Args[2]
	}
	f, err := os.Open(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	img, _, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "decode:", err)
		os.Exit(1)
	}
	spec := raydir.SceneFromImage(img)
	scene := raydir.BuildScene(spec)
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.2, Z: 0}, Pitch: -0.05, FOV: math.Pi / 3}
	rendered := raytrace.PathRender(scene, cam, 720, 480, raytrace.PathOptions{Samples: 96, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true})
	of, err := os.Create(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer of.Close()
	if err := png.Encode(of, rendered); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
	fmt.Printf("imagined %d objects from %s -> %s\n", len(spec.Objects), in, out)
}
