package codec

import (
	"bytes"
	"testing"
)

// Decoding a peer's stream must not change what this codec's encoder emits: the
// SPS/PPS NALs on the wire are not adopted as the encoder's own parameter sets
// (and the caller's input buffer is not retained).
func TestCodecVideoAuditDecodeKeepsEncoderSPS(t *testing.T) {
	c, err := NewH264Codec(DefaultVideoConfig(CodecTypeH264, 640, 480))
	if err != nil {
		t.Fatalf("NewH264Codec: %v", err)
	}
	h := c.(*H264Codec)
	if err := h.SetKeyframeInterval(1); err != nil {
		t.Fatalf("SetKeyframeInterval: %v", err)
	}

	frame := bytes.Repeat([]byte{7}, 640*480*3/2)
	ref, err := c.Encode(frame)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	peer := []byte{
		0x00, 0x00, 0x00, 0x01, 0x67, 0xDE, 0xAD, 0xBE, 0xEF, 0x11, 0x22, 0x33, 0x44, 0x55,
		0x00, 0x00, 0x00, 0x01, 0x68, 0xCA, 0xFE,
		0x00, 0x00, 0x00, 0x01, 0x65, 0x07, 0x07, 0x07, 0x07,
	}
	if _, err := c.Decode(peer); err != nil {
		t.Fatalf("Decode(peer): %v", err)
	}
	for i := range peer {
		peer[i] = 0xEE // the caller reuses its buffer
	}

	got, err := c.Encode(frame)
	if err != nil {
		t.Fatalf("Encode after Decode: %v", err)
	}
	if !bytes.Equal(got, ref) {
		t.Fatalf("keyframe changed after decoding a peer stream:\n got prefix %x\nwant prefix %x",
			got[:min(len(got), 24)], ref[:min(len(ref), 24)])
	}
}
