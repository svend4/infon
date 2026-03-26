package color

import "fmt"

// RGB represents a 24-bit RGB color
type RGB struct {
	R, G, B uint8
}

// RGB565 represents a 16-bit RGB color (5 bits R, 6 bits G, 5 bits B)
type RGB565 uint16

// NewRGB creates a new RGB color
func NewRGB(r, g, b uint8) RGB {
	return RGB{R: r, G: g, B: b}
}

// ToRGB565 converts 24-bit RGB to 16-bit RGB565
func (c RGB) ToRGB565() RGB565 {
	r := uint16(c.R) >> 3 // 8 bits -> 5 bits
	g := uint16(c.G) >> 2 // 8 bits -> 6 bits
	b := uint16(c.B) >> 3 // 8 bits -> 5 bits
	return RGB565((r << 11) | (g << 5) | b)
}

// ToRGB converts RGB565 back to 24-bit RGB
func (c RGB565) ToRGB() RGB {
	r := uint8((c >> 11) << 3)
	g := uint8(((c >> 5) & 0x3F) << 2)
	b := uint8((c & 0x1F) << 3)
	return RGB{R: r, G: g, B: b}
}

// Luminance calculates the perceived brightness (0-255)
// Using ITU-R BT.709 coefficients
func (c RGB) Luminance() uint8 {
	lum := 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
	if lum > 255 {
		return 255
	}
	return uint8(lum)
}

// Distance calculates Euclidean distance between two colors
func (c RGB) Distance(other RGB) float64 {
	dr := float64(c.R) - float64(other.R)
	dg := float64(c.G) - float64(other.G)
	db := float64(c.B) - float64(other.B)
	return dr*dr + dg*dg + db*db // squared distance (no sqrt for performance)
}

// ANSI escape codes for terminal colors

// FgString returns ANSI escape code for 24-bit foreground color
func (c RGB) FgString() string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

// BgString returns ANSI escape code for 24-bit background color
func (c RGB) BgString() string {
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", c.R, c.G, c.B)
}

// Reset returns ANSI escape code to reset colors
const Reset = "\x1b[0m"

// ClearScreen returns ANSI escape code to clear screen
const ClearScreen = "\x1b[2J\x1b[H"

// Grayscale converts RGB to grayscale
func (c RGB) Grayscale() RGB {
	lum := c.Luminance()
	return RGB{R: lum, G: lum, B: lum}
}

// Common colors
var (
	Black   = RGB{0, 0, 0}
	White   = RGB{255, 255, 255}
	Red     = RGB{255, 0, 0}
	Green   = RGB{0, 255, 0}
	Blue    = RGB{0, 0, 255}
	Yellow  = RGB{255, 255, 0}
	Cyan    = RGB{0, 255, 255}
	Magenta = RGB{255, 0, 255}
)

// Blend blends two colors with alpha (0.0 = all c, 1.0 = all other)
func (c RGB) Blend(other RGB, alpha float64) RGB {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	return RGB{
		R: uint8(float64(c.R)*(1-alpha) + float64(other.R)*alpha),
		G: uint8(float64(c.G)*(1-alpha) + float64(other.G)*alpha),
		B: uint8(float64(c.B)*(1-alpha) + float64(other.B)*alpha),
	}
}
