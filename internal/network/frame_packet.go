package network

import (
	"encoding/binary"
	"errors"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// FramePacket represents an encoded video frame
type FramePacket struct {
	Width   uint16       // Frame width in characters
	Height  uint16       // Frame height in characters
	Blocks  []BlockData  // Block data
}

// BlockData represents a single terminal block
type BlockData struct {
	X     uint8  // X position
	Y     uint8  // Y position
	Glyph uint16 // Unicode codepoint (rune as uint16)
	FgR   uint8  // Foreground red
	FgG   uint8  // Foreground green
	FgB   uint8  // Foreground blue
	BgR   uint8  // Background red
	BgG   uint8  // Background green
	BgB   uint8  // Background blue
}

const (
	// BlockDataSize is the size of one encoded block
	BlockDataSize = 2 + 1 + 1 + 2 + 3 + 3 // x(1) + y(1) + glyph(2) + fg(3) + bg(3) = 11 bytes

	// FrameHeaderSize is the size of frame header
	FrameHeaderSize = 2 + 2 // width(2) + height(2) = 4 bytes
)

var (
	ErrFrameTooLarge = errors.New("frame data too large for single packet")
)

// EncodeFrame encodes a terminal frame into bytes
func EncodeFrame(frame *terminal.Frame) ([]byte, error) {
	// Calculate size
	numBlocks := frame.Width * frame.Height
	totalSize := FrameHeaderSize + (numBlocks * BlockDataSize)

	// For now, we encode full frames
	// In production, we'd use delta encoding and compression
	buf := make([]byte, totalSize)

	// Write header
	binary.BigEndian.PutUint16(buf[0:2], uint16(frame.Width))
	binary.BigEndian.PutUint16(buf[2:4], uint16(frame.Height))

	// Write blocks
	offset := FrameHeaderSize
	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			block := frame.Blocks[y][x]

			buf[offset] = uint8(x)
			buf[offset+1] = uint8(y)
			binary.BigEndian.PutUint16(buf[offset+2:offset+4], uint16(block.Glyph))
			buf[offset+4] = block.Fg.R
			buf[offset+5] = block.Fg.G
			buf[offset+6] = block.Fg.B
			buf[offset+7] = block.Bg.R
			buf[offset+8] = block.Bg.G
			buf[offset+9] = block.Bg.B

			offset += BlockDataSize
		}
	}

	return buf, nil
}

// DecodeFrame decodes bytes into a terminal frame
func DecodeFrame(data []byte) (*terminal.Frame, error) {
	if len(data) < FrameHeaderSize {
		return nil, ErrInvalidPacket
	}

	// Read header
	width := int(binary.BigEndian.Uint16(data[0:2]))
	height := int(binary.BigEndian.Uint16(data[2:4]))

	// Create frame
	frame := terminal.NewFrame(width, height)

	// Read blocks
	offset := FrameHeaderSize
	expectedSize := FrameHeaderSize + (width * height * BlockDataSize)

	if len(data) != expectedSize {
		return nil, errors.New("frame data size mismatch")
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if offset+BlockDataSize > len(data) {
				return nil, errors.New("incomplete block data")
			}

			// x and y from packet
			px := int(data[offset])
			py := int(data[offset+1])

			// Sanity check
			if px != x || py != y {
				return nil, errors.New("block position mismatch")
			}

			glyph := rune(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
			fgR := data[offset+4]
			fgG := data[offset+5]
			fgB := data[offset+6]
			bgR := data[offset+7]
			bgG := data[offset+8]
			bgB := data[offset+9]

			frame.Blocks[y][x] = terminal.Block{
				Glyph: glyph,
				Fg:    color.RGB{R: fgR, G: fgG, B: fgB},
				Bg:    color.RGB{R: bgR, G: bgG, B: bgB},
			}

			offset += BlockDataSize
		}
	}

	return frame, nil
}

// EstimateFrameSize estimates the encoded size of a frame
func EstimateFrameSize(width, height int) int {
	return FrameHeaderSize + (width * height * BlockDataSize)
}
