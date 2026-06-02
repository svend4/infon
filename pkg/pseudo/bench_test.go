package pseudo

import "testing"

func benchGrid() Spec {
	pal := []string{"navy", "teal", "gold", "amber", "slate", "skyblue"}
	rows := make([][]string, 8)
	for y := 0; y < 8; y++ {
		rows[y] = make([]string, 12)
		for x := 0; x < 12; x++ {
			rows[y][x] = pal[(x+y)%len(pal)]
		}
	}
	return Spec{Format: FormatGrid, Cols: 64, Rows: 36, Grid: &Grid{Rows: rows}}
}

func BenchmarkGridImage(b *testing.B) {
	s := benchGrid()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Image(320, 180); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGridFrame(b *testing.B) {
	s := benchGrid()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Frame(); err != nil {
			b.Fatal(err)
		}
	}
}
