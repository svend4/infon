package glyphqr

import "testing"

// A framed code must decode from ANY of the four rotations (auto-orientation).
func TestFramedRoundTripAllRotations(t *testing.T) {
	msg := "gate://cat#A5"
	framed, err := EncodeTextFramed(msg, 8)
	if err != nil {
		t.Fatal(err)
	}
	for k := 0; k < 4; k++ {
		g := RotateCW(framed, k)
		rot, ok := Orient(g)
		if !ok {
			t.Fatalf("rot %d: no orientation found", k*90)
		}
		// applying rot to the received grid must restore the canonical frame
		if (rot+k)%4 != 0 {
			t.Fatalf("rot %d: Orient returned %d (not the inverse)", k*90, rot)
		}
		got, err := DecodeTextFramed(g)
		if err != nil {
			t.Fatalf("rot %d: %v", k*90, err)
		}
		if got != msg {
			t.Fatalf("rot %d: got %q want %q", k*90, got, msg)
		}
	}
}

// Interior cell errors are still RS-corrected after auto-orientation.
func TestFramedSurvivesInteriorErrors(t *testing.T) {
	msg := "tangram-number-2000003"
	framed, err := EncodeTextFramed(msg, 12)
	if err != nil {
		t.Fatal(err)
	}
	framed[2][2] = finderTok // corrupt two interior cells (not corner finders)
	framed[3][3] = finderTok
	g := RotateCW(framed, 2) // present it upside-down
	got, err := DecodeTextFramed(g)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != msg {
		t.Fatalf("got %q want %q", got, msg)
	}
}

func TestDecodeFramedRejectsUnframed(t *testing.T) {
	if _, err := DecodeFramed([][]string{{""}}); err == nil {
		t.Fatal("expected error on a grid with no finder pattern")
	}
}
