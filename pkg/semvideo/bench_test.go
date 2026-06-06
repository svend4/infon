package semvideo

import (
	"testing"

	"github.com/svend4/infon/pkg/pseudo"
)

func benchSpecs(n int) []pseudo.Spec {
	specs := make([]pseudo.Spec, n)
	for i := 0; i < n; i++ {
		rows := make([][]string, 8)
		cols := make([][]string, 8)
		for y := 0; y < 8; y++ {
			rows[y] = make([]string, 10)
			cols[y] = make([]string, 10)
		}
		rows[2][i%10], cols[2][i%10] = "full", "skyblue" // one moving mark
		specs[i] = pseudo.Spec{Format: pseudo.FormatMarks, Cols: 10, Rows: 8,
			Marks: &pseudo.MarkArt{Rows: rows, Colors: cols}}
	}
	return specs
}

func BenchmarkEncodeDecode(b *testing.B) {
	specs := benchSpecs(16)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk, err := Encode(specs, 8)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := Decode(pk, 8); err != nil {
			b.Fatal(err)
		}
	}
}
