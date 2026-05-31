package device

import (
	"image"
	"image/color"
	"math"
	"sync"
	"time"
)

// AudioReactiveGenerator is a FrameGenerator whose plasma reacts to live audio
// (roadmap C4): bass scales the pulse, treble shifts the hue, and overall level
// controls brightness. It is the first link between the audio and video
// subsystems — on a call, the other person's voice can drive your visuals.
//
// Audio features are pushed in via SetLevels from whatever produces them (the
// reactive feature extractor over mic/Opus frames); the render loop reads the
// latest values without blocking, so audio and video stay decoupled.
type AudioReactiveGenerator struct {
	mu     sync.Mutex
	level  float64
	bass   float64
	mid    float64
	treble float64
}

// NewAudioReactiveGenerator creates a reactive generator with no audio yet (it
// renders a calm baseline until SetLevels is called).
func NewAudioReactiveGenerator() *AudioReactiveGenerator {
	return &AudioReactiveGenerator{}
}

// Name implements FrameGenerator.
func (a *AudioReactiveGenerator) Name() string { return "audio-reactive" }

// SetLevels updates the audio features driving the visuals. Values are expected
// in 0..1; they are clamped. Safe to call from an audio goroutine.
func (a *AudioReactiveGenerator) SetLevels(level, bass, mid, treble float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.level = clamp01f(level)
	a.bass = clamp01f(bass)
	a.mid = clamp01f(mid)
	a.treble = clamp01f(treble)
}

func (a *AudioReactiveGenerator) snapshot() (level, bass, mid, treble float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.level, a.bass, a.mid, a.treble
}

// Generate implements FrameGenerator: a plasma field modulated by audio.
func (a *AudioReactiveGenerator) Generate(width, height, frameNum int, elapsed time.Duration) (image.Image, error) {
	level, bass, mid, treble := a.snapshot()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	t := elapsed.Seconds()

	// Bass makes the field "breathe" (spatial frequency + amplitude); treble
	// adds a hue rotation; level sets overall brightness with a calm floor.
	pulse := 1.0 + bass*1.5
	hueShift := treble * 180.0
	brightness := 0.35 + 0.65*level

	cx, cy := 0.5, 0.5
	for y := 0; y < height; y++ {
		fy := float64(y) / float64(height)
		for x := 0; x < width; x++ {
			fx := float64(x) / float64(width)
			v := math.Sin(fx*10.0*pulse + t)
			v += math.Sin(fy*10.0*pulse + t*1.3)
			v += math.Sin(math.Hypot(fx-cx, fy-cy)*20.0*pulse - t*2.0*(1.0+mid))
			v /= 3.0

			hue := math.Mod((v*0.5+0.5)*360.0+hueShift+t*20.0, 360.0)
			r, g, b := hsvToRGB(hue, 0.7, brightness)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img, nil
}

func clamp01f(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// hsvToRGB converts HSV (h in degrees, s/v in 0..1) to 0..255 RGB.
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
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
	return to255(r + m), to255(g + m), to255(b + m)
}

func to255(f float64) uint8 {
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	return uint8(f*255 + 0.5)
}
