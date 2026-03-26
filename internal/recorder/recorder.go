package recorder

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/pkg/terminal"
)

// Recording represents a call recording
type Recording struct {
	Metadata RecordingMetadata
	Frames   []RecordedFrame
	Audio    []RecordedAudio
}

// RecordingMetadata contains information about the recording
type RecordingMetadata struct {
	Version      uint16    // Format version
	StartTime    time.Time // Recording start time
	Duration     uint32    // Duration in milliseconds
	FrameWidth   uint16    // Video frame width
	FrameHeight  uint16    // Video frame height
	AudioRate    uint16    // Audio sample rate
	AudioCodec   uint8     // Audio codec (0=PCM, 1=Opus)
	VideoCodec   uint8     // Video codec (0=.babe)
	FrameCount   uint32    // Total video frames
	AudioChunks  uint32    // Total audio chunks
}

// RecordedFrame represents a single video frame with timestamp
type RecordedFrame struct {
	Timestamp uint64           // Milliseconds from start
	Frame     *terminal.Frame  // The actual frame data
}

// RecordedAudio represents an audio chunk with timestamp
type RecordedAudio struct {
	Timestamp uint64 // Milliseconds from start
	Samples   []int16 // Audio samples
}

// Recorder handles recording of calls
type Recorder struct {
	file       *os.File
	startTime  time.Time
	frames     []RecordedFrame
	audio      []RecordedAudio
	metadata   RecordingMetadata
	recording  bool
}

const (
	RecordingVersion = 1
	MagicHeader     = 0x54564350 // "TVCP" in hex
)

// NewRecorder creates a new call recorder
func NewRecorder(width, height int, audioRate int) *Recorder {
	return &Recorder{
		frames: make([]RecordedFrame, 0),
		audio:  make([]RecordedAudio, 0),
		metadata: RecordingMetadata{
			Version:     RecordingVersion,
			FrameWidth:  uint16(width),
			FrameHeight: uint16(height),
			AudioRate:   uint16(audioRate),
			AudioCodec:  network.AudioCodecPCM,
			VideoCodec:  0, // .babe
		},
	}
}

// Start starts recording to a file
func (r *Recorder) Start(filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file for writing
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create recording file: %w", err)
	}

	r.file = file
	r.startTime = time.Now()
	r.metadata.StartTime = r.startTime
	r.recording = true

	return nil
}

// RecordFrame records a video frame
func (r *Recorder) RecordFrame(frame *terminal.Frame) error {
	if !r.recording {
		return fmt.Errorf("not recording")
	}

	elapsed := uint64(time.Since(r.startTime).Milliseconds())

	r.frames = append(r.frames, RecordedFrame{
		Timestamp: elapsed,
		Frame:     frame,
	})

	return nil
}

// RecordAudio records an audio chunk
func (r *Recorder) RecordAudio(samples []int16) error {
	if !r.recording {
		return fmt.Errorf("not recording")
	}

	elapsed := uint64(time.Since(r.startTime).Milliseconds())

	// Make a copy of samples
	samplesCopy := make([]int16, len(samples))
	copy(samplesCopy, samples)

	r.audio = append(r.audio, RecordedAudio{
		Timestamp: elapsed,
		Samples:   samplesCopy,
	})

	return nil
}

// Stop stops recording and saves the file
func (r *Recorder) Stop() error {
	if !r.recording {
		return fmt.Errorf("not recording")
	}

	r.recording = false
	r.metadata.Duration = uint32(time.Since(r.startTime).Milliseconds())
	r.metadata.FrameCount = uint32(len(r.frames))
	r.metadata.AudioChunks = uint32(len(r.audio))

	// Write recording to file
	if err := r.writeRecording(); err != nil {
		r.file.Close()
		return err
	}

	r.file.Close()
	return nil
}

// writeRecording writes the recording to disk
func (r *Recorder) writeRecording() error {
	// File format:
	// [Header]
	// - Magic: 4 bytes (0x54564350 = "TVCP")
	// - Version: 2 bytes
	// - Start time: 8 bytes (Unix timestamp)
	// - Duration: 4 bytes (milliseconds)
	// - Frame width: 2 bytes
	// - Frame height: 2 bytes
	// - Audio rate: 2 bytes
	// - Audio codec: 1 byte
	// - Video codec: 1 byte
	// - Frame count: 4 bytes
	// - Audio chunk count: 4 bytes
	//
	// [Video Frames]
	// For each frame:
	// - Timestamp: 8 bytes
	// - Frame data length: 4 bytes
	// - Frame data: N bytes
	//
	// [Audio Chunks]
	// For each chunk:
	// - Timestamp: 8 bytes
	// - Sample count: 4 bytes
	// - Samples: N * 2 bytes

	// Write header
	header := make([]byte, 32)
	binary.BigEndian.PutUint32(header[0:4], MagicHeader)
	binary.BigEndian.PutUint16(header[4:6], r.metadata.Version)
	binary.BigEndian.PutUint64(header[6:14], uint64(r.metadata.StartTime.Unix()))
	binary.BigEndian.PutUint32(header[14:18], r.metadata.Duration)
	binary.BigEndian.PutUint16(header[18:20], r.metadata.FrameWidth)
	binary.BigEndian.PutUint16(header[20:22], r.metadata.FrameHeight)
	binary.BigEndian.PutUint16(header[22:24], r.metadata.AudioRate)
	header[24] = r.metadata.AudioCodec
	header[25] = r.metadata.VideoCodec
	binary.BigEndian.PutUint32(header[26:30], r.metadata.FrameCount)
	binary.BigEndian.PutUint32(header[30:34], r.metadata.AudioChunks)

	if _, err := r.file.Write(header[:34]); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write video frames
	for _, frame := range r.frames {
		// Serialize frame
		frameData := serializeFrame(frame.Frame)

		// Write frame entry
		entry := make([]byte, 12)
		binary.BigEndian.PutUint64(entry[0:8], frame.Timestamp)
		binary.BigEndian.PutUint32(entry[8:12], uint32(len(frameData)))

		if _, err := r.file.Write(entry); err != nil {
			return fmt.Errorf("failed to write frame header: %w", err)
		}

		if _, err := r.file.Write(frameData); err != nil {
			return fmt.Errorf("failed to write frame data: %w", err)
		}
	}

	// Write audio chunks
	for _, audio := range r.audio {
		// Write audio entry
		entry := make([]byte, 12)
		binary.BigEndian.PutUint64(entry[0:8], audio.Timestamp)
		binary.BigEndian.PutUint32(entry[8:12], uint32(len(audio.Samples)))

		if _, err := r.file.Write(entry); err != nil {
			return fmt.Errorf("failed to write audio header: %w", err)
		}

		// Write samples
		samples := make([]byte, len(audio.Samples)*2)
		for i, sample := range audio.Samples {
			binary.BigEndian.PutUint16(samples[i*2:i*2+2], uint16(sample))
		}

		if _, err := r.file.Write(samples); err != nil {
			return fmt.Errorf("failed to write audio samples: %w", err)
		}
	}

	return nil
}

// serializeFrame converts a terminal.Frame to bytes
func serializeFrame(frame *terminal.Frame) []byte {
	// Simple serialization: width(2) + height(2) + block data
	// Each block: glyph(4) + fg_r(1) + fg_g(1) + fg_b(1) + bg_r(1) + bg_g(1) + bg_b(1) = 10 bytes
	blockCount := frame.Width * frame.Height
	size := 4 + blockCount*10
	data := make([]byte, size)

	binary.BigEndian.PutUint16(data[0:2], uint16(frame.Width))
	binary.BigEndian.PutUint16(data[2:4], uint16(frame.Height))

	offset := 4
	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			block := frame.Blocks[y][x]

			// Write glyph (rune = 4 bytes)
			binary.BigEndian.PutUint32(data[offset:offset+4], uint32(block.Glyph))
			offset += 4

			// Write foreground color (3 bytes)
			data[offset] = block.Fg.R
			data[offset+1] = block.Fg.G
			data[offset+2] = block.Fg.B
			offset += 3

			// Write background color (3 bytes)
			data[offset] = block.Bg.R
			data[offset+1] = block.Bg.G
			data[offset+2] = block.Bg.B
			offset += 3
		}
	}

	return data
}

// IsRecording returns true if currently recording
func (r *Recorder) IsRecording() bool {
	return r.recording
}

// GetStats returns current recording statistics
func (r *Recorder) GetStats() (frameCount, audioCount int, duration time.Duration) {
	if r.recording {
		duration = time.Since(r.startTime)
	} else {
		duration = time.Duration(r.metadata.Duration) * time.Millisecond
	}
	return len(r.frames), len(r.audio), duration
}
