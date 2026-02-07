# Network Streaming

Stream video between two terminals over UDP! This demonstrates real-time video transmission over a network.

## 🌐 Quick Start

### Terminal 1: Receiver

```bash
# Start receiving on port 5000 (default)
./bin/tvcp receive

# Or specify custom port
./bin/tvcp receive 9000
```

### Terminal 2: Sender

```bash
# Send to receiver
./bin/tvcp send localhost:5000

# Send with different pattern
./bin/tvcp send localhost:5000 gradient

# Send to remote host
./bin/tvcp send 192.168.1.100:5000 bounce
```

## 📊 What's Happening

```
Sender Terminal                    Receiver Terminal
─────────────────                  ─────────────────
Camera (simulated)
    ↓
Encode to blocks
    ↓
Fragment frame (17 packets)
    ↓
UDP packets ──────────────────────→ Receive packets
                                        ↓
                                   Reassemble frame
                                        ↓
                                   Render to terminal
```

## 🔧 Technical Details

### Frame Fragmentation

Each 80×24 frame = 1,920 blocks × 11 bytes = **21,120 bytes**

This exceeds UDP MTU (1,400 bytes), so we fragment:
- Each frame → **~17 fragments**
- Each fragment ≤ 1,371 bytes
- ~124 blocks per fragment

### Packet Structure

```
┌─────────────────────────────────┐
│ Packet Header (13 bytes)        │
├─────────────────────────────────┤
│  Type: 0x01 (Frame)              │
│  Sequence: uint32                │
│  Timestamp: uint64               │
├─────────────────────────────────┤
│ Fragment Header (16 bytes)       │
├─────────────────────────────────┤
│  Frame ID: uint32                │
│  Fragment ID: uint16             │
│  Total Fragments: uint16         │
│  Width: uint16                   │
│  Height: uint16                  │
│  Start Block: uint16             │
│  Block Count: uint16             │
├─────────────────────────────────┤
│ Block Data (~1,360 bytes)        │
│  [x, y, glyph, fg, bg] × 124     │
└─────────────────────────────────┘
```

### Performance

**Test Results (localhost):**
- Sender: 15.0 FPS sustained
- Receiver: ~7-8 FPS (limited by reassembly)
- Fragments per frame: 17
- Bandwidth: ~350 KB/s

**Over real network:**
- Expected FPS: 10-12 (with latency)
- Packet loss handling: Basic (drops incomplete frames)
- Future: ARQ, FEC, adaptive bitrate

## 🛠️ Testing

### Local Network Test

```bash
# Automated test script
./test_network.sh
```

### Manual Test

```bash
# Terminal 1
./bin/tvcp receive 7777

# Terminal 2
./bin/tvcp send localhost:7777 bounce

# Watch the bouncing ball stream!
```

### Over LAN

```bash
# Find your IP
ip addr show | grep inet

# Receiver (on machine A)
./bin/tvcp receive 5000

# Sender (on machine B)
./bin/tvcp send 192.168.1.100:5000 gradient
```

## 🎯 Next Steps

Current implementation:
- ✅ UDP transport
- ✅ Frame fragmentation
- ✅ Packet serialization
- ✅ Fragment reassembly
- ⚠️ Basic error handling (drops on packet loss)

**Coming soon:**
- [ ] Packet loss recovery (ARQ)
- [ ] Forward Error Correction (FEC)
- [ ] Congestion control
- [ ] Adaptive bitrate
- [ ] Jitter buffer
- [ ] Yggdrasil P2P integration
- [ ] E2E encryption

## 🐛 Troubleshooting

### "bind: address already in use"
Port is already in use. Try different port:
```bash
./bin/tvcp receive 9999
```

### No frames received
1. Check firewall allows UDP
2. Verify receiver is running first
3. Check IP address/port match
4. Try localhost first

### Low FPS on receiver
Normal for first version. Fragment reassembly adds overhead.
Future optimizations will improve this.

### Partial frames / artifacts
Packet loss. Frame will be dropped and next complete frame displayed.

## 📖 Protocol Specification

See `internal/network/`:
- `packet.go` — Packet structure
- `frame_packet.go` — Frame encoding (deprecated, now uses fragments)
- `frame_fragmenter.go` — Frame fragmentation
- `transport.go` — UDP transport layer

## 🔬 Debugging

Enable verbose logging (future):
```bash
TVCP_DEBUG=1 ./bin/tvcp send localhost:5000
```

Check stats:
- Sender shows: Frames sent, FPS, Fragments per frame
- Receiver shows: Frames received, FPS, Sender address

## 💡 For Developers

### Changing fragment size

Edit `internal/network/frame_fragmenter.go`:
```go
const BlocksPerFragment = 100  // Smaller = more packets, less loss impact
```

### Adding reliability

Future: Implement selective ACK/NACK:
```go
// Receiver sends ACK for received fragments
// Sender retransmits missing fragments
```

---

**See also:** [Preview Guide](PREVIEW.md) | [Demo Guide](DEMO.md)
