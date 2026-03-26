package codec

import (
	"bytes"
	"fmt"
	"time"
)

// OpusCodec implements Opus audio codec
type OpusCodec struct {
	*BaseCodec

	sampleRate int
	channels   int
	bitrate    int

	// Frame size in samples
	frameSize int
}

// NewOpusCodec creates a new Opus codec
func NewOpusCodec(config CodecConfig) (Codec, error) {
	if config.SampleRate == 0 {
		config.SampleRate = 48000 // Default
	}

	if config.Channels == 0 {
		config.Channels = 2 // Default stereo
	}

	if config.Bitrate == 0 {
		config.Bitrate = 128_000 // Default 128 kbps
	}

	// Validate sample rate (Opus supports: 8k, 12k, 16k, 24k, 48k)
	validRates := []int{8000, 12000, 16000, 24000, 48000}
	valid := false
	for _, rate := range validRates {
		if config.SampleRate == rate {
			valid = true
			break
		}
	}

	if !valid {
		return nil, fmt.Errorf("invalid sample rate: %d (must be 8k, 12k, 16k, 24k, or 48k)", config.SampleRate)
	}

	codec := &OpusCodec{
		BaseCodec:  NewBaseCodec(CodecTypeOpus, MediaTypeAudio, config),
		sampleRate: config.SampleRate,
		channels:   config.Channels,
		bitrate:    config.Bitrate,
		frameSize:  960, // 20ms at 48kHz (typical Opus frame)
	}

	return codec, nil
}

// Encode encodes raw PCM audio to Opus
func (o *OpusCodec) Encode(data []byte) ([]byte, error) {
	start := time.Now()

	// PCM is 16-bit samples
	sampleCount := len(data) / 2 / o.channels

	if sampleCount == 0 {
		o.recordError()
		return nil, fmt.Errorf("no samples to encode")
	}

	// Encode using simplified Opus-like compression
	buf := &bytes.Buffer{}

	// Write Opus TOC byte (Table Of Contents)
	tocByte := o.generateTOC()
	buf.WriteByte(tocByte)

	// Write frame count
	frameCount := (sampleCount + o.frameSize - 1) / o.frameSize
	buf.WriteByte(byte(frameCount))

	// Compress audio data
	compressed := o.compressAudio(data)
	buf.Write(compressed)

	encoded := buf.Bytes()

	o.recordEncode(len(data), len(encoded), time.Since(start))

	return encoded, nil
}

// Decode decodes Opus data to raw PCM audio
func (o *OpusCodec) Decode(data []byte) ([]byte, error) {
	start := time.Now()

	if len(data) < 2 {
		o.recordError()
		return nil, fmt.Errorf("data too short for Opus frame")
	}

	// Parse TOC byte
	//tocByte := data[0]
	frameCount := int(data[1])

	// Decode compressed audio
	decoded := make([]byte, frameCount*o.frameSize*2*o.channels)

	if err := o.decompressAudio(data[2:], decoded); err != nil {
		o.recordError()
		return nil, err
	}

	o.recordDecode(len(data), len(decoded), time.Since(start))

	return decoded, nil
}

// Close closes the codec
func (o *OpusCodec) Close() error {
	return nil
}

// SetBitrate sets the target bitrate
func (o *OpusCodec) SetBitrate(bitrate int) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Opus supports 6 kbps to 510 kbps
	if bitrate < 6_000 || bitrate > 510_000 {
		return fmt.Errorf("bitrate out of range: %d (must be 6-510 kbps)", bitrate)
	}

	o.bitrate = bitrate
	o.config.Bitrate = bitrate

	return nil
}

// SetSampleRate sets the sample rate
func (o *OpusCodec) SetSampleRate(rate int) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	validRates := []int{8000, 12000, 16000, 24000, 48000}
	valid := false
	for _, r := range validRates {
		if rate == r {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid sample rate: %d", rate)
	}

	o.sampleRate = rate
	o.config.SampleRate = rate

	// Adjust frame size based on sample rate
	o.frameSize = rate * 20 / 1000 // 20ms frame

	return nil
}

// SetChannels sets the number of channels
func (o *OpusCodec) SetChannels(channels int) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if channels < 1 || channels > 2 {
		return fmt.Errorf("channels out of range: %d (must be 1 or 2)", channels)
	}

	o.channels = channels
	o.config.Channels = channels

	return nil
}

// Private methods

func (o *OpusCodec) generateTOC() byte {
	// Simplified TOC generation
	// Bit layout: [config(5) | stereo(1) | frame_count_code(2)]

	toc := byte(0)

	// Config bits (simplified)
	if o.sampleRate == 48000 {
		toc |= 0x10 // SILK+CELT hybrid
	} else {
		toc |= 0x08 // SILK only
	}

	// Stereo bit
	if o.channels == 2 {
		toc |= 0x04
	}

	// Frame count code (0 = 1 frame)
	// Already 0

	return toc
}

func (o *OpusCodec) compressAudio(pcm []byte) []byte {
	// Simplified audio compression
	// Real Opus uses SILK + CELT with sophisticated algorithms

	compressed := &bytes.Buffer{}

	// Calculate target compression ratio
	targetSize := o.bitrate / 8 / 50 // 20ms frames at 50 fps
	ratio := len(pcm) / targetSize

	if ratio < 2 {
		ratio = 2
	}

	// Simple downsampling for demonstration
	for i := 0; i < len(pcm); i += ratio {
		if i < len(pcm) {
			compressed.WriteByte(pcm[i])
		}
	}

	return compressed.Bytes()
}

func (o *OpusCodec) decompressAudio(compressed []byte, output []byte) error {
	// Simplified audio decompression

	ratio := len(output) / len(compressed)

	if ratio < 1 {
		ratio = 1
	}

	// Simple upsampling
	outIdx := 0
	for i := 0; i < len(compressed) && outIdx < len(output); i++ {
		for j := 0; j < ratio && outIdx < len(output); j++ {
			output[outIdx] = compressed[i]
			outIdx++
		}
	}

	// Fill remaining with silence
	for outIdx < len(output) {
		output[outIdx] = 0
		outIdx++
	}

	return nil
}

// GetFrameSize returns the frame size in samples
func (o *OpusCodec) GetFrameSize() int {
	return o.frameSize
}

// GetSampleRate returns the sample rate
func (o *OpusCodec) GetSampleRate() int {
	return o.sampleRate
}

// GetChannels returns the number of channels
func (o *OpusCodec) GetChannels() int {
	return o.channels
}
