// Command rayterrain activates the Delaunay terrain (raytrace.ScatterTerrain /
// DelaunayMesh / Delaunay) — shipped with tests but reachable from no command. It
// scatters jittered FBM-height points, triangulates them (Bowyer-Watson), shades by
// elevation (water -> grass -> rock -> snow), and path-traces the mesh under a sun.
//
//	go run ./cmd/rayterrain -seed 3 -amp 9
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out    = flag.String("out", "terrain", "output basename")
		n      = flag.Int("n", 900, "terrain sample points (triangulated)")
		extent = flag.Float64("extent", 44, "terrain width (world units)")
		amp    = flag.Float64("amp", 8, "terrain height amplitude")
		seed   = flag.Int64("seed", 1, "terrain seed")
		w      = flag.Int("w", 760, "render width")
		h      = flag.Int("h", 480, "render height")
		spp    = flag.Int("spp", 64, "samples per pixel")
	)
	flag.Parse()

	terrain := raytrace.ScatterTerrain(*n, *extent, *amp, *seed)
	scene := &raytrace.Scene{
		SkyTop:    raytrace.Vec3{X: 0.4, Y: 0.55, Z: 0.85},
		SkyBottom: raytrace.Vec3{X: 0.85, Y: 0.88, Z: 0.95},
		Light:     raytrace.Vec3{X: 6, Y: 9, Z: -4},
		Objects:   terrain,
	}
	scene.Objects = append(scene.Objects, raytrace.Sphere{
		Center: raytrace.Vec3{X: -*extent * 0.3, Y: *amp + 18, Z: -6}, Radius: 3,
		Mat: raytrace.Material{Emit: raytrace.Vec3{X: 16, Y: 16, Z: 15}},
	})
	scene.BuildBVH() // the terrain is hundreds of triangles

	cam := raytrace.Camera{
		Pos:   raytrace.Vec3{X: 0, Y: *amp + 20, Z: -*extent * 0.5},
		Pitch: -0.62, FOV: math.Pi / 3,
	}
	img := raytrace.PathRender(scene, cam, *w, *h, raytrace.PathOptions{
		Samples: *spp, MaxDepth: 5, Seed: 4, NEE: true, MIS: true, Sobol: true,
	})
	writePNG(*out+".png", img)
	fmt.Printf("terrain: %d points -> %d triangles, seed %d -> %s.png\n", *n, len(terrain), *seed, *out)
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
