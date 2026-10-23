package recorder

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
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
//
// It is safe for concurrent use: in a call the video and audio capture
// goroutines record while the main goroutine stops the recording. mu guards
// every field, and Stop holds it while finishing the file, so the frame and
// audio counts written to the header always match the entries written.
//
// A call can last for hours, so nothing is kept in memory: each frame is
// written to the file as it is recorded, and each audio chunk to a temporary
// file next to it (the format stores all audio after all frames). Stop
// appends the audio and fills in the header reserved by Start.
type Recorder struct {
	mu         sync.Mutex
	file       *os.File
	audioFile  *os.File // audio entries until Stop appends them to file
	err        error    // first write error; Stop reports it
	startTime  time.Time
	frameCount int
	audioCount int
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

	// Audio chunks follow all frames in the file; buffer them on disk
	audioFile, err := os.CreateTemp(dir, filepath.Base(filename)+".audio-*")
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to create audio buffer file: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.file = file
	r.audioFile = audioFile
	r.err = nil
	r.frameCount = 0
	r.audioCount = 0
	r.startTime = time.Now()
	r.metadata.StartTime = r.startTime
	r.metadata.Duration = 0
	r.metadata.FrameCount = 0
	r.metadata.AudioChunks = 0

	// Reserve the header; Stop rewrites it with the final counts
	if _, err := file.Write(r.header()); err != nil {
		_ = file.Close()
		_ = audioFile.Close()
		_ = os.Remove(audioFile.Name())
		return fmt.Errorf("failed to write header: %w", err)
	}

	r.recording = true

	return nil
}

// RecordFrame records a video frame
func (r *Recorder) RecordFrame(frame *terminal.Frame) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.recording {
		return fmt.Errorf("not recording")
	}

	if r.err != nil {
		return r.err
	}

	elapsed := uint64(time.Since(r.startTime).Milliseconds())

	// Frame entry: timestamp, data length, frame data
	frameData := serializeFrame(frame)
	entry := make([]byte, 12, 12+len(frameData))
	binary.BigEndian.PutUint64(entry[0:8], elapsed)
	binary.BigEndian.PutUint32(entry[8:12], uint32(len(frameData)))
	entry = append(entry, frameData...)

	if _, err := r.file.Write(entry); err != nil {
		r.err = fmt.Errorf("failed to write frame: %w", err)
		return r.err
	}
	r.frameCount++

	return nil
}

// RecordAudio records an audio chunk
func (r *Recorder) RecordAudio(samples []int16) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.recording {
		return fmt.Errorf("not recording")
	}

	if r.err != nil {
		return r.err
	}

	elapsed := uint64(time.Since(r.startTime).Milliseconds())

	// Audio entry: timestamp, sample count, samples
	entry := make([]byte, 12+len(samples)*2)
	binary.BigEndian.PutUint64(entry[0:8], elapsed)
	binary.BigEndian.PutUint32(entry[8:12], uint32(len(samples)))
	for i, sample := range samples {
		binary.BigEndian.PutUint16(entry[12+i*2:14+i*2], uint16(sample))
	}

	if _, err := r.audioFile.Write(entry); err != nil {
		r.err = fmt.Errorf("failed to write audio chunk: %w", err)
		return r.err
	}
	r.audioCount++

	return nil
}

// Stop stops recording and saves the file
func (r *Recorder) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.recording {
		return fmt.Errorf("not recording")
	}

	r.recording = false
	r.metadata.Duration = uint32(time.Since(r.startTime).Milliseconds())
	r.metadata.FrameCount = uint32(r.frameCount)
	r.metadata.AudioChunks = uint32(r.audioCount)

	// Complete the recording file
	err := r.finishRecording()
	if cerr := r.file.Close(); err == nil && cerr != nil {
		err = fmt.Errorf("failed to close recording: %w", cerr)
	}
	_ = r.audioFile.Close()
	_ = os.Remove(r.audioFile.Name())

	return err
}

// finishRecording appends the buffered audio chunks after the frames already
// in the file and writes the final header
func (r *Recorder) finishRecording() error {
	if r.err != nil {
		return r.err
	}

	if _, err := r.audioFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to read audio buffer: %w", err)
	}
	if _, err := io.Copy(r.file, r.audioFile); err != nil {
		return fmt.Errorf("failed to write audio chunks: %w", err)
	}

	if _, err := r.file.WriteAt(r.header(), 0); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	return nil
}

// header encodes the file header from the metadata
func (r *Recorder) header() []byte {
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

	header := make([]byte, 34)
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

	return header
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
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recording
}

// GetStats returns current recording statistics
func (r *Recorder) GetStats() (frameCount, audioCount int, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.recording {
		duration = time.Since(r.startTime)
	} else {
		duration = time.Duration(r.metadata.Duration) * time.Millisecond
	}
	return r.frameCount, r.audioCount, duration
}
