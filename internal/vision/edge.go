// Package vision provides image-analysis primitives for graphics overlays
// (roadmap C3). It starts with pure-Go, model-free analysis (edge detection)
// and defines the seam (FrameAnalyzer) that external vision models plug into.
package vision

import (
	"image"

	"github.com/svend4/infon/pkg/color"
)

// EdgeMap is a grayscale gradient-magnitude image (0..255), the output of edge
// detection. It is the model-free foundation for overlays: outline a speaker,
// sketch contours, drive Braille line art — no neural network required.
type EdgeMap struct {
	Width, Height int
	Mag           []uint8 // gradient magnitude per pixel, row-major
}

// At returns the magnitude at (x,y), or 0 out of range.
func (e *EdgeMap) At(x, y int) uint8 {
	if x < 0 || y < 0 || x >= e.Width || y >= e.Height {
		return 0
	}
	return e.Mag[y*e.Width+x]
}

// Sobel computes edge magnitude with the 3x3 Sobel operator over the image's
// luminance. This is pure arithmetic (like dithering): no model, deterministic,
// fast — the always-available tier of C3.
func Sobel(img image.Image) *EdgeMap {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	em := &EdgeMap{Width: w, Height: h, Mag: make([]uint8, w*h)}
	if w < 3 || h < 3 {
		return em
	}

	// Precompute luminance to avoid repeated conversions.
	lum := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			lum[y*w+x] = float64(color.RGB{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bl >> 8)}.Luminance())
		}
	}

	at := func(x, y int) float64 {
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		if x >= w {
			x = w - 1
		}
		if y >= h {
			y = h - 1
		}
		return lum[y*w+x]
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Sobel X / Y kernels.
			gx := -at(x-1, y-1) - 2*at(x-1, y) - at(x-1, y+1) +
				at(x+1, y-1) + 2*at(x+1, y) + at(x+1, y+1)
			gy := -at(x-1, y-1) - 2*at(x, y-1) - at(x+1, y-1) +
				at(x-1, y+1) + 2*at(x, y+1) + at(x+1, y+1)
			mag := absF(gx) + absF(gy) // L1 magnitude (cheaper than hypot, fine here)
			if mag > 255 {
				mag = 255
			}
			em.Mag[y*w+x] = uint8(mag)
		}
	}
	return em
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
