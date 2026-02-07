package network

import (
	"encoding/binary"
	"fmt"
)

// AudioPacket represents an encoded audio frame
type AudioPacket struct {
	Timestamp  uint64  // Timestamp in milliseconds
	SampleRate uint16  // Sample rate (e.g., 16000, 48000)
	Channels   uint8   // Number of channels (1=mono, 2=stereo)
	Codec      uint8   // Codec type (0=PCM, 1=Opus, etc.)
	Samples    []int16 // Audio samples (PCM only)
	Data       []byte  // Compressed data (Opus, etc.)
}

// Audio codec types
const (
	AudioCodecPCM  uint8 = 0 // Uncompressed PCM
	AudioCodecOpus uint8 = 1 // Opus codec (future)
)

// EncodeAudioPacket encodes an audio packet to bytes
func EncodeAudioPacket(ap *AudioPacket) ([]byte, error) {
	headerSize := 14 // timestamp(8) + samplerate(2) + channels(1) + codec(1) + length(2)

	var dataSize int
	var buf []byte

	// Handle different codecs
	switch ap.Codec {
	case AudioCodecPCM:
		// PCM: encode samples as int16
		dataSize = len(ap.Samples) * 2
		totalSize := headerSize + dataSize

		if totalSize > MaxPacketSize-PacketHeaderSize {
			return nil, fmt.Errorf("audio packet too large: %d bytes", totalSize)
		}

		buf = make([]byte, totalSize)

		// Write header
		binary.BigEndian.PutUint64(buf[0:8], ap.Timestamp)
		binary.BigEndian.PutUint16(buf[8:10], ap.SampleRate)
		buf[10] = ap.Channels
		buf[11] = ap.Codec
		binary.BigEndian.PutUint16(buf[12:14], uint16(len(ap.Samples)))

		// Write samples
		for i, sample := range ap.Samples {
			offset := 14 + i*2
			binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(sample))
		}

	case AudioCodecOpus:
		// Opus: encode compressed data
		dataSize = len(ap.Data)
		totalSize := headerSize + dataSize

		if totalSize > MaxPacketSize-PacketHeaderSize {
			return nil, fmt.Errorf("audio packet too large: %d bytes", totalSize)
		}

		buf = make([]byte, totalSize)

		// Write header
		binary.BigEndian.PutUint64(buf[0:8], ap.Timestamp)
		binary.BigEndian.PutUint16(buf[8:10], ap.SampleRate)
		buf[10] = ap.Channels
		buf[11] = ap.Codec
		binary.BigEndian.PutUint16(buf[12:14], uint16(len(ap.Data)))

		// Write compressed data
		copy(buf[14:], ap.Data)

	default:
		return nil, fmt.Errorf("unsupported audio codec: %d", ap.Codec)
	}

	return buf, nil
}

// DecodeAudioPacket decodes bytes into an audio packet
func DecodeAudioPacket(data []byte) (*AudioPacket, error) {
	if len(data) < 14 {
		return nil, fmt.Errorf("audio packet too short: %d bytes", len(data))
	}

	ap := &AudioPacket{
		Timestamp:  binary.BigEndian.Uint64(data[0:8]),
		SampleRate: binary.BigEndian.Uint16(data[8:10]),
		Channels:   data[10],
		Codec:      data[11],
	}

	dataLength := binary.BigEndian.Uint16(data[12:14])

	switch ap.Codec {
	case AudioCodecPCM:
		// PCM: decode int16 samples
		sampleCount := dataLength
		expectedSize := 14 + int(sampleCount)*2

		if len(data) < expectedSize {
			return nil, fmt.Errorf("audio packet truncated: expected %d, got %d", expectedSize, len(data))
		}

		// Read samples
		ap.Samples = make([]int16, sampleCount)
		for i := 0; i < int(sampleCount); i++ {
			offset := 14 + i*2
			ap.Samples[i] = int16(binary.BigEndian.Uint16(data[offset : offset+2]))
		}

	case AudioCodecOpus:
		// Opus: decode compressed data
		dataSize := int(dataLength)
		expectedSize := 14 + dataSize

		if len(data) < expectedSize {
			return nil, fmt.Errorf("audio packet truncated: expected %d, got %d", expectedSize, len(data))
		}

		// Read compressed data
		ap.Data = make([]byte, dataSize)
		copy(ap.Data, data[14:14+dataSize])

	default:
		return nil, fmt.Errorf("unsupported audio codec: %d", ap.Codec)
	}

	return ap, nil
}

// GetDuration returns the duration of this audio packet in milliseconds
func (ap *AudioPacket) GetDuration() int {
	if ap.SampleRate == 0 {
		return 0
	}
	return int(float64(len(ap.Samples)) / float64(ap.SampleRate) * 1000.0)
}

// GetSize returns the size of this audio packet in bytes
func (ap *AudioPacket) GetSize() int {
	switch ap.Codec {
	case AudioCodecPCM:
		return 14 + len(ap.Samples)*2
	case AudioCodecOpus:
		return 14 + len(ap.Data)
	default:
		return 14
	}
}
