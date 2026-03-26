# Camera Support

TVCP supports both real webcams (via V4L2 on Linux) and simulated test cameras for development and testing.

## Listing Available Cameras

To see all available camera devices:

```bash
./bin/tvcp list-cameras
```

Example output:
```
🎥 Available Cameras

Found 1 camera(s):

  [0] /dev/video0: HD Pro Webcam C920
      Path: /dev/video0
      Status: Available ✓
      Common resolutions:
        - 640x480
        - 1280x720
        - 1920x1080
```

## Test Cameras (Simulators)

TVCP includes built-in test cameras that generate animated patterns. Perfect for testing without physical hardware.

### Available Patterns

- **bounce**: Animated bouncing ball with gradient colors
- **gradient**: Smooth color gradient (changes over time)
- **noise**: Random noise pattern
- **colorbar**: Standard SMPTE color bars

### Usage

```bash
# Preview with test camera
./bin/tvcp preview bounce
./bin/tvcp preview gradient

# Call with test camera
./bin/tvcp call localhost:5001 bounce 5000
```

## Real Webcams (Linux/V4L2)

### Requirements

- **Linux kernel** with V4L2 support
- **Camera drivers** loaded (usually automatic)
- **Permissions** to access `/dev/video*` devices

### Checking Permissions

If you get "permission denied" errors:

```bash
# Check current permissions
ls -l /dev/video0

# Add yourself to video group
sudo usermod -a -G video $USER

# Log out and back in for changes to take effect
```

### Using Real Cameras

Currently, real camera support is implemented but command-line selection is in development. The system will auto-detect and use available cameras.

**Coming soon:**
```bash
# Select specific camera (planned)
./bin/tvcp preview --camera 0
./bin/tvcp call <host:port> --camera 1
```

## Supported Platforms

| Platform | Status | API |
|----------|--------|-----|
| Linux | ✅ Supported | Video4Linux2 (V4L2) |
| macOS | 🚧 Planned | AVFoundation |
| Windows | 🚧 Planned | DirectShow / Media Foundation |
| BSD | 🚧 Planned | V4L2 / webcamd |

## Technical Details

### Video4Linux2 (V4L2)

TVCP uses direct V4L2 syscalls for minimal overhead and maximum compatibility:

- **No external dependencies** - Pure Go with syscalls
- **YUYV format** - YUV 4:2:2 color space
- **MMAP buffers** - Zero-copy memory-mapped I/O
- **Configurable resolution** - Defaults to 640x480

### Color Conversion

Frames are captured in YUYV format and converted to RGB:

```
YUYV (4:2:2) → RGB (24-bit)
2 pixels per 4 bytes → 2 pixels with 3 bytes each
```

This provides good quality with reasonable bandwidth.

### Buffer Management

- **4 buffers** for frame capture (reduces latency)
- **Memory-mapped I/O** (syscall.Mmap)
- **Circular queuing** - Buffers are reused efficiently

## Troubleshooting

### "No camera device found"

**Cause**: No camera connected or no permissions

**Solutions**:
```bash
# 1. Check if camera is connected
ls /dev/video*

# 2. Check permissions
ls -l /dev/video0

# 3. Load V4L2 drivers (if needed)
sudo modprobe v4l2loopback

# 4. Test with v4l2-ctl
v4l2-ctl --list-devices
```

### "Camera is busy"

**Cause**: Another application is using the camera

**Solutions**:
```bash
# Find processes using the camera
sudo lsof /dev/video0

# Close other applications (browser, Zoom, etc.)
```

### "Failed to open device"

**Cause**: Permission denied

**Solution**:
```bash
# Temporary fix
sudo chmod 666 /dev/video0

# Permanent fix
sudo usermod -a -G video $USER
# Then log out and back in
```

## Future Enhancements

- [ ] Camera selection via `--camera` flag
- [ ] Resolution selection (`--resolution 1280x720`)
- [ ] FPS configuration (`--fps 30`)
- [ ] Multi-camera support (picture-in-picture)
- [ ] macOS support (AVFoundation)
- [ ] Windows support (DirectShow)
- [ ] USB camera hot-plug detection
- [ ] Camera capability detection
- [ ] MJPEG format support (higher quality)

## Examples

### Preview Real Camera (when available)
```bash
# Auto-detect and use first camera
./bin/tvcp preview

# Specific camera (coming soon)
./bin/tvcp preview --camera 0
```

### Video Call with Real Camera
```bash
# Terminal 1: Start call with camera 0
./bin/tvcp call localhost:5001 --camera 0

# Terminal 2: Answer with camera 1
./bin/tvcp call localhost:5000 --camera 1
```

### Test Without Hardware
```bash
# Use simulated cameras
./bin/tvcp preview bounce
./bin/tvcp call localhost:5001 gradient 5000
```

## Related Documentation

- [PREVIEW.md](PREVIEW.md) - Live preview guide
- [CALL.md](CALL.md) - Video calling guide
- [README.md](README.md) - Project overview
