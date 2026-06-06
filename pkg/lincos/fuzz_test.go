package lincos

import (
	"testing"

	"github.com/svend4/infon/pkg/tangram"
)

// Decode must never panic on arbitrary bytes.
func FuzzDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xFF, 0x00, 0xAA})
	if fig, ok := tangram.Named("cat"); ok {
		f.Add(Encode(fig)) // a valid frame as a seed
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = Decode(data)
	})
}

func BenchmarkEncodeDecode(b *testing.B) {
	fig, _ := tangram.Named("cat")
	frame := Encode(fig)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := Decode(frame); err != nil {
			b.Fatal(err)
		}
	}
}
