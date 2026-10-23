package codec

import "testing"

// A VP8 frame consisting only of the 5-byte header carries no payload and
// must be rejected with an error rather than panicking.
func TestVP8AuditDecodeHeaderOnly(t *testing.T) {
	for _, mk := range []func(CodecConfig) (Codec, error){NewVP8Codec, NewVP9Codec} {
		c, err := mk(DefaultVideoConfig(CodecTypeVP8, 640, 480))
		if err != nil {
			t.Fatalf("new codec: %v", err)
		}

		for _, frameType := range []byte{0x00, 0x01} {
			data := []byte{frameType, 0x80, 0x02, 0xE0, 0x01}
			if _, err := decodeNoPanic(t, c, data); err == nil {
				t.Fatalf("%v: Decode of header-only frame (type %#x): expected error", c.Type(), frameType)
			}
		}
	}
}
