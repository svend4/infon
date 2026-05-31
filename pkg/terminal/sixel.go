package terminal

import (
	"image"
	"sort"
	"strconv"
	"strings"
)

// Sixel (roadmap A5) is a true bitmap terminal protocol: pixels, not glyph
// approximation. On a Sixel-capable terminal (detected via Capability.Sixel) the
// neural raster backend's actual image can be shown pixel-perfect instead of
// being quantized into block characters.
//
// Format recap: a sixel encodes a vertical strip of 6 pixels per character as
// one byte ('?'+bitmask). Output is grouped by color: a palette is declared,
// then for each 6-row band each color emits its set sixels. This encoder uses a
// fixed-size median-style palette (quantization by RGB bucket) which keeps it
// dependency-free and good enough for terminal-scale images.

const (
	sixelStart = "\x1bP0;1;0q" // DCS, no aspect scaling, transparent bg off
	sixelEnd   = "\x1b\\"      // ST
)

// EncodeSixel converts an image to a Sixel data string. maxColors caps the
// palette (clamped to 2..256). The result is a complete DCS…ST sequence ready to
// write to a Sixel-capable terminal.
func EncodeSixel(img image.Image, maxColors int) string {
	if maxColors < 2 {
		maxColors = 2
	}
	if maxColors > 256 {
		maxColors = 256
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return sixelStart + sixelEnd
	}

	// Quantize every pixel to a palette and remember each pixel's palette index.
	pal, idx := quantize(img, maxColors)

	var out strings.Builder
	out.WriteString(sixelStart)

	// Declare the palette: "#i;2;R;G;B" with R/G/B in 0..100 (Sixel scale).
	for i, c := range pal {
		out.WriteString("#")
		out.WriteString(strconv.Itoa(i))
		out.WriteString(";2;")
		out.WriteString(strconv.Itoa(int(c[0]) * 100 / 255))
		out.WriteString(";")
		out.WriteString(strconv.Itoa(int(c[1]) * 100 / 255))
		out.WriteString(";")
		out.WriteString(strconv.Itoa(int(c[2]) * 100 / 255))
	}

	// Emit pixels in 6-row bands.
	for band := 0; band*6 < h; band++ {
		y0 := band * 6
		// For each color present in this band, emit a colored run of sixels.
		for ci := range pal {
			// Build the sixel byte for each column for this color.
			line := make([]byte, w)
			any := false
			for x := 0; x < w; x++ {
				var bits byte
				for row := 0; row < 6; row++ {
					y := y0 + row
					if y >= h {
						break
					}
					if idx[y*w+x] == ci {
						bits |= 1 << uint(row)
					}
				}
				line[x] = bits
				if bits != 0 {
					any = true
				}
			}
			if !any {
				continue // this color contributes nothing to this band
			}
			out.WriteString("#")
			out.WriteString(strconv.Itoa(ci))
			writeRunLength(&out, line)
			out.WriteString("$") // carriage return: overlay next color on same band
		}
		out.WriteString("-") // newline: advance to the next 6-row band
	}

	out.WriteString(sixelEnd)
	return out.String()
}

// writeRunLength writes sixel data bytes with RLE ("!count<char>") for runs.
func writeRunLength(out *strings.Builder, line []byte) {
	i := 0
	for i < len(line) {
		j := i + 1
		for j < len(line) && line[j] == line[i] {
			j++
		}
		runLen := j - i
		ch := byte('?') + line[i]
		if runLen >= 4 {
			out.WriteString("!")
			out.WriteString(strconv.Itoa(runLen))
			out.WriteByte(ch)
		} else {
			for k := 0; k < runLen; k++ {
				out.WriteByte(ch)
			}
		}
		i = j
	}
}

// quantize reduces the image to at most maxColors via uniform RGB bucketing,
// then maps each pixel to the nearest used bucket's average color. Returns the
// palette and a per-pixel palette index (row-major).
func quantize(img image.Image, maxColors int) ([][3]uint8, []int) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	// Choose bits/channel so (2^bits)^3 is near maxColors but reasonable.
	bits := 2
	for (1<<(bits+1))*(1<<(bits+1))*(1<<(bits+1)) <= maxColors {
		bits++
	}
	shift := 8 - bits

	type acc struct {
		r, g, bl, n uint64
	}
	buckets := map[int]*acc{}
	raw := make([]int, w*h) // bucket key per pixel
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(bl>>8)
			key := (int(r8>>shift) << (2 * bits)) | (int(g8>>shift) << bits) | int(b8>>shift)
			raw[y*w+x] = key
			a := buckets[key]
			if a == nil {
				a = &acc{}
				buckets[key] = a
			}
			a.r += uint64(r8)
			a.g += uint64(g8)
			a.bl += uint64(b8)
			a.n++
		}
	}

	// Stable palette: sort bucket keys, assign indices, compute average colors.
	keys := make([]int, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	keyToIndex := make(map[int]int, len(keys))
	pal := make([][3]uint8, 0, len(keys))
	for i, k := range keys {
		keyToIndex[k] = i
		a := buckets[k]
		pal = append(pal, [3]uint8{
			uint8(a.r / a.n), uint8(a.g / a.n), uint8(a.bl / a.n),
		})
	}

	idx := make([]int, w*h)
	for i, k := range raw {
		idx[i] = keyToIndex[k]
	}
	return pal, idx
}
