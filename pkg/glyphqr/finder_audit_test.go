package glyphqr

import "testing"

// mustNotPanic runs f and fails the test if it panics.
func mustNotPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	f()
}

// Non-rectangular grids (as produced by the fuzz harness's gridFromBytes when
// the byte count is not a multiple of the row width) must be rejected with an
// error, never a panic, by every decoding entry point.
func TestFinderRejectsRaggedGrids(t *testing.T) {
	F := finderTok
	ragged := map[string][][]string{
		// three finder corners present, last row shorter than the first
		"short-last-row": {{F, "", F}, {F, ""}},
		// later row shorter than the first: rotateCW indexes past its end
		"short-middle-row": {{F, "", "", F}, {""}, {F, "", "", ""}},
		// first row shorter than a later one, still canonical corners on row 0
		"long-later-row": {{F, F}, {"", "", ""}, {F, "", "", ""}},
		// the fuzz harness shape: full rows of w, trailing partial row
		"fuzz-shape": gridFromBytes([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}),
	}
	for name, g := range ragged {
		mustNotPanic(t, name+"/Orient", func() {
			if _, ok := Orient(g); ok {
				t.Errorf("%s: Orient accepted a non-rectangular grid", name)
			}
		})
		mustNotPanic(t, name+"/DecodeFramed", func() {
			if _, err := DecodeFramed(g); err == nil {
				t.Errorf("%s: DecodeFramed accepted a non-rectangular grid", name)
			}
		})
		for k := 0; k < 4; k++ {
			mustNotPanic(t, name+"/RotateCW", func() { _ = RotateCW(g, k) })
		}
	}
}

// A rectangular frame still round-trips through all four rotations, and four
// quarter-turns of a rectangular grid restore it exactly.
func TestFinderRectangularStillWorks(t *testing.T) {
	framed, err := EncodeTextFramed("audit", 4)
	if err != nil {
		t.Fatal(err)
	}
	back := RotateCW(RotateCW(framed, 1), 3)
	if len(back) != len(framed) {
		t.Fatalf("rotation changed height: %d vs %d", len(back), len(framed))
	}
	for y := range framed {
		for x := range framed[y] {
			if back[y][x] != framed[y][x] {
				t.Fatalf("rotation round trip differs at (%d,%d)", x, y)
			}
		}
	}
	for k := 0; k < 4; k++ {
		got, err := DecodeTextFramed(RotateCW(framed, k))
		if err != nil || got != "audit" {
			t.Fatalf("rot %d: got %q, %v", k, got, err)
		}
	}
}
