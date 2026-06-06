package glyphqr

import "testing"

// gridFromBytes builds an arbitrary glyph-QR grid from fuzz bytes.
func gridFromBytes(data []byte) [][]string {
	if len(data) == 0 {
		return nil
	}
	w := 6 + int(data[0])%8
	var grid [][]string
	var row []string
	for _, b := range data {
		row = append(row, cellToken(int(b)))
		if len(row) == w {
			grid = append(grid, row)
			row = nil
		}
	}
	if len(row) > 0 {
		grid = append(grid, row)
	}
	return grid
}

// The decoders must never panic on arbitrary input — they may only return data
// or an error.
func FuzzDecode(f *testing.F) {
	f.Add([]byte("hello glyph-qr"))
	f.Add([]byte{0xA5, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Decode(gridFromBytes(data))
	})
}

func FuzzDecodeFramed(f *testing.F) {
	f.Add([]byte("frame me"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeFramed(gridFromBytes(data))
	})
}
