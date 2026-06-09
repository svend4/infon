// Command rayview is an interactive terminal navigator for the ray tracer. Type a
// command (or several) then Enter to fly the camera; the frame redraws
// flicker-free through the diff renderer. It is portable (line-based stdin), so it
// runs anywhere with no raw-TTY handling.
//
//	a/d orbit   w/s dolly in/out   q/e raise/lower   r/f tilt
//	+/- field of view   p toggle path tracer   n toggle denoiser   x quit
package main

import (
	"bufio"
	"fmt"
	"image"
	"math"
	"os"
	"strings"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/internal/raysource"
	"github.com/svend4/infon/pkg/raytrace"
	"github.com/svend4/infon/pkg/terminal"
)

func main() {
	const cols, rows = 80, 40
	scene := raysource.DemoScene()
	scene.BuildBVH()
	target := raytrace.Vec3{X: 0, Y: 1, Z: 0}

	angle, radius, height, pitch, fov := 0.0, 6.0, 2.6, -0.1, math.Pi/3
	pathT := false
	denoise := 0
	mode := terminal.DetectCapability().BestBlitMode()
	dr := terminal.NewDiffRenderer()

	render := func() {
		pos := raytrace.Vec3{X: math.Sin(angle) * radius, Y: height, Z: math.Cos(angle) * radius}
		d := target.Sub(pos)
		cam := raytrace.Camera{
			Pos:   pos,
			Yaw:   math.Atan2(d.X, d.Z),
			Pitch: math.Atan2(d.Y, math.Hypot(d.X, d.Z)) + pitch,
			FOV:   fov,
		}
		var im image.Image
		if pathT {
			im = raytrace.PathRender(scene, cam, cols*2, rows*4, raytrace.PathOptions{Samples: 8, MaxDepth: 5, Seed: 1, NEE: true})
		} else {
			im = raytrace.Render(scene, cam, cols*2, rows*4, raytrace.Options{Samples: 1})
		}
		if denoise > 0 {
			im = raytrace.Denoise(im, denoise, 0.12)
		}
		rm, _ := babe.ParseRenderMode(mode)
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, cols, rows, rm)))
		fmt.Printf("\n[a/d orbit w/s dolly q/e height r/f tilt +/- fov | p path:%v n denoise:%d | x quit] ", pathT, denoise)
	}

	render()
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		for _, ch := range strings.ToLower(strings.TrimSpace(sc.Text())) {
			switch ch {
			case 'a':
				angle -= 0.2
			case 'd':
				angle += 0.2
			case 'w':
				radius = math.Max(2, radius-0.5)
			case 's':
				radius = math.Min(14, radius+0.5)
			case 'q':
				height = math.Min(6, height+0.3)
			case 'e':
				height = math.Max(0.3, height-0.3)
			case 'r':
				pitch = math.Min(0.6, pitch+0.08)
			case 'f':
				pitch = math.Max(-0.6, pitch-0.08)
			case '+', '=':
				fov = math.Max(0.3, fov-0.08)
			case '-', '_':
				fov = math.Min(2.2, fov+0.08)
			case 'p':
				pathT = !pathT
			case 'n':
				if denoise == 0 {
					denoise = 2
				} else {
					denoise = 0
				}
			case 'x':
				return
			}
		}
		render()
	}
}
