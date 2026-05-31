package babe

import (
	"image"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// Half-block rendering (roadmap A1).
//
// Instead of approximating a 2x2 pixel block with one of 16 quadrant glyphs and
// two clustered colors, half-block mode renders every cell as the upper-half
// block '▀' with:
//
//	foreground = the TOP pixel  (paints the upper half)
//	background = the BOTTOM pixel (shows through the lower half)
//
// So each character cell encodes TWO vertically-stacked pixels at their EXACT
// colors — no quantization. Effective resolution becomes cols x (rows*2) with
// full 24-bit color per sub-pixel. This is the de-facto standard for "images in
// the terminal" (chafa/viu/timg) because color fidelity, not sub-cell shape,
// dominates perceived quality at this scale.
//
// Trade-off vs the quadrant path: 1 wide x 2 tall sub-pixels instead of 2x2, so
// it keeps no horizontal sub-cell detail. It is therefore a selectable mode.

const upperHalfBlock = '▀'

// ImageToFrameHalfBlock converts an image to a terminal frame using half-block
// rendering. The target grid is targetWidth x targetHeight cells, sampling a
// targetWidth x (targetHeight*2) pixel grid (two pixels per cell, stacked).
func ImageToFrameHalfBlock(img image.Image, targetWidth, targetHeight int) *terminal.Frame {
	frame := terminal.NewFrame(targetWidth, targetHeight)

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	if srcWidth == 0 || srcHeight == 0 {
		return frame
	}

	// Each cell is 1 pixel wide and 2 pixels tall in the sampled grid.
	scaleX := float64(srcWidth) / float64(targetWidth)
	scaleY := float64(srcHeight) / float64(targetHeight*2)

	for ty := 0; ty < targetHeight; ty++ {
		for tx := 0; tx < targetWidth; tx++ {
			sx := int((float64(tx) + 0.5) * scaleX)
			topY := int((float64(ty*2) + 0.5) * scaleY)
			botY := int((float64(ty*2+1) + 0.5) * scaleY)

			top := sampleRGB(img, sx, topY, bounds)
			bottom := sampleRGB(img, sx, botY, bounds)

			// '▀': foreground paints the upper pixel, background the lower one.
			frame.SetBlock(tx, ty, upperHalfBlock, top, bottom)
		}
	}

	return frame
}

// sampleRGB reads a clamped pixel from img at (x,y) as color.RGB.
func sampleRGB(img image.Image, x, y int, bounds image.Rectangle) color.RGB {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= bounds.Dx() {
		x = bounds.Dx() - 1
	}
	if y >= bounds.Dy() {
		y = bounds.Dy() - 1
	}
	r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
	return color.RGB{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8)}
}
