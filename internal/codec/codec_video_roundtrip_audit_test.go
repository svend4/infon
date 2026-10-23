package codec

import (
	"bytes"
	"testing"
)

// codecVideoAuditStep mirrors the encoder's sampling step for a frame of n bytes
// under DefaultVideoConfig (2 Mbps, 30 fps): one byte is kept every step bytes.
func codecVideoAuditStep(n int) int {
	step := n / (2_000_000 / 30 / 8)
	if step < 2 {
		step = 2
	}
	return step
}

// codecVideoAuditBlockyFrame returns a frame that is constant over every run of
// step bytes, so the encoder's downsampling loses nothing and Decode(Encode(x))
// must reproduce x exactly. Block values cycle through vals.
func codecVideoAuditBlockyFrame(n, step int, vals []byte) []byte {
	frame := make([]byte, n)
	for i := range frame {
		frame[i] = vals[(i/step)%len(vals)]
	}
	return frame
}

// The decoder must upsample with the same step the encoder sampled with; a
// floor'd ratio drifts every sample and leaves the tail of the frame gray.
func TestCodecVideoAuditRoundTripBlockyFrame(t *testing.T) {
	vals := make([]byte, 200)
	for i := range vals {
		vals[i] = byte(i + 10)
	}

	for _, name := range []string{"h264", "vp8", "vp9"} {
		for _, res := range [][2]int{{640, 480}, {320, 240}, {176, 144}} {
			c, err := codecVideoAuditCtors[name](DefaultVideoConfig(CodecType(name), res[0], res[1]))
			if err != nil {
				t.Fatalf("%s %v: constructor: %v", name, res, err)
			}
			n := res[0] * res[1] * 3 / 2
			frame := codecVideoAuditBlockyFrame(n, codecVideoAuditStep(n), vals)

			for k := 0; k < 2; k++ { // keyframe, then inter frame
				enc, err := c.Encode(frame)
				if err != nil {
					t.Fatalf("%s %v: Encode: %v", name, res, err)
				}
				dec, err := c.Decode(enc)
				if err != nil {
					t.Fatalf("%s %v: Decode: %v", name, res, err)
				}
				if !bytes.Equal(dec, frame) {
					first := 0
					for first < len(dec) && first < len(frame) && dec[first] == frame[first] {
						first++
					}
					t.Fatalf("%s %v frame %d: Decode(Encode(x)) differs from x at byte %d of %d",
						name, res, k, first, n)
				}
			}
		}
	}
}
