package babe

import (
	"image"

	"github.com/svend4/infon/pkg/terminal"
)

// RenderMode selects how an image is converted to terminal cells. Higher-quality
// modes are opt-in because they trade sub-cell shape for color fidelity, or
// require terminal/font support negotiated elsewhere.
type RenderMode int

const (
	// ModeQuadrant is the original 2x2 glyph + 2 clustered colors (luminance).
	ModeQuadrant RenderMode = iota
	// ModePerceptual is ModeQuadrant but clusters/averages in OKLab (A4).
	ModePerceptual
	// ModeHalfBlock renders '▀' with fg=top, bg=bottom: 2 exact colors, 2x
	// vertical resolution, no clustering (A1).
	ModeHalfBlock
	// ModeSextant renders 2x3 legacy-computing glyphs: 6 sub-pixels/cell, denser
	// than quadrants (A2). Needs Unicode 13 support; gate via Capability.
	ModeSextant
	// ModeBraille renders 2x4 Braille dots: highest spatial resolution, per-cell
	// color; best for line art / edges (A3).
	ModeBraille
)

// String returns the mode's CLI name.
func (m RenderMode) String() string {
	switch m {
	case ModePerceptual:
		return "perceptual"
	case ModeHalfBlock:
		return "halfblock"
	case ModeSextant:
		return "sextant"
	case ModeBraille:
		return "braille"
	default:
		return "quadrant"
	}
}

// ParseRenderMode maps a CLI name to a RenderMode (defaults to ModeQuadrant).
func ParseRenderMode(name string) (RenderMode, bool) {
	switch name {
	case "", "quadrant", "quad":
		return ModeQuadrant, true
	case "perceptual", "oklab":
		return ModePerceptual, true
	case "halfblock", "half":
		return ModeHalfBlock, true
	case "sextant", "sext":
		return ModeSextant, true
	case "braille":
		return ModeBraille, true
	default:
		return ModeQuadrant, false
	}
}

// ImageToFrameMode converts an image using the selected RenderMode, so callers
// have a single entry point and the mode is data, not a function choice.
func ImageToFrameMode(img image.Image, w, h int, mode RenderMode) *terminal.Frame {
	switch mode {
	case ModePerceptual:
		return ImageToFramePerceptual(img, w, h)
	case ModeHalfBlock:
		return ImageToFrameHalfBlock(img, w, h)
	case ModeSextant:
		return ImageToFrameSextant(img, w, h)
	case ModeBraille:
		return ImageToFrameBraille(img, w, h)
	default:
		return ImageToFrame(img, w, h)
	}
}
