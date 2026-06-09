// Command ray3d renders a small CPU ray-traced scene — spheres on a checkerboard
// floor with Lambert shading, hard shadows and a mirror — into the terminal using
// truecolor half-blocks: the pkg/raytrace engine driven through infon's own
// renderer (no ASCII ramp). It can instead export a PNG or an orbit GIF, add an
// .obj mesh, and it demonstrates the "scene over Reed-Solomon" codec: the whole
// world is shipped as ~100 bytes and ray-traced locally.
//
//	go run ./cmd/ray3d                      # render to the terminal
//	go run ./cmd/ray3d -spp 2 -w 120 -h 60  # anti-aliased, larger
//	go run ./cmd/ray3d -png out.png         # export a still
//	go run ./cmd/ray3d -gif orbit.gif       # export a turntable
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

func main() {
	cols := flag.Int("w", 90, "width in terminal cells")
	rows := flag.Int("h", 45, "height in terminal cells")
	spp := flag.Int("spp", 1, "anti-aliasing samples per axis (1 = off)")
	angle := flag.Float64("angle", 0.6, "orbit angle in radians")
	objPath := flag.String("obj", "", "optional .obj mesh to add to the scene")
	pngPath := flag.String("png", "", "export a PNG (720x480) instead of drawing")
	gifPath := flag.String("gif", "", "export an orbit GIF instead of drawing")
	frames := flag.Int("frames", 24, "frames in the GIF orbit")
	flag.Parse()

	w := demoWire()

	// Showcase the semantic codec: encode -> bytes over RS -> decode -> render.
	const ecc = 10
	wire := raytrace.Encode(w, ecc)
	dec, err := raytrace.Decode(wire, ecc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wire decode failed, using live scene:", err)
		dec = w
	}
	scene := dec.Build(0.14)

	if *objPath != "" {
		addOBJ(scene, *objPath)
	}

	switch {
	case *gifPath != "":
		writeGIF(*gifPath, scene, *cols, *rows, *spp, *frames)
	case *pngPath != "":
		img := raytrace.Render(scene, orbitCam(*angle), 720, 480, raytrace.Options{Samples: max(*spp, 2)})
		writePNG(*pngPath, img)
	default:
		img := raytrace.Render(scene, orbitCam(*angle), *cols, *rows*2, raytrace.Options{Samples: *spp})
		fmt.Print(terminal.HalfBlock(img, *cols, *rows).Render())
		fmt.Printf("scene-over-RS: %d bytes (ecc=%d) for %d spheres + camera; ray-traced locally\n",
			len(wire), ecc, len(w.Spheres))
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

func writeGIF(path string, scene *raytrace.Scene, cols, rows, spp, frames int) {
	g := &gif.GIF{}
	for i := 0; i < frames; i++ {
		ang := 2 * math.Pi * float64(i) / float64(frames)
		img := raytrace.Render(scene, orbitCam(ang), cols, rows*2, raytrace.Options{Samples: spp})
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
