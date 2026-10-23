package deltastream

import (
	"errors"
	"runtime"
	"testing"

	"github.com/svend4/infon/pkg/glyphqr"
)

func allocatedDuring(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// An unchanged frame costs two bytes on the wire, so a stream of a few hundred
// kilobytes could declare 65535 frames of 65535 bytes and make Decode build about
// 4 GiB of frames. The declared size is now checked against a cap from the header.
func TestDecodeRejectsOversizedHeader(t *testing.T) {
	raw := []byte{0xff, 0xff, 0xff, 0xff} // width 65535, 65535 frames
	raw = append(raw, make([]byte, 65535)...)
	raw = append(raw, make([]byte, 2*65534)...) // every later frame: zero deltas
	wire := glyphqr.RSEncodeBytes(raw, 4)
	var err error
	if n := allocatedDuring(func() { _, err = Decode(wire, 4) }); n > 16<<20 {
		t.Errorf("Decode of a %d-byte stream allocated %d bytes", len(wire), n)
	}
	if err == nil {
		t.Fatal("Decode accepted 65535 frames of 65535 bytes")
	}
}

// DecodeChecked runs the caller's check on the header before building frames, and
// a normal stream still round-trips through it.
func TestDecodeCheckedRunsBeforeFrames(t *testing.T) {
	a := Anim{Width: 3, Frames: [][]byte{{1, 2, 3}, {1, 9, 3}, {4, 9, 3}}}
	wire := Encode(a, 4)
	errNo := errors.New("no")
	var gotW, gotN int
	if _, err := DecodeChecked(wire, 4, func(w, n int) error { gotW, gotN = w, n; return errNo }); err != errNo {
		t.Fatalf("check error not returned: %v", err)
	}
	if gotW != 3 || gotN != 3 {
		t.Errorf("check saw width %d, %d frames; want 3, 3", gotW, gotN)
	}
	got, err := DecodeChecked(wire, 4, func(int, int) error { return nil })
	if err != nil || len(got.Frames) != 3 || got.Frames[2][0] != 4 || got.Frames[1][1] != 9 {
		t.Fatalf("round trip = %v, %v", got, err)
	}
}
