// Package aisource provides an AI-driven video source that satisfies the repo's
// device.Camera interface, so the existing pipeline (preview / send / call /
// record / export) can carry AI-generated video with no changes.
package aisource

import (
	"image"
	"image/color"
	"math"

	"github.com/svend4/infon/internal/device"
)

func hsv(h, s, v float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255)
}

// AICamera is an AI-generated video source (procedural plasma) implementing
// device.Camera. Drop it in anywhere a camera is used.
type AICamera struct {
	w, h int
	fps  float64
	open bool
	t    int
}

func NewAICamera(width, height int, fps float64) *AICamera {
	return &AICamera{w: width, h: height, fps: fps}
}

func (c *AICamera) Open() error                            { c.open = true; return nil }
func (c *AICamera) Close() error                           { c.open = false; return nil }
func (c *AICamera) IsOpen() bool                           { return c.open }
func (c *AICamera) GetWidth() int                          { return c.w }
func (c *AICamera) GetHeight() int                         { return c.h }
func (c *AICamera) GetFPS() float64                        { return c.fps }
func (c *AICamera) SetResolution(width, height int) error  { c.w, c.h = width, height; return nil }
func (c *AICamera) SetFPS(fps float64) error               { c.fps = fps; return nil }

// Read returns the next AI-generated frame (a smoothly evolving plasma field).
func (c *AICamera) Read() (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, c.w, c.h))
	t := float64(c.t)
	cx, cy := float64(c.w)/2, float64(c.h)/2
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			fx, fy := float64(x), float64(y)
			v := math.Sin(fx/16+t*0.2) + math.Sin(fy/12+t*0.15) +
				math.Sin((fx+fy)/16+t*0.1) + math.Sin(math.Hypot(fx-cx, fy-cy)/8-t*0.2)
			r, g, b := hsv((v+4)/8*360+t*4, 0.72, 1.0)
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	c.t++
	return img, nil
}

// Compile-time proof that the AI source IS a drop-in camera.
var _ device.Camera = (*AICamera)(nil)
