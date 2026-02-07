//go:build opus
// +build opus

package audio

import (
	"fmt"

	"gopkg.in/hraban/opus.v2"
)

const (
	// OpusFrameSize is the number of samples per frame at 16kHz
	// 20ms = 320 samples at 16kHz
	OpusFrameSize = 320

	// OpusBitrate for voice-optimized encoding
	OpusBitrate = 12000 // 12 kbps
)

// OpusEncoder wraps the Opus encoder
type OpusEncoder struct {
	encoder *opus.Encoder
	format  AudioFormat
}

// OpusDecoder wraps the Opus decoder
type OpusDecoder struct {
	decoder *opus.Decoder
	format  AudioFormat
}

// NewOpusEncoder creates a new Opus encoder
func NewOpusEncoder(format AudioFormat) (*OpusEncoder, error) {
	if format.Channels != 1 {
		return nil, fmt.Errorf("opus encoder only supports mono (got %d channels)", format.Channels)
	}
	if format.SampleRate != 16000 {
		return nil, fmt.Errorf("opus encoder optimized for 16kHz (got %d Hz)", format.SampleRate)
	}

	// Create encoder
	encoder, err := opus.NewEncoder(format.SampleRate, format.Channels, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus encoder: %w", err)
	}

	// Set bitrate for voice optimization
	if err := encoder.SetBitrate(OpusBitrate); err != nil {
		return nil, fmt.Errorf("failed to set opus bitrate: %w", err)
	}

	// Enable VBR (Variable Bitrate) for better quality
	if err := encoder.SetVbr(true); err != nil {
		return nil, fmt.Errorf("failed to enable VBR: %w", err)
	}

	// Set complexity (0-10, higher = better quality but more CPU)
	// 5 is a good balance for real-time communication
	if err := encoder.SetComplexity(5); err != nil {
		return nil, fmt.Errorf("failed to set complexity: %w", err)
	}

	return &OpusEncoder{
		encoder: encoder,
		format:  format,
	}, nil
}

// Encode encodes PCM samples to Opus format
// Input: PCM samples (int16)
// Output: Opus compressed data ([]byte)
func (e *OpusEncoder) Encode(pcm []int16) ([]byte, error) {
	if len(pcm) != OpusFrameSize {
		return nil, fmt.Errorf("opus expects %d samples, got %d", OpusFrameSize, len(pcm))
	}

	// Opus can compress to ~1/10th the size
	// PCM: 320 samples × 2 bytes = 640 bytes
	// Opus: ~30-60 bytes (depending on content)
	output := make([]byte, 1024) // Generous buffer

	n, err := e.encoder.Encode(pcm, output)
	if err != nil {
		return nil, fmt.Errorf("failed to encode opus: %w", err)
	}

	return output[:n], nil
}

// NewOpusDecoder creates a new Opus decoder
func NewOpusDecoder(format AudioFormat) (*OpusDecoder, error) {
	if format.Channels != 1 {
		return nil, fmt.Errorf("opus decoder only supports mono (got %d channels)", format.Channels)
	}
	if format.SampleRate != 16000 {
		return nil, fmt.Errorf("opus decoder optimized for 16kHz (got %d Hz)", format.SampleRate)
	}

	decoder, err := opus.NewDecoder(format.SampleRate, format.Channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create opus decoder: %w", err)
	}

	return &OpusDecoder{
		decoder: decoder,
		format:  format,
	}, nil
}

// Decode decodes Opus data to PCM samples
// Input: Opus compressed data ([]byte)
// Output: PCM samples (int16)
func (d *OpusDecoder) Decode(data []byte) ([]int16, error) {
	// Opus frames are always 320 samples at 16kHz (20ms)
	pcm := make([]int16, OpusFrameSize)

	n, err := d.decoder.Decode(data, pcm)
	if err != nil {
		return nil, fmt.Errorf("failed to decode opus: %w", err)
	}

	return pcm[:n], nil
}

// DecodeFEC decodes with forward error correction (for packet loss)
func (d *OpusDecoder) DecodeFEC(data []byte, fec bool) ([]int16, error) {
	pcm := make([]int16, OpusFrameSize)

	n, err := d.decoder.DecodeFEC(data, pcm)
	if err != nil {
		return nil, fmt.Errorf("failed to decode opus with FEC: %w", err)
	}

	return pcm[:n], nil
}
