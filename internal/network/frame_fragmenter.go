package network

import (
	"fmt"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// FrameFragment represents a fragment of a frame
type FrameFragment struct {
	FrameID      uint32 // Unique frame ID
	FragmentID   uint16 // Fragment number (0-based)
	TotalFrags   uint16 // Total number of fragments
	Width        uint16 // Frame width
	Height       uint16 // Frame height
	StartBlock   uint16 // Starting block index
	BlockCount   uint16 // Number of blocks in this fragment
	Blocks       []byte // Block data
}

const (
	// FragmentHeaderSize is the size of fragment header
	FragmentHeaderSize = 4 + 2 + 2 + 2 + 2 + 2 + 2 // 16 bytes

	// MaxFragmentPayload is the maximum payload size per fragment
	MaxFragmentPayload = MaxPacketSize - PacketHeaderSize - FragmentHeaderSize // ~1371 bytes

	// BlocksPerFragment is approximately how many blocks fit in one fragment
	BlocksPerFragment = MaxFragmentPayload / BlockDataSize // ~124 blocks
)

// FragmentFrame splits a frame into multiple fragments that fit in UDP packets
func FragmentFrame(frame *terminal.Frame, frameID uint32) ([][]byte, error) {
	totalBlocks := frame.Width * frame.Height
	numFragments := (totalBlocks + BlocksPerFragment - 1) / BlocksPerFragment

	if numFragments > 65535 {
		return nil, fmt.Errorf("frame too large: requires %d fragments", numFragments)
	}

	fragments := make([][]byte, 0, numFragments)

	for fragID := 0; fragID < numFragments; fragID++ {
		startBlock := fragID * BlocksPerFragment
		endBlock := startBlock + BlocksPerFragment
		if endBlock > totalBlocks {
			endBlock = totalBlocks
		}
		blockCount := endBlock - startBlock

		// Encode fragment header
		fragData := make([]byte, FragmentHeaderSize+blockCount*BlockDataSize)

		// Header
		putUint32(fragData[0:4], frameID)
		putUint16(fragData[4:6], uint16(fragID))
		putUint16(fragData[6:8], uint16(numFragments))
		putUint16(fragData[8:10], uint16(frame.Width))
		putUint16(fragData[10:12], uint16(frame.Height))
		putUint16(fragData[12:14], uint16(startBlock))
		putUint16(fragData[14:16], uint16(blockCount))

		// Encode blocks
		offset := FragmentHeaderSize
		for i := startBlock; i < endBlock; i++ {
			y := i / frame.Width
			x := i % frame.Width
			block := frame.Blocks[y][x]

			fragData[offset] = uint8(x)
			fragData[offset+1] = uint8(y)
			putUint16(fragData[offset+2:offset+4], uint16(block.Glyph))
			fragData[offset+4] = block.Fg.R
			fragData[offset+5] = block.Fg.G
			fragData[offset+6] = block.Fg.B
			fragData[offset+7] = block.Bg.R
			fragData[offset+8] = block.Bg.G
			fragData[offset+9] = block.Bg.B

			offset += BlockDataSize
		}

		fragments = append(fragments, fragData)
	}

	return fragments, nil
}

// AssembleFrame reassembles fragments into a complete frame
func AssembleFrame(fragments [][]byte) (*terminal.Frame, error) {
	if len(fragments) == 0 {
		return nil, fmt.Errorf("no fragments provided")
	}

	// Decode first fragment to get frame dimensions
	if len(fragments[0]) < FragmentHeaderSize {
		return nil, fmt.Errorf("invalid fragment header")
	}

	firstFrag := fragments[0]
	width := int(getUint16(firstFrag[8:10]))
	height := int(getUint16(firstFrag[10:12]))

	// Create frame
	frame := terminal.NewFrame(width, height)

	// Process each fragment
	for _, fragData := range fragments {
		if len(fragData) < FragmentHeaderSize {
			return nil, fmt.Errorf("invalid fragment")
		}

		_ = int(getUint16(fragData[12:14])) // startBlock (not used in reassembly)
		blockCount := int(getUint16(fragData[14:16]))

		offset := FragmentHeaderSize
		for i := 0; i < blockCount; i++ {
			if offset+BlockDataSize > len(fragData) {
				return nil, fmt.Errorf("incomplete block data in fragment")
			}

			x := int(fragData[offset])
			y := int(fragData[offset+1])
			glyph := rune(getUint16(fragData[offset+2 : offset+4]))
			fgR := fragData[offset+4]
			fgG := fragData[offset+5]
			fgB := fragData[offset+6]
			bgR := fragData[offset+7]
			bgG := fragData[offset+8]
			bgB := fragData[offset+9]

			if x < width && y < height {
				frame.Blocks[y][x] = terminal.Block{
					Glyph: glyph,
					Fg:    color.NewRGB(fgR, fgG, fgB),
					Bg:    color.NewRGB(bgR, bgG, bgB),
				}
			}

			offset += BlockDataSize
		}
	}

	return frame, nil
}

// Helper functions for binary encoding
func putUint32(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}

func putUint16(b []byte, v uint16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

func getUint32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func getUint16(b []byte) uint16 {
	return uint16(b[0])<<8 | uint16(b[1])
}
