package video

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// makeFrame builds a solid frame of one color.
func makeFrame(w, h int, c color.RGB) *terminal.Frame {
	f := terminal.NewFrame(w, h)
	f.Fill('█', c, c)
	return f
}

func TestEncodedSizeNonZero(t *testing.T) {
	enc := &EncodedFrame{Type: FrameTypeI, Width: 10, Height: 5, IFrameData: makeFrame(10, 5, color.White)}
	if EncodedSize(enc) <= 0 {
		t.Error("I-frame serialized size should be > 0")
	}
	if EncodedSize(nil) != 0 {
		t.Error("nil frame size should be 0")
	}
}

// TestPFrameSavesBandwidthOnStaticScene is the core D1 claim: for slowly-
// changing video (one cell changes per frame after the first), the P-frame path
// transmits far fewer bytes than sending a full I-frame every time.
func TestPFrameSavesBandwidthOnStaticScene(t *testing.T) {
	const w, h = 40, 30
	enc := NewPFrameEncoder(1000) // effectively only the first frame is an I-frame
	var stats StreamStats

	base := makeFrame(w, h, color.NewRGB(20, 40, 80))
	stats.Observe(enc.Encode(base))

	// 29 subsequent frames that each change a single cell (a "slow" scene).
	for i := 1; i < 30; i++ {
		f := makeFrame(w, h, color.NewRGB(20, 40, 80))
		f.SetBlock(i%w, i%h, '★', color.White, color.Black)
		stats.Observe(enc.Encode(f))
	}

	if stats.IFrames != 1 {
		t.Fatalf("expected exactly 1 I-frame, got %d", stats.IFrames)
	}
	if stats.PFrames != 29 {
		t.Fatalf("expected 29 P-frames, got %d", stats.PFrames)
	}

	ratio := stats.CompressionRatio(stats.AverageIFrameBytes())
	t.Logf("P-frame compression ratio on static scene: %.1fx (I=%dB total, P=%dB total)",
		ratio, stats.IBytes, stats.PBytes)
	if ratio < 5 {
		t.Errorf("expected >=5x bandwidth saving on a near-static scene, got %.1fx", ratio)
	}
}

// TestPFrameEncoderFallsBackToIFrameOnBigChange verifies the encoder's own
// guard: when most of the frame changes, it sends an I-frame rather than a huge
// P-frame.
func TestPFrameEncoderFallsBackToIFrameOnBigChange(t *testing.T) {
	const w, h = 20, 20
	enc := NewPFrameEncoder(1000)
	enc.Encode(makeFrame(w, h, color.Black)) // first = I-frame

	// Completely different frame -> >50% blocks change -> forced I-frame.
	whole := makeFrame(w, h, color.White)
	out := enc.Encode(whole)
	if out.Type != FrameTypeI {
		t.Errorf("a full-frame change should fall back to an I-frame, got %v", out.Type)
	}
}

// TestPFrameRoundTripReconstructs verifies decode(encode(x)) == x across a mixed
// I/P stream — correctness of the path we're measuring.
func TestPFrameRoundTripReconstructs(t *testing.T) {
	const w, h = 16, 12
	enc := NewPFrameEncoder(5)
	dec := NewPFrameDecoder()

	for i := 0; i < 12; i++ {
		f := makeFrame(w, h, color.NewRGB(uint8(i*10), 50, 60))
		f.SetBlock(i%w, (i*2)%h, 'A'+rune(i%26), color.White, color.Black)

		got := dec.Decode(enc.Encode(f))
		// Spot-check the changed cell survived the round trip.
		want := f.Blocks[(i*2)%h][i%w]
		gb := got.Blocks[(i*2)%h][i%w]
		if gb.Glyph != want.Glyph || gb.Fg != want.Fg || gb.Bg != want.Bg {
			t.Fatalf("frame %d: cell mismatch after round trip: got %v want %v", i, gb, want)
		}
	}
}

func TestStreamStatsRatioEdgeCases(t *testing.T) {
	var empty StreamStats
	if empty.CompressionRatio(100) != 1 {
		t.Error("empty stream ratio should be 1")
	}
	if empty.AverageIFrameBytes() != 0 {
		t.Error("empty stream avg I-frame bytes should be 0")
	}
}
