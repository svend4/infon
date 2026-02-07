package export

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/svend4/infon/internal/recorder"
	"github.com/svend4/infon/pkg/terminal"
)

// VideoExporter exports .tvcp recordings to standard video formats
type VideoExporter struct {
	recording *recorder.Recording
	tempDir   string
	fps       int
	scale     int // Pixel scale factor for terminal blocks
}

// ExportFormat represents output video format
type ExportFormat string

const (
	FormatMP4  ExportFormat = "mp4"
	FormatWebM ExportFormat = "webm"
)

// ExportOptions contains export configuration
type ExportOptions struct {
	Format      ExportFormat
	OutputPath  string
	FPS         int    // Target FPS (default: 15)
	Scale       int    // Scale factor (default: 8)
	VideoCodec  string // Video codec (default: libx264 for MP4, libvpx-vp9 for WebM)
	AudioCodec  string // Audio codec (default: aac for MP4, libopus for WebM)
	Quality     int    // Quality (0-100, default: 75)
	Preset      string // Encoding preset (default: medium)
}

// NewVideoExporter creates a new video exporter
func NewVideoExporter(recording *recorder.Recording) *VideoExporter {
	return &VideoExporter{
		recording: recording,
		fps:       15,
		scale:     8,
	}
}

// Export exports the recording to video format
func (ve *VideoExporter) Export(options ExportOptions) error {
	// Set defaults
	if options.FPS == 0 {
		options.FPS = 15
	}
	if options.Scale == 0 {
		options.Scale = 8
	}
	if options.Quality == 0 {
		options.Quality = 75
	}
	if options.Preset == "" {
		options.Preset = "medium"
	}

	ve.fps = options.FPS
	ve.scale = options.Scale

	// Set codec defaults based on format
	if options.VideoCodec == "" {
		if options.Format == FormatMP4 {
			options.VideoCodec = "libx264"
		} else {
			options.VideoCodec = "libvpx-vp9"
		}
	}
	if options.AudioCodec == "" {
		if options.Format == FormatMP4 {
			options.AudioCodec = "aac"
		} else {
			options.AudioCodec = "libopus"
		}
	}

	// Check FFmpeg availability
	if !ve.checkFFmpeg() {
		return fmt.Errorf("FFmpeg not found. Please install FFmpeg to export videos")
	}

	// Create temporary directory for frames
	tempDir, err := os.MkdirTemp("", "tvcp-export-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	ve.tempDir = tempDir
	defer os.RemoveAll(tempDir)

	fmt.Println("🎬 Exporting recording to video...")
	fmt.Printf("   Format: %s\n", options.Format)
	fmt.Printf("   Resolution: %dx%d\n", int(ve.recording.Metadata.FrameWidth)*ve.scale, int(ve.recording.Metadata.FrameHeight)*ve.scale)
	fmt.Printf("   FPS: %d\n", ve.fps)

	// Step 1: Render frames to images
	fmt.Println("📸 Rendering frames...")
	if err := ve.renderFrames(); err != nil {
		return fmt.Errorf("failed to render frames: %w", err)
	}

	// Step 2: Export audio to WAV
	fmt.Println("🔊 Exporting audio...")
	audioPath := filepath.Join(tempDir, "audio.wav")
	if err := ve.exportAudio(audioPath); err != nil {
		return fmt.Errorf("failed to export audio: %w", err)
	}

	// Step 3: Encode video with FFmpeg
	fmt.Println("🎥 Encoding video...")
	if err := ve.encodeVideo(options, audioPath); err != nil {
		return fmt.Errorf("failed to encode video: %w", err)
	}

	fmt.Printf("✅ Export complete: %s\n", options.OutputPath)
	return nil
}

// checkFFmpeg checks if FFmpeg is available
func (ve *VideoExporter) checkFFmpeg() bool {
	cmd := exec.Command("ffmpeg", "-version")
	return cmd.Run() == nil
}

// renderFrames renders all frames to PNG images
func (ve *VideoExporter) renderFrames() error {
	frameCount := len(ve.recording.Frames)

	for i, recordedFrame := range ve.recording.Frames {
		// Progress indicator
		if i%100 == 0 {
			fmt.Printf("   Frame %d/%d\r", i+1, frameCount)
		}

		// Render frame to image
		img := ve.frameToImage(recordedFrame.Frame)

		// Save as PNG
		framePath := filepath.Join(ve.tempDir, fmt.Sprintf("frame_%06d.png", i))
		if err := ve.saveImage(img, framePath); err != nil {
			return err
		}
	}

	fmt.Printf("   Frame %d/%d ✓\n", frameCount, frameCount)
	return nil
}

// frameToImage converts a terminal frame to an image
func (ve *VideoExporter) frameToImage(frame *terminal.Frame) image.Image {
	// Calculate image dimensions
	width := frame.Width * ve.scale
	height := frame.Height * ve.scale

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Render each block
	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			block := frame.Blocks[y][x]

			// Get block colors
			fg := color.RGBA{R: block.Fg.R, G: block.Fg.G, B: block.Fg.B, A: 255}
			bg := color.RGBA{R: block.Bg.R, G: block.Bg.G, B: block.Bg.B, A: 255}

			// Draw block (simplified - use background color)
			// In a full implementation, would render the glyph
			blockColor := bg
			if block.Glyph != ' ' && block.Glyph != 0 {
				// If there's a glyph, blend with foreground
				blockColor = ve.blendColors(fg, bg, 0.5)
			}

			// Fill block area
			for dy := 0; dy < ve.scale; dy++ {
				for dx := 0; dx < ve.scale; dx++ {
					px := x*ve.scale + dx
					py := y*ve.scale + dy
					img.Set(px, py, blockColor)
				}
			}
		}
	}

	return img
}

// blendColors blends two colors
func (ve *VideoExporter) blendColors(c1, c2 color.RGBA, alpha float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c1.R)*alpha + float64(c2.R)*(1-alpha)),
		G: uint8(float64(c1.G)*alpha + float64(c2.G)*(1-alpha)),
		B: uint8(float64(c1.B)*alpha + float64(c2.B)*(1-alpha)),
		A: 255,
	}
}

// saveImage saves an image as PNG
func (ve *VideoExporter) saveImage(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// exportAudio exports audio to WAV file
func (ve *VideoExporter) exportAudio(outputPath string) error {
	// Create WAV file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write WAV header
	sampleRate := ve.recording.Metadata.AudioRate
	channels := 1 // Mono
	bitsPerSample := 16

	// Calculate total audio samples
	var totalSamples int
	for _, audio := range ve.recording.Audio {
		totalSamples += len(audio.Samples)
	}

	dataSize := totalSamples * channels * (bitsPerSample / 8)

	// Write RIFF header
	file.Write([]byte("RIFF"))
	ve.writeUint32(file, uint32(36+dataSize)) // File size - 8
	file.Write([]byte("WAVE"))

	// Write fmt chunk
	file.Write([]byte("fmt "))
	ve.writeUint32(file, 16) // Chunk size
	ve.writeUint16(file, 1)  // Audio format (PCM)
	ve.writeUint16(file, uint16(channels))
	ve.writeUint32(file, uint32(sampleRate))
	ve.writeUint32(file, uint32(int(sampleRate)*channels*bitsPerSample/8)) // Byte rate
	ve.writeUint16(file, uint16(channels*bitsPerSample/8))                 // Block align
	ve.writeUint16(file, uint16(bitsPerSample))

	// Write data chunk
	file.Write([]byte("data"))
	ve.writeUint32(file, uint32(dataSize))

	// Write audio samples
	for _, audio := range ve.recording.Audio {
		for _, sample := range audio.Samples {
			ve.writeInt16(file, sample)
		}
	}

	return nil
}

// writeUint32 writes uint32 in little-endian
func (ve *VideoExporter) writeUint32(file *os.File, val uint32) {
	buf := []byte{
		byte(val),
		byte(val >> 8),
		byte(val >> 16),
		byte(val >> 24),
	}
	file.Write(buf)
}

// writeUint16 writes uint16 in little-endian
func (ve *VideoExporter) writeUint16(file *os.File, val uint16) {
	buf := []byte{
		byte(val),
		byte(val >> 8),
	}
	file.Write(buf)
}

// writeInt16 writes int16 in little-endian
func (ve *VideoExporter) writeInt16(file *os.File, val int16) {
	buf := []byte{
		byte(val),
		byte(val >> 8),
	}
	file.Write(buf)
}

// encodeVideo encodes video using FFmpeg
func (ve *VideoExporter) encodeVideo(options ExportOptions, audioPath string) error {
	inputPattern := filepath.Join(ve.tempDir, "frame_%06d.png")

	// Build FFmpeg command
	args := []string{
		"-framerate", fmt.Sprintf("%d", ve.fps),
		"-i", inputPattern,
		"-i", audioPath,
		"-c:v", options.VideoCodec,
		"-c:a", options.AudioCodec,
		"-preset", options.Preset,
		"-pix_fmt", "yuv420p",
		"-shortest", // Match shortest stream duration
		"-y", // Overwrite output
	}

	// Add quality settings
	if options.Format == FormatMP4 {
		crf := 51 - (options.Quality * 51 / 100) // Convert 0-100 to CRF 51-0
		args = append(args, "-crf", fmt.Sprintf("%d", crf))
	} else if options.Format == FormatWebM {
		bitrate := options.Quality * 10 // Rough bitrate estimation
		args = append(args, "-b:v", fmt.Sprintf("%dK", bitrate))
	}

	args = append(args, options.OutputPath)

	// Execute FFmpeg
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FFmpeg encoding failed: %w", err)
	}

	return nil
}

// GetExportInfo returns information about the export
func (ve *VideoExporter) GetExportInfo(options ExportOptions) ExportInfo {
	duration := time.Duration(ve.recording.Metadata.Duration) * time.Millisecond
	frameCount := len(ve.recording.Frames)
	audioSamples := 0
	for _, audio := range ve.recording.Audio {
		audioSamples += len(audio.Samples)
	}

	width := int(ve.recording.Metadata.FrameWidth) * options.Scale
	height := int(ve.recording.Metadata.FrameHeight) * options.Scale

	return ExportInfo{
		Duration:     duration,
		FrameCount:   frameCount,
		AudioSamples: audioSamples,
		Width:        width,
		Height:       height,
		FPS:          options.FPS,
		Format:       string(options.Format),
	}
}

// ExportInfo contains export information
type ExportInfo struct {
	Duration     time.Duration
	FrameCount   int
	AudioSamples int
	Width        int
	Height       int
	FPS          int
	Format       string
}
