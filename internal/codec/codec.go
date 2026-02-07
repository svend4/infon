package codec

import (
	"fmt"
	"sync"
	"time"
)

// CodecType represents the type of codec
type CodecType string

const (
	CodecTypeH264  CodecType = "h264"
	CodecTypeH265  CodecType = "h265"
	CodecTypeVP8   CodecType = "vp8"
	CodecTypeVP9   CodecType = "vp9"
	CodecTypeOpus  CodecType = "opus"
	CodecTypeAAC   CodecType = "aac"
	CodecTypePCM   CodecType = "pcm"
)

// MediaType represents whether codec is for audio or video
type MediaType string

const (
	MediaTypeAudio MediaType = "audio"
	MediaTypeVideo MediaType = "video"
)

// Codec represents a generic codec interface
type Codec interface {
	// Type returns the codec type
	Type() CodecType

	// MediaType returns audio or video
	MediaType() MediaType

	// Encode encodes raw data
	Encode(data []byte) ([]byte, error)

	// Decode decodes encoded data
	Decode(data []byte) ([]byte, error)

	// GetStatistics returns codec statistics
	GetStatistics() CodecStatistics

	// Reset resets the codec state
	Reset() error

	// Close closes the codec
	Close() error
}

// CodecStatistics represents codec statistics
type CodecStatistics struct {
	FramesEncoded    uint64
	FramesDecoded    uint64
	BytesEncoded     uint64
	BytesDecoded     uint64
	Errors           uint64
	AverageEncodeTime time.Duration
	AverageDecodeTime time.Duration
	CompressionRatio float64
}

// VideoCodec represents a video codec with additional parameters
type VideoCodec interface {
	Codec

	// SetBitrate sets the target bitrate
	SetBitrate(bitrate int) error

	// SetFramerate sets the target framerate
	SetFramerate(fps float64) error

	// SetResolution sets the video resolution
	SetResolution(width, height int) error

	// SetKeyframeInterval sets keyframe interval
	SetKeyframeInterval(interval int) error
}

// AudioCodec represents an audio codec with additional parameters
type AudioCodec interface {
	Codec

	// SetBitrate sets the target bitrate
	SetBitrate(bitrate int) error

	// SetSampleRate sets the sample rate
	SetSampleRate(rate int) error

	// SetChannels sets the number of channels
	SetChannels(channels int) error
}

// CodecConfig represents codec configuration
type CodecConfig struct {
	Type         CodecType
	Bitrate      int     // Target bitrate in bps
	Width        int     // Video width (video only)
	Height       int     // Video height (video only)
	Framerate    float64 // Video framerate (video only)
	SampleRate   int     // Audio sample rate (audio only)
	Channels     int     // Audio channels (audio only)
	KeyframeInterval int // Keyframe interval in frames (video only)
}

// BaseCodec provides common functionality for codecs
type BaseCodec struct {
	mu sync.RWMutex

	codecType CodecType
	mediaType MediaType
	config    CodecConfig

	// Statistics
	framesEncoded    uint64
	framesDecoded    uint64
	bytesEncoded     uint64
	bytesDecoded     uint64
	errors           uint64
	totalEncodeTime  time.Duration
	totalDecodeTime  time.Duration
}

// NewBaseCodec creates a new base codec
func NewBaseCodec(codecType CodecType, mediaType MediaType, config CodecConfig) *BaseCodec {
	return &BaseCodec{
		codecType: codecType,
		mediaType: mediaType,
		config:    config,
	}
}

// Type returns the codec type
func (bc *BaseCodec) Type() CodecType {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.codecType
}

// MediaType returns the media type
func (bc *BaseCodec) MediaType() MediaType {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.mediaType
}

// GetStatistics returns codec statistics
func (bc *BaseCodec) GetStatistics() CodecStatistics {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	avgEncodeTime := time.Duration(0)
	if bc.framesEncoded > 0 {
		avgEncodeTime = bc.totalEncodeTime / time.Duration(bc.framesEncoded)
	}

	avgDecodeTime := time.Duration(0)
	if bc.framesDecoded > 0 {
		avgDecodeTime = bc.totalDecodeTime / time.Duration(bc.framesDecoded)
	}

	compressionRatio := 0.0
	if bc.bytesDecoded > 0 {
		compressionRatio = float64(bc.bytesDecoded) / float64(bc.bytesEncoded)
	}

	return CodecStatistics{
		FramesEncoded:     bc.framesEncoded,
		FramesDecoded:     bc.framesDecoded,
		BytesEncoded:      bc.bytesEncoded,
		BytesDecoded:      bc.bytesDecoded,
		Errors:            bc.errors,
		AverageEncodeTime: avgEncodeTime,
		AverageDecodeTime: avgDecodeTime,
		CompressionRatio:  compressionRatio,
	}
}

// Reset resets codec statistics
func (bc *BaseCodec) Reset() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.framesEncoded = 0
	bc.framesDecoded = 0
	bc.bytesEncoded = 0
	bc.bytesDecoded = 0
	bc.errors = 0
	bc.totalEncodeTime = 0
	bc.totalDecodeTime = 0

	return nil
}

// recordEncode records encoding statistics
func (bc *BaseCodec) recordEncode(inputSize, outputSize int, duration time.Duration) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.framesEncoded++
	bc.bytesDecoded += uint64(inputSize)
	bc.bytesEncoded += uint64(outputSize)
	bc.totalEncodeTime += duration
}

// recordDecode records decoding statistics
func (bc *BaseCodec) recordDecode(inputSize, outputSize int, duration time.Duration) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.framesDecoded++
	bc.bytesDecoded += uint64(outputSize)
	bc.totalDecodeTime += duration
}

// recordError records an error
func (bc *BaseCodec) recordError() {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.errors++
}

// GetConfig returns the codec configuration
func (bc *BaseCodec) GetConfig() CodecConfig {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return bc.config
}

// CodecFactory creates codecs
type CodecFactory struct {
	mu sync.RWMutex

	// Registered codec constructors
	constructors map[CodecType]func(CodecConfig) (Codec, error)
}

// NewCodecFactory creates a new codec factory
func NewCodecFactory() *CodecFactory {
	factory := &CodecFactory{
		constructors: make(map[CodecType]func(CodecConfig) (Codec, error)),
	}

	// Register built-in codecs
	factory.Register(CodecTypeH264, NewH264Codec)
	factory.Register(CodecTypeVP8, NewVP8Codec)
	factory.Register(CodecTypeVP9, NewVP9Codec)
	factory.Register(CodecTypeOpus, NewOpusCodec)
	factory.Register(CodecTypeAAC, NewAACCodec)
	factory.Register(CodecTypePCM, NewPCMCodec)

	return factory
}

// Register registers a codec constructor
func (cf *CodecFactory) Register(codecType CodecType, constructor func(CodecConfig) (Codec, error)) {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	cf.constructors[codecType] = constructor
}

// Create creates a codec instance
func (cf *CodecFactory) Create(codecType CodecType, config CodecConfig) (Codec, error) {
	cf.mu.RLock()
	constructor, ok := cf.constructors[codecType]
	cf.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("codec type %s not registered", codecType)
	}

	return constructor(config)
}

// ListCodecs returns available codec types
func (cf *CodecFactory) ListCodecs() []CodecType {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	types := make([]CodecType, 0, len(cf.constructors))
	for t := range cf.constructors {
		types = append(types, t)
	}

	return types
}

// Global codec factory
var globalFactory = NewCodecFactory()

// CreateCodec creates a codec using the global factory
func CreateCodec(codecType CodecType, config CodecConfig) (Codec, error) {
	return globalFactory.Create(codecType, config)
}

// ListAvailableCodecs returns available codec types
func ListAvailableCodecs() []CodecType {
	return globalFactory.ListCodecs()
}

// DefaultVideoConfig returns default video codec configuration
func DefaultVideoConfig(codecType CodecType, width, height int) CodecConfig {
	return CodecConfig{
		Type:             codecType,
		Bitrate:          2_000_000, // 2 Mbps
		Width:            width,
		Height:           height,
		Framerate:        30.0,
		KeyframeInterval: 60, // Keyframe every 2 seconds at 30fps
	}
}

// DefaultAudioConfig returns default audio codec configuration
func DefaultAudioConfig(codecType CodecType) CodecConfig {
	return CodecConfig{
		Type:       codecType,
		Bitrate:    128_000, // 128 kbps
		SampleRate: 48000,
		Channels:   2, // Stereo
	}
}
