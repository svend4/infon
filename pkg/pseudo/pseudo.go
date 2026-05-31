// Package pseudo turns the compact, text-friendly "pseudo-image" descriptions
// that ANY ordinary (non-diffusion) language model can emit into real graphics:
// a terminal.Frame for the live renderer, or an image.Image for the BABE video
// path. It is the answer to "let a plain chat model paint" — the model speaks a
// few tokens of JSON (a coarse color grid, a glyph map, a list of sigils, or
// vector shapes) and this package deterministically expands it into a picture.
//
// Formats (all producible by a text model with no image head):
//
//   - grid    : a coarse grid of named colors -> bilinear "pseudo-diffusion"
//   - pixels  : the same grid rendered crisp (mosaic / pixel art)
//   - glyphs  : a character grid + a char->color palette (colored ASCII/Unicode art)
//   - sigils  : named pictographs (sun, moon, star, mountain...) placed on a canvas
//   - vector  : the full draw-DSL (pkg/scene): rects, discs, gradients, text
//   - sketch  : the weak-model sketch format (pkg/sketch): named scene shapes
//
// "Pseudo-diffusion" is the grid path: a 12x8-ish color seed is upscaled and
// blurred into a smooth, painterly raster — visually diffusion-like, but with no
// diffusion model, no GPU, and fully deterministic.
package pseudo

import (
	"encoding/json"
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/scene"
	"github.com/svend4/infon/pkg/sketch"
	"github.com/svend4/infon/pkg/terminal"
)

// Format names a pseudo-image encoding.
type Format string

const (
	FormatGrid   Format = "grid"
	FormatPixels Format = "pixels"
	FormatGlyphs Format = "glyphs"
	FormatSigils Format = "sigils"
	FormatVector Format = "vector"
	FormatSketch Format = "sketch"
	FormatMixed  Format = "mixed"
)

// Spec is the unified request a text model fills in. It picks one Format and
// supplies the matching payload; Cols/Rows are the target terminal size.
type Spec struct {
	Format  Format          `json:"format"`
	Cols    int             `json:"cols,omitempty"`
	Rows    int             `json:"rows,omitempty"`
	Title   string          `json:"title,omitempty"`
	Grid    *Grid           `json:"grid,omitempty"`
	Pixels  *Grid           `json:"pixels,omitempty"`
	Glyphs  *GlyphArt       `json:"glyphs,omitempty"`
	Sigils  *SigilScene     `json:"sigils,omitempty"`
	Vector  json.RawMessage `json:"vector,omitempty"`
	Sketch  json.RawMessage `json:"sketch,omitempty"`
	Mixed   *Mixed          `json:"mixed,omitempty"`
	Palette string          `json:"palette,omitempty"` // preset: dawn|dusk|neon|mono|forest|ocean
}

// Canvas clamps keep a misbehaving model from blowing up the frame.
const (
	defCols, defRows = 64, 32
	minCols, minRows = 4, 2
	maxCols, maxRows = 400, 200
)

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (s Spec) cols() int {
	if s.Cols == 0 {
		return defCols
	}
	return clampi(s.Cols, minCols, maxCols)
}

func (s Spec) rows() int {
	if s.Rows == 0 {
		return defRows
	}
	return clampi(s.Rows, minRows, maxRows)
}

// Frame renders the spec to a terminal.Frame for the live renderer.
func (s Spec) Frame() (*terminal.Frame, error) {
	c, r := s.cols(), s.rows()
	pal := presetPalette(s.Palette)
	switch s.Format {
	case FormatGlyphs:
		if s.Glyphs == nil {
			return nil, fmt.Errorf("pseudo: glyphs format with no glyphs payload")
		}
		s.Glyphs.pal = pal
		return s.Glyphs.frame(c, r), nil
	case FormatSigils:
		if s.Sigils == nil {
			return nil, fmt.Errorf("pseudo: sigils format with no sigils payload")
		}
		s.Sigils.pal = pal
		return s.Sigils.frame(c, r), nil
	case FormatMixed:
		if s.Mixed == nil {
			return nil, fmt.Errorf("pseudo: mixed format with no mixed payload")
		}
		s.Mixed.pal = pal
		return s.Mixed.frame(c, r), nil
	case FormatVector:
		var sc scene.Scene
		if err := json.Unmarshal(nonNil(s.Vector), &sc); err != nil {
			return nil, fmt.Errorf("pseudo: vector: %w", err)
		}
		if sc.Width == 0 {
			sc.Width = c
		}
		if sc.Height == 0 {
			sc.Height = r
		}
		return scene.RenderScene(&sc), nil
	case FormatSketch:
		var sk sketch.Sketch
		if err := json.Unmarshal(nonNil(s.Sketch), &sk); err != nil {
			return nil, fmt.Errorf("pseudo: sketch: %w", err)
		}
		sc := sketch.ToScene(sk, c, r)
		return scene.RenderScene(&sc), nil
	case FormatPixels:
		g := s.gridPayload()
		if g == nil {
			return nil, fmt.Errorf("pseudo: pixels format with no grid payload")
		}
		g.pal = pal
		return g.frameNearest(c, r), nil
	case FormatGrid, "":
		g := s.gridPayload()
		if g == nil {
			return nil, fmt.Errorf("pseudo: grid format with no grid payload")
		}
		g.pal = pal
		return g.frameSmooth(c, r), nil
	default:
		return nil, fmt.Errorf("pseudo: unknown format %q", s.Format)
	}
}

// Image renders the spec to an RGBA raster of the given pixel size, suitable for
// the BABE -> terminal video path (downsampled later by the renderer).
func (s Spec) Image(pxW, pxH int) (image.Image, error) {
	pxW = clampi(pxW, 8, 4096)
	pxH = clampi(pxH, 8, 4096)
	pal := presetPalette(s.Palette)
	switch s.Format {
	case FormatGrid, "":
		g := s.gridPayload()
		if g == nil {
			return nil, fmt.Errorf("pseudo: grid format with no grid payload")
		}
		g.pal = pal
		return g.diffuse(pxW, pxH), nil
	case FormatPixels:
		g := s.gridPayload()
		if g == nil {
			return nil, fmt.Errorf("pseudo: pixels format with no grid payload")
		}
		g.pal = pal
		return g.mosaic(pxW, pxH), nil
	default:
		f, err := s.Frame()
		if err != nil {
			return nil, err
		}
		return RasterizeFrame(f, pxW, pxH), nil
	}
}

func (s Spec) gridPayload() *Grid {
	if s.Grid != nil {
		return s.Grid
	}
	return s.Pixels
}

// Render parses a pseudo-image spec document and renders it to a Frame.
func Render(doc []byte) (*terminal.Frame, error) {
	var s Spec
	if err := json.Unmarshal(doc, &s); err != nil {
		return nil, err
	}
	return s.Frame()
}

func nonNil(r json.RawMessage) json.RawMessage {
	if len(r) == 0 {
		return json.RawMessage("{}")
	}
	return r
}

// --- color tokens -----------------------------------------------------------

func mk(r, g, b uint8) color.RGB { return color.RGB{R: r, G: g, B: b} }

// palette mirrors pkg/sketch's named colors (plus a few extras) so models can
// use the same friendly names across every format.
var palette = map[string]color.RGB{
	"black": mk(10, 10, 14), "white": mk(240, 240, 245), "gray": mk(130, 130, 140), "grey": mk(130, 130, 140),
	"red": mk(230, 70, 70), "orange": mk(240, 140, 60), "orangered": mk(240, 95, 55),
	"yellow": mk(250, 220, 110), "gold": mk(245, 200, 90), "amber": mk(235, 170, 70),
	"green": mk(90, 200, 110), "darkgreen": mk(32, 110, 62), "teal": mk(70, 200, 180),
	"blue": mk(80, 150, 235), "navy": mk(28, 38, 88), "cyan": mk(120, 220, 235), "skyblue": mk(120, 185, 235),
	"purple": mk(150, 110, 220), "pink": mk(235, 130, 170), "brown": mk(120, 80, 50), "dusk": mk(120, 80, 140),
	"sand": mk(220, 200, 150), "slate": mk(70, 80, 100), "mint": mk(150, 230, 190), "coral": mk(240, 120, 100),
}

// Color resolves a token to an RGB: a palette name, "#rrggbb"/"#rgb", or "r,g,b".
func Color(tok string, def color.RGB) color.RGB {
	t := strings.ToLower(strings.TrimSpace(tok))
	if t == "" {
		return def
	}
	if c, ok := palette[t]; ok {
		return c
	}
	if strings.HasPrefix(t, "#") {
		if c, ok := parseHex(t); ok {
			return c
		}
		return def
	}
	if strings.Contains(t, ",") {
		parts := strings.Split(t, ",")
		if len(parts) == 3 {
			r, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			g, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			b, e3 := strconv.Atoi(strings.TrimSpace(parts[2]))
			if e1 == nil && e2 == nil && e3 == nil {
				return color.NewRGB(byteOf(r), byteOf(g), byteOf(b))
			}
		}
	}
	return def
}

func parseHex(t string) (color.RGB, bool) {
	h := strings.TrimPrefix(t, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return color.RGB{}, false
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return color.RGB{}, false
	}
	return color.RGB{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v)}, true
}

func byteOf(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
