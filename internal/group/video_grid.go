package group

import (
	"fmt"
	"strings"
	"time"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// GridLayout represents different grid layout modes
type GridLayout int

const (
	// LayoutAuto automatically chooses the best layout
	LayoutAuto GridLayout = iota
	// Layout1x1 shows 1 participant (full screen)
	Layout1x1
	// Layout2x1 shows 2 participants (side by side)
	Layout2x1
	// Layout2x2 shows 4 participants (2x2 grid)
	Layout2x2
	// Layout3x2 shows 6 participants (3x2 grid)
	Layout3x2
	// Layout3x3 shows 9 participants (3x3 grid)
	Layout3x3
)

// VideoGrid manages multiple video feeds in a grid layout
type VideoGrid struct {
	width     int
	height    int
	layout    GridLayout
	frames    map[string]*terminal.Frame // Map of peer ID to video frame
	peerNames map[string]string          // Map of peer ID to display name
}

// NewVideoGrid creates a new video grid
func NewVideoGrid(width, height int) *VideoGrid {
	return &VideoGrid{
		width:     width,
		height:    height,
		layout:    LayoutAuto,
		frames:    make(map[string]*terminal.Frame),
		peerNames: make(map[string]string),
	}
}

// SetFrame sets a video frame for a peer
func (vg *VideoGrid) SetFrame(peerID string, frame *terminal.Frame) {
	vg.frames[peerID] = frame
}

// SetPeerName sets a display name for a peer
func (vg *VideoGrid) SetPeerName(peerID string, name string) {
	vg.peerNames[peerID] = name
}

// RemovePeer removes a peer's video frame
func (vg *VideoGrid) RemovePeer(peerID string) {
	delete(vg.frames, peerID)
	delete(vg.peerNames, peerID)
}

// SetLayout sets the grid layout
func (vg *VideoGrid) SetLayout(layout GridLayout) {
	vg.layout = layout
}

// getOptimalLayout determines the best layout for the number of peers
func (vg *VideoGrid) getOptimalLayout(peerCount int) (rows, cols int) {
	if vg.layout != LayoutAuto {
		return vg.getFixedLayout()
	}

	switch peerCount {
	case 0:
		return 1, 1
	case 1:
		return 1, 1
	case 2:
		return 1, 2
	case 3, 4:
		return 2, 2
	case 5, 6:
		return 2, 3
	case 7, 8, 9:
		return 3, 3
	default:
		// For more than 9 peers, use 4x4 or larger
		rows = int(float64(peerCount)/4.0 + 0.5)
		if rows < 4 {
			rows = 4
		}
		cols = 4
		return rows, cols
	}
}

// getFixedLayout returns rows/cols for a fixed layout
func (vg *VideoGrid) getFixedLayout() (rows, cols int) {
	switch vg.layout {
	case Layout1x1:
		return 1, 1
	case Layout2x1:
		return 1, 2
	case Layout2x2:
		return 2, 2
	case Layout3x2:
		return 2, 3
	case Layout3x3:
		return 3, 3
	default:
		return 1, 1
	}
}

// Render renders all video feeds in a grid
func (vg *VideoGrid) Render() *terminal.Frame {
	peerCount := len(vg.frames)
	rows, cols := vg.getOptimalLayout(peerCount)

	// Calculate cell dimensions
	cellWidth := vg.width / cols
	cellHeight := vg.height / rows

	// Create output frame
	output := terminal.NewFrame(vg.width, vg.height)

	// Get peer IDs in sorted order for consistency
	peerIDs := make([]string, 0, len(vg.frames))
	for id := range vg.frames {
		peerIDs = append(peerIDs, id)
	}

	// Render each peer's video in a cell
	idx := 0
	for row := 0; row < rows && idx < len(peerIDs); row++ {
		for col := 0; col < cols && idx < len(peerIDs); col++ {
			peerID := peerIDs[idx]
			frame := vg.frames[peerID]

			// Calculate cell position
			cellX := col * cellWidth
			cellY := row * cellHeight

			// Render frame in cell
			vg.renderCell(output, frame, cellX, cellY, cellWidth, cellHeight)

			// Add peer name label at top of cell
			name := vg.peerNames[peerID]
			if name == "" {
				name = peerID
			}
			vg.addLabel(output, name, cellX, cellY, cellWidth)

			idx++
		}
	}

	// Draw grid lines
	vg.drawGridLines(output, rows, cols, cellWidth, cellHeight)

	return output
}

// renderCell renders a video frame into a grid cell
func (vg *VideoGrid) renderCell(output, frame *terminal.Frame, x, y, width, height int) {
	if frame == nil {
		// Render black placeholder
		for dy := 0; dy < height; dy++ {
			for dx := 0; dx < width; dx++ {
				if x+dx < vg.width && y+dy < vg.height {
					output.SetBlock(x+dx, y+dy, ' ',
						color.Black,
						color.Black,
					)
				}
			}
		}
		return
	}

	// Calculate scale to fit frame in cell
	scaleX := float64(width) / float64(frame.Width)
	scaleY := float64(height) / float64(frame.Height)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Center frame in cell
	scaledWidth := int(float64(frame.Width) * scale)
	scaledHeight := int(float64(frame.Height) * scale)
	offsetX := (width - scaledWidth) / 2
	offsetY := (height - scaledHeight) / 2

	// Render scaled frame
	for dy := 0; dy < scaledHeight; dy++ {
		for dx := 0; dx < scaledWidth; dx++ {
			// Map to source frame coordinates
			srcX := int(float64(dx) / scale)
			srcY := int(float64(dy) / scale)

			if srcX >= 0 && srcX < frame.Width && srcY >= 0 && srcY < frame.Height {
				block := frame.Blocks[srcY][srcX]

				destX := x + offsetX + dx
				destY := y + offsetY + dy

				if destX >= 0 && destX < vg.width && destY >= 0 && destY < vg.height {
					output.SetBlock(destX, destY, block.Glyph, block.Fg, block.Bg)
				}
			}
		}
	}
}

// addLabel adds a text label at the top of a cell
func (vg *VideoGrid) addLabel(output *terminal.Frame, text string, x, y, width int) {
	// Truncate text if too long
	maxLen := width - 2
	if len(text) > maxLen {
		text = text[:maxLen]
	}

	// Center text
	startX := x + (width-len(text))/2

	// Draw label background
	for dx := 0; dx < width; dx++ {
		if x+dx < vg.width && y < vg.height {
			output.SetBlock(x+dx, y, ' ',
				color.White,
				color.RGB{R: 50, G: 50, B: 50}, // Dark gray background
			)
		}
	}

	// Draw label text
	for i, char := range text {
		if startX+i < vg.width && y < vg.height {
			output.SetBlock(startX+i, y, char,
				color.White,
				color.RGB{R: 50, G: 50, B: 50},
			)
		}
	}
}

// drawGridLines draws separator lines between cells
func (vg *VideoGrid) drawGridLines(output *terminal.Frame, rows, cols, cellWidth, cellHeight int) {
	lineColor := color.RGB{R: 100, G: 100, B: 100} // Gray lines

	// Draw vertical lines
	for col := 1; col < cols; col++ {
		x := col * cellWidth
		for y := 0; y < vg.height; y++ {
			if x < vg.width && y < vg.height {
				output.SetBlock(x, y, '│', lineColor, color.Black)
			}
		}
	}

	// Draw horizontal lines
	for row := 1; row < rows; row++ {
		y := row * cellHeight
		for x := 0; x < vg.width; x++ {
			if x < vg.width && y < vg.height {
				output.SetBlock(x, y, '─', lineColor, color.Black)
			}
		}
	}

	// Draw intersections
	for row := 1; row < rows; row++ {
		for col := 1; col < cols; col++ {
			x := col * cellWidth
			y := row * cellHeight
			if x < vg.width && y < vg.height {
				output.SetBlock(x, y, '┼', lineColor, color.Black)
			}
		}
	}
}

// Clear clears all video frames
func (vg *VideoGrid) Clear() {
	vg.frames = make(map[string]*terminal.Frame)
	vg.peerNames = make(map[string]string)
}

// PeerCount returns the number of peers
func (vg *VideoGrid) PeerCount() int {
	return len(vg.frames)
}

// RenderStats renders statistics overlay
func (vg *VideoGrid) RenderStats(peers []*PeerConnection) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("👥 Group Call - %d participants\n", len(peers)))
	sb.WriteString("─────────────────────────────\n")

	for i, peer := range peers {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, peer.ID))
		sb.WriteString(fmt.Sprintf("   📹 Frames: %d | 🎤 Audio: %d\n",
			peer.FrameCount, peer.AudioCount))

		active := "✅"
		if !peer.IsActive() {
			active = "❌"
		}
		sb.WriteString(fmt.Sprintf("   Status: %s | Last seen: %s ago\n",
			active, formatTimeSince(peer.LastSeen)))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatTimeSince formats time since as human-readable string
func formatTimeSince(t time.Time) string {
	d := time.Since(t)

	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
