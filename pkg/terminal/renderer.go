package terminal

import (
	"fmt"
	"strings"

	"github.com/svend4/infon/internal/codec/glyphs"
	"github.com/svend4/infon/pkg/color"
)

// Block represents a single terminal character block with colors
type Block struct {
	Glyph rune
	Fg    color.RGB // foreground color
	Bg    color.RGB // background color
}

// Frame represents a terminal frame (grid of blocks)
type Frame struct {
	Width  int
	Height int
	Blocks [][]Block
}

// NewFrame creates a new empty frame
func NewFrame(width, height int) *Frame {
	blocks := make([][]Block, height)
	for i := range blocks {
		blocks[i] = make([]Block, width)
		for j := range blocks[i] {
			blocks[i][j] = Block{
				Glyph: ' ',
				Fg:    color.Black,
				Bg:    color.Black,
			}
		}
	}
	return &Frame{
		Width:  width,
		Height: height,
		Blocks: blocks,
	}
}

// SetBlock sets a block at the given position
func (f *Frame) SetBlock(x, y int, glyph rune, fg, bg color.RGB) {
	if x >= 0 && x < f.Width && y >= 0 && y < f.Height {
		f.Blocks[y][x] = Block{
			Glyph: glyph,
			Fg:    fg,
			Bg:    bg,
		}
	}
}

// Render renders the frame to a string with ANSI escape codes
func (f *Frame) Render() string {
	var buf strings.Builder
	buf.WriteString(color.ClearScreen)

	var lastFg, lastBg color.RGB
	needsReset := false

	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			block := f.Blocks[y][x]

			// Only emit color codes if they changed
			if !needsReset || block.Fg != lastFg {
				buf.WriteString(block.Fg.FgString())
				lastFg = block.Fg
			}
			if !needsReset || block.Bg != lastBg {
				buf.WriteString(block.Bg.BgString())
				lastBg = block.Bg
			}
			needsReset = true

			buf.WriteRune(block.Glyph)
		}
		buf.WriteString(color.Reset)
		buf.WriteRune('\n')
		needsReset = false
	}

	return buf.String()
}

// RenderToTerminal renders the frame directly to stdout
func (f *Frame) RenderToTerminal() {
	fmt.Print(f.Render())
}

// Clear clears the frame to black
func (f *Frame) Clear() {
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			f.Blocks[y][x] = Block{
				Glyph: ' ',
				Fg:    color.Black,
				Bg:    color.Black,
			}
		}
	}
}

// Fill fills the frame with a single character and colors
func (f *Frame) Fill(glyph rune, fg, bg color.RGB) {
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			f.Blocks[y][x] = Block{
				Glyph: glyph,
				Fg:    fg,
				Bg:    bg,
			}
		}
	}
}

// DrawText draws text at the given position
func (f *Frame) DrawText(x, y int, text string, fg, bg color.RGB) {
	for i, ch := range text {
		if x+i < f.Width && y < f.Height {
			f.Blocks[y][x+i] = Block{
				Glyph: ch,
				Fg:    fg,
				Bg:    bg,
			}
		}
	}
}

// DrawBox draws a box using Unicode box drawing characters
func (f *Frame) DrawBox(x, y, width, height int, fg, bg color.RGB) {
	if width < 2 || height < 2 {
		return
	}

	// Top border
	f.SetBlock(x, y, '┌', fg, bg)
	for i := 1; i < width-1; i++ {
		f.SetBlock(x+i, y, '─', fg, bg)
	}
	f.SetBlock(x+width-1, y, '┐', fg, bg)

	// Sides
	for i := 1; i < height-1; i++ {
		f.SetBlock(x, y+i, '│', fg, bg)
		f.SetBlock(x+width-1, y+i, '│', fg, bg)
	}

	// Bottom border
	f.SetBlock(x, y+height-1, '└', fg, bg)
	for i := 1; i < width-1; i++ {
		f.SetBlock(x+i, y+height-1, '─', fg, bg)
	}
	f.SetBlock(x+width-1, y+height-1, '┘', fg, bg)
}

// DrawQuadrant draws a quadrant block based on 4 boolean values
func (f *Frame) DrawQuadrant(x, y int, tl, tr, bl, br bool, fg, bg color.RGB) {
	glyph := glyphs.GetGlyphFromBits(tl, tr, bl, br)
	f.SetBlock(x, y, glyph.Char, fg, bg)
}
