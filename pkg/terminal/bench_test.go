package terminal

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
)

// fillGradient writes a distinct color per cell so Render can't trivially
// coalesce everything (a realistic worst-ish case).
func fillGradient(f *Frame) {
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			c := color.NewRGB(uint8(x*3), uint8(y*7), uint8((x+y)*2))
			f.SetBlock(x, y, '█', c, color.Black)
		}
	}
}

// BenchmarkFullRender measures a complete repaint (the baseline cost).
func BenchmarkFullRender(b *testing.B) {
	f := NewFrame(80, 24)
	fillGradient(f)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Render()
	}
}

// BenchmarkDiffRenderNoChange measures the diff renderer when nothing changed —
// this should be dramatically cheaper than a full repaint (the whole point).
func BenchmarkDiffRenderNoChange(b *testing.B) {
	f := NewFrame(80, 24)
	fillGradient(f)
	d := NewDiffRenderer()
	_ = d.Render(f) // prime
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Render(f)
	}
}

// BenchmarkDiffRenderSmallChange measures the diff renderer when one cell
// changed per frame — the realistic interactive/animation case.
func BenchmarkDiffRenderSmallChange(b *testing.B) {
	f := NewFrame(80, 24)
	fillGradient(f)
	d := NewDiffRenderer()
	_ = d.Render(f)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.SetBlock(i%f.Width, (i/f.Width)%f.Height, 'X', color.White, color.Black)
		_ = d.Render(f)
	}
}
