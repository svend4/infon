# BABE Codec

Bi-Level Adaptive Block Encoding — custom video codec for terminal rendering.

## Responsibilities

- Encode video frames to .babe format
- Decode .babe frames to terminal blocks
- Delta encoding between frames
- RLE compression
- Huffman coding

## Format Specification

Each block (2×2 pixels):
- 4 bits: glyph pattern index (0-15)
- 16 bits: foreground color RGB565
- 16 bits: background color RGB565
- Total: 36 bits per block

## Files

- `encoder.go` — Frame to .babe encoder
- `decoder.go` — .babe to terminal blocks decoder
- `delta.go` — Inter-frame delta encoding
- `rle.go` — Run-length encoding
- `format.go` — Binary format definitions
