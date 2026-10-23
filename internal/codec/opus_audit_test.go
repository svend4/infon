package codec

import "testing"

// An Opus packet consisting only of the TOC and frame-count bytes has an
// empty payload; Decode must not panic and must return either an error or
// silence of the advertised length.
func TestOpusAuditDecodeEmptyPayload(t *testing.T) {
	c, err := NewOpusCodec(DefaultAudioConfig(CodecTypeOpus))
	if err != nil {
		t.Fatalf("NewOpusCodec: %v", err)
	}
	opus := c.(*OpusCodec)

	for _, frameCount := range []byte{0, 1, 3} {
		data := []byte{opus.generateTOC(), frameCount}
		decoded, err := decodeNoPanic(t, c, data)
		if err != nil {
			continue
		}
		want := int(frameCount) * opus.GetFrameSize() * 2 * opus.GetChannels()
		if len(decoded) != want {
			t.Fatalf("frameCount=%d: decoded %d bytes, want %d", frameCount, len(decoded), want)
		}
		for i, b := range decoded {
			if b != 0 {
				t.Fatalf("frameCount=%d: decoded[%d] = %d, want silence", frameCount, i, b)
			}
		}
	}
}
