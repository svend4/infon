package glyphqr_test

import (
	"bytes"
	"testing"

	"github.com/svend4/infon/pkg/glyphqr"
)

// flipCell corrupts the glyph at flat index idx to a different glyph.
func flipCell(grid [][]string, idx int) {
	w := len(grid[0])
	y, x := idx/w, idx%w
	cur := grid[y][x]
	for _, alt := range []string{"█", "◤", "◢", "▌", "□"} {
		if alt != cur {
			grid[y][x] = alt
			return
		}
	}
}

func TestRoundTrip(t *testing.T) {
	for _, s := range []string{"", "cat", "TVCP-AI glyph-QR", "the quick brown fox 0123456789"} {
		g, err := glyphqr.EncodeText(s, 12)
		if err != nil {
			t.Fatalf("encode %q: %v", s, err)
		}
		got, err := glyphqr.DecodeText(g)
		if err != nil || got != s {
			t.Fatalf("round-trip %q -> %q err=%v", s, got, err)
		}
	}
}

// Corrupting up to ecc/2 distinct bytes must still decode (Reed-Solomon).
func TestCorrectsUpToBudget(t *testing.T) {
	data := []byte("address: cat  number: 3291027959...")
	ecc := 12 // corrects up to 6 byte errors
	g, err := glyphqr.Encode(data, ecc)
	if err != nil {
		t.Fatal(err)
	}
	for b := 0; b < ecc/2; b++ { // one hi-nibble cell per distinct codeword byte
		flipCell(g, 12+2*b)
	}
	got, err := glyphqr.Decode(g)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("did not recover from %d errors: err=%v", ecc/2, err)
	}
}

// A corrupted header cell is tolerated (the header is replicated 3x).
func TestHeaderRobust(t *testing.T) {
	data := []byte("hello world")
	g, err := glyphqr.Encode(data, 10)
	if err != nil {
		t.Fatal(err)
	}
	flipCell(g, 0)  // corrupt one header cell
	flipCell(g, 12) // plus one codeword byte
	got, err := glyphqr.Decode(g)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("header+1 error not tolerated: %q err=%v", got, err)
	}
}

func TestRenderNonNil(t *testing.T) {
	g, _ := glyphqr.EncodeText("x", 6)
	img := glyphqr.Render(g, 16)
	if img == nil || img.Bounds().Dx() == 0 {
		t.Fatal("render returned empty image")
	}
}
