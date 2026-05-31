package babe

import (
	"image"
	gocolor "image/color"
	"testing"

	"github.com/svend4/infon/pkg/color"
)

// benchImage builds a representative gradient+noise image to encode.
func benchImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, gocolor.RGBA{
				R: uint8((x * 255) / w),
				G: uint8((y * 255) / h),
				B: uint8((x ^ y) & 0xFF),
				A: 255,
			})
		}
	}
	return img
}

// Benchmark each render mode at a typical source size -> 80x24 terminal, so the
// relative cost of the A-series modes is measurable (roadmap D6).
func benchmarkMode(b *testing.B, mode RenderMode) {
	img := benchImage(320, 240)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ImageToFrameMode(img, 80, 24, mode)
	}
}

func BenchmarkModeQuadrant(b *testing.B)   { benchmarkMode(b, ModeQuadrant) }
func BenchmarkModePerceptual(b *testing.B) { benchmarkMode(b, ModePerceptual) }
func BenchmarkModeOptimal(b *testing.B)    { benchmarkMode(b, ModeOptimal) }
func BenchmarkModeHalfBlock(b *testing.B)  { benchmarkMode(b, ModeHalfBlock) }
func BenchmarkModeSextant(b *testing.B)    { benchmarkMode(b, ModeSextant) }
func BenchmarkModeBraille(b *testing.B)    { benchmarkMode(b, ModeBraille) }

// Benchmark the per-block encoders directly (the inner loop of the A-series).
func benchmarkEncoder(b *testing.B, enc func([4]color.RGB, [4]bool) (rune, color.RGB, color.RGB)) {
	px := [4]color.RGB{
		{R: 200, G: 50, B: 50}, {R: 50, G: 200, B: 50},
		{R: 50, G: 50, B: 200}, {R: 200, G: 200, B: 50},
	}
	valid := [4]bool{true, true, true, true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = enc(px, valid)
	}
}

func BenchmarkEncodeBlock(b *testing.B)           { benchmarkEncoder(b, EncodeBlock) }
func BenchmarkEncodeBlockPerceptual(b *testing.B) { benchmarkEncoder(b, EncodeBlockPerceptual) }
func BenchmarkEncodeBlockOptimal(b *testing.B)    { benchmarkEncoder(b, EncodeBlockOptimal) }
