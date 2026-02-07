package audio

import (
	"encoding/binary"
	"errors"
)

// EncodePCM encodes int16 samples to PCM byte stream (little-endian)
func EncodePCM(samples []int16) []byte {
	if len(samples) == 0 {
		return []byte{}
	}

	data := make([]byte, len(samples)*2)

	for i, sample := range samples {
		binary.LittleEndian.PutUint16(data[i*2:], uint16(sample))
	}

	return data
}

// DecodePCM decodes PCM byte stream (little-endian) to int16 samples
func DecodePCM(data []byte) ([]int16, error) {
	if len(data)%2 != 0 {
		return nil, errors.New("invalid PCM data length (must be even)")
	}

	if len(data) == 0 {
		return []int16{}, nil
	}

	samples := make([]int16, len(data)/2)

	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
	}

	return samples, nil
}
