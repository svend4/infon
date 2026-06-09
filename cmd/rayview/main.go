// Command rayview is an interactive terminal navigator for the ray tracer. Type a
// command (or several) then Enter to fly the camera; the frame redraws
// flicker-free through the diff renderer. In path-trace mode an empty line
// (just Enter) PROGRESSIVELY REFINES the current view (accumulates more samples,
// less noise) instead of restarting. Portable: line-based stdin, no raw-TTY.
//
//	a/d orbit   w/s dolly in/out   q/e raise/lower   r/f tilt
//	+/- field of view   p toggle path tracer   n toggle denoiser
//	(path mode) <Enter> refine   x quit
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
	const pxW, pxH = cols * 2, rows * 4
	scene := raysource.DemoScene()
	scene.BuildBVH()
	target := raytrace.Vec3{X: 0, Y: 1, Z: 0}

	angle, radius, height, pitch, fov := 0.0, 6.0, 2.6, -0.1, math.Pi/3
	pathT := false
	denoise := 0
	mode := terminal.DetectCapability().BestBlitMode()
	dr := terminal.NewDiffRenderer()
	pathOpt := raytrace.PathOptions{Samples: 6, MaxDepth: 5, Seed: 1, MIS: true}
	var acc *raytrace.Accumulator

	camera := func() raytrace.Camera {
		pos := raytrace.Vec3{X: math.Sin(angle) * radius, Y: height, Z: math.Cos(angle) * radius}
		d := target.Sub(pos)
		return raytrace.Camera{
			Pos:   pos,
			Yaw:   math.Atan2(d.X, d.Z),
			Pitch: math.Atan2(d.Y, math.Hypot(d.X, d.Z)) + pitch,
			FOV:   fov,
		}
	}

	render := func(dirty bool) {
		cam := camera()
		var im image.Image
		status := ""
		if pathT {
			if acc == nil || dirty {
				acc = raytrace.NewAccumulator(pxW, pxH)
			}
			acc.AddSamples(scene, cam, pathOpt)
			im = acc.Image()
			status = fmt.Sprintf("path %d spp", acc.Samples())
		} else {
			im = raytrace.Render(scene, cam, pxW, pxH, raytrace.Options{Samples: 1})
			status = "raster"
		}
		if denoise > 0 {
			im = raytrace.Denoise(im, denoise, 0.12)
		}
		rm, _ := babe.ParseRenderMode(mode)
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, cols, rows, rm)))
		fmt.Printf("\n[a/d orbit w/s dolly q/e height r/f tilt +/- fov | p:%v n:%d | %s | Enter=refine x=quit] ", pathT, denoise, status)
	}

	render(true)
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := strings.ToLower(strings.TrimSpace(sc.Text()))
		dirty := false
		for _, ch := range line {
			dirty = true
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
		render(dirty)
	}
}
