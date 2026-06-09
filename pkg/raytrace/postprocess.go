// postprocess.go adds two camera-style post effects: auto-exposure (scale the
// image so its mean luminance lands on mid-grey) and bloom/glare (extract bright
// pixels above a threshold, blur them and add the glow back, so highlights and
// caustics bleed light). It works on a rendered image, decoding sRGB toward
// linear, applying the effects, then re-applying ACES tone mapping + gamma.
package raytrace

import (
	"image"
	"image/color"
	"math"
)

// GradeOptions configures the post-grade pipeline (Grade): exposure, bloom,
// vignette, and a choice of tone map.
type GradeOptions struct {
	Exposure      float64 // <=0 auto-exposes to mid-grey; else a fixed multiplier
	BloomThresh   float64 // brightness above which pixels glow
	BloomStrength float64 // <=0 disables bloom
	Vignette      float64 // 0 none .. ~1 strong corner darkening
	AgX           bool    // AgX tone map (filmic) instead of ACES
}

// Grade is the "pretty" post pipeline: decode to linear, expose, add bloom, apply
// a vignette, then tone-map (ACES or AgX) and encode. It turns a flat render into a
// graded frame for the walk.
func Grade(img image.Image, o GradeOptions) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return img
	}
	dec := func(c uint32) float64 { v := float64(c) / 65535; return v * v } // sRGB -> ~linear
	lin := make([]Vec3, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			lin[y*w+x] = Vec3{X: dec(r), Y: dec(g), Z: dec(bl)}
		}
	}
	exp := o.Exposure
	if exp <= 0 {
		var sumL float64
		for _, c := range lin {
			sumL += lumOf(c)
		}
		if avg := sumL / float64(len(lin)); avg > 1e-4 {
			exp = 0.5 / avg
		} else {
			exp = 1
		}
	}
	for i := range lin {
		lin[i] = lin[i].Scale(exp)
	}
	if o.BloomStrength > 0 {
		bright := make([]Vec3, w*h)
		for i, c := range lin {
			if l := lumOf(c); l > o.BloomThresh {
				bright[i] = c.Scale((l - o.BloomThresh) / l)
			}
		}
		blurred := gaussBlur(bright, w, h, 2.0)
		for i := range lin {
			lin[i] = lin[i].Add(blurred[i].Scale(o.BloomStrength))
		}
	}
	if o.Vignette > 0 {
		cx, cy := float64(w-1)/2, float64(h-1)/2
		maxr2 := cx*cx + cy*cy
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dx, dy := float64(x)-cx, float64(y)-cy
				f := 1 - o.Vignette*((dx*dx+dy*dy)/maxr2)
				if f < 0 {
					f = 0
				}
				lin[y*w+x] = lin[y*w+x].Scale(f)
			}
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if o.AgX {
				out.SetRGBA(x, y, agxRGBA(lin[y*w+x]))
			} else {
				out.SetRGBA(x, y, toRGBA(lin[y*w+x]))
			}
		}
	}
	return out
}

// agx maps a linear colour through a compact AgX tone curve (matrix inset, log2
// encode, contrast sigmoid, matrix outset), returning display values in [0,1]. AgX
// rolls highlights off gently and keeps saturated colours from clipping.
func agx(c Vec3) Vec3 {
	dot := func(a, v Vec3) float64 { return a.X*v.X + a.Y*v.Y + a.Z*v.Z }
	in := Vec3{
		X: dot(Vec3{X: 0.842, Y: 0.0784, Z: 0.0792}, c),
		Y: dot(Vec3{X: 0.0423, Y: 0.878, Z: 0.0792}, c),
		Z: dot(Vec3{X: 0.0424, Y: 0.0784, Z: 0.879}, c),
	}
	const lo, hi = -12.47393, 4.026069
	enc := func(x float64) float64 {
		if x <= 0 {
			x = 1e-10
		}
		return clamp01((math.Log2(x) - lo) / (hi - lo))
	}
	poly := func(x float64) float64 {
		x2 := x * x
		x4 := x2 * x2
		return 15.5*x4*x2 - 40.14*x4*x + 31.96*x4 - 6.868*x2*x + 0.4298*x2 + 0.1191*x - 0.00232
	}
	v := Vec3{X: poly(enc(in.X)), Y: poly(enc(in.Y)), Z: poly(enc(in.Z))}
	out := Vec3{
		X: dot(Vec3{X: 1.1976, Y: -0.0980, Z: -0.0990}, v),
		Y: dot(Vec3{X: -0.0528, Y: 1.1519, Z: -0.0961}, v),
		Z: dot(Vec3{X: -0.0529, Y: -0.0983, Z: 1.1510}, v),
	}
	return Vec3{X: clamp01(out.X), Y: clamp01(out.Y), Z: clamp01(out.Z)}
}

func agxRGBA(c Vec3) color.RGBA {
	g := agx(c)
	u := func(x float64) uint8 { return uint8(x*255 + 0.5) }
	return color.RGBA{R: u(g.X), G: u(g.Y), B: u(g.Z), A: 255}
}

// PostProcess applies auto-exposure and bloom to a rendered image. exposure <= 0
// auto-exposes to mid-grey; otherwise it is a fixed multiplier. bloomStrength <= 0
// disables bloom; bloomThresh is the brightness above which pixels glow.
func PostProcess(img image.Image, exposure, bloomThresh, bloomStrength float64) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return img
	}
	dec := func(c uint32) float64 { v := float64(c) / 65535; return v * v } // sRGB -> ~linear
	lin := make([]Vec3, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			lin[y*w+x] = Vec3{X: dec(r), Y: dec(g), Z: dec(bl)}
		}
	}

	exp := exposure
	if exp <= 0 {
		var sumL float64
		for _, c := range lin {
			sumL += 0.2126*c.X + 0.7152*c.Y + 0.0722*c.Z
		}
		avg := sumL / float64(len(lin))
		exp = 1.0
		if avg > 1e-4 {
			exp = 0.5 / avg // target mid-grey
		}
	}
	for i := range lin {
		lin[i] = lin[i].Scale(exp)
	}

	if bloomStrength > 0 {
		bright := make([]Vec3, w*h)
		for i, c := range lin {
			l := 0.2126*c.X + 0.7152*c.Y + 0.0722*c.Z
			if l > bloomThresh {
				bright[i] = c.Scale((l - bloomThresh) / l) // soft knee
			}
		}
		blurred := gaussBlur(bright, w, h, 2.0)
		for i := range lin {
			lin[i] = lin[i].Add(blurred[i].Scale(bloomStrength))
		}
	}

	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.SetRGBA(x, y, toRGBA(lin[y*w+x]))
		}
	}
	return out
}

// gaussBlur separably blurs a Vec3 buffer with the given sigma (clamped edges).
func gaussBlur(src []Vec3, w, h int, sigma float64) []Vec3 {
	if sigma <= 0 {
		return src
	}
	radius := int(3 * sigma)
	if radius < 1 {
		radius = 1
	}
	kern := make([]float64, 2*radius+1)
	var ksum float64
	for i := -radius; i <= radius; i++ {
		v := math.Exp(-float64(i*i) / (2 * sigma * sigma))
		kern[i+radius] = v
		ksum += v
	}
	for i := range kern {
		kern[i] /= ksum
	}
	clampi := func(v, hi int) int {
		if v < 0 {
			return 0
		}
		if v > hi {
			return hi
		}
		return v
	}
	tmp := make([]Vec3, w*h)
	for y := 0; y < h; y++ { // horizontal
		for x := 0; x < w; x++ {
			var acc Vec3
			for k := -radius; k <= radius; k++ {
				acc = acc.Add(src[y*w+clampi(x+k, w-1)].Scale(kern[k+radius]))
			}
			tmp[y*w+x] = acc
		}
	}
	out := make([]Vec3, w*h)
	for y := 0; y < h; y++ { // vertical
		for x := 0; x < w; x++ {
			var acc Vec3
			for k := -radius; k <= radius; k++ {
				acc = acc.Add(tmp[clampi(y+k, h-1)*w+x].Scale(kern[k+radius]))
			}
			out[y*w+x] = acc
		}
	}
	return out
}
