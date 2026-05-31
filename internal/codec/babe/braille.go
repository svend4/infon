package babe

import (
	"image"

	"github.com/svend4/infon/internal/codec/glyphs"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// ImageToFrameBraille renders using Braille patterns (roadmap A3): each cell is
// a 2×4 dot matrix (8 dots), giving the highest *spatial* resolution of the
// glyph modes (cols×2 wide, rows×4 tall) with the widest font support. Color is
// per-cell (one foreground over a background), so it suits line art, edges,
// plots, and high-contrast scenes rather than full-color photos.
//
// A dot is "on" when its sub-pixel is brighter than the cell's mean lightness;
// the foreground is the average of the lit dots, the background the average of
// the unlit ones.
func ImageToFrameBraille(img image.Image, targetWidth, targetHeight int) *terminal.Frame {
	frame := terminal.NewFrame(targetWidth, targetHeight)

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	if srcWidth == 0 || srcHeight == 0 {
		return frame
	}

	// Each cell covers 2 sub-pixels wide × 4 tall.
	scaleX := float64(srcWidth) / float64(targetWidth*2)
	scaleY := float64(srcHeight) / float64(targetHeight*4)

	for ty := 0; ty < targetHeight; ty++ {
		for tx := 0; tx < targetWidth; tx++ {
			// BraillePattern dot order (dots[0..7]):
			//   1(0) 4(3)
			//   2(1) 5(4)
			//   3(2) 6(5)
			//   7(6) 8(7)
			// i.e. left column top->bottom is dots 0,1,2,6; right is 3,4,5,7.
			var px [8]color.RGB
			// (col,row) -> dots index
			layout := [8][2]int{
				{0, 0}, {0, 1}, {0, 2}, {1, 0},
				{1, 1}, {1, 2}, {0, 3}, {1, 3},
			}
			var sumL float64
			ls := [8]float64{}
			for i, lr := range layout {
				sx := int((float64(tx*2+lr[0]) + 0.5) * scaleX)
				sy := int((float64(ty*4+lr[1]) + 0.5) * scaleY)
				px[i] = sampleRGB(img, sx, sy, bounds)
				ls[i] = px[i].ToOKLab().L
				sumL += ls[i]
			}
			avgL := sumL / 8.0

			var dots [8]bool
			var fgGroup, bgGroup []color.RGB
			for i := range px {
				if ls[i] >= avgL {
					dots[i] = true
					fgGroup = append(fgGroup, px[i])
				} else {
					bgGroup = append(bgGroup, px[i])
				}
			}

			glyph := glyphs.BraillePattern(dots)
			fg := averageColorOKLab(fgGroup)
			bg := averageColorOKLab(bgGroup)
			frame.SetBlock(tx, ty, glyph, fg, bg)
		}
	}
	return frame
}
