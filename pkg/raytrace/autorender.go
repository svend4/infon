package raytrace

import (
	"image"
	"math"
)

// autorender.go is a small "renderer brain": it probes a scene's difficulty with
// two cheap path-traced passes and, when their seed-to-seed disagreement is high
// (hard light — caustics through glass, tiny emitters, specular-diffuse paths),
// renders with Metropolis light transport (PSSMLT), which spends its samples on
// the few high-energy paths the unidirectional tracer finds only rarely, so the
// caustic resolves with far less noise at the same budget. Easy scenes stay on the
// plain path tracer. This brings MLT — shipped and tested but reachable only from
// the ray3d demo — to any caller, chosen by content. (MLT was chosen over BDPT and
// PPM because it traces camera rays normally, so it renders specular/glass surfaces
// correctly where this BDPT blacks them out and PPM speckles them.)

// IntegratorChoice is the picked integrator and the probe noise that drove it.
type IntegratorChoice struct {
	Name  string  // "path" or "mlt"
	Noise float64 // mean per-pixel disagreement between two low-spp probes (0..1)
}

// PickIntegrator renders two low-sample path-traced probes (different seeds) and
// returns "mlt" when their mean per-pixel disagreement exceeds tol, else "path".
func PickIntegrator(s *Scene, cam Camera, w, h int, tol float64) IntegratorChoice {
	probe := PathOptions{Samples: 8, MaxDepth: 6, Seed: 1, NEE: true, MIS: true}
	a := PathRender(s, cam, w, h, probe)
	probe.Seed = 9973
	b := PathRender(s, cam, w, h, probe)
	noise := probeNoise(a, b)
	name := "path"
	if noise > tol {
		name = "mlt"
	}
	return IntegratorChoice{Name: name, Noise: noise}
}

// RenderBest picks an integrator for the scene (see PickIntegrator) and renders at
// an equal budget, returning the image and the chosen integrator's name.
func RenderBest(s *Scene, cam Camera, w, h int, opt PathOptions) (image.Image, string) {
	c := PickIntegrator(s, cam, w, h, 0.065)
	if c.Name == "mlt" {
		mut := opt.Samples * w * h
		return MLTRender(s, cam, w, h, MLTOptions{
			Mutations: mut, Bootstrap: 10000, MaxDepth: opt.MaxDepth, NEE: true, MIS: true, Seed: opt.Seed,
		}), "mlt"
	}
	return PathRender(s, cam, w, h, opt), "path"
}

// probeNoise is the mean per-pixel RGB disagreement between two images (0..1).
func probeNoise(a, b image.Image) float64 {
	bnd := a.Bounds()
	if bnd.Empty() {
		return 0
	}
	var sum float64
	var n float64
	for y := bnd.Min.Y; y < bnd.Max.Y; y++ {
		for x := bnd.Min.X; x < bnd.Max.X; x++ {
			ar, ag, ab, _ := a.At(x, y).RGBA()
			br, bg, bb, _ := b.At(x, y).RGBA()
			d := math.Abs(float64(ar)-float64(br)) + math.Abs(float64(ag)-float64(bg)) + math.Abs(float64(ab)-float64(bb))
			sum += d / (3 * 65535)
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / n
}
