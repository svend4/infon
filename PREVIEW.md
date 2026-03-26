# Live Video Preview

Real-time video streaming in your terminal! This demonstrates the core video encoding/rendering pipeline at 10-15 FPS.

## 🎬 Quick Start

```bash
# Build the project
make build

# Start live preview with default pattern (bouncing ball)
./bin/tvcp preview

# Try different patterns
./bin/tvcp preview gradient
./bin/tvcp preview noise
./bin/tvcp preview colorbar
```

Press **Ctrl+C** to stop.

## 🎨 Available Patterns

### bounce (default)
Animated bouncing ball with color gradients. Best for demonstrating smooth motion.

```bash
./bin/tvcp preview bounce
```

### gradient
Flowing color gradients. Shows color encoding quality.

```bash
./bin/tvcp preview gradient
```

### noise
Random noise pattern (TV static). Stress test for the encoder.

```bash
./bin/tvcp preview noise
```

### colorbar
Standard SMPTE color bars. Good for testing terminal color accuracy.

```bash
./bin/tvcp preview colorbar
```

## 📊 What You're Seeing

**The pipeline in action:**

```
1. Test Camera (640×480)
   ↓ 15 FPS
2. Frame Capture
   ↓
3. BABE Encoder (2×2 pixel blocks)
   ↓ k-means clustering
4. Unicode Glyphs + RGB565 Colors
   ↓
5. ANSI Escape Codes
   ↓
6. Terminal (80×24 chars = 160×48 effective pixels)
```

**Performance:**
- Resolution: 80×24 characters (3,840 subpixels)
- FPS: 15 frames/second
- Encoding time: ~5-10ms per frame
- Bandwidth: ~300 KB/s (if transmitted)

## 🔬 Technical Details

### Frame Timing
```go
frameDuration := 1000ms / 15 FPS = 66.6ms per frame
```

Each iteration:
1. Read frame from camera (~0ms, it's simulated)
2. Convert to terminal blocks (~5-10ms)
3. Render ANSI codes (<1ms)
4. Display in terminal (<1ms)
5. Wait for next frame

### Color Encoding
Each 2×2 pixel block becomes:
- 1 Unicode character (16 quadrant patterns)
- 2 RGB colors (foreground + background)
- Total: ~36 bits of data

For 80×24 terminal = 1,920 blocks = ~8.6 KB per frame uncompressed.

### Why 15 FPS?

Terminal rendering is fast, but has limits:
- 10-15 FPS feels smooth for video calls
- Higher FPS = more bandwidth needed
- 15 FPS is standard for low-bitrate video

In production, we can adaptively adjust based on network conditions.

## 🎯 Next Steps

This preview demonstrates:
- ✅ Real-time encoding pipeline
- ✅ Frame timing and synchronization
- ✅ ANSI terminal rendering
- ✅ Multiple test patterns

**What's still needed for Phase 1 MVP:**
- [ ] Actual camera capture (currently simulated)
- [ ] Network transport (UDP)
- [ ] Audio encoding (G.722)
- [ ] Yggdrasil P2P connection
- [ ] Two-way communication

## 🐛 Troubleshooting

### Colors look wrong
Make sure your terminal supports 24-bit True Color:
```bash
echo $COLORTERM
# Should output: truecolor
```

### Low FPS / stuttering
This is expected on slow terminals. The encoding itself is fast (~10ms).

### Preview doesn't start
Make sure you built the project:
```bash
make clean && make build
```

## 💡 For Developers

### Adding New Patterns

Edit `internal/device/camera_simulator.go` and add a new generate function:

```go
func (c *TestCamera) generateMyPattern() image.Image {
    img := image.NewRGBA(image.Rect(0, 0, c.width, c.height))

    // Your pattern logic here

    return img
}
```

Then update `Read()` to handle the new pattern name.

### Adjusting FPS

In `cmd/tvcp/preview.go`:
```go
fps := 30.0  // Change this value
```

### Changing Terminal Size

In `cmd/tvcp/preview.go`:
```go
width := 120   // characters
height := 40   // characters
```

## 🎥 Real Camera Support (Future)

To capture from actual camera, we'll use:

**Option 1: FFmpeg (cross-platform)**
```bash
ffmpeg -i /dev/video0 -f rawvideo -pix_fmt rgb24 -s 640x480 -
```

**Option 2: Go OpenCV (gocv)**
```go
import "gocv.io/x/gocv"

webcam, _ := gocv.VideoCaptureDevice(0)
defer webcam.Close()
```

For now, the simulator is perfect for testing!

---

**See also:** [Demo Guide](DEMO.md) | [Business Plan](tvcp-business-plan.md)
