package glyphqr_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/svend4/infon/pkg/glyphqr"
)

func TestRSBytesRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for _, n := range []int{0, 1, 50, 223, 224, 600, 2000} {
		data := make([]byte, n)
		r.Read(data)
		w := glyphqr.RSEncodeBytes(data, 16)
		got, err := glyphqr.RSDecodeBytes(w, 16)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("len %d round-trip failed: err=%v", n, err)
		}
	}
}

// A contiguous burst (a torn-out chunk) is recovered thanks to interleaving.
func TestRSBytesBurst(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	data := make([]byte, 1500) // ~7 blocks at ecc=16
	r.Read(data)
	ecc := 16
	w := glyphqr.RSEncodeBytes(data, ecc)
	nb := len(w) / 255      // blocks are inferred from the stream length now
	burst := nb * (ecc / 2) // within the interleaved budget
	off := 0                // burst starts at the very front (over the length) to prove it is protected
	for i := off; i < off+burst && i < len(w); i++ {
		w[i] ^= 0xFF // wipe a contiguous run
	}
	got, err := glyphqr.RSDecodeBytes(w, ecc)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("burst of %d bytes (%d blocks) not recovered: err=%v", burst, nb, err)
	}
}
