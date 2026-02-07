# Getting Started with TVCP

Welcome to TVCP - Terminal Video Communication Platform! This guide will help you get started with P2P audio+video calls.

## Quick Start (5 Minutes)

### 1. Build TVCP

```bash
# Clone the repository
git clone https://github.com/svend4/infon
cd infon

# Build
make build

# Verify installation
./bin/tvcp version
```

### 2. Test Video

```bash
# Try the demo with a static image
./bin/tvcp demo test_pattern.png

# Test live video preview
./bin/tvcp preview bounce
```

You should see animated video in your terminal!

### 3. Test Audio

```bash
# Test audio generation
./bin/tvcp audio-test
```

You should see audio chunks being generated (test tones).

### 4. Local Video Call

Open two terminals:

**Terminal 1 (Alice):**
```bash
./bin/tvcp call localhost:5001 gradient 5000
```

**Terminal 2 (Bob):**
```bash
./bin/tvcp call localhost:5000 bounce 5001
```

You should see:
- Split-screen video (remote on top, local on bottom)
- Real-time statistics
- Audio transmission indicators

Press `Ctrl+C` to end the call.

## Next Steps

### Setup Yggdrasil for P2P

For real P2P calls over the internet, install Yggdrasil:

**Ubuntu/Debian:**
```bash
# Install Yggdrasil
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 6FF19A7F
echo 'deb https://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/ debian yggdrasil' | \
  sudo tee /etc/apt/sources.list.d/yggdrasil.list
sudo apt update
sudo apt install yggdrasil

# Start daemon
sudo systemctl enable yggdrasil
sudo systemctl start yggdrasil

# Check status
./bin/tvcp yggdrasil
```

See [YGGDRASIL.md](YGGDRASIL.md) for detailed setup.

### Add Contacts

```bash
# Get your address
./bin/tvcp yggdrasil

# Add a friend
./bin/tvcp contacts add alice 200:1234:5678:90ab:cdef:1234:5678:90ab

# List contacts
./bin/tvcp contacts list

# Call by name
./bin/tvcp call alice
```

## Understanding the Interface

### During a Call

```
┌─────────────────────────────────────┐
│         Remote Video                │
│  (What they're sending you)         │
│                                     │
└─────────────────────────────────────┘
────────────────────────────────────────
┌─────────────────────────────────────┐
│         Local Preview               │
│  (What you're sending them)         │
│                                     │
└─────────────────────────────────────┘

[Call] Video: 75/68 (15.0/13.6 FPS) | Audio: 250/242 | Loss: 0.2% | Time: 5s
```

**What the stats mean:**
- `Video: 75/68` - Sent 75 frames, received 68
- `(15.0/13.6 FPS)` - Send/receive frame rates
- `Audio: 250/242` - Sent 250 chunks, received 242
- `Loss: 0.2%` - Packet loss percentage
- `Time: 5s` - Call duration

### End of Call Statistics

```
✓ Call ended
Duration: 10.0s

Video:
  Sent: 151 frames (15.1 FPS)
  Received: 135 frames (13.5 FPS)

Audio:
  Sent: 499 chunks (49.9 chunks/s)
  Received: 482 chunks (48.2 chunks/s)

Network Quality:
  Packets received: 2150
  Packets lost: 15 (0.69%)
  Out of order: 12
  Duplicates: 0
  Retransmissions: 8
  NACKs sent: 3
```

## Terminal Requirements

### Minimum Requirements
- 256-color terminal support
- UTF-8 encoding
- 80×24 minimum size (recommended: 120×40)

### Recommended Terminals
- ✅ **iTerm2** (macOS) - Excellent support
- ✅ **Terminal.app** (macOS) - Good support
- ✅ **GNOME Terminal** (Linux) - Excellent support
- ✅ **Konsole** (Linux) - Excellent support
- ✅ **Windows Terminal** (Windows) - Good support
- ⚠️ **PuTTY** - Limited color support
- ❌ **Classic cmd.exe** - Not supported

### Testing Your Terminal

```bash
# Test 256 colors
./bin/tvcp demo test_pattern.png

# Test Unicode blocks
./bin/tvcp preview colorbar

# If you see clear, colorful output, your terminal is good!
```

## Bandwidth Recommendations

### Network Requirements

| Connection Type | Quality | Notes |
|----------------|---------|-------|
| **Wired Ethernet** | Excellent | Best experience |
| **WiFi 5GHz** | Excellent | Recommended |
| **WiFi 2.4GHz** | Good | May have occasional drops |
| **4G/LTE** | Good | Works well |
| **3G** | Fair | High latency possible |
| **Satellite** | Poor | High latency (500ms+) |

### Bandwidth Usage

- **Video**: ~350 KB/s (2.8 Mbps)
- **Audio**: ~32 KB/s (256 kbps)
- **Total**: ~382 KB/s (3.056 Mbps)

Compare to:
- Zoom: 1.8 Mbps minimum (5× more)
- Skype: 1.5 Mbps typical (4× more)

## Troubleshooting

### "Connection timeout"

**Problem**: Can't connect to remote peer

**Solutions:**
1. Check firewall allows UDP port 5000
2. Verify both peers are running
3. Check Yggdrasil is connected (`tvcp yggdrasil`)
4. Try local test first (`localhost:5001`)

### "No video received"

**Problem**: Call connects but no video appears

**Solutions:**
1. Check network stats for high packet loss
2. Try wired connection instead of WiFi
3. Reduce distance to WiFi router
4. Check if firewall is blocking UDP

### "Choppy video"

**Problem**: Video stutters or freezes

**Solutions:**
1. Check packet loss (`Loss: X%` in stats)
2. Close other network-heavy applications
3. Use wired Ethernet if possible
4. Check CPU usage (`top` or `htop`)

### Terminal doesn't show colors

**Problem**: Output is black/white or garbled

**Solutions:**
1. Verify terminal supports 256 colors
2. Check UTF-8 encoding is enabled
3. Try a different terminal emulator
4. Update terminal software

## Advanced Usage

### Custom Video Patterns

```bash
# Available patterns
./bin/tvcp preview bounce     # Bouncing ball
./bin/tvcp preview gradient   # Color gradient
./bin/tvcp preview noise      # Random noise
./bin/tvcp preview colorbar   # SMPTE bars
```

### One-Way Streaming

```bash
# Terminal 1: Receiver
./bin/tvcp receive 5000

# Terminal 2: Sender
./bin/tvcp send localhost:5000 bounce
```

### Network Statistics

For detailed network analysis, watch the live stats during a call:

```
[Call] Video: 45/43 (15.0/14.3 FPS) | Audio: 150/148 | Loss: 1.2% | Time: 3s
```

- Loss < 1%: Excellent
- Loss 1-3%: Good
- Loss 3-10%: Fair (may have quality issues)
- Loss > 10%: Poor (consider switching networks)

## Learning More

### Documentation

- [README.md](README.md) - Project overview
- [CAMERAS.md](CAMERAS.md) - Camera support
- [AUDIO.md](AUDIO.md) - Audio system
- [YGGDRASIL.md](YGGDRASIL.md) - P2P networking
- [LOSS_RECOVERY.md](LOSS_RECOVERY.md) - Network resilience
- [CHANGELOG.md](CHANGELOG.md) - Version history

### Getting Help

- 📖 Check documentation in this repository
- 🐛 Report issues: https://github.com/svend4/infon/issues
- 💬 Discussions: https://github.com/svend4/infon/discussions

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## What's Next?

After you're comfortable with the basics:

1. **Setup P2P calling** with Yggdrasil mesh network
2. **Add contacts** for easy calling
3. **Experiment with patterns** and settings
4. **Share your address** with friends for real P2P calls
5. **Contribute** improvements or bug fixes!

---

**Welcome to the future of terminal communication!** 🚀📞🎥
