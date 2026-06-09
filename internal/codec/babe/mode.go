package babe

import (
	"image"

	"github.com/svend4/infon/pkg/glyphset"
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
	// ModeOptimal is ModeQuadrant with an exhaustive perceptual-optimal block
	// encoder (searches all 16 partitions to minimize OKLab error) (C2).
	ModeOptimal
	// ModeOctant renders 2x4 legacy-computing glyphs: 8 sub-pixels/cell, the
	// densest glyph mode (A2 extended). Needs Unicode 16 support; gate via
	// Capability.
	ModeOctant
	// ModeTriangle renders diagonal/triangle glyphs (◤◥◣◢) chosen by best-match,
	// for crisp slanted edges (mountains, sails, sun discs). From pkg/glyphset.
	ModeTriangle
	// ModeASCII renders one glyph per cell from a dark->light luminance ramp
	// (classic ASCII art), area-averaged and colored. Lowest bandwidth and the
	// most portable mode: no Unicode or sub-cell color pairs required.
	ModeASCII
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
	case ModeOptimal:
		return "optimal"
	case ModeOctant:
		return "octant"
	case ModeTriangle:
		return "triangle"
	case ModeASCII:
		return "ascii"
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
	case "optimal", "best":
		return ModeOptimal, true
	case "octant", "oct":
		return ModeOctant, true
	case "triangle", "tri", "diagonal", "diag":
		return ModeTriangle, true
	case "ascii", "ramp", "text", "art":
		return ModeASCII, true
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
	case ModeOptimal:
		return imageToFrameWithEncoder(img, w, h, EncodeBlockOptimal)
	case ModeOctant:
		return ImageToFrameOctant(img, w, h)
	case ModeTriangle:
		return glyphset.RenderTriangles(img, w, h)
	case ModeASCII:
		return ImageToFrameASCII(img, w, h)
	default:
		return ImageToFrame(img, w, h)
	}
}
