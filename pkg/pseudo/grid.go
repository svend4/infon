package pseudo

import (
	"image"
	stdcolor "image/color"
	"math/rand"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// defGridColor is used for blank / unknown cells.
var defGridColor = color.RGB{R: 20, G: 20, B: 28}

// Grid is a coarse grid of color tokens — the smallest "image" a text model can
// emit. Rows may be ragged; they are normalized to a rectangle on render.
type Grid struct {
	Rows [][]string `json:"rows"`
}

// rect normalizes the (possibly ragged) token grid into a rectangle of RGB.
func (g *Grid) rect() (cells [][]color.RGB, w, h int) {
	h = len(g.Rows)
	for _, r := range g.Rows {
		if len(r) > w {
			w = len(r)
		}
	}
	if w == 0 || h == 0 {
		return [][]color.RGB{{defGridColor}}, 1, 1
	}
	cells = make([][]color.RGB, h)
	for y := 0; y < h; y++ {
		cells[y] = make([]color.RGB, w)
		row := g.Rows[y]
		for x := 0; x < w; x++ {
			tok := ""
			switch {
			case x < len(row):
				tok = row[x]
			case len(row) > 0:
				tok = row[len(row)-1] // extend the last color to pad ragged rows
			}
			cells[y][x] = Color(tok, defGridColor)
		}
	}
	return cells, w, h
}

// frameSmooth renders the grid bilinearly upscaled to the terminal size — the
// terminal-resolution form of pseudo-diffusion (smooth color transitions).
func (g *Grid) frameSmooth(cols, rows int) *terminal.Frame {
	cells, sw, sh := g.rect()
	up := upscaleBilinear(cells, sw, sh, cols, rows)
	return cellsToFrame(up)
}

// frameNearest renders the grid as crisp mosaic cells (pixel art).
func (g *Grid) frameNearest(cols, rows int) *terminal.Frame {
	cells, sw, sh := g.rect()
	f := terminal.NewFrame(cols, rows)
	for y := 0; y < rows; y++ {
		sy := y * sh / rows
		for x := 0; x < cols; x++ {
			sx := x * sw / cols
			c := cells[sy][sx]
			f.SetBlock(x, y, ' ', c, c)
		}
	}
	return f
}

// diffuse is the pseudo-diffusion raster: upscale the coarse seed and blur it
// into a smooth painterly image (no diffusion model, fully deterministic).
func (g *Grid) diffuse(pxW, pxH int) image.Image {
	cells, sw, sh := g.rect()
	up := upscaleBilinear(cells, sw, sh, pxW, pxH)
	rad := pxW / sw / 2
	if rad < 2 {
		rad = 2
	}
	up = blur(up, rad, 3)
	return cellsToImage(up)
}

// mosaic is the crisp pixel-art raster (nearest neighbor, no blur).
func (g *Grid) mosaic(pxW, pxH int) image.Image {
	cells, sw, sh := g.rect()
	out := make([][]color.RGB, pxH)
	for y := 0; y < pxH; y++ {
		sy := y * sh / pxH
		out[y] = make([]color.RGB, pxW)
		for x := 0; x < pxW; x++ {
			sx := x * sw / pxW
			out[y][x] = cells[sy][sx]
		}
	}
	return cellsToImage(out)
}

// DiffuseFrames produces a deterministic "denoising" animation: seeded RGB noise
// resolves into the pseudo-diffusion target over the given number of steps, with
// blur fading out as the picture sharpens. Handy for transitions and demos.
func (g *Grid) DiffuseFrames(pxW, pxH, steps int) []image.Image {
	if steps < 2 {
		steps = 2
	}
	cells, sw, sh := g.rect()
	target := upscaleBilinear(cells, sw, sh, pxW, pxH)
	maxRad := pxW / sw / 2
	if maxRad < 3 {
		maxRad = 3
	}
	target = blur(target, maxRad, 2)

	rng := rand.New(rand.NewSource(1))
	noise := make([][]color.RGB, pxH)
	for y := 0; y < pxH; y++ {
		noise[y] = make([]color.RGB, pxW)
		for x := 0; x < pxW; x++ {
			noise[y][x] = color.RGB{R: uint8(rng.Intn(256)), G: uint8(rng.Intn(256)), B: uint8(rng.Intn(256))}
		}
	}

	frames := make([]image.Image, steps)
	for i := 0; i < steps; i++ {
		alpha := float64(i) / float64(steps-1) // 0 (noise) .. 1 (target)
		blended := make([][]color.RGB, pxH)
		for y := 0; y < pxH; y++ {
			blended[y] = make([]color.RGB, pxW)
			for x := 0; x < pxW; x++ {
				blended[y][x] = noise[y][x].Blend(target[y][x], alpha)
			}
		}
		if rad := int(float64(maxRad) * (1 - alpha)); rad > 0 {
			blended = blur(blended, rad, 1)
		}
		frames[i] = cellsToImage(blended)
	}
	return frames
}

// --- sampling helpers -------------------------------------------------------

func upscaleBilinear(src [][]color.RGB, sw, sh, ow, oh int) [][]color.RGB {
	dst := make([][]color.RGB, oh)
	for oy := 0; oy < oh; oy++ {
		fy := 0.0
		if oh > 1 {
			fy = float64(oy) / float64(oh-1) * float64(sh-1)
		}
		y0 := int(fy)
		y1 := y0 + 1
		if y1 >= sh {
			y1 = sh - 1
		}
		ty := fy - float64(y0)
		dst[oy] = make([]color.RGB, ow)
		for ox := 0; ox < ow; ox++ {
			fx := 0.0
			if ow > 1 {
				fx = float64(ox) / float64(ow-1) * float64(sw-1)
			}
			x0 := int(fx)
			x1 := x0 + 1
			if x1 >= sw {
				x1 = sw - 1
			}
			tx := fx - float64(x0)
			top := src[y0][x0].Blend(src[y0][x1], tx)
			bot := src[y1][x0].Blend(src[y1][x1], tx)
			dst[oy][ox] = top.Blend(bot, ty)
		}
	}
	return dst
}

func blur(src [][]color.RGB, radius, passes int) [][]color.RGB {
	if radius < 1 || passes < 1 || len(src) == 0 {
		return src
	}
	cur := src
	for p := 0; p < passes; p++ {
		cur = blurAxis(cur, radius, true)
		cur = blurAxis(cur, radius, false)
	}
	return cur
}

func blurAxis(src [][]color.RGB, radius int, horizontal bool) [][]color.RGB {
	h := len(src)
	w := len(src[0])
	dst := make([][]color.RGB, h)
	for y := 0; y < h; y++ {
		dst[y] = make([]color.RGB, w)
		for x := 0; x < w; x++ {
			var rs, gs, bs, n int
			for d := -radius; d <= radius; d++ {
				xx, yy := x, y
				if horizontal {
					xx = x + d
				} else {
					yy = y + d
				}
				if xx < 0 || xx >= w || yy < 0 || yy >= h {
					continue
				}
				c := src[yy][xx]
				rs += int(c.R)
				gs += int(c.G)
				bs += int(c.B)
				n++
			}
			if n == 0 {
				n = 1
			}
			dst[y][x] = color.RGB{R: uint8(rs / n), G: uint8(gs / n), B: uint8(bs / n)}
		}
	}
	return dst
}

func cellsToImage(cells [][]color.RGB) *image.RGBA {
	h := len(cells)
	w := 0
	if h > 0 {
		w = len(cells[0])
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cells[y][x]
			img.SetRGBA(x, y, stdcolor.RGBA{R: c.R, G: c.G, B: c.B, A: 255})
		}
	}
	return img
}

func cellsToFrame(cells [][]color.RGB) *terminal.Frame {
	h := len(cells)
	w := 0
	if h > 0 {
		w = len(cells[0])
	}
	f := terminal.NewFrame(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cells[y][x]
			f.SetBlock(x, y, ' ', c, c)
		}
	}
	return f
}
