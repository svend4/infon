// Command ray3d renders a small CPU ray-traced scene — spheres on a checkerboard
// floor with Lambert/Blinn-Phong shading, hard shadows, a mirror and glass —
// into the terminal using truecolor glyph modes (no ASCII ramp). With -renderer
// it switches among all the engines: the Monte-Carlo path tracer, bidirectional
// path tracing, Metropolis light transport, the light tracer, progressive photon
// mapping and ReSTIR direct lighting. It can instead export a PNG or an orbit
// GIF, add an .obj mesh, and it demonstrates the "scene over Reed-Solomon" codec:
// the whole world is shipped as ~100 bytes.
//
//	go run ./cmd/ray3d                      # raster render to the terminal
//	go run ./cmd/ray3d -path -spp 64        # path-traced global illumination
//	go run ./cmd/ray3d -renderer bdpt -spp 32   # bidirectional path tracing
//	go run ./cmd/ray3d -renderer mlt -spp 64    # Metropolis light transport
//	go run ./cmd/ray3d -renderer ppm            # progressive photon mapping
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
	renderer string // raster | path | bdpt | mlt | lighttrace | ppm | restir
	spp      int
	depth    int
	nee      bool
	mis      bool
	denoise  int
	gdenoise bool
	bloom    float64
	exposure float64
}

// renderImage dispatches to the selected renderer, then optionally denoises.
func renderImage(scene *raytrace.Scene, cam raytrace.Camera, w, h int, o opts) image.Image {
	var img image.Image
	switch o.renderer {
	case "path":
		img = raytrace.PathRender(scene, cam, w, h, raytrace.PathOptions{Samples: o.spp, MaxDepth: o.depth, Seed: 1, NEE: o.nee, MIS: o.mis, Sobol: true})
	case "bdpt":
		img = raytrace.BDPTRender(scene, cam, w, h, raytrace.BDPTOptions{Samples: o.spp, MaxDepth: o.depth, Seed: 1})
	case "mlt":
		img = raytrace.MLTRender(scene, cam, w, h, raytrace.MLTOptions{Mutations: o.spp * w * h, Bootstrap: 10000, MaxDepth: o.depth, NEE: o.nee, MIS: o.mis, Seed: 1})
	case "lighttrace":
		img = raytrace.LightTraceRender(scene, cam, w, h, raytrace.LightTraceOptions{Paths: o.spp * w * h, MaxBounce: o.depth, Seed: 1})
	case "ppm":
		img = raytrace.PPMRender(scene, cam, w, h, raytrace.PPMOptions{Passes: max(o.spp, 8), PhotonsPerPass: 100000, Radius: 0.4, MaxBounce: o.depth, Seed: 1})
	case "restir":
		img = raytrace.ReSTIRRender(scene, cam, w, h, raytrace.ReSTIROptions{Candidates: max(o.spp, 8), SpatialPasses: 2, SpatialK: 5, Radius: 16, Seed: 1})
	default: // raster
		img = raytrace.Render(scene, cam, w, h, raytrace.Options{Samples: o.spp})
	}
	if o.denoise > 0 {
		if o.gdenoise {
			al, nm := raytrace.GBuffer(scene, cam, w, h)
			img = raytrace.DenoiseGuided(img, al, nm, o.denoise, 0.12, 0.1, 0.2)
		} else {
			img = raytrace.Denoise(img, o.denoise, 0.12)
		}
	}
	if o.bloom > 0 || o.exposure != 1 {
		img = raytrace.PostProcess(img, o.exposure, 0.8, o.bloom)
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
	mode := flag.String("mode", "auto", "terminal mode: auto|halfblock|sextant|octant|braille|perceptual|optimal|quadrant|ascii|sixel|kitty")
	pathT := flag.Bool("path", false, "use the Monte-Carlo path tracer (global illumination)")
	renderer := flag.String("renderer", "", "renderer: raster|path|bdpt|mlt|lighttrace|ppm|restir (default raster; -path forces path)")
	depth := flag.Int("depth", 6, "path tracer max bounces")
	nee := flag.Bool("nee", true, "path tracer: next-event estimation (direct light sampling)")
	mis := flag.Bool("mis", false, "path tracer: multiple importance sampling (light + BSDF)")
	denoise := flag.Int("denoise", 0, "à-trous denoiser passes applied to the result (0 = off)")
	gdenoise := flag.Bool("gdenoise", false, "denoise with albedo/normal guides (sharper edges)")
	bloom := flag.Float64("bloom", 0, "bloom/glare strength (0 = off)")
	exposure := flag.Float64("exposure", 1, "exposure multiplier (0 = auto to mid-grey)")
	flag.Parse()

	rname := *renderer
	if rname == "" {
		if *pathT {
			rname = "path"
		} else {
			rname = "raster"
		}
	}
	o := opts{renderer: rname, spp: *spp, depth: *depth, nee: *nee, mis: *mis, denoise: *denoise, gdenoise: *gdenoise, bloom: *bloom, exposure: *exposure}
	w := demoWire()

	const ecc = 10
	wire := raytrace.Encode(w, ecc)
	dec, err := raytrace.Decode(wire, ecc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wire decode failed, using live scene:", err)
		dec = w
	}
	scene := dec.Build(0.14)
	// An emissive "sun" area light so the global-illumination renderers
	// (path/bdpt/mlt/lighttrace/ppm/restir) have an emitter to sample; the raster
	// renderer keeps using Scene.Light.
	scene.Objects = append(scene.Objects, raytrace.Sphere{
		Center: raytrace.Vec3{X: 6, Y: 9, Z: -4}, Radius: 1.5,
		Mat: raytrace.Material{Color: raytrace.Vec3{X: 1, Y: 1, Z: 0.95}, Emit: raytrace.Vec3{X: 12, Y: 12, Z: 11}},
	})
	scene.BuildBVH()

	if *objPath != "" {
		addOBJ(scene, *objPath)
	}

	switch {
	case *gifPath != "":
		writeGIF(*gifPath, scene, *cols, *rows, *frames, o)
	case *pngPath != "":
		so := o
		if so.renderer == "raster" && so.spp < 2 {
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
	if o.renderer != "raster" {
		return fmt.Sprintf(" (%s, %d spp)", o.renderer, o.spp)
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
