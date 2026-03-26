package color

import (
	"math"
	"testing"
)

func TestRGB565Conversion(t *testing.T) {
	tests := []struct {
		name string
		rgb  RGB
	}{
		{"Black", RGB{0, 0, 0}},
		{"White", RGB{255, 255, 255}},
		{"Red", RGB{255, 0, 0}},
		{"Green", RGB{0, 255, 0}},
		{"Blue", RGB{0, 0, 255}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rgb565 := tt.rgb.ToRGB565()
			back := rgb565.ToRGB()

			// RGB565 loses precision, so check within tolerance
			tolerance := uint8(8)
			if diff(tt.rgb.R, back.R) > tolerance ||
				diff(tt.rgb.G, back.G) > tolerance ||
				diff(tt.rgb.B, back.B) > tolerance {
				t.Errorf("RGB565 roundtrip failed: %v -> %v -> %v", tt.rgb, rgb565, back)
			}
		})
	}
}

func diff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func TestLuminance(t *testing.T) {
	tests := []struct {
		name string
		rgb  RGB
		want uint8
	}{
		{"Black", RGB{0, 0, 0}, 0},
		{"White", RGB{255, 255, 255}, 255},
		{"Red", RGB{255, 0, 0}, 54},   // ~0.2126 * 255
		{"Green", RGB{0, 255, 0}, 182}, // ~0.7152 * 255
		{"Blue", RGB{0, 0, 255}, 18},   // ~0.0722 * 255
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rgb.Luminance()
			// Allow 5% tolerance
			tolerance := uint8(13)
			if diff(got, tt.want) > tolerance {
				t.Errorf("Luminance(%v) = %d, want ~%d", tt.rgb, got, tt.want)
			}
		})
	}
}

func TestDistance(t *testing.T) {
	c1 := RGB{100, 100, 100}
	c2 := RGB{100, 100, 100}
	if c1.Distance(c2) != 0 {
		t.Errorf("Distance between identical colors should be 0")
	}

	c3 := RGB{0, 0, 0}
	c4 := RGB{255, 255, 255}
	dist := c3.Distance(c4)
	expected := 3.0 * 255.0 * 255.0
	if math.Abs(dist-expected) > 0.1 {
		t.Errorf("Distance(%v, %v) = %f, want %f", c3, c4, dist, expected)
	}
}

func TestBlend(t *testing.T) {
	black := RGB{0, 0, 0}
	white := RGB{255, 255, 255}

	// Blend 0% white (all black)
	got := black.Blend(white, 0.0)
	if got != black {
		t.Errorf("Blend(0.0) = %v, want %v", got, black)
	}

	// Blend 100% white (all white)
	got = black.Blend(white, 1.0)
	if got != white {
		t.Errorf("Blend(1.0) = %v, want %v", got, white)
	}

	// Blend 50%
	got = black.Blend(white, 0.5)
	want := RGB{127, 127, 127}
	tolerance := uint8(2)
	if diff(got.R, want.R) > tolerance ||
		diff(got.G, want.G) > tolerance ||
		diff(got.B, want.B) > tolerance {
		t.Errorf("Blend(0.5) = %v, want ~%v", got, want)
	}
}

func TestGrayscale(t *testing.T) {
	red := RGB{255, 0, 0}
	gray := red.Grayscale()

	// All components should be equal
	if gray.R != gray.G || gray.G != gray.B {
		t.Errorf("Grayscale should have equal R,G,B: got %v", gray)
	}

	// Should equal luminance
	if gray.R != red.Luminance() {
		t.Errorf("Grayscale RGB should equal Luminance: got %d, want %d",
			gray.R, red.Luminance())
	}
}

func TestANSICodes(t *testing.T) {
	red := RGB{255, 0, 0}

	fg := red.FgString()
	if fg != "\x1b[38;2;255;0;0m" {
		t.Errorf("FgString() = %q, want \\x1b[38;2;255;0;0m", fg)
	}

	bg := red.BgString()
	if bg != "\x1b[48;2;255;0;0m" {
		t.Errorf("BgString() = %q, want \\x1b[48;2;255;0;0m", bg)
	}
}
