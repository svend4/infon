package device

import (
	"image"
	"image/color"
	"math"
	"time"
)

// This file implements Layer 1 of the synthesis architecture: cheap,
// deterministic, GPU-free generators that always run in real time. They are the
// reference implementations of FrameGenerator and the fallback that works on a
// Raspberry Pi, over SSH, or anywhere a heavy model cannot run.

// PlasmaGenerator renders a classic animated "plasma" effect by summing a few
// sine fields. It is smooth, colorful and a good stress test for the color path.
type PlasmaGenerator struct{}

// Name implements FrameGenerator.
func (PlasmaGenerator) Name() string { return "plasma" }

// Generate implements FrameGenerator.
func (PlasmaGenerator) Generate(width, height, frameNum int, elapsed time.Duration) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	t := elapsed.Seconds()

	cx, cy := 0.5, 0.5
	for y := 0; y < height; y++ {
		fy := float64(y) / float64(height)
		for x := 0; x < width; x++ {
			fx := float64(x) / float64(width)

			v := math.Sin(fx*10.0 + t)
			v += math.Sin(fy*10.0 + t*1.3)
			v += math.Sin((fx+fy)*10.0 + t*0.7)
			v += math.Sin(math.Hypot(fx-cx, fy-cy)*20.0 - t*2.0)
			v /= 4.0 // normalize to roughly [-1, 1]

			img.SetRGBA(x, y, palette(v))
		}
	}
	return img, nil
}

// RippleGenerator renders concentric rings expanding from the center, like a
// drop hitting water. Good for showing smooth motion at low resolution.
type RippleGenerator struct{}

// Name implements FrameGenerator.
func (RippleGenerator) Name() string { return "ripple" }

// Generate implements FrameGenerator.
func (RippleGenerator) Generate(width, height, frameNum int, elapsed time.Duration) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	t := elapsed.Seconds()

	cx, cy := float64(width)/2.0, float64(height)/2.0
	maxDist := math.Hypot(cx, cy)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dist := math.Hypot(float64(x)-cx, float64(y)-cy) / maxDist
			v := math.Sin(dist*24.0 - t*4.0)
			img.SetRGBA(x, y, palette(v*0.5+0.5*math.Sin(t)))
		}
	}
	return img, nil
}

// palette maps a value in roughly [-1, 1] to a smooth RGB color using three
// phase-shifted cosines (a cheap, vivid rainbow).
func palette(v float64) color.RGBA {
	phase := math.Pi * v
	r := uint8(128.0 + 127.0*math.Sin(phase))
	g := uint8(128.0 + 127.0*math.Sin(phase+2.0944)) // +120 degrees
	b := uint8(128.0 + 127.0*math.Sin(phase+4.1888)) // +240 degrees
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
