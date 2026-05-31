// Package avatar implements the ultra-low-bandwidth "neural avatar" path
// (roadmap C5): instead of transmitting video, the sender extracts compact face
// KEYPOINTS and ships only those; the receiver reconstructs the face from the
// keypoints plus a one-time reference image, using a generative model. This is
// the logical extreme of the semantic codec for calls — kilobits/sec for a
// talking face — and falls back to ordinary block video when no model is present.
//
// This file implements the buildable, model-free core: the keypoint container,
// its compact wire encoding, and the bandwidth accounting that proves the win.
// The landmark extractor (sender) and the face generator (receiver) are external
// models behind HTTP sidecars (see ai/adapters/avatar_*_sidecar.py).
package avatar

import (
	"encoding/binary"
	"errors"
	"math"
)

// Keypoints is a frame of face landmarks in normalized coordinates (0..1), plus
// a small set of expression/pose scalars. 68 points is the classic dlib layout;
// any count works. This is what travels per frame — typically a few hundred
// bytes vs hundreds of KB for video.
type Keypoints struct {
	Frame  uint32       // monotonic frame index
	Points [][2]float32 // normalized landmark coordinates
	// Pose: yaw, pitch, roll in radians (head orientation), optional.
	Yaw, Pitch, Roll float32
}

// magic identifies the compact keypoint wire format.
const magic = 0x4B50 // "KP"

// Encode packs keypoints into a compact binary frame: a small header plus two
// uint16 per point (each normalized coord quantized to 0..65535). At 68 points
// that is ~10 + 272 = ~282 bytes — independent of image resolution.
func (k Keypoints) Encode() []byte {
	n := len(k.Points)
	buf := make([]byte, 0, 4+2+4+12+n*4)
	var hdr [22]byte
	binary.BigEndian.PutUint16(hdr[0:], magic)
	binary.BigEndian.PutUint16(hdr[2:], uint16(n))
	binary.BigEndian.PutUint32(hdr[4:], k.Frame)
	binary.BigEndian.PutUint32(hdr[8:], math.Float32bits(k.Yaw))
	binary.BigEndian.PutUint32(hdr[12:], math.Float32bits(k.Pitch))
	binary.BigEndian.PutUint32(hdr[16:], math.Float32bits(k.Roll))
	// hdr[20:22] reserved (padding/version).
	buf = append(buf, hdr[:]...)

	for _, p := range k.Points {
		var q [4]byte
		binary.BigEndian.PutUint16(q[0:], quantize(p[0]))
		binary.BigEndian.PutUint16(q[2:], quantize(p[1]))
		buf = append(buf, q[:]...)
	}
	return buf
}

// ErrBadKeypoints is returned for malformed wire data.
var ErrBadKeypoints = errors.New("avatar: malformed keypoint frame")

// DecodeKeypoints parses a compact keypoint frame.
func DecodeKeypoints(data []byte) (Keypoints, error) {
	if len(data) < 22 {
		return Keypoints{}, ErrBadKeypoints
	}
	if binary.BigEndian.Uint16(data[0:]) != magic {
		return Keypoints{}, ErrBadKeypoints
	}
	n := int(binary.BigEndian.Uint16(data[2:]))
	if len(data) < 22+n*4 {
		return Keypoints{}, ErrBadKeypoints
	}
	k := Keypoints{
		Frame: binary.BigEndian.Uint32(data[4:]),
		Yaw:   math.Float32frombits(binary.BigEndian.Uint32(data[8:])),
		Pitch: math.Float32frombits(binary.BigEndian.Uint32(data[12:])),
		Roll:  math.Float32frombits(binary.BigEndian.Uint32(data[16:])),
	}
	k.Points = make([][2]float32, n)
	off := 22
	for i := 0; i < n; i++ {
		x := binary.BigEndian.Uint16(data[off:])
		y := binary.BigEndian.Uint16(data[off+2:])
		k.Points[i] = [2]float32{dequantize(x), dequantize(y)}
		off += 4
	}
	return k, nil
}

// EncodedSize returns the wire size for n keypoints (header + 4 bytes/point).
func EncodedSize(n int) int { return 22 + n*4 }

func quantize(v float32) uint16 {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return uint16(v*65535 + 0.5)
}

func dequantize(q uint16) float32 {
	return float32(q) / 65535.0
}
