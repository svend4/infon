// postprocess.go adds two camera-style post effects: auto-exposure (scale the
// image so its mean luminance lands on mid-grey) and bloom/glare (extract bright
// pixels above a threshold, blur them and add the glow back, so highlights and
// caustics bleed light). It works on a rendered image, decoding sRGB toward
// linear, applying the effects, then re-applying ACES tone mapping + gamma.
package raytrace

import (
	"image"
	"math"
)

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
