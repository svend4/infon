package aisource

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

// bombPNG is about a hundred bytes of PNG declaring a w x h 8-bit grayscale image
// whose pixel data stops after the first row. The 64 MiB body cap doesn't bound it:
// image.Decode allocates the whole w*h buffer before reading any pixel data.
func bombPNG(t *testing.T, w, h uint32) []byte {
	t.Helper()
	return truncatedPNG(t, w, h, 0, 8)
}

// truncatedPNG is a PNG declaring a w x h image of the given PNG color type and bit
// depth whose pixel data stops after the first row.
func truncatedPNG(t *testing.T, w, h uint32, colorType, depth byte) []byte {
	t.Helper()
	channels := map[byte]uint32{0: 1, 2: 3, 4: 2, 6: 4}[colorType]
	chunk := func(out *bytes.Buffer, typ string, data []byte) {
		_ = binary.Write(out, binary.BigEndian, uint32(len(data)))
		crc := crc32.NewIEEE()
		crc.Write([]byte(typ))
		crc.Write(data)
		out.WriteString(typ)
		out.Write(data)
		_ = binary.Write(out, binary.BigEndian, crc.Sum32())
	}
	var out bytes.Buffer
	out.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := binary.BigEndian.AppendUint32(nil, w)
	ihdr = binary.BigEndian.AppendUint32(ihdr, h)
	ihdr = append(ihdr, depth, colorType, 0, 0, 0) // deflate, no filter, no interlace
	chunk(&out, "IHDR", ihdr)
	var idat bytes.Buffer
	zw := zlib.NewWriter(&idat)
	if _, err := zw.Write(make([]byte, 1+w*channels*uint32(depth)/8)); err != nil { // one row: filter byte + pixels
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	chunk(&out, "IDAT", idat.Bytes())
	chunk(&out, "IEND", nil)
	return out.Bytes()
}

func allocatedDuring(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

func serveImage(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// A 16384x16384 PNG of ~100 bytes used to make Generate allocate 256 MiB
// (and a 100000x100000 one, 10 GB) before failing; its declared size is now
// checked against a pixel cap first.
func TestImageBackendRejectsHugeDimensions(t *testing.T) {
	bomb := bombPNG(t, 1<<14, 1<<14)
	be := NewImageBackend(ImageConfig{URL: serveImage(t, bomb).URL})
	var err error
	if n := allocatedDuring(func() { _, err = be.Generate(context.Background(), "x", 64, 64) }); n > 32<<20 {
		t.Errorf("Generate of a %d-byte PNG allocated %d bytes", len(bomb), n)
	}
	if err == nil {
		t.Error("oversized image accepted")
	}

	// An image within the cap still decodes.
	ok := NewImageBackend(ImageConfig{URL: serveImage(t, tinyPNG(t)).URL})
	if img, err := ok.Generate(context.Background(), "x", 2, 2); err != nil || img.Bounds().Dx() != 2 {
		t.Errorf("small image: %v, %v", img, err)
	}
}

// The restore API's output is decoded the same way; an oversized one falls back to
// the unrestored frame.
func TestRestoreBackendRejectsHugeDimensions(t *testing.T) {
	be := NewRestoreBackend(greenBackend{}, RestoreConfig{URL: serveImage(t, bombPNG(t, 1<<14, 1<<14)).URL})
	var dx int
	var err error
	if n := allocatedDuring(func() {
		out, e := be.Generate(context.Background(), "x", 8, 8)
		err = e
		if out != nil {
			dx = out.Bounds().Dx()
		}
	}); n > 32<<20 {
		t.Errorf("restore of an oversized PNG allocated %d bytes", n)
	}
	if err != nil || dx != 8 {
		t.Errorf("restore fallback = width %d, %v; want the 8-wide base frame", dx, err)
	}
}

// The cap counts decoded bytes, not pixels: a 16-bit RGBA pixel takes 8 bytes, so
// a 4096x4096 one (128 MiB decoded) is refused while the same size in 8-bit
// grayscale (16 MiB) gets past the check to the decoder.
func TestDecodeImageCapsDecodedBytes(t *testing.T) {
	wide := truncatedPNG(t, 4096, 4096, 6, 16)
	var err error
	if n := allocatedDuring(func() { _, err = decodeImage(wide) }); n > 32<<20 {
		t.Errorf("decode of a 16-bit RGBA 4096x4096 PNG allocated %d bytes before being refused", n)
	}
	if err == nil || !strings.Contains(err.Error(), "decode limit") {
		t.Errorf("16-bit RGBA 4096x4096: err = %v, want the decode limit", err)
	}

	// 2048x2048 16-bit RGBA is 32 MiB: within the limit, so the truncated data is
	// what fails, not the cap.
	if _, err := decodeImage(truncatedPNG(t, 2048, 2048, 6, 16)); err == nil || strings.Contains(err.Error(), "decode limit") {
		t.Errorf("16-bit RGBA 2048x2048: err = %v, want a decode error, not the limit", err)
	}
	if _, err := decodeImage(bombPNG(t, 4096, 4096)); err == nil || strings.Contains(err.Error(), "decode limit") {
		t.Errorf("8-bit gray 4096x4096: err = %v, want a decode error, not the limit", err)
	}
}
