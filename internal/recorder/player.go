package recorder

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// Player plays back recorded calls
type Player struct {
	recording *Recording
	position  time.Duration
	playing   bool
}

// NewPlayer creates a new recording player
func NewPlayer() *Player {
	return &Player{}
}

// Load loads a recording from file
func (p *Player) Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open recording: %w", err)
	}
	defer file.Close()

	recording, err := readRecording(file)
	if err != nil {
		return fmt.Errorf("failed to read recording: %w", err)
	}

	p.recording = recording
	return nil
}

// Play plays the recording
func (p *Player) Play() error {
	if p.recording == nil {
		return fmt.Errorf("no recording loaded")
	}

	p.playing = true
	defer func() { p.playing = false }()

	startTime := time.Now()
	frameIndex := 0
	audioIndex := 0

	fmt.Println("▶️  Playing recording...")
	fmt.Printf("Duration: %.1fs | Frames: %d | Audio chunks: %d\n",
		float64(p.recording.Metadata.Duration)/1000.0,
		p.recording.Metadata.FrameCount,
		p.recording.Metadata.AudioChunks)
	fmt.Println("Press Ctrl+C to stop\n")

	time.Sleep(500 * time.Millisecond)

	// Playback loop
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for p.playing {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime)
			elapsedMs := uint64(elapsed.Milliseconds())

			// Play video frames
			for frameIndex < len(p.recording.Frames) {
				frame := p.recording.Frames[frameIndex]
				if frame.Timestamp > elapsedMs {
					break
				}

				// Render frame
				renderFrame(frame.Frame)

				frameIndex++
			}

			// Update progress
			if frameIndex%15 == 0 { // Every ~1 second
				progress := float64(elapsedMs) / float64(p.recording.Metadata.Duration) * 100
				fmt.Printf("\r[Playback] %.1f%% | %d/%d frames | %.1fs    ",
					progress,
					frameIndex,
					p.recording.Metadata.FrameCount,
					float64(elapsedMs)/1000.0)
			}

			// Check if done
			if elapsedMs >= uint64(p.recording.Metadata.Duration) ||
				(frameIndex >= len(p.recording.Frames) && audioIndex >= len(p.recording.Audio)) {
				p.playing = false
			}
		}
	}

	fmt.Println("\n\n⏹️  Playback complete")
	return nil
}

// Stop stops playback
func (p *Player) Stop() {
	p.playing = false
}

// readRecording reads a recording from file
func readRecording(file *os.File) (*Recording, error) {
	// Read header
	header := make([]byte, 34)
	if _, err := io.ReadFull(file, header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Verify magic header
	magic := binary.BigEndian.Uint32(header[0:4])
	if magic != MagicHeader {
		return nil, fmt.Errorf("invalid recording file: bad magic header")
	}

	// Parse metadata
	metadata := RecordingMetadata{
		Version:     binary.BigEndian.Uint16(header[4:6]),
		StartTime:   time.Unix(int64(binary.BigEndian.Uint64(header[6:14])), 0),
		Duration:    binary.BigEndian.Uint32(header[14:18]),
		FrameWidth:  binary.BigEndian.Uint16(header[18:20]),
		FrameHeight: binary.BigEndian.Uint16(header[20:22]),
		AudioRate:   binary.BigEndian.Uint16(header[22:24]),
		AudioCodec:  header[24],
		VideoCodec:  header[25],
		FrameCount:  binary.BigEndian.Uint32(header[26:30]),
		AudioChunks: binary.BigEndian.Uint32(header[30:34]),
	}

	recording := &Recording{
		Metadata: metadata,
		Frames:   make([]RecordedFrame, 0, metadata.FrameCount),
		Audio:    make([]RecordedAudio, 0, metadata.AudioChunks),
	}

	// Read video frames
	for i := 0; i < int(metadata.FrameCount); i++ {
		entry := make([]byte, 12)
		if _, err := io.ReadFull(file, entry); err != nil {
			return nil, fmt.Errorf("failed to read frame %d header: %w", i, err)
		}

		timestamp := binary.BigEndian.Uint64(entry[0:8])
		dataLen := binary.BigEndian.Uint32(entry[8:12])

		frameData := make([]byte, dataLen)
		if _, err := io.ReadFull(file, frameData); err != nil {
			return nil, fmt.Errorf("failed to read frame %d data: %w", i, err)
		}

		frame := deserializeFrame(frameData)

		recording.Frames = append(recording.Frames, RecordedFrame{
			Timestamp: timestamp,
			Frame:     frame,
		})
	}

	// Read audio chunks
	for i := 0; i < int(metadata.AudioChunks); i++ {
		entry := make([]byte, 12)
		if _, err := io.ReadFull(file, entry); err != nil {
			return nil, fmt.Errorf("failed to read audio %d header: %w", i, err)
		}

		timestamp := binary.BigEndian.Uint64(entry[0:8])
		sampleCount := binary.BigEndian.Uint32(entry[8:12])

		sampleData := make([]byte, sampleCount*2)
		if _, err := io.ReadFull(file, sampleData); err != nil {
			return nil, fmt.Errorf("failed to read audio %d samples: %w", i, err)
		}

		samples := make([]int16, sampleCount)
		for j := 0; j < int(sampleCount); j++ {
			samples[j] = int16(binary.BigEndian.Uint16(sampleData[j*2 : j*2+2]))
		}

		recording.Audio = append(recording.Audio, RecordedAudio{
			Timestamp: timestamp,
			Samples:   samples,
		})
	}

	return recording, nil
}

// deserializeFrame converts bytes back to terminal.Frame
func deserializeFrame(data []byte) *terminal.Frame {
	if len(data) < 4 {
		return terminal.NewFrame(0, 0)
	}

	width := int(binary.BigEndian.Uint16(data[0:2]))
	height := int(binary.BigEndian.Uint16(data[2:4]))

	frame := terminal.NewFrame(width, height)
	offset := 4

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if offset+10 > len(data) {
				break
			}

			// Read glyph (4 bytes)
			glyph := rune(binary.BigEndian.Uint32(data[offset : offset+4]))
			offset += 4

			// Read foreground color (3 bytes)
			fg := color.RGB{
				R: data[offset],
				G: data[offset+1],
				B: data[offset+2],
			}
			offset += 3

			// Read background color (3 bytes)
			bg := color.RGB{
				R: data[offset],
				G: data[offset+1],
				B: data[offset+2],
			}
			offset += 3

			frame.SetBlock(x, y, glyph, fg, bg)
		}
	}

	return frame
}

// renderFrame renders a frame to the terminal
func renderFrame(frame *terminal.Frame) {
	// Simple rendering: display as colored blocks
	fmt.Print(color.ClearScreen)

	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			block := frame.Blocks[y][x]
			// Use ANSI RGB color for background and display the glyph
			fmt.Printf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm%c",
				block.Fg.R, block.Fg.G, block.Fg.B,
				block.Bg.R, block.Bg.G, block.Bg.B,
				block.Glyph)
		}
		fmt.Print(color.Reset + "\n")
	}
}

// GetInfo returns information about the loaded recording
func (p *Player) GetInfo() RecordingMetadata {
	if p.recording == nil {
		return RecordingMetadata{}
	}
	return p.recording.Metadata
}
