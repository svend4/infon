# TVCP — Terminal Video Communication Platform

> 🎥 **Video calls inside your terminal. 9× less bandwidth than Zoom. No GUI required.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/svend4/infon)](https://goreportcard.com/report/github.com/svend4/infon)
[![Status: Pre-Alpha](https://img.shields.io/badge/Status-Pre--Alpha-red)](https://github.com/svend4/infon)

**TVCP** is the first video communication platform designed to work entirely inside a text terminal, using Unicode block elements and ANSI colors to render video at ultra-low bandwidth.

## ⚡ Key Features

- **🚀 Ultra-low bandwidth:** 200 kbps minimum (vs 1.8 Mbps for Zoom)
- **📊 90 MB/hour** traffic (vs 540-1620 MB for Zoom)
- **🔒 P2P mesh networking** via Yggdrasil — no servers, no accounts
- **🖥️ Headless-first** — perfect for SSH sessions and remote servers
- **🎨 True Color support** — 24-bit color in modern terminals
- **🌐 Works offline** — local mesh network without internet
- **🔐 E2E encrypted** — built-in Yggdrasil encryption
- **🪶 Minimal resources** — 256 MB RAM, runs on Raspberry Pi

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

**Note:** TVCP is currently in pre-alpha development stage. Installation instructions will be available soon.

```bash
# Coming soon
# go install github.com/svend4/infon/cmd/tvcp@latest
```

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

### Future: Making Calls (Coming Soon)

```bash
# Start TVCP daemon
tvcp daemon

# Make a call
tvcp call [yggdrasil-address]

# Or use contact name
tvcp call "Alice/laptop"
```

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
- **Audio codec:** G.722 / Opus / Codec2
- **Networking:** Yggdrasil P2P mesh network
- **Transport:** UDP with congestion control
- **Rendering:** ANSI escape codes + Unicode block elements

## 📚 Documentation

- 🎨 [**Proof-of-Concept Demo**](DEMO.md) — Try the working prototype! ⭐
- 📖 [Business Plan](tvcp-business-plan.md) — Full market analysis and roadmap
- 🔬 [Technical Appendix](tvcp-appendix.md) — Deep dive into algorithms and protocols
- 📋 [Repository Review](REPOSITORY_REVIEW.md) — Current status and recommendations

## 🛣️ Roadmap

### Phase 0: Preparation (Weeks 1-2) — **✓ Complete**
- [x] Create documentation
- [x] Repository setup
- [x] Setup CI/CD (GitHub Actions)
- [x] **Proof-of-concept demo** — Image-to-terminal renderer working!
- [ ] Fork [Say](https://github.com/svanichkin/say) project

### Phase 1: MVP (Weeks 3-8)
- [ ] Stable 1:1 audio+video calls
- [ ] Yggdrasil integration
- [ ] Adaptive bitrate
- [ ] Auto-reconnect

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
| Min bandwidth | 1.8 Mbps | 1.5 Mbps | **200 kbps** |
| Traffic/hour | 540-1620 MB | 450-900 MB | **90 MB** |
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

**Status:** 🚧 Pre-Alpha — Documentation stage, seeking contributors and early adopters.
