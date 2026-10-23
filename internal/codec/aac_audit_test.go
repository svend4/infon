package codec

import (
	"bytes"
	"testing"
)

// adtsFrameLength extracts the 13-bit frame length field from an ADTS header.
func adtsFrameLength(h []byte) int {
	return ((int(h[3]) & 0x03) << 11) | (int(h[4]) << 3) | ((int(h[5]) >> 5) & 0x07)
}

// decodeNoPanic calls Decode and converts a panic into a test failure.
func decodeNoPanic(t *testing.T, c Codec, data []byte) (out []byte, err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Decode(%x) panicked: %v", data, r)
		}
	}()
	return c.Decode(data)
}

// The ADTS frame length must describe the bytes actually written (header +
// compressed payload), so that Decode(Encode(x)) consumes the whole frame.
func TestAACAuditEncodeDecodeRoundTrip(t *testing.T) {
	c, err := NewAACCodec(DefaultAudioConfig(CodecTypeAAC))
	if err != nil {
		t.Fatalf("NewAACCodec: %v", err)
	}
	aac := c.(*AACCodec)

	for _, n := range []int{64, 1000, 4096, 8192, 20000} {
		pcm := make([]byte, n)
		for i := range pcm {
			pcm[i] = byte(i*7 + 3)
		}

		encoded, err := c.Encode(pcm)
		if err != nil {
			t.Fatalf("n=%d: Encode: %v", n, err)
		}
		if got := adtsFrameLength(encoded); got != len(encoded) {
			t.Fatalf("n=%d: ADTS frame length = %d, want %d (len(encoded))", n, got, len(encoded))
		}

		decoded, err := decodeNoPanic(t, c, encoded)
		if err != nil {
			t.Fatalf("n=%d: Decode(Encode(x)): %v", n, err)
		}

		// Decode must reproduce exactly what the shared decompressor yields
		// for the whole compressed payload.
		payload := aac.compressAudio(pcm)
		want := make([]byte, len(payload)*8)
		if err := aac.decompressAudio(payload, want); err != nil {
			t.Fatalf("n=%d: decompressAudio: %v", n, err)
		}
		if !bytes.Equal(decoded, want) {
			t.Fatalf("n=%d: decoded %d bytes, want %d bytes from full payload", n, len(decoded), len(want))
		}
	}
}

// A compressed frame that cannot be described by the 13-bit ADTS frame
// length must not be emitted with a silently truncated length.
func TestAACAuditEncodeOversizedFrame(t *testing.T) {
	cfg := DefaultAudioConfig(CodecTypeAAC)
	cfg.Bitrate = 10_000_000 // large target size -> large compressed frame
	c, err := NewAACCodec(cfg)
	if err != nil {
		t.Fatalf("NewAACCodec: %v", err)
	}

	pcm := make([]byte, 40000)
	encoded, err := c.Encode(pcm)
	if err != nil {
		return // rejected: correct
	}
	if got := adtsFrameLength(encoded); got != len(encoded) {
		t.Fatalf("oversized frame encoded with ADTS frame length %d, actual length %d", got, len(encoded))
	}
}

// A header whose frame length is smaller than the 7-byte header itself is
// malformed and must be rejected without panicking.
func TestAACAuditDecodeShortFrameLength(t *testing.T) {
	c, err := NewAACCodec(DefaultAudioConfig(CodecTypeAAC))
	if err != nil {
		t.Fatalf("NewAACCodec: %v", err)
	}

	for _, fl := range []int{0, 1, 6} {
		h := []byte{0xFF, 0xF1, 0x4C, 0x80, 0x00, 0x1F, 0xFC}
		h[3] = (h[3] &^ 0x03) | byte((fl>>11)&0x03)
		h[4] = byte((fl >> 3) & 0xFF)
		h[5] = byte(((fl & 0x07) << 5) | 0x1F)
		data := append(h, 0x01, 0x02, 0x03)

		if got := adtsFrameLength(data); got != fl {
			t.Fatalf("test setup: frame length %d, want %d", got, fl)
		}
		if _, err := decodeNoPanic(t, c, data); err == nil {
			t.Fatalf("Decode with frame length %d: expected error", fl)
		}
	}

	// A frame length of exactly 7 (header only) is well-formed.
	h := []byte{0xFF, 0xF1, 0x4C, 0x80, 0x00, 0xFF, 0xFC}
	if fl := adtsFrameLength(h); fl != 7 {
		t.Fatalf("test setup: frame length %d, want 7", fl)
	}
	if _, err := decodeNoPanic(t, c, h); err != nil {
		t.Fatalf("Decode header-only frame: %v", err)
	}
}
