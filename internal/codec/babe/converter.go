package babe

import (
	"image"
	gocolor "image/color"

	"github.com/svend4/infon/internal/codec/glyphs"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// ImageToFrame converts an image to a terminal frame using quadrant blocks
// Each terminal character represents a 2x2 block of pixels
func ImageToFrame(img image.Image, targetWidth, targetHeight int) *terminal.Frame {
	frame := terminal.NewFrame(targetWidth, targetHeight)

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate scaling factors
	scaleX := float64(srcWidth) / float64(targetWidth*2)   // *2 because each char = 2 pixels wide
	scaleY := float64(srcHeight) / float64(targetHeight*2) // *2 because each char = 2 pixels high

	// Process each terminal character position
	for ty := 0; ty < targetHeight; ty++ {
		for tx := 0; tx < targetWidth; tx++ {
			// Each terminal character represents 2x2 pixels in the source image
			// Calculate the corresponding source pixel positions
			sx := int(float64(tx*2) * scaleX)
			sy := int(float64(ty*2) * scaleY)

			// Sample 2x2 pixels for this block
			var pixels [4]color.RGB // TL, TR, BL, BR
			var validPixels [4]bool

			positions := [4][2]int{
				{sx, sy},         // TL
				{sx + 1, sy},     // TR
				{sx, sy + 1},     // BL
				{sx + 1, sy + 1}, // BR
			}

			for i, pos := range positions {
				px, py := pos[0], pos[1]
				if px < srcWidth && py < srcHeight {
					r, g, b, _ := img.At(px, py).RGBA()
					pixels[i] = color.RGB{
						R: uint8(r >> 8),
						G: uint8(g >> 8),
						B: uint8(b >> 8),
					}
					validPixels[i] = true
				}
			}

			// Determine the best glyph and colors for this block
			glyph, fg, bg := EncodeBlock(pixels, validPixels)
			frame.SetBlock(tx, ty, glyph, fg, bg)
		}
	}

	return frame
}

// ImageToFramePerceptual is like ImageToFrame but uses EncodeBlockPerceptual
// (OKLab clustering) for higher color fidelity. It is a drop-in alternative for
// callers that prefer quality over the marginally cheaper luminance path.
func ImageToFramePerceptual(img image.Image, targetWidth, targetHeight int) *terminal.Frame {
	frame := terminal.NewFrame(targetWidth, targetHeight)

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	scaleX := float64(srcWidth) / float64(targetWidth*2)
	scaleY := float64(srcHeight) / float64(targetHeight*2)

	for ty := 0; ty < targetHeight; ty++ {
		for tx := 0; tx < targetWidth; tx++ {
			sx := int(float64(tx*2) * scaleX)
			sy := int(float64(ty*2) * scaleY)

			var pixels [4]color.RGB
			var validPixels [4]bool
			positions := [4][2]int{
				{sx, sy}, {sx + 1, sy}, {sx, sy + 1}, {sx + 1, sy + 1},
			}
			for i, pos := range positions {
				px, py := pos[0], pos[1]
				if px < srcWidth && py < srcHeight {
					r, g, b, _ := img.At(px, py).RGBA()
					pixels[i] = color.RGB{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}
					validPixels[i] = true
				}
			}

			glyph, fg, bg := EncodeBlockPerceptual(pixels, validPixels)
			frame.SetBlock(tx, ty, glyph, fg, bg)
		}
	}

	return frame
}

// blockEncoder is the signature shared by EncodeBlock / EncodeBlockPerceptual /
// EncodeBlockOptimal, so converters can be parameterized by the encoder.
type blockEncoder func(pixels [4]color.RGB, valid [4]bool) (rune, color.RGB, color.RGB)

// imageToFrameWithEncoder runs the standard 2x2 sampling loop using the given
// block encoder, avoiding copy-paste across the quadrant-family converters.
func imageToFrameWithEncoder(img image.Image, targetWidth, targetHeight int, enc blockEncoder) *terminal.Frame {
	frame := terminal.NewFrame(targetWidth, targetHeight)

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	scaleX := float64(srcWidth) / float64(targetWidth*2)
	scaleY := float64(srcHeight) / float64(targetHeight*2)

	for ty := 0; ty < targetHeight; ty++ {
		for tx := 0; tx < targetWidth; tx++ {
			sx := int(float64(tx*2) * scaleX)
			sy := int(float64(ty*2) * scaleY)

			var pixels [4]color.RGB
			var validPixels [4]bool
			positions := [4][2]int{
				{sx, sy}, {sx + 1, sy}, {sx, sy + 1}, {sx + 1, sy + 1},
			}
			for i, pos := range positions {
				px, py := pos[0], pos[1]
				if px < srcWidth && py < srcHeight {
					r, g, b, _ := img.At(px, py).RGBA()
					pixels[i] = color.RGB{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}
					validPixels[i] = true
				}
			}
			glyph, fg, bg := enc(pixels, validPixels)
			frame.SetBlock(tx, ty, glyph, fg, bg)
		}
	}
	return frame
}

// EncodeBlock determines the best glyph and colors for a 2x2 pixel block
// Uses simple 2-means clustering (k=2)
func EncodeBlock(pixels [4]color.RGB, valid [4]bool) (rune, color.RGB, color.RGB) {
	// Count valid pixels
	validCount := 0
	for _, v := range valid {
		if v {
			validCount++
		}
	}

	if validCount == 0 {
		return ' ', color.Black, color.Black
	}

	// Simple approach: cluster into two colors using luminance
	// Calculate average luminance
	totalLum := 0.0
	for i := 0; i < 4; i++ {
		if valid[i] {
			totalLum += float64(pixels[i].Luminance())
		}
	}
	avgLum := totalLum / float64(validCount)

	// Split pixels into two groups based on luminance
	var group0, group1 []color.RGB
	var bits [4]bool

	for i := 0; i < 4; i++ {
		if valid[i] {
			lum := float64(pixels[i].Luminance())
			if lum >= avgLum {
				group1 = append(group1, pixels[i])
				bits[i] = true
			} else {
				group0 = append(group0, pixels[i])
				bits[i] = false
			}
		}
	}

	// Calculate average colors for each group
	bg := averageColor(group0) // darker group = background
	fg := averageColor(group1) // lighter group = foreground

	// Get the glyph based on bit pattern
	glyph := glyphs.GetGlyphFromBits(bits[0], bits[1], bits[2], bits[3])

	return glyph.Char, fg, bg
}

// EncodeBlockPerceptual is a higher-quality variant of EncodeBlock that clusters
// the 2x2 block in the OKLab perceptual space instead of by raw luminance, and
// averages each group's color perceptually. The split threshold is the mean
// OKLab lightness, so the two chosen colors better match how the block looks to
// the eye — which is what dominates perceived quality at terminal resolution.
func EncodeBlockPerceptual(pixels [4]color.RGB, valid [4]bool) (rune, color.RGB, color.RGB) {
	// Convert each valid pixel to OKLab ONCE and reuse (the conversion is the hot
	// path — cbrt/pow — so doing it per-pixel-per-use was the bottleneck D6 found).
	var lab [4]color.OKLab
	validCount := 0
	totalL := 0.0
	for i := 0; i < 4; i++ {
		if valid[i] {
			lab[i] = pixels[i].ToOKLab()
			validCount++
			totalL += lab[i].L
		}
	}
	if validCount == 0 {
		return ' ', color.Black, color.Black
	}
	avgL := totalL / float64(validCount)

	// Accumulate group means directly in OKLab (no slices, no re-conversion).
	var l0, a0, b0, l1, a1, b1 float64
	var n0, n1 int
	var bits [4]bool
	for i := 0; i < 4; i++ {
		if !valid[i] {
			continue
		}
		if lab[i].L >= avgL {
			l1 += lab[i].L
			a1 += lab[i].A
			b1 += lab[i].B
			n1++
			bits[i] = true
		} else {
			l0 += lab[i].L
			a0 += lab[i].A
			b0 += lab[i].B
			n0++
		}
	}

	bg := meanOKLab(l0, a0, b0, n0) // darker group = background
	fg := meanOKLab(l1, a1, b1, n1) // lighter group = foreground
	glyph := glyphs.GetGlyphFromBits(bits[0], bits[1], bits[2], bits[3])
	return glyph.Char, fg, bg
}

// meanOKLab returns the RGB of the averaged OKLab sums over n samples (black if
// n==0). Avoids slice allocation in the per-block hot path.
func meanOKLab(l, a, b float64, n int) color.RGB {
	if n == 0 {
		return color.Black
	}
	fn := float64(n)
	return color.OKLab{L: l / fn, A: a / fn, B: b / fn}.ToRGB()
}

// averageColorOKLab averages colors in OKLab space (perceptually even mean),
// avoiding the muddy results of averaging gamma-encoded sRGB directly.
func averageColorOKLab(colors []color.RGB) color.RGB {
	if len(colors) == 0 {
		return color.Black
	}
	var l, a, b float64
	for _, c := range colors {
		o := c.ToOKLab()
		l += o.L
		a += o.A
		b += o.B
	}
	n := float64(len(colors))
	return color.OKLab{L: l / n, A: a / n, B: b / n}.ToRGB()
}

// averageColor calculates the average of a slice of colors
func averageColor(colors []color.RGB) color.RGB {
	if len(colors) == 0 {
		return color.Black
	}

	var r, g, b uint32
	for _, c := range colors {
		r += uint32(c.R)
		g += uint32(c.G)
		b += uint32(c.B)
	}

	n := uint32(len(colors))
	return color.RGB{
		R: uint8(r / n),
		G: uint8(g / n),
		B: uint8(b / n),
	}
}

// SimpleImageToFrame converts an image using simple luminance mapping (faster, lower quality)
func SimpleImageToFrame(img image.Image, targetWidth, targetHeight int) *terminal.Frame {
	frame := terminal.NewFrame(targetWidth, targetHeight)

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	scaleX := float64(srcWidth) / float64(targetWidth)
	scaleY := float64(srcHeight) / float64(targetHeight)

	// Simple ASCII ramp from dark to light
	ramp := []rune(" .:-=+*#%@")

	for ty := 0; ty < targetHeight; ty++ {
		for tx := 0; tx < targetWidth; tx++ {
			sx := int(float64(tx) * scaleX)
			sy := int(float64(ty) * scaleY)

			if sx < srcWidth && sy < srcHeight {
				r, g, b, _ := img.At(sx, sy).RGBA()
				rgb := color.RGB{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
				}

				lum := rgb.Luminance()
				// Map luminance to ramp
				idx := int(float64(lum) / 255.0 * float64(len(ramp)-1))
				if idx >= len(ramp) {
					idx = len(ramp) - 1
				}

				frame.SetBlock(tx, ty, ramp[idx], rgb, color.Black)
			}
		}
	}

	return frame
}

// ColorDistance calculates the perceptual distance between two colors
func ColorDistance(c1, c2 gocolor.Color) float64 {
	r1, g1, b1, _ := c1.RGBA()
	r2, g2, b2, _ := c2.RGBA()

	dr := float64(r1>>8) - float64(r2>>8)
	dg := float64(g1>>8) - float64(g2>>8)
	db := float64(b1>>8) - float64(b2>>8)

	// Weighted Euclidean distance (closer to human perception)
	return 0.3*dr*dr + 0.59*dg*dg + 0.11*db*db
}
