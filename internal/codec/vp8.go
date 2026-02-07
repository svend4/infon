package codec

import (
	"bytes"
	"fmt"
	"time"
)

// VP8Codec implements VP8 video codec
type VP8Codec struct {
	*BaseCodec

	width  int
	height int
	fps    float64

	bitrate          int
	keyframeInterval int
	frameCount       int
}

// NewVP8Codec creates a new VP8 codec
func NewVP8Codec(config CodecConfig) (Codec, error) {
	if config.Width == 0 || config.Height == 0 {
		return nil, fmt.Errorf("width and height must be specified")
	}

	codec := &VP8Codec{
		BaseCodec:        NewBaseCodec(CodecTypeVP8, MediaTypeVideo, config),
		width:            config.Width,
		height:           config.Height,
		fps:              config.Framerate,
		bitrate:          config.Bitrate,
		keyframeInterval: config.KeyframeInterval,
	}

	if codec.keyframeInterval == 0 {
		codec.keyframeInterval = 60
	}

	return codec, nil
}

// Encode encodes raw YUV frame to VP8
func (v *VP8Codec) Encode(data []byte) ([]byte, error) {
	start := time.Now()

	expectedSize := v.width * v.height * 3 / 2
	if len(data) < expectedSize {
		v.recordError()
		return nil, fmt.Errorf("input data too small")
	}

	isKeyframe := (v.frameCount % v.keyframeInterval) == 0

	buf := &bytes.Buffer{}

	// Write VP8 frame header
	if isKeyframe {
		buf.WriteByte(0x00) // Keyframe
	} else {
		buf.WriteByte(0x01) // Inter frame
	}

	// Write frame size
	buf.WriteByte(byte(v.width & 0xFF))
	buf.WriteByte(byte(v.width >> 8))
	buf.WriteByte(byte(v.height & 0xFF))
	buf.WriteByte(byte(v.height >> 8))

	// Compress frame data
	compressed := v.compressFrame(data, isKeyframe)
	buf.Write(compressed)

	v.frameCount++
	encoded := buf.Bytes()

	v.recordEncode(len(data), len(encoded), time.Since(start))

	return encoded, nil
}

// Decode decodes VP8 data to raw YUV frame
func (v *VP8Codec) Decode(data []byte) ([]byte, error) {
	start := time.Now()

	if len(data) < 5 {
		v.recordError()
		return nil, fmt.Errorf("data too short for VP8 frame")
	}

	// Parse header
	frameType := data[0]
	isKeyframe := (frameType == 0x00)

	// Decode frame
	decoded := make([]byte, v.width*v.height*3/2)
	if err := v.decompressFrame(data[5:], decoded, isKeyframe); err != nil {
		v.recordError()
		return nil, err
	}

	v.recordDecode(len(data), len(decoded), time.Since(start))

	return decoded, nil
}

// Close closes the codec
func (v *VP8Codec) Close() error {
	return nil
}

// SetBitrate sets the target bitrate
func (v *VP8Codec) SetBitrate(bitrate int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if bitrate < 100_000 || bitrate > 50_000_000 {
		return fmt.Errorf("bitrate out of range")
	}

	v.bitrate = bitrate
	v.config.Bitrate = bitrate

	return nil
}

// SetFramerate sets the target framerate
func (v *VP8Codec) SetFramerate(fps float64) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if fps < 1.0 || fps > 120.0 {
		return fmt.Errorf("framerate out of range")
	}

	v.fps = fps
	v.config.Framerate = fps

	return nil
}

// SetResolution sets the video resolution
func (v *VP8Codec) SetResolution(width, height int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if width < 128 || width > 7680 || height < 128 || height > 4320 {
		return fmt.Errorf("resolution out of range")
	}

	v.width = width
	v.height = height
	v.config.Width = width
	v.config.Height = height

	return nil
}

// SetKeyframeInterval sets keyframe interval
func (v *VP8Codec) SetKeyframeInterval(interval int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if interval < 1 || interval > 600 {
		return fmt.Errorf("keyframe interval out of range")
	}

	v.keyframeInterval = interval
	v.config.KeyframeInterval = interval

	return nil
}

func (v *VP8Codec) compressFrame(yuv []byte, isKeyframe bool) []byte {
	// Simplified VP8 compression
	targetSize := v.bitrate / int(v.fps) / 8
	ratio := len(yuv) / targetSize

	if ratio < 2 {
		ratio = 2
	}

	compressed := &bytes.Buffer{}
	for i := 0; i < len(yuv); i += ratio {
		if i < len(yuv) {
			compressed.WriteByte(yuv[i])
		}
	}

	return compressed.Bytes()
}

func (v *VP8Codec) decompressFrame(compressed []byte, output []byte, isKeyframe bool) error {
	expectedSize := v.width * v.height * 3 / 2
	ratio := expectedSize / len(compressed)

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

	for outIdx < len(output) {
		output[outIdx] = 128
		outIdx++
	}

	return nil
}

// VP9Codec implements VP9 video codec (similar to VP8 but more efficient)
type VP9Codec struct {
	*VP8Codec
}

// NewVP9Codec creates a new VP9 codec
func NewVP9Codec(config CodecConfig) (Codec, error) {
	vp8, err := NewVP8Codec(config)
	if err != nil {
		return nil, err
	}

	vp9 := &VP9Codec{
		VP8Codec: vp8.(*VP8Codec),
	}

	// Update codec type
	vp9.codecType = CodecTypeVP9

	return vp9, nil
}
