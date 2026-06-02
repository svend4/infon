package semvideo_test

import (
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
)

func spec(rows [][]string) pseudo.Spec {
	return pseudo.Spec{Format: pseudo.FormatMarks, Cols: len(rows[0]), Rows: len(rows),
		Marks: &pseudo.MarkArt{Rows: rows, Bg: "navy"}}
}

func frames() []pseudo.Spec {
	a := [][]string{{"full", "", ""}, {"", "tri-ul", ""}, {"", "", "full"}}
	b := [][]string{{"", "full", ""}, {"", "tri-ul", ""}, {"", "", "full"}}
	c := [][]string{{"", "", "full"}, {"", "tri-ur", ""}, {"full", "", ""}}
	return []pseudo.Spec{spec(a), spec(b), spec(c)}
}

func TestStreamRoundTrip(t *testing.T) {
	fs := frames()
	pkts, err := semvideo.Encode(fs, 12)
	if err != nil {
		t.Fatal(err)
	}
	got, err := semvideo.Decode(pkts, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(fs) {
		t.Fatalf("got %d frames, want %d", len(got), len(fs))
	}
	for i := range fs {
		if !reflect.DeepEqual(got[i].Marks.Rows, fs[i].Marks.Rows) {
			t.Errorf("frame %d rows mismatch", i)
		}
	}
}

// A burst corruption inside a delta packet is corrected; the stream still decodes.
func TestStreamSurvivesBurst(t *testing.T) {
	fs := frames()
	ecc := 16
	pkts, _ := semvideo.Encode(fs, ecc)
	p := pkts[1] // a P-frame
	for i := 6; i < 6+ecc/2 && i < len(p); i++ {
		p[i] ^= 0xFF
	}
	got, err := semvideo.Decode(pkts, ecc)
	if err != nil {
		t.Fatalf("stream did not survive burst: %v", err)
	}
	if !reflect.DeepEqual(got[2].Marks.Rows, fs[2].Marks.Rows) {
		t.Fatal("post-burst frame not recovered")
	}
}
