// Command raymandala activates raydir.Mandala — dihedral (D_n) replication of a
// motif — which shipped with tests but no command. A small motif of coloured forms
// is rotated `fold` times about a centre (and mirrored, for 2*fold copies), making a
// kaleidoscopic, symmetric world, then rendered from above.
//
//	go run ./cmd/raymandala -fold 8 -mirror
//	go run ./cmd/raymandala -fold 6 -mirror=false -out star.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out    = flag.String("out", "mandala", "output basename")
		fold   = flag.Int("fold", 8, "rotational symmetry order (D_n)")
		mirror = flag.Bool("mirror", true, "also mirror each motif (doubles the copies)")
		w      = flag.Int("w", 560, "render width")
		h      = flag.Int("h", 560, "render height")
		spp    = flag.Int("spp", 64, "samples per pixel")
	)
	flag.Parse()

	center := raytrace.Vec3{X: 0, Y: 0, Z: 6}
	// a motif offset from the centre, so replication sweeps it into a ring/flower.
	motif := []raytrace.Object{
		raytrace.Sphere{Center: center.Add(raytrace.Vec3{X: 3.4, Y: 0.7}), Radius: 0.8, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.9, Y: 0.3, Z: 0.32}, Rough: 0.4}},
		raytrace.Sphere{Center: center.Add(raytrace.Vec3{X: 5.2, Y: 0.5}), Radius: 0.5, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.95, Y: 0.82, Z: 0.35}, Emit: raytrace.Vec3{X: 1.4, Y: 1.1, Z: 0.4}}},
		raytrace.Sphere{Center: center.Add(raytrace.Vec3{X: 4.3, Y: 0.4, Z: 1.3}), Radius: 0.34, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.3, Y: 0.6, Z: 0.95}, Reflect: 0.4}},
	}
	mandala := raydir.Mandala(motif, *fold, *mirror, center)

	scene := raydir.BuildScene(brain.SceneSpec{
		Objects: []brain.ObjSpec{{Kind: "plane", Color: [3]float64{0.12, 0.13, 0.18}}},
		Light:   [3]float64{4, 10, 2},
		SkyTop:  [3]float64{0.06, 0.07, 0.1}, SkyBot: [3]float64{0.1, 0.11, 0.15},
	})
	scene.Objects = append(scene.Objects, mandala...)
	scene.Objects = append(scene.Objects, raytrace.Sphere{Center: center.Add(raytrace.Vec3{X: -9, Y: 18, Z: -4}), Radius: 2, Mat: raytrace.Material{Emit: raytrace.Vec3{X: 13, Y: 13, Z: 12}}})
	scene.BuildBVH()

	cam := raytrace.Camera{Pos: raytrace.Vec3{X: center.X, Y: 11, Z: center.Z - 9}, Pitch: -0.82, FOV: math.Pi / 3}
	img := raytrace.PathRender(scene, cam, *w, *h, raytrace.PathOptions{Samples: *spp, MaxDepth: 5, Seed: 4, NEE: true, MIS: true, Sobol: true})
	writePNG(*out+".png", img)

	copies := *fold
	if *mirror {
		copies = 2 * *fold
	}
	fmt.Printf("mandala: D_%d, %d motif copies (%d objects) -> %s.png\n", *fold, copies, len(mandala), *out)
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
