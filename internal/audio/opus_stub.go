//go:build !opus
// +build !opus

package audio

import "fmt"

const (
	// OpusFrameSize is the number of samples per frame at 16kHz
	// 20ms = 320 samples at 16kHz
	OpusFrameSize = 320

	// OpusBitrate for voice-optimized encoding
	OpusBitrate = 12000 // 12 kbps
)

// OpusEncoder wraps the Opus encoder (stub version - Opus not available)
type OpusEncoder struct {
	format AudioFormat
}

// OpusDecoder wraps the Opus decoder (stub version - Opus not available)
type OpusDecoder struct {
	format AudioFormat
}

// NewOpusEncoder creates a new Opus encoder (stub - returns error)
func NewOpusEncoder(format AudioFormat) (*OpusEncoder, error) {
	return nil, fmt.Errorf("opus support not compiled in (rebuild with -tags opus and install libopus)")
}

// Encode encodes PCM samples to Opus format (stub - returns error)
func (e *OpusEncoder) Encode(pcm []int16) ([]byte, error) {
	return nil, fmt.Errorf("opus support not available")
}

// NewOpusDecoder creates a new Opus decoder (stub - returns error)
func NewOpusDecoder(format AudioFormat) (*OpusDecoder, error) {
	return nil, fmt.Errorf("opus support not compiled in (rebuild with -tags opus and install libopus)")
}

// Decode decodes Opus data to PCM samples (stub - returns error)
func (d *OpusDecoder) Decode(data []byte) ([]int16, error) {
	return nil, fmt.Errorf("opus support not available")
}

// DecodeFEC decodes with forward error correction (stub - returns error)
func (d *OpusDecoder) DecodeFEC(data []byte, fec bool) ([]int16, error) {
	return nil, fmt.Errorf("opus support not available")
}
