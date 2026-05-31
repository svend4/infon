package vision

import (
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// DetectionOverlay draws labeled bounding boxes (and any landmark points) from a
// vision model onto a terminal.Frame: a box outline per detection with its label
// above it, plus markers for keypoints. This is the visible half of C3 — turning
// model analysis into terminal graphics. Only the box edges/labels are drawn, so
// the underlying picture remains visible inside each box.
func DetectionOverlay(frame *terminal.Frame, dets []Detection, boxColor color.RGB) {
	bg := color.Black
	for _, d := range dets {
		x0, y0, x1, y1 := d.PixelBox(frame.Width, frame.Height)

		// Box edges.
		for x := x0; x <= x1; x++ {
			frame.SetBlock(x, y0, '─', boxColor, bgAt(frame, x, y0))
			frame.SetBlock(x, y1, '─', boxColor, bgAt(frame, x, y1))
		}
		for y := y0; y <= y1; y++ {
			frame.SetBlock(x0, y, '│', boxColor, bgAt(frame, x0, y))
			frame.SetBlock(x1, y, '│', boxColor, bgAt(frame, x1, y))
		}
		// Corners.
		frame.SetBlock(x0, y0, '┌', boxColor, bg)
		frame.SetBlock(x1, y0, '┐', boxColor, bg)
		frame.SetBlock(x0, y1, '└', boxColor, bg)
		frame.SetBlock(x1, y1, '┘', boxColor, bg)

		// Label above the box (or inside if at the top edge).
		if d.Label != "" {
			ly := y0 - 1
			if ly < 0 {
				ly = y0 + 1
			}
			label := d.Label
			if len(label) > x1-x0+1 && x1-x0+1 > 1 {
				label = label[:x1-x0+1]
			}
			frame.DrawText(x0, ly, label, boxColor, bg)
		}

		// Landmark points as small markers.
		for _, p := range d.Points {
			px := clampInt(int(p[0]*float64(frame.Width)), 0, frame.Width-1)
			py := clampInt(int(p[1]*float64(frame.Height)), 0, frame.Height-1)
			frame.SetBlock(px, py, '•', boxColor, bgAt(frame, px, py))
		}
	}
}

// bgAt returns the existing background color of a cell, so overlays preserve the
// underlying picture instead of painting it black.
func bgAt(f *terminal.Frame, x, y int) color.RGB {
	if x < 0 || y < 0 || x >= f.Width || y >= f.Height {
		return color.Black
	}
	return f.Blocks[y][x].Bg
}
