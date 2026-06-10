// painterly.go gives the renderer a non-photorealistic "look": an oil-paint, a
// pen-and-ink, or a flat poster treatment, applied as a post-process over a
// finished frame. The pieces are classic image operators — a Kuwahara filter
// (edge-preserving smoothing that flattens regions into brush strokes), a Sobel
// ink-edge pass (dark contours), and colour quantisation (palette reduction) —
// combined into named styles. It works in display space on any rendered image.
package raytrace

import (
	"image"
	"image/color"
	"math"
)

// PainterlyStyle selects a non-photoreal look.
type PainterlyStyle int

const (
	StyleNone   PainterlyStyle = iota // pass through unchanged
	StyleOil                          // soft painted regions with faint outlines
	StyleInk                          // bold ink contours over flat colour
	StylePoster                       // few flat colours with strong outlines
)

// ParsePainterly maps a name ("oil", "ink", "poster") to a style; ok=false for an
// empty or unknown name (meaning: leave the frame alone).
func ParsePainterly(name string) (PainterlyStyle, bool) {
	switch name {
	case "oil":
		return StyleOil, true
	case "ink":
		return StyleInk, true
	case "poster":
		return StylePoster, true
	}
	return StyleNone, false
}

// imgToBuf loads an image into a display-space float buffer (channels 0..1).
func imgToBuf(img image.Image) ([]Vec3, int, int) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	buf := make([]Vec3, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			buf[y*w+x] = Vec3{X: float64(r) / 65535, Y: float64(g) / 65535, Z: float64(bl) / 65535}
		}
	}
	return buf, w, h
}

// bufToImg writes a display-space float buffer back to an RGBA image (clamped).
func bufToImg(buf []Vec3, w, h int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	u := func(v float64) uint8 { return uint8(clamp01(v)*255 + 0.5) }
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := buf[y*w+x]
			out.SetRGBA(x, y, color.RGBA{R: u(c.X), G: u(c.Y), B: u(c.Z), A: 255})
		}
	}
	return out
}

// clampedAt returns a buffer sample with edge-clamped coordinates.
func clampedAt(buf []Vec3, w, h, x, y int) Vec3 {
	if x < 0 {
		x = 0
	} else if x >= w {
		x = w - 1
	}
	if y < 0 {
		y = 0
	} else if y >= h {
		y = h - 1
	}
	return buf[y*w+x]
}

// Kuwahara is the classic edge-preserving smoothing that gives a painted look:
// for each pixel it considers the four overlapping quadrant windows around it,
// and outputs the mean colour of whichever window has the least colour variance.
// Flat regions flatten into brush strokes while edges stay crisp.
func Kuwahara(img image.Image, radius int) image.Image {
	src, w, h := imgToBuf(img)
	if radius < 1 {
		radius = 1
	}
	out := make([]Vec3, w*h)
	quads := [4][4]int{ // x0, x1, y0, y1 relative to the pixel
		{-radius, 0, -radius, 0}, {0, radius, -radius, 0},
		{-radius, 0, 0, radius}, {0, radius, 0, radius},
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			best := math.Inf(1)
			var bestMean Vec3
			for _, q := range quads {
				var sum, sum2 Vec3
				n := 0.0
				for yy := q[2]; yy <= q[3]; yy++ {
					for xx := q[0]; xx <= q[1]; xx++ {
						c := clampedAt(src, w, h, x+xx, y+yy)
						sum = sum.Add(c)
						sum2 = sum2.Add(c.Mul(c))
						n++
					}
				}
				mean := sum.Scale(1 / n)
				varr := sum2.Scale(1 / n).Sub(mean.Mul(mean)) // E[c^2]-E[c]^2 per channel
				if v := varr.X + varr.Y + varr.Z; v < best {
					best, bestMean = v, mean
				}
			}
			out[y*w+x] = bestMean
		}
	}
	return bufToImg(out, w, h)
}

// InkEdges darkens Sobel edges into ink contours (strength 0 none .. 1 black).
func InkEdges(img image.Image, strength float64) image.Image {
	src, w, h := imgToBuf(img)
	out := make([]Vec3, w*h)
	lum := func(c Vec3) float64 { return 0.2126*c.X + 0.7152*c.Y + 0.0722*c.Z }
	l := func(x, y int) float64 { return lum(clampedAt(src, w, h, x, y)) }
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			gx := -l(x-1, y-1) - 2*l(x-1, y) - l(x-1, y+1) + l(x+1, y-1) + 2*l(x+1, y) + l(x+1, y+1)
			gy := -l(x-1, y-1) - 2*l(x, y-1) - l(x+1, y-1) + l(x-1, y+1) + 2*l(x, y+1) + l(x+1, y+1)
			e := clamp01(math.Hypot(gx, gy))
			out[y*w+x] = src[y*w+x].Scale(1 - strength*e)
		}
	}
	return bufToImg(out, w, h)
}

// Quantize posterises each channel to `levels` steps (palette reduction).
func Quantize(img image.Image, levels int) image.Image {
	src, w, h := imgToBuf(img)
	if levels < 2 {
		levels = 2
	}
	s := float64(levels - 1)
	q := func(v float64) float64 { return math.Round(clamp01(v)*s) / s }
	out := make([]Vec3, w*h)
	for i, c := range src {
		out[i] = Vec3{X: q(c.X), Y: q(c.Y), Z: q(c.Z)}
	}
	return bufToImg(out, w, h)
}

// Painterly applies a named non-photoreal look to a finished frame.
func Painterly(img image.Image, style PainterlyStyle) image.Image {
	switch style {
	case StyleOil:
		return InkEdges(Quantize(Kuwahara(img, 3), 12), 0.35)
	case StyleInk:
		return InkEdges(Quantize(Kuwahara(img, 2), 8), 0.9)
	case StylePoster:
		return InkEdges(Quantize(img, 5), 0.6)
	default:
		return img
	}
}
