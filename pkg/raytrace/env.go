// env.go importance-samples an equirectangular environment map by luminance, so
// the path tracer can find bright sky regions (a sun, a window) directly instead
// of waiting for a random bounce to hit them - far less noise for image-based
// lighting. A flat cumulative table over texels (weighted by luminance and the
// sin(theta) solid-angle measure) inverts to a direction; the pdf is returned in
// solid angle (the equirect Jacobian's sin(theta) cancels). A small epsilon keeps
// every direction sample-able, so the estimator stays unbiased over the sphere.
package raytrace

import (
	"image"
	"math"
	"sort"
)

// EnvSampler importance-samples an environment image (consistent with Scene.sky's
// equirectangular mapping).
type EnvSampler struct {
	w, h  int
	pix   []Vec3
	cum   []float64 // cumulative texel weights, len w*h+1
	total float64
}

// BuildEnvSampler precomputes the luminance distribution of an env image.
func BuildEnvSampler(img image.Image) *EnvSampler {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	e := &EnvSampler{w: w, h: h, pix: make([]Vec3, w*h), cum: make([]float64, w*h+1)}
	srgb := func(c uint32) float64 { v := float64(c) / 65535; return v * v }
	for y := 0; y < h; y++ {
		st := math.Sin(math.Pi * (float64(y) + 0.5) / float64(h))
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			col := Vec3{X: srgb(r), Y: srgb(g), Z: srgb(bl)}
			e.pix[y*w+x] = col
			lum := 0.2126*col.X + 0.7152*col.Y + 0.0722*col.Z
			e.total += (lum + 1e-4) * st // +eps keeps full support
			e.cum[y*w+x+1] = e.total
		}
	}
	return e
}

// BuildEnvSamplerFromSky builds an importance sampler for an analytic sky function
// (e.g. PreethamSky.At) at w x h resolution, keeping HDR radiance so a bright sun
// is sampled often. Its directions/pdf drive env-NEE; the radiance used in shading
// is still read from Scene.sky, so it stays exact (and MIS-consistent).
func BuildEnvSamplerFromSky(skyAt func(Vec3) Vec3, w, h int) *EnvSampler {
	if w < 2 || h < 2 || skyAt == nil {
		return nil
	}
	e := &EnvSampler{w: w, h: h, pix: make([]Vec3, w*h), cum: make([]float64, w*h+1)}
	for y := 0; y < h; y++ {
		st := math.Sin(math.Pi * (float64(y) + 0.5) / float64(h))
		v := (float64(y) + 0.5) / float64(h)
		for x := 0; x < w; x++ {
			u := (float64(x) + 0.5) / float64(w)
			col := skyAt(e.dir(u, v))
			e.pix[y*w+x] = col
			lum := 0.2126*col.X + 0.7152*col.Y + 0.0722*col.Z
			e.total += (lum + 1e-4) * st
			e.cum[y*w+x+1] = e.total
		}
	}
	return e
}

func (e *EnvSampler) dir(u, v float64) Vec3 {
	y := math.Cos(math.Pi * v)
	s := math.Sqrt(math.Max(0, 1-y*y))
	phi := 2 * math.Pi * (u - 0.5)
	return Vec3{X: s * math.Cos(phi), Y: y, Z: s * math.Sin(phi)}.Norm()
}

func (e *EnvSampler) pdfTexel(idx int) float64 {
	wgt := e.cum[idx+1] - e.cum[idx]
	st := math.Sin(math.Pi * (float64(idx/e.w) + 0.5) / float64(e.h))
	if st < 1e-6 {
		st = 1e-6
	}
	// pdf_omega = wgt * (w*h) / (total * 2*pi^2 * sin(theta)); the texel weight's
	// own sin(theta) factor is divided back out here.
	return wgt * float64(e.w*e.h) / (e.total * 2 * math.Pi * math.Pi * st)
}

// Sample draws a direction proportional to env luminance and returns the
// direction, its radiance, and the solid-angle pdf.
func (e *EnvSampler) Sample(rg *rng) (Vec3, Vec3, float64) {
	i := sort.SearchFloat64s(e.cum, rg.f()*e.total)
	if i < 1 {
		i = 1
	}
	if i > e.w*e.h {
		i = e.w * e.h
	}
	idx := i - 1
	x, y := idx%e.w, idx/e.w
	d := e.dir((float64(x)+rg.f())/float64(e.w), (float64(y)+rg.f())/float64(e.h))
	return d, e.pix[idx], e.pdfTexel(idx)
}

// Pdf returns the solid-angle pdf this sampler assigns to direction d (for MIS).
func (e *EnvSampler) Pdf(d Vec3) float64 {
	u := 0.5 + math.Atan2(d.Z, d.X)/(2*math.Pi)
	v := 0.5 - math.Asin(math.Max(-1, math.Min(1, d.Y)))/math.Pi
	x := int(u * float64(e.w))
	y := int(v * float64(e.h))
	if x < 0 {
		x = 0
	} else if x >= e.w {
		x = e.w - 1
	}
	if y < 0 {
		y = 0
	} else if y >= e.h {
		y = e.h - 1
	}
	return e.pdfTexel(y*e.w + x)
}
