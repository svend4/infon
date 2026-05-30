# TVCP — Terminal Video Communication Platform

> 🎥 **Video calls inside your terminal. 9× less bandwidth than Zoom. No GUI required.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/svend4/infon)](https://goreportcard.com/report/github.com/svend4/infon)
[![Status: Alpha MVP](https://img.shields.io/badge/Status-Alpha%20MVP-orange)](https://github.com/svend4/infon)

**TVCP** is the first video communication platform designed to work entirely inside a text terminal, using Unicode block elements and ANSI colors to render video at ultra-low bandwidth.

## 🤖 AI integration (tvcp-ai/1)

This repo also includes an **AI layer**: an open **`tvcp-ai/1`** format (JSON over HTTP) that lets **any neural network** be a partner for the terminal — drawing (draw-DSL + a simple sketch format), playing games (tic-tac-toe, Wordle, UNO), reacting with glyph cards, and acting as a video source. Swap brains by changing a URL — a reference brain (no model), local Ollama, or any OpenAI/Anthropic endpoint (demonstrated live with Claude Haiku).

See **[ai/IMPLEMENTED.md](ai/IMPLEMENTED.md)** for everything that was built, **[ai/BRAIN_PROTOCOL.md](ai/BRAIN_PROTOCOL.md)** for the spec, and **[ai/showcase.html](ai/showcase.html)** for a visual gallery. New code lives in `pkg/scene`, `pkg/sketch`, `pkg/brain`, `internal/aisource`, and the `cmd/ai*` commands; `tvcp ai` streams an AI video source.

## ⚡ Key Features

- **🚀 Ultra-low bandwidth:** 382 kbps total (vs 1.8 Mbps for Zoom)
- **📊 170 MB/hour** traffic (vs 540-1620 MB for Zoom)
- **🎧 Full audio+video** — 16 kHz voice + 15 FPS video
- **🔒 P2P mesh networking** via Yggdrasil — no servers, no accounts
- **🖥️ Headless-first** — perfect for SSH sessions and remote servers
- **🎨 True Color support** — 24-bit color in modern terminals
- **🌐 Works offline** — local mesh network without internet
- **🔐 E2E encrypted** — built-in Yggdrasil encryption
- **🪶 Minimal resources** — 256 MB RAM, runs on Raspberry Pi
- **🛡️ Packet loss recovery** — automatic retransmission (NACK)

## 🎯 Use Cases

### 🛠️ DevOps / SRE
```bash
# You're in an SSH session, incident happens
ssh production-server
# Start video call without leaving terminal
tvcp call alice@200:abc:def::1
```

### 📹 IoT Video Monitoring
Monitor cameras on cellular/satellite connection at 1/10th the bandwidth of traditional IP cameras.

### 🛰️ Satellite Internet
Video calls at $90/hour instead of $2,700+ (on Iridium/Inmarsat).

### 🔒 Privacy & Anonymity
No accounts, no servers, no metadata. Works over Tor and mesh networks.

### 🎖️ Military / Tactical Communications
Autonomous mesh network, works without infrastructure.

## 📦 Installation

### Build from Source

**Linux / macOS:**
```bash
# Clone the repository
git clone https://github.com/svend4/infon.git
cd infon

# Build the binary
make build

# Binary will be available at ./bin/tvcp
./bin/tvcp --help
```

**Windows (PowerShell):**
```powershell
# Clone the repository
git clone https://github.com/svend4/infon.git
cd infon

# Run setup script (one-click install)
.\setup.ps1

# Or build manually
go build -o bin/tvcp.exe ./cmd/tvcp

# Binary will be available at .\bin\tvcp.exe
.\bin\tvcp.exe --help
```

📖 **Platform-specific guides:**
- **Windows:** [BUILD_WINDOWS.md](BUILD_WINDOWS.md)
- **Linux/macOS:** [GETTING_STARTED.md](GETTING_STARTED.md)
- **Android (Termux):** [TERMUX_ANDROID.md](TERMUX_ANDROID.md) 📱

### Requirements
- Go 1.21 or higher
- **Linux:** ALSA audio (libasound2-dev), V4L2 for cameras
- **macOS:** CoreAudio (built-in), cameras via future implementation
- **Windows:** WASAPI (built-in), cameras via future implementation
- **Android (Termux):** Works as relay server (no camera/audio)
- Terminal with TrueColor support (recommended)

## 🚀 Quick Start

### Try the Proof-of-Concept Demo

```bash
# Clone and build
git clone https://github.com/svend4/infon.git
cd infon
make build

# Generate a test image
./bin/tvcp generate test.png

# Display it in your terminal!
./bin/tvcp demo test.png
```

📖 **[Read the full demo guide →](DEMO.md)**

### Try Live Video Preview

```bash
# Watch live animated video in your terminal!
./bin/tvcp preview

# Try different patterns
./bin/tvcp preview bounce    # bouncing ball
./bin/tvcp preview gradient  # flowing colors
./bin/tvcp preview noise     # TV static
```

🎥 **[Read the preview guide →](PREVIEW.md)**

### Stream Video Over Network

```bash
# Terminal 1: Start receiver
./bin/tvcp receive 5000

# Terminal 2: Send video stream
./bin/tvcp send localhost:5000 bounce

# Works over LAN too!
./bin/tvcp send 192.168.1.100:5000 gradient
```

🌐 **[Read the network guide →](NETWORK.md)**

### Make Video+Audio Calls

```bash
# Make a P2P call to Yggdrasil address
./bin/tvcp call [200:abc:def::1]:5000

# Or call over local network
./bin/tvcp call localhost:5000

# Receive calls on port 5000
./bin/tvcp call --listen 5000

# Full audio+video communication with:
#   - 15 FPS video at ~350 KB/s
#   - 16 kHz mono audio at 32 KB/s
#   - Automatic packet loss recovery
#   - Real-time statistics display
```

🎧 **[Learn more about audio →](AUDIO.md)** | 📹 **[Camera setup guide →](CAMERAS.md)**

## 🏗️ Architecture

TVCP uses a custom block-based video codec (**`.babe`** format) that encodes video frames as Unicode quadrant blocks with foreground/background colors:

```
Each block = 2×2 pixels encoded as:
- 4 bits: glyph pattern (16 Unicode quadrants: ▀▄▌▐▖▗▘▝ etc.)
- 16 bits: foreground color (RGB565)
- 16 bits: background color (RGB565)
= 36 bits per block
```

**Tech stack:**
- **Video codec:** Custom `.babe` format (Bi-Level Adaptive Block Encoding)
- **Audio codec:** PCM 16 kHz mono (Opus planned)
- **Networking:** Yggdrasil P2P mesh network
- **Transport:** UDP with automatic retransmission (NACK)
- **Rendering:** ANSI escape codes + Unicode block elements

## 📚 Documentation

### Getting Started
- 🚀 [**Getting Started Guide**](GETTING_STARTED.md) — Installation, setup, and first call
- 📋 [**Changelog**](CHANGELOG.md) — Version history and release notes

### Core Features
- 📹 [**Camera Support**](CAMERAS.md) — Real webcams + test cameras (V4L2)
- 🔊 [**Audio Support**](AUDIO.md) — Voice transmission with PCM/Opus encoding
- 💬 [**Text Chat**](TEXT_CHAT.md) — Real-time messaging during calls
- 📺 [**Screen Sharing**](SCREEN_SHARING.md) — Share terminal output (logs, monitoring, builds)
- 🎬 [**Call Recording**](RECORDING.md) — Save and playback video+audio calls
- 🎥 [**Video Export**](EXPORT.md) — Export recordings to MP4/WebM format
- 🌐 [**Yggdrasil P2P Integration**](YGGDRASIL.md) — Mesh networking and contacts
- 🛡️ [**Packet Loss Recovery**](LOSS_RECOVERY.md) — Automatic retransmission and network resilience

### Guides
- 🌐 [**Network Streaming**](NETWORK.md) — Stream video over UDP
- 🎥 [**Live Video Preview**](PREVIEW.md) — Real-time video rendering
- 🎨 [**Proof-of-Concept Demo**](DEMO.md) — Static image rendering
- 🧠 [**Generative Graphics**](GENERATIVE.md) — Synthesized frames (procedural + neural hook)

### Planning & Analysis
- 📖 [Business Plan](tvcp-business-plan.md) — Full market analysis and roadmap
- 🔬 [Technical Appendix](tvcp-appendix.md) — Deep dive into algorithms and protocols
- 📋 [Repository Review](REPOSITORY_REVIEW.md) — Current status and recommendations

## 🛣️ Roadmap

### Phase 0: Preparation (Weeks 1-2) — **✓ Complete**
- [x] Create documentation
- [x] Repository setup
- [x] Setup CI/CD (GitHub Actions)
- [x] **Proof-of-concept demo** — Image-to-terminal renderer working!
- [x] **Live video preview** — Real-time streaming at 15 FPS! 🎉
- [ ] Fork [Say](https://github.com/svanichkin/say) project

### Phase 1: MVP (Weeks 3-8) — **✓ Complete**
- [x] Video capture interface and simulator
- [x] Real-time frame encoding (15 FPS)
- [x] Frame timing and synchronization
- [x] **Network transport (UDP)** — Video streaming works! 🎉
- [x] **Frame fragmentation** — Handles MTU limits
- [x] **Packet loss recovery (NACK)** — Automatic retransmission! 🎉
- [x] **V4L2 camera capture** — Real webcam support! 🎉
- [x] **Yggdrasil integration** — P2P mesh networking! 🎉
- [x] **Audio encoding (PCM)** — Voice transmission! 🎉
- [x] **Two-way audio+video calls** — Full communication! 🎉

### Phase 2: Expansion (Weeks 9-16)
- [ ] Text chat
- [ ] Screen sharing
- [ ] Call recording
- [ ] Package managers (Homebrew, apt)

### Phase 3: Production (Weeks 17-28)
- [ ] Group calls (up to 5 participants)
- [ ] Web portal
- [ ] SaaS managed relay nodes
- [ ] IoT SDK

### Phase 4: Scale (Months 7-12)
- [ ] Mobile clients (iOS/Android)
- [ ] P-frames (inter-frame compression)
- [ ] Enterprise integrations
- [ ] Hardware partnerships

## 🤝 Contributing

TVCP is in early development. Contributions are welcome!

**Areas where we need help:**
- Go development (codec, networking)
- Video encoding algorithms
- Terminal rendering optimization
- Testing on various terminals
- Documentation

## 📊 Comparison

| Feature | Zoom | WebRTC | TVCP |
|---------|------|--------|------|
| Min bandwidth | 1.8 Mbps | 1.5 Mbps | **382 kbps** |
| Video quality | 720p | 720p | **Terminal (40x30)** |
| Audio quality | 48 kHz stereo | 48 kHz | **16 kHz mono** |
| Traffic/hour | 540-1620 MB | 450-900 MB | **170 MB** |
| GUI required | ✓ | ✓ | **✗** |
| Central server | ✓ | Signaling only | **✗ (P2P)** |
| Account needed | ✓ | Varies | **✗** |
| RAM usage | 8-16 GB | 2-4 GB | **256 MB** |
| Works headless | ✗ | ✗ | **✓** |
| IoT/Embedded | ✗ | Limited | **✓** |

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Based on [Say](https://github.com/svanichkin/say) by Sergey Vanichkin
- Inspired by [.babe codec](https://github.com/svanichkin/babe)
- Powered by [Yggdrasil Network](https://yggdrasil-network.github.io/)

## 📧 Contact

- GitHub Issues: [github.com/svend4/infon/issues](https://github.com/svend4/infon/issues)
- Author: Stefan Engel (stefan.engel.de@gmail.com)

---

**Status:** 🎉 **Alpha MVP Complete** — Full audio+video P2P calling works! Try it out and report issues.
