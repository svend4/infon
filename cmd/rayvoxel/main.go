// Command rayvoxel renders a voxel-space landscape — the blocky height-field look
// the dream hackers describe as how dreams form ("Воксельная графика в
// сновидениях"). It draws a procedural fractal terrain with a height-banded
// palette (water, sand, grass, rock, snow), front-to-back with a y-buffer and
// aerial haze, and writes a PNG.
//
//	go run ./cmd/rayvoxel -seed 7 -out voxel.png
//	go run ./cmd/rayvoxel -w 900 -h 500 -dist 1200 -yaw 0.4
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"

	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	seed := flag.Float64("seed", 7, "terrain seed")
	scale := flag.Float64("scale", 0.008, "terrain frequency (smaller = broader hills)")
	amp := flag.Float64("amp", 90, "terrain height amplitude")
	w := flag.Int("w", 800, "image width")
	h := flag.Int("h", 460, "image height")
	x := flag.Float64("x", 0, "camera X on the map")
	z := flag.Float64("z", 0, "camera Z on the map")
	height := flag.Float64("height", 115, "camera height")
	yaw := flag.Float64("yaw", 0, "view direction (radians)")
	dist := flag.Float64("dist", 900, "far view distance")
	out := flag.String("out", "voxel.png", "output PNG path")
	flag.Parse()

	terrain := raytrace.FBMTerrain(*seed, *scale, *amp)
	cam := raytrace.VoxelCamera{
		X: *x, Z: *z, Height: *height, Yaw: *yaw,
		Horizon: float64(*h) * 0.33, Scale: float64(*h) * 0.75, Distance: *dist,
	}
	img := raytrace.RenderVoxel(terrain, cam, *w, *h)
	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		return
	}
	fmt.Printf("wrote %s (%dx%d voxel terrain)\n", *out, *w, *h)
}
