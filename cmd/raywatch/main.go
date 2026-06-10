// Command raywatch is the camera observer of Part 1 (variant A): a camera frame
// of a machine goes through the same engine as a telemetry stream — visual
// signatures (heat, occlusion, glare, blur) are extracted, assessed, and the
// verdict is composited back onto the frame as a status border and banner, with a
// bloom/dream pass so the hot motor glows. It prints the cue (text) and writes the
// annotated frame (graphics) — a real object surrounded by cinematic effects.
//
//	go run ./cmd/raywatch                 # render a robot with a hot motor, observe it
//	go run ./cmd/raywatch -heat 0.9       # hotter motor -> higher severity
//	go run ./cmd/raywatch photo.png       # observe a real image instead
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/fleet"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out  = flag.String("out", "watch", "output basename (writes <out>.png)")
		heat = flag.Float64("heat", 0.7, "motor heat 0..1 for the rendered robot (ignored with a file)")
		w    = flag.Int("w", 720, "render width")
		h    = flag.Int("h", 480, "render height")
		spp  = flag.Int("spp", 64, "samples per pixel")
	)
	flag.Parse()

	// The camera frame: a real image if one is given, else a rendered robot whose
	// motor glows redder as -heat rises.
	var frame image.Image
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, "open:", err)
			os.Exit(1)
		}
		frame, _, err = image.Decode(f)
		_ = f.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, "decode:", err)
			os.Exit(1)
		}
	} else {
		scene := raydir.BuildScene(robotFrameSpec(*heat))
		cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 1.9, Z: -4}, Pitch: -0.08, FOV: math.Pi / 3}
		frame = raytrace.PathRender(scene, cam, *w, *h, raytrace.PathOptions{
			Samples: *spp, MaxDepth: 6, Seed: 5, NEE: true, MIS: true, Sobol: true,
		})
	}

	// Observe: extract signatures and assess, exactly like a telemetry reading.
	sigs := fleet.SignalsFromImage(frame)
	a := fleet.NewMonitoringEngine().Assess(fleet.Reading{Unit: "cam-robot", Signals: sigs})

	fmt.Printf("camera observe — %s\n", a.Cue)
	fmt.Printf("  signatures: %s\n", a.Explain)

	// React, on the frame: bloom the hot motor, dream the lens, then composite a
	// crisp HUD (severity border + banner).
	vfx := raytrace.Dream(
		raytrace.PostProcess(frame, 1.0, 0.75, 0.7),
		raytrace.DreamOptions{Chroma: 0.005, Grain: 0.02, Vignette: 0.4, Seed: 1},
	)
	annotated := composite(vfx, a)
	writePNG(*out+".png", annotated)
	fmt.Printf("wrote %s.png\n", *out)
}

// robotFrameSpec is a single robot whose motor patch emits red light scaled by
// heat — so a hotter robot makes a redder frame, which the vision front-end reads
// as a higher thermal signature.
func robotFrameSpec(heat float64) brain.SceneSpec {
	if heat < 0 {
		heat = 0
	}
	if heat > 1 {
		heat = 1
	}
	// The body warms toward red as heat rises and glows (a thermal-camera look),
	// kept matte so the colour shows; the emission is modest so it stays red rather
	// than clipping to white.
	lerp := func(a, b float64) float64 { return a*(1-heat) + b*heat }
	body := [3]float64{lerp(0.45, 0.9), lerp(0.5, 0.22), lerp(0.62, 0.13)}
	bEmit := [3]float64{2.6 * heat, 0.12 * heat, 0.05 * heat} // red-dominant so it survives tonemapping
	mcol := [3]float64{lerp(0.5, 0.9), lerp(0.52, 0.13), lerp(0.58, 0.08)}
	glow := 0.9 * heat // a cool robot's motor is dark; it reddens with heat
	return brain.SceneSpec{
		Light:  [3]float64{5, 8, -4},
		SkyTop: [3]float64{0.16, 0.18, 0.24}, SkyBot: [3]float64{0.36, 0.38, 0.44},
		Objects: []brain.ObjSpec{
			{Kind: "plane", Color: [3]float64{0.32, 0.33, 0.36}},
			{Kind: "box", X: 0, Y: 0.85, Z: 1.7, S: [3]float64{1.05, 0.9, 0.7}, Color: body, Emit: bEmit, Metal: 0.1, Rough: 0.6},
			{Kind: "sphere", X: 0, Y: 2.0, Z: 1.7, R: 0.55, Color: [3]float64{0.6, 0.62, 0.66}, Metal: 0.4, Rough: 0.35},
			// the motor: a small patch, dark when cool and red when hot
			{Kind: "sphere", X: 0.85, Y: 0.7, Z: 1.05, R: 0.34, Color: mcol, Emit: [3]float64{glow, glow * 0.12, glow * 0.06}},
			{Y: 6, Z: -1, R: 0.7, Emit: [3]float64{10, 10, 10}},
		},
	}
}

// composite draws a severity border and a status banner over the frame.
func composite(img image.Image, a fleet.Assessment) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)

	c := fleet.SeverityColor(a.Severity)
	border := color.RGBA{R: u8(c[0]), G: u8(c[1]), B: u8(c[2]), A: 255}
	const t = 9
	fillRect(rgba, 0, 0, w, t, border)
	fillRect(rgba, 0, h-t, w, t, border)
	fillRect(rgba, 0, 0, t, h, border)
	fillRect(rgba, w-t, 0, t, h, border)

	// a dim banner so the text reads over any frame
	blend(rgba, 0, 0, w, 38, color.RGBA{R: 12, G: 12, B: 16, A: 255}, 0.55)
	white := color.RGBA{R: 235, G: 235, B: 240, A: 255}
	microfont.Draw(rgba, t+6, 8, 2, "CAM "+a.Level.String(), border)
	microfont.Draw(rgba, t+6, 26, 1, a.Cue, white)
	return rgba
}

func u8(v float64) uint8 {
	v = math.Round(v * 255)
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return uint8(v)
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			img.SetRGBA(i, j, c)
		}
	}
}

// blend alpha-composites a flat colour over a rectangle.
func blend(img *image.RGBA, x, y, w, h int, c color.RGBA, alpha float64) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			o := img.RGBAAt(i, j)
			img.SetRGBA(i, j, color.RGBA{
				R: u8((float64(o.R)*(1-alpha) + float64(c.R)*alpha) / 255),
				G: u8((float64(o.G)*(1-alpha) + float64(c.G)*alpha) / 255),
				B: u8((float64(o.B)*(1-alpha) + float64(c.B)*alpha) / 255),
				A: 255,
			})
		}
	}
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
