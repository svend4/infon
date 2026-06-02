package semvideo_test

import (
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
)

// moving builds n constant-shape marks frames: a "full" mark stepping along the
// middle row of a 3x5 grid.
func moving(n int) []pseudo.Spec {
	out := make([]pseudo.Spec, n)
	for i := 0; i < n; i++ {
		rows := [][]string{
			{"", "", "", "", ""},
			{"", "", "", "", ""},
			{"", "", "", "", ""},
		}
		rows[1][i%5] = "full"
		out[i] = spec(rows)
	}
	return out
}

func TestBiRoundTrip(t *testing.T) {
	src := moving(5)
	pk, disp, err := semvideo.BiEncode(src, 2, 12)
	if err != nil {
		t.Fatal(err)
	}
	got, err := semvideo.BiDecode(pk, disp, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(src) {
		t.Fatalf("got %d frames, want %d", len(got), len(src))
	}
	for i := range src {
		if !reflect.DeepEqual(got[i].Marks.Rows, src[i].Marks.Rows) {
			t.Errorf("frame %d mismatch", i)
		}
	}
}

func TestBiCodingOrderDiffersFromDisplay(t *testing.T) {
	src := moving(5)
	_, disp, err := semvideo.BiEncode(src, 2, 12)
	if err != nil {
		t.Fatal(err)
	}
	// coding order K0,P2,B1,P4,B3 -> display permutation [0,2,1,4,3]
	if want := []int{0, 2, 1, 4, 3}; !reflect.DeepEqual(disp, want) {
		t.Fatalf("display perm = %v, want %v", disp, want)
	}
	identity := true
	for i, d := range disp {
		if d != i {
			identity = false
		}
	}
	if identity {
		t.Fatal("coding order should differ from display order (DTS != PTS)")
	}
}

func TestBFramesAreDisposable(t *testing.T) {
	src := moving(5)
	pk, disp, err := semvideo.BiEncode(src, 2, 12)
	if err != nil {
		t.Fatal(err)
	}
	ap, ad := semvideo.DropBFrames(pk, disp)
	got, err := semvideo.BiDecode(ap, ad, 12)
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{0, 2, 4} { // anchors survive
		if got[i].Marks == nil || !reflect.DeepEqual(got[i].Marks.Rows, src[i].Marks.Rows) {
			t.Errorf("anchor frame %d not recovered from anchors-only stream", i)
		}
	}
	for _, i := range []int{1, 3} { // dropped B-frames are simply absent
		if got[i].Marks != nil {
			t.Errorf("dropped B-frame %d should be absent", i)
		}
	}
}

func TestBiSurvivesBurstInBFrame(t *testing.T) {
	src := moving(5)
	ecc := 16
	pk, disp, err := semvideo.BiEncode(src, 2, ecc)
	if err != nil {
		t.Fatal(err)
	}
	corrupted := false
	for i := range pk {
		if pk[i][0] == semvideo.BFrame {
			p := pk[i]
			for j := 8; j < 8+ecc/2 && j < len(p); j++ {
				p[j] ^= 0xFF
			}
			corrupted = true
			break
		}
	}
	if !corrupted {
		t.Fatal("no B-frame found to corrupt")
	}
	got, err := semvideo.BiDecode(pk, disp, ecc)
	if err != nil {
		t.Fatalf("burst not corrected: %v", err)
	}
	for i := range src {
		if !reflect.DeepEqual(got[i].Marks.Rows, src[i].Marks.Rows) {
			t.Errorf("frame %d not recovered after burst", i)
		}
	}
}

func TestBiRejectsShapeChange(t *testing.T) {
	bad := moving(3)
	bad[2] = spec([][]string{{"full", ""}, {"", ""}}) // different shape
	if _, _, err := semvideo.BiEncode(bad, 2, 12); err == nil {
		t.Fatal("expected error on non-constant grid shape")
	}
}
