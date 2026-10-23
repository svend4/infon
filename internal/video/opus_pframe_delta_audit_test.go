package video

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// wireRoundTrip encodes f, sends it through Serialize/Deserialize as the
// network path does, and decodes it.
func wireRoundTrip(t *testing.T, enc *PFrameEncoder, dec *PFrameDecoder, f *terminal.Frame) (*EncodedFrame, *terminal.Frame) {
	t.Helper()
	encoded := enc.Encode(f)
	wire := DeserializeEncodedFrame(SerializeEncodedFrame(encoded))
	if wire == nil {
		t.Fatalf("DeserializeEncodedFrame rejected a frame produced by the encoder (type %v)", encoded.Type)
	}
	return encoded, dec.Decode(wire)
}

func requireSameFrame(t *testing.T, got, want *terminal.Frame) {
	t.Helper()
	if got.Width != want.Width || got.Height != want.Height {
		t.Fatalf("decoded frame is %dx%d, want %dx%d", got.Width, got.Height, want.Width, want.Height)
	}
	for y := 0; y < want.Height; y++ {
		for x := 0; x < want.Width; x++ {
			if g, w := got.Blocks[y][x], want.Blocks[y][x]; !blocksEqual(g, w) {
				t.Fatalf("cell (%d,%d) = %v, want %v", x, y, g, w)
			}
		}
	}
}

// A frame whose size differs from the previous one cannot be described by a
// delta; the encoder must send an I-frame so the decoder adopts the new size.
func TestOpusPframeDimensionChangeSendsIFrame(t *testing.T) {
	for _, tc := range []struct{ w0, h0, w1, h1 int }{
		{4, 3, 8, 6}, // grow
		{8, 6, 4, 3}, // shrink
		{6, 4, 4, 6}, // same area, different shape
	} {
		enc := NewPFrameEncoder(1000)
		dec := NewPFrameDecoder()
		wireRoundTrip(t, enc, dec, makeFrame(tc.w0, tc.h0, color.Black))

		next := makeFrame(tc.w1, tc.h1, color.Black)
		encoded, got := wireRoundTrip(t, enc, dec, next)
		if encoded.Type != FrameTypeI {
			t.Errorf("%dx%d -> %dx%d: encoder sent frame type %v, want an I-frame",
				tc.w0, tc.h0, tc.w1, tc.h1, encoded.Type)
			continue
		}
		requireSameFrame(t, got, next)
	}
}

// A P-frame whose delta count exceeds the 16-bit count field must not lose
// deltas on the wire: the decoder must reconstruct the frame exactly.
func TestOpusPframeLargeDeltaCountSurvivesWire(t *testing.T) {
	const w, h = 400, 400 // 160000 cells: half is 80000 > 65535
	const changed = 70000 // below the 50% I-frame threshold, above MaxUint16

	enc := NewPFrameEncoder(1000)
	dec := NewPFrameDecoder()
	wireRoundTrip(t, enc, dec, makeFrame(w, h, color.Black))

	next := makeFrame(w, h, color.Black)
	n := 0
	for y := 0; y < h && n < changed; y++ {
		for x := 0; x < w && n < changed; x++ {
			next.SetBlock(x, y, '#', color.White, color.Black)
			n++
		}
	}

	_, got := wireRoundTrip(t, enc, dec, next)
	requireSameFrame(t, got, next)
}
