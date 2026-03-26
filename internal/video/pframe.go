package video

import (
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// FrameType represents the type of encoded frame
type FrameType uint8

const (
	// FrameTypeI represents an I-frame (Intra-frame, full frame)
	FrameTypeI FrameType = 0

	// FrameTypeP represents a P-frame (Predicted frame, delta from previous)
	FrameTypeP FrameType = 1
)

// DeltaBlock represents a changed block in a P-frame
type DeltaBlock struct {
	X     uint16    // Block X position
	Y     uint16    // Block Y position
	Glyph rune      // New glyph
	Fg    color.RGB // New foreground color
	Bg    color.RGB // New background color
}

// EncodedFrame represents an encoded frame (I-frame or P-frame)
type EncodedFrame struct {
	Type        FrameType     // I-frame or P-frame
	Width       int           // Frame width (I-frames only)
	Height      int           // Frame height (I-frames only)
	IFrameData  *terminal.Frame // Full frame data (I-frames only)
	DeltaBlocks []DeltaBlock   // Changed blocks (P-frames only)
}

// PFrameEncoder handles P-frame encoding
type PFrameEncoder struct {
	lastFrame *terminal.Frame // Previous frame for delta calculation
	iFrameInterval int        // Send I-frame every N frames
	frameCount     int        // Frames since last I-frame
}

// NewPFrameEncoder creates a new P-frame encoder
func NewPFrameEncoder(iFrameInterval int) *PFrameEncoder {
	return &PFrameEncoder{
		iFrameInterval: iFrameInterval,
		frameCount:     0,
	}
}

// Encode encodes a frame as either I-frame or P-frame
func (enc *PFrameEncoder) Encode(frame *terminal.Frame) *EncodedFrame {
	enc.frameCount++

	// Force I-frame every N frames or if no previous frame
	if enc.lastFrame == nil || enc.frameCount >= enc.iFrameInterval {
		enc.frameCount = 0
		enc.lastFrame = copyFrame(frame)
		return &EncodedFrame{
			Type:       FrameTypeI,
			Width:      frame.Width,
			Height:     frame.Height,
			IFrameData: frame,
		}
	}

	// Calculate delta (P-frame)
	deltaBlocks := calculateDelta(enc.lastFrame, frame)

	// If delta is too large (>50% blocks changed), send I-frame instead
	totalBlocks := frame.Width * frame.Height
	if len(deltaBlocks) > totalBlocks/2 {
		enc.frameCount = 0
		enc.lastFrame = copyFrame(frame)
		return &EncodedFrame{
			Type:       FrameTypeI,
			Width:      frame.Width,
			Height:     frame.Height,
			IFrameData: frame,
		}
	}

	// Send P-frame
	enc.lastFrame = copyFrame(frame)
	return &EncodedFrame{
		Type:        FrameTypeP,
		DeltaBlocks: deltaBlocks,
	}
}

// Reset resets the encoder (forces next frame to be I-frame)
func (enc *PFrameEncoder) Reset() {
	enc.lastFrame = nil
	enc.frameCount = 0
}

// PFrameDecoder handles P-frame decoding
type PFrameDecoder struct {
	currentFrame *terminal.Frame // Reconstructed current frame
}

// NewPFrameDecoder creates a new P-frame decoder
func NewPFrameDecoder() *PFrameDecoder {
	return &PFrameDecoder{}
}

// Decode decodes an encoded frame (I-frame or P-frame)
func (dec *PFrameDecoder) Decode(encoded *EncodedFrame) *terminal.Frame {
	if encoded.Type == FrameTypeI {
		// I-frame: replace current frame entirely
		dec.currentFrame = copyFrame(encoded.IFrameData)
		return dec.currentFrame
	}

	// P-frame: apply delta to current frame
	if dec.currentFrame == nil {
		// No base frame yet, cannot decode P-frame
		// Return empty frame (should not happen in practice)
		return terminal.NewFrame(40, 30)
	}

	// Apply delta blocks
	for _, delta := range encoded.DeltaBlocks {
		if int(delta.Y) < dec.currentFrame.Height && int(delta.X) < dec.currentFrame.Width {
			dec.currentFrame.SetBlock(int(delta.X), int(delta.Y), delta.Glyph, delta.Fg, delta.Bg)
		}
	}

	return dec.currentFrame
}

// Reset resets the decoder
func (dec *PFrameDecoder) Reset() {
	dec.currentFrame = nil
}

// GetCurrentFrame returns the current reconstructed frame
func (dec *PFrameDecoder) GetCurrentFrame() *terminal.Frame {
	return dec.currentFrame
}

// calculateDelta calculates the difference between two frames
func calculateDelta(oldFrame, newFrame *terminal.Frame) []DeltaBlock {
	if oldFrame.Width != newFrame.Width || oldFrame.Height != newFrame.Height {
		// Dimensions changed, cannot calculate delta
		return nil
	}

	deltaBlocks := make([]DeltaBlock, 0)

	for y := 0; y < newFrame.Height; y++ {
		for x := 0; x < newFrame.Width; x++ {
			oldBlock := oldFrame.Blocks[y][x]
			newBlock := newFrame.Blocks[y][x]

			// Check if block changed
			if !blocksEqual(oldBlock, newBlock) {
				deltaBlocks = append(deltaBlocks, DeltaBlock{
					X:     uint16(x),
					Y:     uint16(y),
					Glyph: newBlock.Glyph,
					Fg:    newBlock.Fg,
					Bg:    newBlock.Bg,
				})
			}
		}
	}

	return deltaBlocks
}

// blocksEqual checks if two blocks are identical
func blocksEqual(a, b terminal.Block) bool {
	return a.Glyph == b.Glyph &&
		a.Fg.R == b.Fg.R && a.Fg.G == b.Fg.G && a.Fg.B == b.Fg.B &&
		a.Bg.R == b.Bg.R && a.Bg.G == b.Bg.G && a.Bg.B == b.Bg.B
}

// copyFrame creates a deep copy of a frame
func copyFrame(frame *terminal.Frame) *terminal.Frame {
	copy := terminal.NewFrame(frame.Width, frame.Height)
	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			block := frame.Blocks[y][x]
			copy.SetBlock(x, y, block.Glyph, block.Fg, block.Bg)
		}
	}
	return copy
}

// SerializeEncodedFrame serializes an encoded frame to bytes
func SerializeEncodedFrame(encoded *EncodedFrame) []byte {
	if encoded.Type == FrameTypeI {
		// I-frame format:
		// - Type: 1 byte (0)
		// - Width: 2 bytes
		// - Height: 2 bytes
		// - Block data: width*height*10 bytes
		blockCount := encoded.Width * encoded.Height
		size := 1 + 2 + 2 + blockCount*10
		data := make([]byte, size)

		data[0] = byte(FrameTypeI)
		data[1] = byte(encoded.Width >> 8)
		data[2] = byte(encoded.Width & 0xFF)
		data[3] = byte(encoded.Height >> 8)
		data[4] = byte(encoded.Height & 0xFF)

		offset := 5
		for y := 0; y < encoded.Height; y++ {
			for x := 0; x < encoded.Width; x++ {
				block := encoded.IFrameData.Blocks[y][x]
				writeBlock(data[offset:offset+10], &block)
				offset += 10
			}
		}

		return data
	}

	// P-frame format:
	// - Type: 1 byte (1)
	// - Delta count: 2 bytes
	// - Delta blocks: count*12 bytes (x:2, y:2, glyph:4, fg:3, bg:3)
	size := 1 + 2 + len(encoded.DeltaBlocks)*12
	data := make([]byte, size)

	data[0] = byte(FrameTypeP)
	data[1] = byte(len(encoded.DeltaBlocks) >> 8)
	data[2] = byte(len(encoded.DeltaBlocks) & 0xFF)

	offset := 3
	for _, delta := range encoded.DeltaBlocks {
		// X position (2 bytes)
		data[offset] = byte(delta.X >> 8)
		data[offset+1] = byte(delta.X & 0xFF)
		// Y position (2 bytes)
		data[offset+2] = byte(delta.Y >> 8)
		data[offset+3] = byte(delta.Y & 0xFF)
		// Glyph (4 bytes)
		data[offset+4] = byte(delta.Glyph >> 24)
		data[offset+5] = byte(delta.Glyph >> 16)
		data[offset+6] = byte(delta.Glyph >> 8)
		data[offset+7] = byte(delta.Glyph & 0xFF)
		// Foreground RGB (3 bytes)
		data[offset+8] = delta.Fg.R
		data[offset+9] = delta.Fg.G
		data[offset+10] = delta.Fg.B
		// Background RGB (3 bytes)
		data[offset+11] = delta.Bg.R
		data[offset+12] = delta.Bg.G
		data[offset+13] = delta.Bg.B

		offset += 14
	}

	return data
}

// DeserializeEncodedFrame deserializes an encoded frame from bytes
func DeserializeEncodedFrame(data []byte) *EncodedFrame {
	if len(data) < 1 {
		return nil
	}

	frameType := FrameType(data[0])

	if frameType == FrameTypeI {
		// I-frame
		if len(data) < 5 {
			return nil
		}

		width := int(uint16(data[1])<<8 | uint16(data[2]))
		height := int(uint16(data[3])<<8 | uint16(data[4]))

		frame := terminal.NewFrame(width, height)
		offset := 5

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if offset+10 > len(data) {
					return nil
				}
				block := readBlock(data[offset : offset+10])
				frame.SetBlock(x, y, block.Glyph, block.Fg, block.Bg)
				offset += 10
			}
		}

		return &EncodedFrame{
			Type:       FrameTypeI,
			Width:      width,
			Height:     height,
			IFrameData: frame,
		}
	}

	// P-frame
	if len(data) < 3 {
		return nil
	}

	deltaCount := int(uint16(data[1])<<8 | uint16(data[2]))
	deltaBlocks := make([]DeltaBlock, deltaCount)

	offset := 3
	for i := 0; i < deltaCount; i++ {
		if offset+14 > len(data) {
			return nil
		}

		x := uint16(data[offset])<<8 | uint16(data[offset+1])
		y := uint16(data[offset+2])<<8 | uint16(data[offset+3])
		glyph := rune(uint32(data[offset+4])<<24 | uint32(data[offset+5])<<16 |
			uint32(data[offset+6])<<8 | uint32(data[offset+7]))
		fg := color.RGB{
			R: data[offset+8],
			G: data[offset+9],
			B: data[offset+10],
		}
		bg := color.RGB{
			R: data[offset+11],
			G: data[offset+12],
			B: data[offset+13],
		}

		deltaBlocks[i] = DeltaBlock{
			X:     x,
			Y:     y,
			Glyph: glyph,
			Fg:    fg,
			Bg:    bg,
		}

		offset += 14
	}

	return &EncodedFrame{
		Type:        FrameTypeP,
		DeltaBlocks: deltaBlocks,
	}
}

// writeBlock writes a block to a byte array
func writeBlock(data []byte, block *terminal.Block) {
	// Glyph (4 bytes)
	data[0] = byte(block.Glyph >> 24)
	data[1] = byte(block.Glyph >> 16)
	data[2] = byte(block.Glyph >> 8)
	data[3] = byte(block.Glyph & 0xFF)
	// Foreground RGB (3 bytes)
	data[4] = block.Fg.R
	data[5] = block.Fg.G
	data[6] = block.Fg.B
	// Background RGB (3 bytes)
	data[7] = block.Bg.R
	data[8] = block.Bg.G
	data[9] = block.Bg.B
}

// readBlock reads a block from a byte array
func readBlock(data []byte) terminal.Block {
	glyph := rune(uint32(data[0])<<24 | uint32(data[1])<<16 |
		uint32(data[2])<<8 | uint32(data[3]))
	fg := color.RGB{
		R: data[4],
		G: data[5],
		B: data[6],
	}
	bg := color.RGB{
		R: data[7],
		G: data[8],
		B: data[9],
	}
	return terminal.Block{
		Glyph: glyph,
		Fg:    fg,
		Bg:    bg,
	}
}

// GetCompressionRatio returns the compression ratio for an encoded frame
func (encoded *EncodedFrame) GetCompressionRatio(width, height int) float64 {
	totalBlocks := width * height

	if encoded.Type == FrameTypeI {
		return 1.0 // No compression for I-frames
	}

	// P-frame compression ratio
	changedBlocks := len(encoded.DeltaBlocks)
	if totalBlocks == 0 {
		return 1.0
	}

	return float64(changedBlocks) / float64(totalBlocks)
}
