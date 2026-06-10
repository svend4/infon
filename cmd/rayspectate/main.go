// Command rayspectate renders the raymeet shared world populated like an MMORPG:
// several player avatars and a swarm of patrolling robots in one world, shown both
// from a spectator's overhead view and through one player's eyes — the same scene
// raymeet builds (AvatarSpheres over World.SceneWith), so it is the multiplayer
// world of Part 1, with the robots of block H sharing it.
//
//	go run ./cmd/rayspectate              # writes spectate.png (spectator | player view)
//	go run ./cmd/rayspectate -players 5 -robots 4 -spp 64
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

	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out      = flag.String("out", "spectate", "output basename")
		nPlayers = flag.Int("players", 4, "player avatars in the world")
		nRobots  = flag.Int("robots", 4, "patrolling robots in the world")
		w        = flag.Int("w", 420, "panel width")
		h        = flag.Int("h", 300, "panel height")
		spp      = flag.Int("spp", 56, "samples per pixel")
	)
	flag.Parse()

	world := raydir.NewWorld()
	world.SetTime(0.4) // morning sun

	// A swarm of robots on patrol loops, status spread green -> red.
	for i := 0; i < *nRobots; i++ {
		a := float64(i) / float64(max1(*nRobots)) * 2 * math.Pi
		c := raytrace.Vec3{X: math.Cos(a) * 4.5, Z: 9 + math.Sin(a)*3}
		loop := []raytrace.Vec3{c, c.Add(raytrace.Vec3{X: 3}), c.Add(raytrace.Vec3{X: 3, Z: 3}), c.Add(raytrace.Vec3{Z: 3})}
		status := float64(i) / float64(max1(*nRobots-1))
		world.SpawnRobot(raydir.NewRobot(c, loop, status))
	}
	for k := 0; k < 25; k++ { // advance the swarm ~2.5 s into its rounds
		world.StepRobots(0.1)
	}

	// Player avatars scattered through the world, facing various ways. Player 0 is
	// the viewpoint: it stands at the front looking into the crowd, so its POV is
	// populated.
	players := make([]raydir.Pose, *nPlayers)
	players[0] = raydir.Pose{Pos: raytrace.Vec3{X: 0, Z: 2.5}, Yaw: 0}
	for i := 1; i < *nPlayers; i++ {
		a := float64(i)/float64(max1(*nPlayers))*2*math.Pi + 0.5
		players[i] = raydir.Pose{
			Pos: raytrace.Vec3{X: math.Cos(a) * 3.0, Y: 0, Z: 8 + math.Sin(a)*2.5},
			Yaw: a + math.Pi,
		}
	}
	var extra []raytrace.Object
	for id, p := range players {
		extra = append(extra, raydir.AvatarSpheres(p, raydir.AvatarColor(uint32(id)))...)
	}

	scene := world.SceneWith(extra)
	opt := raytrace.PathOptions{Samples: *spp, MaxDepth: 5, Seed: 7, NEE: true, MIS: true, Sobol: true}

	// Spectator: a high oblique view over the whole crowd.
	specCam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 13, Z: -1}, Pitch: -0.85, FOV: math.Pi / 3}
	specImg := bloom(raytrace.PathRender(scene, specCam, *w, *h, opt))

	// Player view: through player 0's eyes (the shared world from inside).
	p0 := players[0]
	povCam := raytrace.Camera{Pos: p0.Pos.Add(raytrace.Vec3{Y: 1.5}), Yaw: p0.Yaw, Pitch: -0.05, FOV: math.Pi / 3}
	povImg := bloom(raytrace.PathRender(scene, povCam, *w, *h, opt))

	sheet := montage([]panel{{specImg, "spectator"}, {povImg, "player view"}})
	writePNG(*out+".png", sheet)
	fmt.Printf("shared world: %d players + %d robots -> %s.png\n", *nPlayers, *nRobots, *out)
}

func bloom(img image.Image) image.Image { return raytrace.PostProcess(img, 1.0, 0.75, 0.6) }

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
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
	H := ph + labelH
	out := image.NewRGBA(image.Rect(0, 0, W, H))
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
