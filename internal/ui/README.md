# UI Package

Terminal rendering and user interface.

## Responsibilities

- ANSI escape sequence rendering
- Terminal size detection
- Frame buffer management
- Sixel/Kitty graphics protocol support

## Files

- `renderer.go` — ANSI terminal renderer
- `renderer_sixel.go` — Sixel protocol support
- `renderer_kitty.go` — Kitty graphics protocol support
- `framebuffer.go` — Frame buffer management
- `tty.go` — Terminal control and detection
