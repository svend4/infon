package device

import (
	"image"
	"image/color"
	"math"
	"sync"
	"time"
)

// TestCamera is a simulated camera for testing
type TestCamera struct {
	width     int
	height    int
	fps       float64
	isOpen    bool
	frameNum  int
	mu        sync.Mutex
	pattern   string // "gradient", "noise", "colorbar"
	startTime time.Time
}

// NewTestCamera creates a new test camera with animated patterns
func NewTestCamera(width, height int, fps float64, pattern string) *TestCamera {
	return &TestCamera{
		width:   width,
		height:  height,
		fps:     fps,
		pattern: pattern,
	}
}

func (c *TestCamera) Open() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isOpen {
		return ErrCameraBusy
	}

	c.isOpen = true
	c.frameNum = 0
	c.startTime = time.Now()
	return nil
}

func (c *TestCamera) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isOpen {
		return ErrCameraNotOpen
	}

	c.isOpen = false
	return nil
}

func (c *TestCamera) Read() (image.Image, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isOpen {
		return nil, ErrCameraNotOpen
	}

	// Generate test pattern
	var img image.Image
	switch c.pattern {
	case "gradient":
		img = c.generateGradient()
	case "noise":
		img = c.generateNoise()
	case "colorbar":
		img = c.generateColorBars()
	case "bounce":
		img = c.generateBouncingBall()
	default:
		img = c.generateGradient()
	}

	c.frameNum++
	return img, nil
}

func (c *TestCamera) IsOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isOpen
}

func (c *TestCamera) GetWidth() int {
	return c.width
}

func (c *TestCamera) GetHeight() int {
	return c.height
}

func (c *TestCamera) GetFPS() float64 {
	return c.fps
}

func (c *TestCamera) SetResolution(width, height int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.width = width
	c.height = height
	return nil
}

func (c *TestCamera) SetFPS(fps float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fps = fps
	return nil
}

// generateGradient creates an animated gradient pattern
func (c *TestCamera) generateGradient() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, c.width, c.height))

	// Animated offset
	offset := float64(c.frameNum) * 2.0

	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			r := uint8((float64(x)+offset)*255.0/float64(c.width)) % 255
			g := uint8(float64(y)*255.0/float64(c.height))
			b := uint8((float64(x+y)+offset)*255.0/float64(c.width+c.height)) % 255

			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	return img
}

// generateNoise creates animated noise pattern
func (c *TestCamera) generateNoise() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, c.width, c.height))

	seed := c.frameNum
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			// Simple pseudo-random based on position and frame
			hash := (x*73856093 ^ y*19349663 ^ seed*83492791) & 0xFFFFFF
			r := uint8((hash >> 16) & 0xFF)
			g := uint8((hash >> 8) & 0xFF)
			b := uint8(hash & 0xFF)

			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	return img
}

// generateColorBars creates standard SMPTE color bars
func (c *TestCamera) generateColorBars() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, c.width, c.height))

	colors := []color.RGBA{
		{255, 255, 255, 255}, // White
		{255, 255, 0, 255},   // Yellow
		{0, 255, 255, 255},   // Cyan
		{0, 255, 0, 255},     // Green
		{255, 0, 255, 255},   // Magenta
		{255, 0, 0, 255},     // Red
		{0, 0, 255, 255},     // Blue
		{0, 0, 0, 255},       // Black
	}

	barWidth := c.width / len(colors)

	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			barIdx := x / barWidth
			if barIdx >= len(colors) {
				barIdx = len(colors) - 1
			}
			img.Set(x, y, colors[barIdx])
		}
	}

	return img
}

// generateBouncingBall creates an animated bouncing ball
func (c *TestCamera) generateBouncingBall() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, c.width, c.height))

	// Black background
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}

	// Bouncing ball animation
	t := float64(c.frameNum) / c.fps
	ballRadius := 30

	// Horizontal bounce
	ballX := int(math.Abs(math.Sin(t*2.0)) * float64(c.width-ballRadius*2))
	ballX += ballRadius

	// Vertical bounce with gravity-like motion
	ballY := int(math.Abs(math.Sin(t*3.0)) * float64(c.height-ballRadius*2))
	ballY += ballRadius

	// Draw ball
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			dx := x - ballX
			dy := y - ballY
			dist := math.Sqrt(float64(dx*dx + dy*dy))

			if dist < float64(ballRadius) {
				// Color based on position
				r := uint8(float64(x) / float64(c.width) * 255)
				g := uint8(float64(y) / float64(c.height) * 255)
				b := uint8(200)
				img.Set(x, y, color.RGBA{r, g, b, 255})
			}
		}
	}

	return img
}
