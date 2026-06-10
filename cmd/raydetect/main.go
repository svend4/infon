// Command raydetect closes the semantic-video chain: a vision model's detections
// become a rayscene, which is then path-traced. With VISION_API_URL set it runs the
// real analyzer on an image (camera -> detections); otherwise it uses a built-in
// street scene, so it always runs. It prints the wire size of the SceneSpec — the
// whole world in about one UDP packet — the point of TVCP: send meaning, not pixels.
//
//	go run ./cmd/raydetect                         # built-in detections -> world
//	VISION_API_URL=http://127.0.0.1:8088 go run ./cmd/raydetect photo.png
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/internal/vision"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out = flag.String("out", "detect", "output basename")
		w   = flag.Int("w", 720, "render width")
		h   = flag.Int("h", 480, "render height")
		spp = flag.Int("spp", 64, "samples per pixel")
	)
	flag.Parse()

	dets, src := detections()
	spec := raydir.SceneFromDetections(dets)

	fmt.Printf("detections (%s):\n", src)
	for _, d := range dets {
		fmt.Printf("  %-14s conf %.2f  box (%.2f,%.2f %.2fx%.2f)\n", d.Label, d.Confidence, d.X, d.Y, d.W, d.H)
	}
	if wire, err := json.Marshal(spec); err == nil {
		fmt.Printf("rayscene wire size: %d bytes (%d objects) — the whole world in ~one packet\n", len(wire), len(spec.Objects))
	}

	scene := raydir.BuildScene(spec)
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.4, Z: -3.5}, Pitch: -0.12, FOV: math.Pi / 3}
	img := raytrace.PathRender(scene, cam, *w, *h, raytrace.PathOptions{
		Samples: *spp, MaxDepth: 6, Seed: 4, NEE: true, MIS: true, Sobol: true,
	})
	writePNG(*out+".png", img)
	fmt.Printf("wrote %s.png\n", *out)
}

// detections runs the real analyzer on an image when VISION_API_URL is set and a
// path is given, else returns a built-in street scene.
func detections() ([]vision.Detection, string) {
	if an, ok := vision.NewHTTPAnalyzerFromEnv(); ok && flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err == nil {
			img, _, derr := image.Decode(f)
			_ = f.Close()
			if derr == nil {
				if ds, aerr := an.Analyze(context.Background(), img); aerr == nil {
					return ds, an.Name()
				}
			}
		}
		fmt.Fprintln(os.Stderr, "vision analyze failed; using the built-in scene")
	}
	return []vision.Detection{
		{Label: "tree", Confidence: 0.92, X: 0.04, Y: 0.18, W: 0.18, H: 0.62},
		{Label: "person", Confidence: 0.96, X: 0.42, Y: 0.46, W: 0.10, H: 0.40},
		{Label: "dog", Confidence: 0.81, X: 0.56, Y: 0.70, W: 0.12, H: 0.18},
		{Label: "car", Confidence: 0.90, X: 0.68, Y: 0.54, W: 0.30, H: 0.24},
	}, "built-in"
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
