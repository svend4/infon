package network

import (
	"runtime"
	"testing"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// DecodeFrame reads width and height from an untrusted header. A 4-byte packet
// claiming a large frame must be rejected by the size check before the frame is
// allocated, not after.
func TestDecodeFrameRejectsTruncatedFrameBeforeAllocating(t *testing.T) {
	data := []byte{0x07, 0xD0, 0x07, 0xD0} // 2000x2000 blocks, no block data

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	frame, err := DecodeFrame(data)
	runtime.ReadMemStats(&after)

	if err == nil || frame != nil {
		t.Fatalf("DecodeFrame(4-byte header) = %v, %v; want nil, error", frame, err)
	}
	if grown := after.TotalAlloc - before.TotalAlloc; grown > 1<<20 {
		t.Errorf("DecodeFrame allocated %d bytes for a 4-byte packet; want < 1 MiB", grown)
	}
}

func patternFrame(w, h int) *terminal.Frame {
	f := terminal.NewFrame(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			f.Blocks[y][x] = terminal.Block{
				Glyph: rune('A' + (x+y)%26),
				Fg:    color.RGB{R: uint8(x), G: uint8(y), B: uint8(x >> 8)},
				Bg:    color.RGB{R: uint8(y >> 8), G: 1, B: 2},
			}
		}
	}
	return f
}

func sameFrame(a, b *terminal.Frame) bool {
	if a.Width != b.Width || a.Height != b.Height {
		return false
	}
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			if a.Blocks[y][x] != b.Blocks[y][x] {
				return false
			}
		}
	}
	return true
}

// Block positions are carried in one byte each, so the encoders must either
// reproduce a frame exactly or refuse it; silently wrapping columns or rows past
// 255 onto earlier ones is not acceptable.
func TestFrameEncodersRejectUnrepresentableDimensions(t *testing.T) {
	for _, d := range [][2]int{{256, 2}, {2, 256}, {300, 2}, {2, 300}} {
		w, h := d[0], d[1]
		frame := patternFrame(w, h)
		representable := w <= 256 && h <= 256

		enc, err := EncodeFrame(frame)
		if err != nil {
			if representable {
				t.Errorf("EncodeFrame(%dx%d): unexpected error %v", w, h, err)
			}
		} else {
			dec, err := DecodeFrame(enc)
			switch {
			case err != nil:
				t.Errorf("EncodeFrame(%dx%d) succeeded but DecodeFrame failed: %v", w, h, err)
			case !sameFrame(frame, dec):
				t.Errorf("EncodeFrame/DecodeFrame(%dx%d) did not round trip", w, h)
			}
		}

		frags, err := FragmentFrame(frame, 7)
		if err != nil {
			if representable {
				t.Errorf("FragmentFrame(%dx%d): unexpected error %v", w, h, err)
			}
			continue
		}
		got, err := AssembleFrame(frags)
		switch {
		case err != nil:
			t.Errorf("FragmentFrame(%dx%d) succeeded but AssembleFrame failed: %v", w, h, err)
		case !sameFrame(frame, got):
			t.Errorf("FragmentFrame/AssembleFrame(%dx%d) did not round trip", w, h)
		}
	}
}
