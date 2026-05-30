package device

import (
	"fmt"
	"image"
	"sync"
	"time"
)

// FrameGenerator is the pluggable "frame producer" behind a GenerativeSource.
//
// It is the single integration point that lets graphics be *synthesized* rather
// than captured. A procedural library (see PlasmaGenerator, RippleGenerator) and
// a neural network (see NeuralGenerator) are both just implementations of this
// interface, so the rest of the TVCP pipeline — BABE encoding, ANSI rendering and
// P2P transport — works unchanged regardless of where the pixels come from.
type FrameGenerator interface {
	// Name returns a short identifier for the generator (e.g. "plasma").
	Name() string

	// Generate produces a single RGB frame of size width x height.
	//
	// frameNum is the monotonically increasing frame index since Open and
	// elapsed is the wall-clock time since Open, so generators can animate
	// deterministically (frameNum) or in real time (elapsed).
	Generate(width, height, frameNum int, elapsed time.Duration) (image.Image, error)
}

// GenerativeSource is a synthetic video source that implements the device.Camera
// interface. Where TestCamera and V4L2Camera read pixels from a fixed pattern or
// a webcam, GenerativeSource delegates every frame to a FrameGenerator.
//
// Because it satisfies device.Camera, it is a drop-in replacement anywhere a
// camera is used: preview, send, call, group calls, recording, etc.
type GenerativeSource struct {
	width     int
	height    int
	fps       float64
	isOpen    bool
	frameNum  int
	startTime time.Time
	mu        sync.Mutex
	gen       FrameGenerator
}

// Compile-time guarantee that a GenerativeSource is a usable camera.
var _ Camera = (*GenerativeSource)(nil)

// NewGenerativeSource creates a synthetic camera driven by the given generator.
// width and height are the resolution of the produced frames (the "sensor"
// resolution); the renderer downsamples these to the terminal grid later.
func NewGenerativeSource(width, height int, fps float64, gen FrameGenerator) *GenerativeSource {
	return &GenerativeSource{
		width:  width,
		height: height,
		fps:    fps,
		gen:    gen,
	}
}

// Open starts the source.
func (g *GenerativeSource) Open() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.isOpen {
		return ErrCameraBusy
	}
	if g.gen == nil {
		return fmt.Errorf("generative source: no frame generator configured")
	}

	g.isOpen = true
	g.frameNum = 0
	g.startTime = time.Now()
	return nil
}

// Close stops the source.
func (g *GenerativeSource) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.isOpen {
		return ErrCameraNotOpen
	}

	g.isOpen = false
	return nil
}

// Read produces the next synthesized frame.
func (g *GenerativeSource) Read() (image.Image, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.isOpen {
		return nil, ErrCameraNotOpen
	}

	elapsed := time.Since(g.startTime)
	img, err := g.gen.Generate(g.width, g.height, g.frameNum, elapsed)
	if err != nil {
		return nil, fmt.Errorf("generator %q: %w", g.gen.Name(), err)
	}

	g.frameNum++
	return img, nil
}

// IsOpen reports whether the source is currently open.
func (g *GenerativeSource) IsOpen() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.isOpen
}

// GetWidth returns the frame width.
func (g *GenerativeSource) GetWidth() int { return g.width }

// GetHeight returns the frame height.
func (g *GenerativeSource) GetHeight() int { return g.height }

// GetFPS returns the configured frame rate.
func (g *GenerativeSource) GetFPS() float64 { return g.fps }

// SetResolution changes the produced frame size.
func (g *GenerativeSource) SetResolution(width, height int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.width = width
	g.height = height
	return nil
}

// SetFPS changes the configured frame rate.
func (g *GenerativeSource) SetFPS(fps float64) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.fps = fps
	return nil
}

// Generator returns the underlying frame generator.
func (g *GenerativeSource) Generator() FrameGenerator {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.gen
}

// NewProceduralGenerator returns a built-in procedural (Layer 1) generator by
// name. These run anywhere in real time with no GPU and no model — they are the
// deterministic, always-available synthesis layer.
func NewProceduralGenerator(name string) (FrameGenerator, error) {
	switch name {
	case "plasma":
		return PlasmaGenerator{}, nil
	case "ripple":
		return RippleGenerator{}, nil
	default:
		return nil, fmt.Errorf("unknown procedural generator %q (available: plasma, ripple)", name)
	}
}
