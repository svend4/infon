// Command raygather is a meeting INSIDE a shared world (Block A: metaverse): N
// participants, each an avatar with a face (raydir.AvatarFace from keypoints), sit
// in a circle in a hexagram-authored world and look at one another. It renders the
// gathering from one seat's eyes and from a spectator above, and reports how little
// each participant's "presence" costs on the wire — a pose plus face keypoints —
// the bandwidth a real call would carry (the rest is rebuilt and ray-traced
// locally). Joins AvatarFace + AuthorScene (the director) + the avatar wire format.
//
//	go run ./cmd/raygather -n 5 -hexagram 110010
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

	"github.com/svend4/infon/internal/avatar"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out  = flag.String("out", "gather", "output basename")
		n    = flag.Int("n", 5, "participants")
		hexS = flag.String("hexagram", "110010", "hexagram world to meet in")
		w    = flag.Int("w", 380, "panel width")
		h    = flag.Int("h", 260, "panel height")
		spp  = flag.Int("spp", 56, "samples per pixel")
	)
	flag.Parse()

	hex, ok := raydir.ParseHexagram(*hexS)
	if !ok {
		hex = raydir.HexagramFromNumber(0b110010)
	}
	_, spec, err := raydir.AuthorScene(brain.Local{}, hex.Prompt())
	if err != nil {
		fmt.Fprintln(os.Stderr, "author:", err)
		os.Exit(1)
	}

	center := raytrace.Vec3{X: 0, Y: 1.6, Z: 6}
	const radius = 2.7
	seats := make([]raydir.Pose, *n)
	for i := 0; i < *n; i++ {
		th := -math.Pi/2 + float64(i)/float64(*n)*2*math.Pi
		pos := raytrace.Vec3{X: center.X + math.Cos(th)*radius, Y: 1.6, Z: center.Z + math.Sin(th)*radius}
		seats[i] = raydir.Pose{Pos: pos, Yaw: math.Atan2(center.X-pos.X, center.Z-pos.Z)} // face the centre
	}
	faces := func(skip int) []raytrace.Object {
		var objs []raytrace.Object
		for i, s := range seats {
			if i == skip {
				continue
			}
			objs = append(objs, raydir.AvatarFace(s, raydir.DemoFace(i), raydir.AvatarColor(uint32(i)))...)
		}
		return objs
	}
	buildWith := func(extra []raytrace.Object) *raytrace.Scene {
		s := raydir.BuildScene(spec)
		s.Objects = append(s.Objects, extra...)
		s.BuildBVH()
		return s
	}

	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 5, Seed: 6, NEE: true, MIS: true, Sobol: true}
	// a seat's eyes: from seat 0, looking at the centre (its own face omitted).
	pov := raytrace.Camera{Pos: seats[0].Pos.Add(raytrace.Vec3{Y: 0.1}), Yaw: seats[0].Yaw, Pitch: -0.04, FOV: math.Pi / 3}
	povImg := raytrace.PathRender(buildWith(faces(0)), pov, *w, *h, opt)
	// the spectator: above the ring.
	spec3 := raytrace.Camera{Pos: raytrace.Vec3{X: center.X, Y: 8, Z: center.Z - 5}, Pitch: -0.72, FOV: math.Pi / 3}
	specImg := raytrace.PathRender(buildWith(faces(-1)), spec3, *w, *h, opt)
	specImg = raytrace.PostProcess(specImg, 1.0, 0.85, 0.5)
	povImg = raytrace.PostProcess(povImg, 1.0, 0.85, 0.5)

	// presence cost: a pose (40 B) + face keypoints, per participant per frame.
	face := avatar.Keypoints{Points: raydir.DemoFace(0)}
	per := 40 + len(face.Encode())
	fmt.Printf("meeting in %q: %d participants\n", hex.Name(), *n)
	fmt.Printf("  presence on the wire: ~%d B/participant/frame (pose + %d face keypoints) -> ~%d B/frame for the room\n",
		per, len(face.Points), per**n)
	fmt.Printf("  the world and everyone's faces are rebuilt and ray-traced locally — pixels never cross the wire\n")

	writePNG(*out+".png", montage(povImg, "from your seat", specImg, "spectator"))
	fmt.Printf("wrote %s.png\n", *out)
}

func montage(a image.Image, la string, b image.Image, lb string) image.Image {
	pw, ph := a.Bounds().Dx(), a.Bounds().Dy()
	const gap, labelH = 8, 16
	out := image.NewRGBA(image.Rect(0, 0, 2*pw+gap, ph+labelH))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 20, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(out, image.Rect(0, labelH, pw, labelH+ph), a, a.Bounds().Min, draw.Src)
	draw.Draw(out, image.Rect(pw+gap, labelH, 2*pw+gap, labelH+ph), b, b.Bounds().Min, draw.Src)
	white := color.RGBA{R: 230, G: 230, B: 235, A: 255}
	microfont.Draw(out, 4, 3, 1, la, white)
	microfont.Draw(out, pw+gap+4, 3, 1, lb, white)
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
