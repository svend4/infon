package codec

import (
	"bytes"
	"fmt"
	"time"
)

// AACCodec implements AAC audio codec
type AACCodec struct {
	*BaseCodec

	sampleRate int
	channels   int
	bitrate    int

	profile AACProfile
}

// AACProfile represents AAC encoding profile
type AACProfile int

const (
	AACProfileLC   AACProfile = 1 // Low Complexity (most common)
	AACProfileHE   AACProfile = 2 // High Efficiency (HE-AAC)
	AACProfileHEv2 AACProfile = 3 // High Efficiency v2
)

// NewAACCodec creates a new AAC codec
func NewAACCodec(config CodecConfig) (Codec, error) {
	if config.SampleRate == 0 {
		config.SampleRate = 48000
	}

	if config.Channels == 0 {
		config.Channels = 2
	}

	if config.Bitrate == 0 {
		config.Bitrate = 128_000
	}

	codec := &AACCodec{
		BaseCodec:  NewBaseCodec(CodecTypeAAC, MediaTypeAudio, config),
		sampleRate: config.SampleRate,
		channels:   config.Channels,
		bitrate:    config.Bitrate,
		profile:    AACProfileLC, // Default to LC
	}

	return codec, nil
}

// Encode encodes raw PCM audio to AAC
func (a *AACCodec) Encode(data []byte) ([]byte, error) {
	start := time.Now()

	if len(data) == 0 {
		a.recordError()
		return nil, fmt.Errorf("no data to encode")
	}

	buf := &bytes.Buffer{}

	// Write ADTS header (Audio Data Transport Stream)
	adtsHeader := a.generateADTSHeader(len(data))
	buf.Write(adtsHeader)

	// Compress audio data
	compressed := a.compressAudio(data)
	buf.Write(compressed)

	encoded := buf.Bytes()

	a.recordEncode(len(data), len(encoded), time.Since(start))

	return encoded, nil
}

// Decode decodes AAC data to raw PCM audio
func (a *AACCodec) Decode(data []byte) ([]byte, error) {
	start := time.Now()

	if len(data) < 7 {
		a.recordError()
		return nil, fmt.Errorf("data too short for AAC frame")
	}

	// Parse ADTS header
	if !a.isADTSHeader(data) {
		a.recordError()
		return nil, fmt.Errorf("invalid ADTS header")
	}

	// Get frame length from ADTS header
	frameLength := a.getADTSFrameLength(data)

	if len(data) < frameLength {
		a.recordError()
		return nil, fmt.Errorf("incomplete AAC frame")
	}

	// Decode audio (skip ADTS header)
	audioData := data[7:frameLength]

	// Decompress
	decoded := make([]byte, len(audioData)*8) // Estimate decompressed size
	if err := a.decompressAudio(audioData, decoded); err != nil {
		a.recordError()
		return nil, err
	}

	a.recordDecode(len(data), len(decoded), time.Since(start))

	return decoded, nil
}

// Close closes the codec
func (a *AACCodec) Close() error {
	return nil
}

// SetBitrate sets the target bitrate
func (a *AACCodec) SetBitrate(bitrate int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// AAC supports 8 kbps to 320 kbps per channel
	maxBitrate := 320_000 * a.channels
	if bitrate < 8_000 || bitrate > maxBitrate {
		return fmt.Errorf("bitrate out of range: %d", bitrate)
	}

	a.bitrate = bitrate
	a.config.Bitrate = bitrate

	return nil
}

// SetSampleRate sets the sample rate
func (a *AACCodec) SetSampleRate(rate int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// AAC supports various sample rates
	validRates := []int{8000, 11025, 12000, 16000, 22050, 24000, 32000, 44100, 48000, 88200, 96000}
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

	a.sampleRate = rate
	a.config.SampleRate = rate

	return nil
}

// SetChannels sets the number of channels
func (a *AACCodec) SetChannels(channels int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if channels < 1 || channels > 8 {
		return fmt.Errorf("channels out of range: %d", channels)
	}

	a.channels = channels
	a.config.Channels = channels

	return nil
}

// SetProfile sets the AAC profile
func (a *AACCodec) SetProfile(profile AACProfile) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.profile = profile
}

// Private methods

func (a *AACCodec) generateADTSHeader(dataSize int) []byte {
	// ADTS header is 7 bytes (without CRC)
	header := make([]byte, 7)

	// Syncword (12 bits): 0xFFF
	header[0] = 0xFF
	header[1] = 0xF1 // 0xF (syncword) + 0 (MPEG-4) + 00 (Layer) + 1 (no CRC)

	// Profile (2 bits) + Sample rate index (4 bits) + private (1 bit) + channel config start (1 bit)
	sampleRateIndex := a.getSampleRateIndex()
	header[2] = byte((int(a.profile)-1)<<6) | byte(sampleRateIndex<<2) | byte((a.channels>>2)&0x01)

	// Channel config (2 bits) + original (1 bit) + home (1 bit) + copyrighted (1 bit) + copyright start (1 bit) + frame length start (2 bits)
	frameLength := dataSize + 7 // Data + header
	header[3] = byte((a.channels&0x03)<<6) | byte((frameLength>>11)&0x03)

	// Frame length (11 bits continued)
	header[4] = byte((frameLength >> 3) & 0xFF)
	header[5] = byte(((frameLength & 0x07) << 5) | 0x1F)

	// Buffer fullness (11 bits) + number of frames (2 bits)
	header[6] = 0xFC

	return header
}

func (a *AACCodec) getSampleRateIndex() int {
	rates := []int{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000}
	for i, rate := range rates {
		if rate == a.sampleRate {
			return i
		}
	}
	return 4 // Default to 44.1kHz
}

func (a *AACCodec) isADTSHeader(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	// Check syncword
	return data[0] == 0xFF && (data[1]&0xF0) == 0xF0
}

func (a *AACCodec) getADTSFrameLength(data []byte) int {
	if len(data) < 6 {
		return 0
	}

	// Extract frame length from ADTS header
	length := ((int(data[3]) & 0x03) << 11) |
		(int(data[4]) << 3) |
		((int(data[5]) >> 5) & 0x07)

	return length
}

func (a *AACCodec) compressAudio(pcm []byte) []byte {
	// Simplified AAC compression
	// Real AAC uses MDCT, psychoacoustic model, and Huffman coding

	compressed := &bytes.Buffer{}

	// Calculate target compression ratio
	targetSize := a.bitrate / 8 / 50 // Assuming ~20ms frames
	ratio := len(pcm) / targetSize

	if ratio < 2 {
		ratio = 2
	}

	// Simple downsampling
	for i := 0; i < len(pcm); i += ratio {
		if i < len(pcm) {
			compressed.WriteByte(pcm[i])
		}
	}

	return compressed.Bytes()
}

func (a *AACCodec) decompressAudio(compressed []byte, output []byte) error {
	// Simplified AAC decompression

	if len(compressed) == 0 {
		// Fill with silence
		for i := range output {
			output[i] = 0
		}
		return nil
	}

	ratio := len(output) / len(compressed)

	if ratio < 1 {
		ratio = 1
	}

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

// PCMCodec implements raw PCM audio (no compression)
type PCMCodec struct {
	*BaseCodec

	sampleRate int
	channels   int
	bitDepth   int // 8, 16, 24, or 32 bits
}

// NewPCMCodec creates a new PCM codec
func NewPCMCodec(config CodecConfig) (Codec, error) {
	if config.SampleRate == 0 {
		config.SampleRate = 48000
	}

	if config.Channels == 0 {
		config.Channels = 2
	}

	codec := &PCMCodec{
		BaseCodec:  NewBaseCodec(CodecTypePCM, MediaTypeAudio, config),
		sampleRate: config.SampleRate,
		channels:   config.Channels,
		bitDepth:   16, // Default 16-bit
	}

	return codec, nil
}

// Encode "encodes" PCM (just passes through)
func (p *PCMCodec) Encode(data []byte) ([]byte, error) {
	start := time.Now()

	// PCM encoding is just a passthrough
	encoded := make([]byte, len(data))
	copy(encoded, data)

	p.recordEncode(len(data), len(encoded), time.Since(start))

	return encoded, nil
}

// Decode "decodes" PCM (just passes through)
func (p *PCMCodec) Decode(data []byte) ([]byte, error) {
	start := time.Now()

	// PCM decoding is just a passthrough
	decoded := make([]byte, len(data))
	copy(decoded, data)

	p.recordDecode(len(data), len(decoded), time.Since(start))

	return decoded, nil
}

// Close closes the codec
func (p *PCMCodec) Close() error {
	return nil
}

// SetBitrate is a no-op for PCM
func (p *PCMCodec) SetBitrate(bitrate int) error {
	// PCM doesn't have a bitrate setting
	return nil
}

// SetSampleRate sets the sample rate
func (p *PCMCodec) SetSampleRate(rate int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if rate < 8000 || rate > 192000 {
		return fmt.Errorf("sample rate out of range: %d", rate)
	}

	p.sampleRate = rate
	p.config.SampleRate = rate

	return nil
}

// SetChannels sets the number of channels
func (p *PCMCodec) SetChannels(channels int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if channels < 1 || channels > 8 {
		return fmt.Errorf("channels out of range: %d", channels)
	}

	p.channels = channels
	p.config.Channels = channels

	return nil
}

// SetBitDepth sets the bit depth
func (p *PCMCodec) SetBitDepth(bitDepth int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	validDepths := []int{8, 16, 24, 32}
	valid := false
	for _, d := range validDepths {
		if bitDepth == d {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid bit depth: %d (must be 8, 16, 24, or 32)", bitDepth)
	}

	p.bitDepth = bitDepth

	return nil
}
