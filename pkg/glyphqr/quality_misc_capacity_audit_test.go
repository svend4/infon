package glyphqr

import (
	"bytes"
	"testing"
)

// A Reed-Solomon codeword over GF(256) holds at most 255 bytes, and its length
// is stored in one header byte, so Encode must either produce a grid that
// decodes back to the data or refuse the payload. It must never hand back an
// undecodable grid.
func TestQualityMiscAudit_EncodeCapacity(t *testing.T) {
	for _, ecc := range []int{2, 10, 30} {
		for _, n := range []int{0, 1, 200, 251 - ecc, 252 - ecc, 253 - ecc, 240, 251, 252, 253} {
			if n < 0 {
				continue
			}
			data := make([]byte, n)
			for i := range data {
				data[i] = byte(i*7 + 3)
			}
			g, err := Encode(data, ecc)
			if err != nil {
				if n+2+ecc <= 255 {
					t.Errorf("Encode(%d bytes, ecc=%d) = %v, want success (codeword %d <= 255)", n, ecc, err, n+2+ecc)
				}
				continue
			}
			got, err := Decode(g)
			if err != nil || !bytes.Equal(got, data) {
				t.Errorf("Encode(%d bytes, ecc=%d) succeeded but Decode = (%d bytes, %v), want the data back", n, ecc, len(got), err)
			}
		}
	}
}
