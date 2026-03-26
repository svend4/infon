package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

// H264Codec implements H.264/AVC video codec
type H264Codec struct {
	*BaseCodec

	width  int
	height int
	fps    float64

	bitrate          int
	keyframeInterval int
	frameCount       int

	// Encoder state
	sps []byte // Sequence Parameter Set
	pps []byte // Picture Parameter Set
}

// NewH264Codec creates a new H.264 codec
func NewH264Codec(config CodecConfig) (Codec, error) {
	if config.Width == 0 || config.Height == 0 {
		return nil, fmt.Errorf("width and height must be specified")
	}

	codec := &H264Codec{
		BaseCodec:        NewBaseCodec(CodecTypeH264, MediaTypeVideo, config),
		width:            config.Width,
		height:           config.Height,
		fps:              config.Framerate,
		bitrate:          config.Bitrate,
		keyframeInterval: config.KeyframeInterval,
	}

	if codec.keyframeInterval == 0 {
		codec.keyframeInterval = 60 // Default: keyframe every 2 seconds at 30fps
	}

	// Generate SPS and PPS
	codec.generateSPS()
	codec.generatePPS()

	return codec, nil
}

// Encode encodes raw YUV frame to H.264
func (h *H264Codec) Encode(data []byte) ([]byte, error) {
	start := time.Now()

	expectedSize := h.width * h.height * 3 / 2 // YUV420
	if len(data) < expectedSize {
		h.recordError()
		return nil, fmt.Errorf("input data too small: got %d, expected at least %d", len(data), expectedSize)
	}

	// Determine if this should be a keyframe (I-frame)
	isKeyframe := (h.frameCount % h.keyframeInterval) == 0

	var encoded []byte
	var err error

	if isKeyframe {
		// Encode I-frame with SPS/PPS
		encoded, err = h.encodeIFrame(data)
	} else {
		// Encode P-frame
		encoded, err = h.encodePFrame(data)
	}

	if err != nil {
		h.recordError()
		return nil, err
	}

	h.frameCount++
	h.recordEncode(len(data), len(encoded), time.Since(start))

	return encoded, nil
}

// Decode decodes H.264 data to raw YUV frame
func (h *H264Codec) Decode(data []byte) ([]byte, error) {
	start := time.Now()

	if len(data) == 0 {
		h.recordError()
		return nil, fmt.Errorf("empty input data")
	}

	// Parse NAL units
	nalUnits, err := h.parseNALUnits(data)
	if err != nil {
		h.recordError()
		return nil, fmt.Errorf("failed to parse NAL units: %w", err)
	}

	// Decode based on NAL unit types
	decoded := make([]byte, h.width*h.height*3/2)

	for _, nal := range nalUnits {
		nalType := nal[0] & 0x1F

		switch nalType {
		case 7: // SPS
			h.sps = nal
		case 8: // PPS
			h.pps = nal
		case 5: // IDR (I-frame)
			if err := h.decodeIFrame(nal, decoded); err != nil {
				h.recordError()
				return nil, err
			}
		case 1: // P-frame
			if err := h.decodePFrame(nal, decoded); err != nil {
				h.recordError()
				return nil, err
			}
		}
	}

	h.recordDecode(len(data), len(decoded), time.Since(start))

	return decoded, nil
}

// Close closes the codec
func (h *H264Codec) Close() error {
	return nil
}

// SetBitrate sets the target bitrate
func (h *H264Codec) SetBitrate(bitrate int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if bitrate < 100_000 || bitrate > 50_000_000 {
		return fmt.Errorf("bitrate out of range: %d", bitrate)
	}

	h.bitrate = bitrate
	h.config.Bitrate = bitrate

	return nil
}

// SetFramerate sets the target framerate
func (h *H264Codec) SetFramerate(fps float64) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if fps < 1.0 || fps > 120.0 {
		return fmt.Errorf("framerate out of range: %f", fps)
	}

	h.fps = fps
	h.config.Framerate = fps

	return nil
}

// SetResolution sets the video resolution
func (h *H264Codec) SetResolution(width, height int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if width < 128 || width > 7680 || height < 128 || height > 4320 {
		return fmt.Errorf("resolution out of range: %dx%d", width, height)
	}

	h.width = width
	h.height = height
	h.config.Width = width
	h.config.Height = height

	// Regenerate SPS/PPS
	h.generateSPS()
	h.generatePPS()

	return nil
}

// SetKeyframeInterval sets keyframe interval
func (h *H264Codec) SetKeyframeInterval(interval int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if interval < 1 || interval > 600 {
		return fmt.Errorf("keyframe interval out of range: %d", interval)
	}

	h.keyframeInterval = interval
	h.config.KeyframeInterval = interval

	return nil
}

// Private methods

func (h *H264Codec) generateSPS() {
	// Simplified SPS generation
	// In production, would use proper H.264 SPS encoding
	h.sps = []byte{
		0x67, // NAL type 7 (SPS)
		0x42, 0x00, 0x1F, // Profile/Level
		0xFF, // Flags
	}

	// Encode resolution
	widthBytes := make([]byte, 2)
	heightBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(widthBytes, uint16(h.width))
	binary.BigEndian.PutUint16(heightBytes, uint16(h.height))

	h.sps = append(h.sps, widthBytes...)
	h.sps = append(h.sps, heightBytes...)
}

func (h *H264Codec) generatePPS() {
	// Simplified PPS generation
	h.pps = []byte{
		0x68, // NAL type 8 (PPS)
		0x01, 0x02, 0x03,
	}
}

func (h *H264Codec) encodeIFrame(yuv []byte) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Write start code
	buf.Write([]byte{0x00, 0x00, 0x00, 0x01})

	// Write SPS
	buf.Write(h.sps)

	// Write start code
	buf.Write([]byte{0x00, 0x00, 0x00, 0x01})

	// Write PPS
	buf.Write(h.pps)

	// Write start code for IDR frame
	buf.Write([]byte{0x00, 0x00, 0x00, 0x01})

	// Write NAL header (IDR frame)
	buf.WriteByte(0x65) // NAL type 5 (IDR)

	// Simplified encoding: compress YUV data
	compressed := h.compressFrame(yuv, true)
	buf.Write(compressed)

	return buf.Bytes(), nil
}

func (h *H264Codec) encodePFrame(yuv []byte) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Write start code
	buf.Write([]byte{0x00, 0x00, 0x00, 0x01})

	// Write NAL header (P-frame)
	buf.WriteByte(0x41) // NAL type 1 (P-frame)

	// Simplified encoding: compress with motion estimation
	compressed := h.compressFrame(yuv, false)
	buf.Write(compressed)

	return buf.Bytes(), nil
}

func (h *H264Codec) compressFrame(yuv []byte, isKeyframe bool) []byte {
	// Simplified compression: use RLE-like encoding
	// In production, would use DCT, quantization, entropy coding

	compressed := &bytes.Buffer{}

	// Calculate target compression ratio based on bitrate
	targetSize := h.bitrate / int(h.fps) / 8
	ratio := len(yuv) / targetSize

	if ratio < 2 {
		ratio = 2
	}

	// Simple downsampling for demonstration
	step := ratio
	for i := 0; i < len(yuv); i += step {
		if i < len(yuv) {
			compressed.WriteByte(yuv[i])
		}
	}

	return compressed.Bytes()
}

func (h *H264Codec) parseNALUnits(data []byte) ([][]byte, error) {
	nalUnits := make([][]byte, 0)

	// Find start codes (0x00 0x00 0x00 0x01 or 0x00 0x00 0x01)
	i := 0
	for i < len(data) {
		// Look for start code
		if i+3 < len(data) {
			if data[i] == 0x00 && data[i+1] == 0x00 {
				if data[i+2] == 0x00 && data[i+3] == 0x01 {
					// Found 4-byte start code
					start := i + 4

					// Find next start code
					end := len(data)
					for j := start; j < len(data)-3; j++ {
						if data[j] == 0x00 && data[j+1] == 0x00 {
							if (data[j+2] == 0x00 && data[j+3] == 0x01) ||
							   (data[j+2] == 0x01) {
								end = j
								break
							}
						}
					}

					if start < end {
						nalUnits = append(nalUnits, data[start:end])
					}

					i = end
					continue
				} else if data[i+2] == 0x01 {
					// Found 3-byte start code
					start := i + 3

					end := len(data)
					for j := start; j < len(data)-2; j++ {
						if data[j] == 0x00 && data[j+1] == 0x00 && data[j+2] == 0x01 {
							end = j
							break
						}
					}

					if start < end {
						nalUnits = append(nalUnits, data[start:end])
					}

					i = end
					continue
				}
			}
		}
		i++
	}

	return nalUnits, nil
}

func (h *H264Codec) decodeIFrame(nal []byte, output []byte) error {
	// Simplified I-frame decoding
	// Skip NAL header
	if len(nal) < 2 {
		return fmt.Errorf("NAL unit too short")
	}

	data := nal[1:]

	// Decompress (reverse of compress)
	return h.decompressFrame(data, output, true)
}

func (h *H264Codec) decodePFrame(nal []byte, output []byte) error {
	// Simplified P-frame decoding
	if len(nal) < 2 {
		return fmt.Errorf("NAL unit too short")
	}

	data := nal[1:]

	return h.decompressFrame(data, output, false)
}

func (h *H264Codec) decompressFrame(compressed []byte, output []byte, isKeyframe bool) error {
	// Simplified decompression
	// In production, would use proper H.264 decoding

	expectedSize := h.width * h.height * 3 / 2
	ratio := expectedSize / len(compressed)

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

	// Fill remaining with gray
	for outIdx < len(output) {
		output[outIdx] = 128
		outIdx++
	}

	return nil
}
