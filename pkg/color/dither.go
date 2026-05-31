package color

// Dithering reduces visible banding when an image is quantized to few colors by
// spreading the quantization error spatially, so the eye averages it back into
// the original tone. At terminal resolution (few cells, 2 colors per cell) this
// is one of the cheapest, most visible quality wins.
//
// Two methods are provided:
//   - Bayer (ordered): a fast, stateless threshold matrix; deterministic and
//     parallelizable; good for live video.
//   - Floyd–Steinberg (error diffusion): higher quality, serial; good for stills.
//
// Both operate on a Field — a small RGB buffer — so they compose with the BABE
// encoder (which samples a Field of pixels) without touching image/* types.

// Field is a width×height grid of RGB pixels (row-major).
type Field struct {
	Width, Height int
	Pix           []RGB
}

// NewField allocates a black Field.
func NewField(w, h int) *Field {
	return &Field{Width: w, Height: h, Pix: make([]RGB, w*h)}
}

// At returns the pixel at (x,y); out-of-range returns black.
func (f *Field) At(x, y int) RGB {
	if x < 0 || y < 0 || x >= f.Width || y >= f.Height {
		return Black
	}
	return f.Pix[y*f.Width+x]
}

// Set writes the pixel at (x,y) (no-op if out of range).
func (f *Field) Set(x, y int, c RGB) {
	if x < 0 || y < 0 || x >= f.Width || y >= f.Height {
		return
	}
	f.Pix[y*f.Width+x] = c
}

// Quantizer maps an arbitrary color to the nearest color the output can show.
// For the terminal this is "snap to one of the cell's chosen colors" or a fixed
// palette; the dither functions call it and diffuse the resulting error.
type Quantizer func(RGB) RGB

// bayer4 is the normalized 4×4 ordered-dither threshold matrix (values 0..15).
var bayer4 = [4][4]int{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

// OrderedDither applies 4×4 Bayer dithering in place: each pixel is nudged by a
// position-dependent threshold before quantization, which breaks up flat bands.
// amount scales the perturbation (a typical value is ~32 on the 0..255 scale).
func (f *Field) OrderedDither(q Quantizer, amount float64) {
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			// Bayer threshold centered on zero in [-0.5, 0.5).
			t := (float64(bayer4[y&3][x&3])+0.5)/16.0 - 0.5
			bias := t * amount
			c := f.At(x, y)
			nudged := RGB{
				R: clampToByte(float64(c.R) + bias),
				G: clampToByte(float64(c.G) + bias),
				B: clampToByte(float64(c.B) + bias),
			}
			f.Set(x, y, q(nudged))
		}
	}
}

// FloydSteinberg applies Floyd–Steinberg error diffusion in place: each pixel is
// quantized and its error is pushed to not-yet-visited neighbors with the classic
//
//	      X   7/16
//	3/16 5/16 1/16
//
// weights, yielding smoother gradients than ordered dithering.
func (f *Field) FloydSteinberg(q Quantizer) {
	// Work in float to accumulate fractional error.
	er := make([]float64, len(f.Pix))
	eg := make([]float64, len(f.Pix))
	eb := make([]float64, len(f.Pix))

	idx := func(x, y int) int { return y*f.Width + x }
	add := func(x, y int, dr, dg, db float64) {
		if x < 0 || y < 0 || x >= f.Width || y >= f.Height {
			return
		}
		i := idx(x, y)
		er[i] += dr
		eg[i] += dg
		eb[i] += db
	}

	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			i := idx(x, y)
			c := f.At(x, y)
			want := RGB{
				R: clampToByte(float64(c.R) + er[i]),
				G: clampToByte(float64(c.G) + eg[i]),
				B: clampToByte(float64(c.B) + eb[i]),
			}
			got := q(want)
			f.Set(x, y, got)

			// Quantization error (can be negative).
			dr := float64(want.R) - float64(got.R)
			dg := float64(want.G) - float64(got.G)
			db := float64(want.B) - float64(got.B)

			add(x+1, y, dr*7/16, dg*7/16, db*7/16)
			add(x-1, y+1, dr*3/16, dg*3/16, db*3/16)
			add(x, y+1, dr*5/16, dg*5/16, db*5/16)
			add(x+1, y+1, dr*1/16, dg*1/16, db*1/16)
		}
	}
}

func clampToByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}
