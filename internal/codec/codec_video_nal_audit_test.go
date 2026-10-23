package codec

import (
	"bytes"
	"testing"
)

// Pixel data that happens to contain 00 00 01 / 00 00 00 01 (or 00 00 03) must
// not be mistaken for NAL start codes: an encoded keyframe is exactly SPS, PPS
// and one IDR NAL, an inter frame exactly one non-IDR NAL, and both decode back
// to the original frame.
func TestCodecVideoAuditNALEmulationPrevention(t *testing.T) {
	c, err := NewH264Codec(DefaultVideoConfig(CodecTypeH264, 640, 480))
	if err != nil {
		t.Fatalf("NewH264Codec: %v", err)
	}
	h := c.(*H264Codec)
	if err := h.SetKeyframeInterval(2); err != nil {
		t.Fatalf("SetKeyframeInterval: %v", err)
	}

	n := 640 * 480 * 3 / 2
	vals := []byte{9, 0, 0, 1, 0, 0, 0, 1, 0, 0, 3, 0, 0, 2, 0x65, 0, 0, 0, 0, 0x41, 7}
	frame := codecVideoAuditBlockyFrame(n, codecVideoAuditStep(n), vals)

	wantTypes := [][]byte{{7, 8, 5}, {1}}
	for k, want := range wantTypes {
		enc, err := c.Encode(frame)
		if err != nil {
			t.Fatalf("frame %d: Encode: %v", k, err)
		}

		nals, err := h.parseNALUnits(enc)
		if err != nil {
			t.Fatalf("frame %d: parseNALUnits: %v", k, err)
		}
		got := make([]byte, len(nals))
		for i, nal := range nals {
			got[i] = nal[0] & 0x1F
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("frame %d: %d NAL units with types %v..., want types %v",
				k, len(got), got[:min(len(got), 8)], want)
		}

		dec, err := codecVideoAuditCall(t, "Decode", func() ([]byte, error) { return c.Decode(enc) })
		if err != nil {
			t.Fatalf("frame %d: Decode: %v", k, err)
		}
		if !bytes.Equal(dec, frame) {
			t.Fatalf("frame %d: Decode(Encode(x)) != x", k)
		}
	}
}

// The emulation-prevention escape must round trip arbitrary byte strings and
// never leave a 00 00 0x (x <= 2) sequence in its output.
func TestCodecVideoAuditEmulationEscapeRoundTrip(t *testing.T) {
	cases := [][]byte{
		nil,
		{0},
		{0, 0},
		{0, 0, 0},
		{0, 0, 1},
		{0, 0, 3},
		{0, 0, 0, 0, 0, 0},
		{1, 0, 0, 2, 0, 0, 3, 0, 0, 4, 0, 0},
		{0, 0, 3, 0, 0, 3, 3},
	}
	for _, in := range cases {
		esc := escapeEmulation(in)
		for i := 0; i+2 < len(esc); i++ {
			if esc[i] == 0 && esc[i+1] == 0 && esc[i+2] <= 2 {
				t.Errorf("escapeEmulation(%x) = %x contains 00 00 %02x", in, esc, esc[i+2])
			}
		}
		if got := unescapeEmulation(esc); !bytes.Equal(got, in) {
			t.Errorf("unescapeEmulation(escapeEmulation(%x)) = %x", in, got)
		}
	}
}
