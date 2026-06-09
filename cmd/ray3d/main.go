// Command ray3d renders a small CPU ray-traced scene — spheres on a checkerboard
// floor with Lambert/Blinn-Phong shading, hard shadows, a mirror and glass —
// into the terminal using truecolor glyph modes (no ASCII ramp). With -path it
// switches to the Monte-Carlo path tracer (global illumination). It can instead
// export a PNG or an orbit GIF, add an .obj mesh, and it demonstrates the "scene
// over Reed-Solomon" codec: the whole world is shipped as ~100 bytes.
//
//	go run ./cmd/ray3d                      # raster render to the terminal
//	go run ./cmd/ray3d -path -spp 64        # path-traced global illumination
//	go run ./cmd/ray3d -png out.png         # export a still
//	go run ./cmd/ray3d -gif orbit.gif       # export a turntable
//	go run ./cmd/ray3d -mode sextant        # denser glyph mode
//	go run ./cmd/ray3d -obj model.obj       # drop a mesh into the scene
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/pkg/raytrace"
	"github.com/svend4/infon/pkg/terminal"
)

func demoWire() raytrace.SceneWire {
	return raytrace.SceneWire{
		Light: raytrace.Vec3{X: 6, Y: 9, Z: -4},
		Spheres: []raytrace.Sphere{
			{Center: raytrace.Vec3{X: -1.3, Y: 1.0, Z: 0}, Radius: 1.0, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.85, Y: 0.25, Z: 0.25}}},
			{Center: raytrace.Vec3{X: 1.3, Y: 1.0, Z: 0.4}, Radius: 1.0, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.85, Y: 0.85, Z: 0.9}, Reflect: 0.75}},
			{Center: raytrace.Vec3{X: 0, Y: 0.55, Z: 2.0}, Radius: 0.55, Mat: raytrace.Material{Color: raytrace.Vec3{X: 0.25, Y: 0.5, Z: 0.9}}},
		},
	}
}

// orbitCam places the camera on a ring looking at the scene centre (0,1,0).
func orbitCam(angle float64) raytrace.Camera {
	const radius, height = 6.0, 2.4
	pos := raytrace.Vec3{X: math.Sin(angle) * radius, Y: height, Z: math.Cos(angle) * radius}
	d := raytrace.Vec3{X: 0, Y: 1, Z: 0}.Sub(pos)
	return raytrace.Camera{
		Pos:   pos,
		Yaw:   math.Atan2(d.X, d.Z),
		Pitch: math.Atan2(d.Y, math.Hypot(d.X, d.Z)),
		FOV:   math.Pi / 3,
	}
}

type opts struct {
	spp     int
	path    bool
	depth   int
	nee     bool
	denoise int
}

// renderImage picks the raster or the path tracer, then optionally denoises.
func renderImage(scene *raytrace.Scene, cam raytrace.Camera, w, h int, o opts) image.Image {
	var img image.Image
	if o.path {
		img = raytrace.PathRender(scene, cam, w, h, raytrace.PathOptions{Samples: o.spp, MaxDepth: o.depth, Seed: 1, NEE: o.nee})
	} else {
		img = raytrace.Render(scene, cam, w, h, raytrace.Options{Samples: o.spp})
	}
	if o.denoise > 0 {
		img = raytrace.Denoise(img, o.denoise, 0.12)
	}
	return img
}

func main() {
	cols := flag.Int("w", 90, "width in terminal cells")
	rows := flag.Int("h", 45, "height in terminal cells")
	spp := flag.Int("spp", 1, "samples per pixel (AA for raster; paths for -path)")
	angle := flag.Float64("angle", 0.6, "orbit angle in radians")
	objPath := flag.String("obj", "", "optional .obj mesh to add to the scene")
	pngPath := flag.String("png", "", "export a PNG (720x480) instead of drawing")
	gifPath := flag.String("gif", "", "export an orbit GIF instead of drawing")
	frames := flag.Int("frames", 24, "frames in the GIF orbit")
	mode := flag.String("mode", "auto", "terminal mode: auto|halfblock|sextant|octant|braille|perceptual|optimal|quadrant|sixel|kitty")
	pathT := flag.Bool("path", false, "use the Monte-Carlo path tracer (global illumination)")
	depth := flag.Int("depth", 6, "path tracer max bounces")
	nee := flag.Bool("nee", true, "path tracer: next-event estimation (direct light sampling)")
	denoise := flag.Int("denoise", 0, "à-trous denoiser passes applied to the result (0 = off)")
	flag.Parse()

	o := opts{spp: *spp, path: *pathT, depth: *depth, nee: *nee, denoise: *denoise}
	w := demoWire()

	const ecc = 10
	wire := raytrace.Encode(w, ecc)
	dec, err := raytrace.Decode(wire, ecc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wire decode failed, using live scene:", err)
		dec = w
	}
	scene := dec.Build(0.14)
	scene.BuildBVH()

	if *objPath != "" {
		addOBJ(scene, *objPath)
	}

	switch {
	case *gifPath != "":
		writeGIF(*gifPath, scene, *cols, *rows, *frames, o)
	case *pngPath != "":
		so := o
		if !so.path && so.spp < 2 {
			so.spp = 2
		}
		writePNG(*pngPath, renderImage(scene, orbitCam(*angle), 720, 480, so))
	default:
		fmt.Print(renderToTerminal(scene, orbitCam(*angle), *cols, *rows, *mode, o))
		fmt.Printf("scene-over-RS: %d bytes (ecc=%d) for %d spheres + camera; mode=%s%s; ray-traced locally\n",
			len(wire), ecc, len(w.Spheres), *mode, pathTag(o))
	}
}

func pathTag(o opts) string {
	if o.path {
		return fmt.Sprintf(" (path tracer, %d spp)", o.spp)
	}
	return ""
}

// renderToTerminal ray-traces and converts to a terminal string in the requested
// mode ("auto" picks the best glyph mode; "sixel"/"kitty" emit true bitmaps).
func renderToTerminal(scene *raytrace.Scene, cam raytrace.Camera, cols, rows int, mode string, o opts) string {
	switch mode {
	case "kitty":
		return terminal.EncodeKitty(renderImage(scene, cam, cols*8, rows*16, o))
	case "sixel":
		return terminal.EncodeSixel(renderImage(scene, cam, cols*6, rows*12, o), 256)
	default:
		name := mode
		if name == "" || name == "auto" {
			name = terminal.DetectCapability().BestBlitMode()
		}
		rm, _ := babe.ParseRenderMode(name)
		img := renderImage(scene, cam, cols*2, rows*4, o)
		return babe.ImageToFrameMode(img, cols, rows, rm).Render()
	}
}

func addOBJ(scene *raytrace.Scene, path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "obj:", err)
		return
	}
	defer func() { _ = f.Close() }()
	m, err := raytrace.LoadOBJ(f, raytrace.Material{Color: raytrace.Vec3{X: 0.9, Y: 0.8, Z: 0.3}})
	if err != nil {
		fmt.Fprintln(os.Stderr, "obj:", err)
		return
	}
	scene.Objects = append(scene.Objects, m)
	scene.BuildBVH()
	fmt.Fprintf(os.Stderr, "loaded %s (%d triangles)\n", path, len(m.Tris))
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "png:", err)
		return
	}
	defer func() { _ = f.Close() }()
	_ = png.Encode(f, img)
	fmt.Printf("wrote %s\n", path)
}

func writeGIF(path string, scene *raytrace.Scene, cols, rows, frames int, o opts) {
	g := &gif.GIF{}
	for i := 0; i < frames; i++ {
		ang := 2 * math.Pi * float64(i) / float64(frames)
		img := renderImage(scene, orbitCam(ang), cols*2, rows*4, o)
		pal := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pal, img.Bounds(), img, image.Point{})
		g.Image = append(g.Image, pal)
		g.Delay = append(g.Delay, 6)
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gif:", err)
		return
	}
	defer func() { _ = f.Close() }()
	_ = gif.EncodeAll(f, g)
	fmt.Printf("wrote %s (%d frames)\n", path, frames)
}
