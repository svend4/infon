package lincos

import (
	"testing"

	"github.com/svend4/infon/pkg/tangram"
)

func figs(t *testing.T) map[string]tangram.Figure {
	t.Helper()
	out := map[string]tangram.Figure{}
	for _, n := range []string{"cat", "swan", "boat", "house", "fox", "turtle"} {
		f, ok := tangram.Named(n)
		if !ok {
			t.Fatalf("figure %q not in catalog", n)
		}
		out[n] = f
	}
	return out
}

// A frame must decode back to the same figure using only the bytes.
func TestRoundTripNumber(t *testing.T) {
	for name, f := range figs(t) {
		frame := Encode(f)
		got, L, err := Decode(frame)
		if err != nil {
			t.Fatalf("%s: decode: %v", name, err)
		}
		if !L.OK {
			t.Fatalf("%s: checksum not OK on clean frame", name)
		}
		if got.Number().Cmp(f.Number()) != 0 {
			t.Fatalf("%s: reconstructed Number %s != %s", name, got.Number(), f.Number())
		}
		if L.H != f.H || L.W != f.W {
			t.Fatalf("%s: dims %dx%d != %dx%d", name, L.H, L.W, f.H, f.W)
		}
	}
}

// The decoder shares no constants in advance: it learns the radix and bit width
// from the stream itself.
func TestDecoderLearnsParams(t *testing.T) {
	f, _ := tangram.Named("cat")
	_, L, err := Decode(Encode(f))
	if err != nil {
		t.Fatal(err)
	}
	if L.Radix != tangram.AlphabetSize {
		t.Fatalf("learned radix %d, want %d", L.Radix, tangram.AlphabetSize)
	}
	if L.Bits != 4 {
		t.Fatalf("bits/cell = %d, want 4 for radix 16", L.Bits)
	}
	if len(L.Cells) != f.H*f.W {
		t.Fatalf("read %d cells, want %d", len(L.Cells), f.H*f.W)
	}
}

func flip(frame []byte, bit int) []byte {
	out := make([]byte, len(frame))
	copy(out, frame)
	out[bit/8] ^= 1 << uint(7-bit%8)
	return out
}

// A single flipped payload bit must be caught by the self-stated checksum — no
// external key needed to know the frame was misread.
func TestChecksumCatchesCorruption(t *testing.T) {
	f, _ := tangram.Named("cat")
	frame := Encode(f)
	// header bits: preamble(1,2,3)=9, radix tally, H tally, W tally.
	hdr := 9 + (tangram.AlphabetSize + 1) + (f.H + 1) + (f.W + 1)
	bad := flip(frame, hdr) // first payload bit
	_, L, err := Decode(bad)
	if err == nil && L.OK {
		t.Fatal("corrupted payload decoded as clean — checksum failed to catch it")
	}
	if L.OK {
		t.Fatal("Lesson.OK true on corrupted frame")
	}
}

func TestGarbageRejected(t *testing.T) {
	if _, _, err := Decode(nil); err == nil {
		t.Fatal("nil frame should error")
	}
	if _, _, err := Decode([]byte{0, 0, 0, 0}); err == nil {
		t.Fatal("all-zero frame should fail the 1,2,3 preamble")
	}
}

func TestBitIO(t *testing.T) {
	w := &bitWriter{}
	w.tally(5)
	w.field(11, 4)
	w.tally(0)
	r := &bitReader{buf: w.bytes()}
	if n, err := r.tally(); err != nil || n != 5 {
		t.Fatalf("tally got %d,%v want 5", n, err)
	}
	if v, err := r.field(4); err != nil || v != 11 {
		t.Fatalf("field got %d,%v want 11", v, err)
	}
	if n, err := r.tally(); err != nil || n != 0 {
		t.Fatalf("tally got %d,%v want 0", n, err)
	}
}
