package device

import (
	"testing"
	"time"
)

func TestAudioReactiveImplementsFrameGenerator(t *testing.T) {
	var _ FrameGenerator = NewAudioReactiveGenerator()
}

func TestAudioReactiveGeneratesSizedFrame(t *testing.T) {
	g := NewAudioReactiveGenerator()
	img, err := g.Generate(64, 48, 0, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 64 || b.Dy() != 48 {
		t.Fatalf("frame = %dx%d, want 64x48", b.Dx(), b.Dy())
	}
}

func TestAudioReactiveLevelsClamped(t *testing.T) {
	g := NewAudioReactiveGenerator()
	g.SetLevels(5, -1, 2, 0.5) // out-of-range
	level, bass, mid, treble := g.snapshot()
	if level != 1 || bass != 0 || mid != 1 || treble != 0.5 {
		t.Errorf("levels not clamped: %v %v %v %v", level, bass, mid, treble)
	}
}

func TestAudioReactiveBrightnessTracksLevel(t *testing.T) {
	// Higher overall level => brighter average frame.
	avg := func(level float64) float64 {
		g := NewAudioReactiveGenerator()
		g.SetLevels(level, 0, 0, 0)
		img, _ := g.Generate(32, 32, 0, 0)
		var sum float64
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, gg, bb, _ := img.At(x, y).RGBA()
				sum += float64(r>>8) + float64(gg>>8) + float64(bb>>8)
			}
		}
		return sum
	}
	quiet := avg(0.0)
	loud := avg(1.0)
	if loud <= quiet {
		t.Errorf("louder audio should brighten the frame: quiet=%.0f loud=%.0f", quiet, loud)
	}
}

func TestHSVToRGBPrimaries(t *testing.T) {
	// Spot-check hue extremes.
	r, g, b := hsvToRGB(0, 1, 1) // red
	if r != 255 || g != 0 || b != 0 {
		t.Errorf("hue 0 = %d,%d,%d, want 255,0,0", r, g, b)
	}
	r, g, b = hsvToRGB(120, 1, 1) // green
	if r != 0 || g != 255 || b != 0 {
		t.Errorf("hue 120 = %d,%d,%d, want 0,255,0", r, g, b)
	}
	r, g, b = hsvToRGB(240, 1, 1) // blue
	if r != 0 || g != 0 || b != 255 {
		t.Errorf("hue 240 = %d,%d,%d, want 0,0,255", r, g, b)
	}
	// Zero saturation => gray.
	r, g, b = hsvToRGB(123, 0, 0.5)
	if r != g || g != b {
		t.Errorf("zero saturation should be gray, got %d,%d,%d", r, g, b)
	}
}
