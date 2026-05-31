package terminal

import (
	"encoding/base64"
	"image"
	"strconv"
	"strings"
)

// Kitty graphics protocol (roadmap A5, second half).
//
// Like Sixel, this is a true-bitmap path: on a Kitty-capable terminal the actual
// image (e.g. the neural raster backend's output) is shown pixel-perfect instead
// of glyph approximation. The Kitty protocol transmits raw RGB/RGBA as base64
// inside APC escape sequences (ESC _ G ... ESC \), chunked to <=4096 base64
// bytes per chunk with m=1 on all but the last chunk.
//
// We send format 24 (RGB), action a=T (transmit-and-display). The result is a
// complete, ready-to-write escape sequence.

const kittyChunk = 4096 // max base64 payload per APC chunk (protocol limit)

// EncodeKitty converts an image to a Kitty graphics protocol escape sequence
// (RGB, transmit-and-display). Returns "" for an empty image.
func EncodeKitty(img image.Image) string {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return ""
	}

	// Pack pixels as tightly-interleaved RGB (3 bytes/pixel), row-major.
	rgb := make([]byte, 0, w*h*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			rgb = append(rgb, byte(r>>8), byte(g>>8), byte(bl>>8))
		}
	}
	payload := base64.StdEncoding.EncodeToString(rgb)

	var out strings.Builder
	// Chunk the base64 payload; the first chunk carries the image control keys.
	for i := 0; i < len(payload); i += kittyChunk {
		end := i + kittyChunk
		if end > len(payload) {
			end = len(payload)
		}
		chunk := payload[i:end]
		more := 0
		if end < len(payload) {
			more = 1
		}

		out.WriteString("\x1b_G")
		if i == 0 {
			// First chunk: format=24 (RGB), action=T (transmit+display), w/h.
			out.WriteString("a=T,f=24,s=")
			out.WriteString(strconv.Itoa(w))
			out.WriteString(",v=")
			out.WriteString(strconv.Itoa(h))
			out.WriteString(",m=")
			out.WriteString(strconv.Itoa(more))
		} else {
			// Continuation chunks carry only the more-data flag.
			out.WriteString("m=")
			out.WriteString(strconv.Itoa(more))
		}
		out.WriteString(";")
		out.WriteString(chunk)
		out.WriteString("\x1b\\")
	}
	return out.String()
}
