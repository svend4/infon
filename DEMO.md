# TVCP Proof-of-Concept Demo

This demo showcases the core technology behind TVCP: converting images to Unicode block characters with ANSI True Color.

## 🎨 What's Working

The prototype includes:
- ✅ **Unicode quadrant blocks** — 16 patterns (▀▄▌▐▖▗▘▝ etc.)
- ✅ **24-bit True Color** — Full RGB color support via ANSI escape codes
- ✅ **Image-to-blocks converter** — k-means clustering for optimal glyph selection
- ✅ **Terminal renderer** — ANSI escape sequence output
- ✅ **Test image generator** — Colorful test patterns

## 🚀 Quick Start

### 1. Build the Project

```bash
make build
```

### 2. Generate a Test Image

```bash
./bin/tvcp generate test_pattern.png
```

This creates a 640×480 test image with gradients and geometric shapes.

### 3. Display in Terminal

```bash
./bin/tvcp demo test_pattern.png
```

You should see the image rendered in your terminal using Unicode blocks!

## 📸 Try With Your Own Images

```bash
# Works with PNG, JPEG, GIF
./bin/tvcp demo photo.jpg
./bin/tvcp demo screenshot.png
./bin/tvcp demo avatar.gif
```

## 🎯 How It Works

### Image Conversion Pipeline

```
1. Load image → image.Image (Go standard library)
   ↓
2. Resize to terminal dimensions (80×24 chars = 160×48 pixels)
   ↓
3. For each 2×2 pixel block:
   - Extract 4 pixels (TL, TR, BL, BR)
   - Run k-means clustering (k=2) by luminance
   - Assign pixels to foreground/background groups
   - Calculate average colors for each group
   - Select matching Unicode glyph
   ↓
4. Render with ANSI escape codes:
   \x1b[38;2;{R};{G};{B}m  ← foreground color
   \x1b[48;2;{R};{G};{B}m  ← background color
   {glyph}                  ← Unicode character
```

### Example: Single Block Encoding

```
Input 2×2 pixels:          Output:
┌──────┬──────┐
│ dark │ dark │           Glyph: ▄ (bottom half)
├──────┼──────┤           Fg: RGB(200,200,200) ← average of light pixels
│light │light │           Bg: RGB(50,50,50)    ← average of dark pixels
└──────┴──────┘
                           ANSI: \x1b[38;2;200;200;200m\x1b[48;2;50;50;50m▄
```

## 🧪 Testing

Run the test suite:

```bash
# All tests
go test ./...

# Specific package
go test -v ./internal/codec/glyphs
go test -v ./pkg/color
```

### Test Coverage

```bash
go test -cover ./...
```

Current coverage:
- `internal/codec/glyphs`: 100%
- `pkg/color`: 100%

## 🎨 Terminal Compatibility

The demo works best with terminals supporting:
- ✅ **True Color (24-bit)** — iTerm2, WezTerm, kitty, GNOME Terminal, Windows Terminal
- ⚠️ **256 colors** — PuTTY, older xterm (degraded quality)
- ❌ **16 colors** — cmd.exe, very old terminals (not recommended)

### Check Your Terminal

```bash
echo $COLORTERM
# Should output: truecolor or 24bit
```

## 📊 Performance

Rendering a 640×480 image (80×24 terminal):
- **Conversion time:** ~5-10ms
- **Render time:** <1ms
- **Output size:** ~20KB ANSI codes

## 🔬 Code Structure

```
internal/codec/
├── glyphs/
│   ├── glyphs.go       ← 16 Unicode quadrant definitions
│   └── glyphs_test.go  ← Unit tests
└── babe/
    └── converter.go    ← Image-to-blocks converter

pkg/
├── color/
│   ├── color.go        ← RGB, RGB565, ANSI escape codes
│   └── color_test.go   ← Unit tests
└── terminal/
    └── renderer.go     ← Frame buffer & ANSI rendering

cmd/tvcp/
├── main.go             ← CLI entry point
├── demo.go             ← Demo command
└── generate_test_image.go ← Test image generator
```

## 🎯 Next Steps

This proof-of-concept demonstrates the feasibility of terminal-based video rendering. Next:

### Phase 1A: Video Stream (Current Goal)
- [ ] Capture frames from webcam (using ffmpeg or gocv)
- [ ] Real-time encoding at 10-15 FPS
- [ ] Display video stream in terminal

### Phase 1B: Network Transport
- [ ] Send encoded frames over UDP
- [ ] Basic congestion control
- [ ] Frame synchronization

### Phase 1C: Two-Way Communication
- [ ] Bidirectional video stream
- [ ] Audio integration (G.722 codec)
- [ ] Yggdrasil P2P connection

## 🐛 Known Limitations

Current demo:
- Static images only (no video yet)
- Fixed terminal size (80×24)
- No camera capture (needs ffmpeg/gocv)
- No network transport
- Simple 2-means clustering (could be optimized)

## 💡 Try It!

```bash
# Generate and view in one go
./bin/tvcp generate demo.png && ./bin/tvcp demo demo.png

# Try with different images
curl -o cat.jpg https://placekitten.com/640/480
./bin/tvcp demo cat.jpg

# Or use any image from your system
./bin/tvcp demo ~/Pictures/photo.jpg
```

## 🎉 Success Criteria

If you can see:
- ✓ Colors in the terminal output
- ✓ Recognizable shapes/patterns
- ✓ Smooth gradients (no harsh banding)

Then the core technology works! 🚀

---

**Next:** [Business Plan](tvcp-business-plan.md) | [Technical Appendix](tvcp-appendix.md)
