# Yggdrasil P2P Integration

TVCP uses [Yggdrasil](https://yggdrasil-network.github.io/) for peer-to-peer mesh networking. This enables video calls without servers, accounts, or centralized infrastructure.

## Why Yggdrasil?

- **🌐 True P2P**: Direct connections between users, no middleman
- **🔒 End-to-End Encrypted**: Built-in encryption using crypto_box
- **🗺️ Mesh Routing**: Self-healing network topology
- **📡 Works Offline**: Local mesh networks without internet
- **🌍 Global Reach**: Connect to anyone on the Yggdrasil network
- **🔐 No Accounts**: Just share your IPv6 address
- **🛡️ Privacy-First**: No tracking, no metadata collection

## Installation

### Linux

**Debian/Ubuntu:**
```bash
# Add repository
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 6FF19A7F
echo 'deb https://neilalexander.s3.dualstack.eu-west-2.amazonaws.com/deb/ debian yggdrasil' | sudo tee /etc/apt/sources.list.d/yggdrasil.list

# Install
sudo apt update
sudo apt install yggdrasil

# Start and enable
sudo systemctl enable yggdrasil
sudo systemctl start yggdrasil
```

**Arch Linux:**
```bash
yay -S yggdrasil
sudo systemctl enable yggdrasil
sudo systemctl start yggdrasil
```

**From Source:**
```bash
git clone https://github.com/yggdrasil-network/yggdrasil-go
cd yggdrasil-go
./build
sudo cp yggdrasil /usr/local/bin/
sudo cp yggdrasilctl /usr/local/bin/
```

### macOS

```bash
brew install yggdrasil-go
brew services start yggdrasil-go
```

### Windows

Download from [releases](https://github.com/yggdrasil-network/yggdrasil-go/releases) and run installer.

## Configuration

### 1. Check Yggdrasil Status

```bash
tvcp yggdrasil
```

Expected output:
```
🌐 Yggdrasil Network Status

✓ Yggdrasil daemon is running

Your Yggdrasil Address:
  200:1234:5678:90ab:cdef:1234:5678:90ab

Connected Peers: 3
  • tls://1.2.3.4:443
  • tcp://5.6.7.8:12345
  • tcp://9.10.11.12:54321
```

### 2. Add Peers

Edit `/etc/yggdrasil/yggdrasil.conf` and add public peers:

```json
{
  "Peers": [
    "tls://ygg-01.paraskov.ru:443",
    "tcp://ygg.ace.ctrl-c.liu.se:9999",
    "tls://[2a01:4f9:c010:664d::1]:61995"
  ]
}
```

**Public Peers:** Find more at [publicpeers.neilalexander.dev](https://publicpeers.neilalexander.dev/)

Restart after editing:
```bash
sudo systemctl restart yggdrasil
```

### 3. Get Your Address

```bash
tvcp yggdrasil
# or
yggdrasilctl getSelf | grep address
```

Your address looks like: `200:1234:5678:90ab:cdef:1234:5678:90ab`

## Using TVCP with Yggdrasil

### Contact Management

**Add a contact:**
```bash
tvcp contacts add alice 200:1234:5678:90ab:cdef:1234:5678:90ab
```

**List contacts:**
```bash
tvcp contacts list
```

**Call a contact:**
```bash
tvcp call alice
```

### Direct Calls

You can also call directly using IPv6 addresses:

```bash
# Call with brackets (recommended)
tvcp call [200:1234:5678:90ab:cdef:1234:5678:90ab]:5000

# Call without port (uses default 5000)
tvcp call [200:1234:5678:90ab:cdef:1234:5678:90ab]
```

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                    Yggdrasil Mesh Network                   │
│                                                             │
│   ┌──────────┐         Internet/LAN          ┌──────────┐  │
│   │  Alice   │◄──────────────────────────────►│   Bob    │  │
│   │  (TVCP)  │                                │  (TVCP)  │  │
│   └──────────┘                                └──────────┘  │
│        │                                            │        │
│        │ Encrypted E2E                     Encrypted│E2E     │
│        ▼                                            ▼        │
│   ┌──────────┐         Mesh Peers         ┌──────────┐     │
│   │Yggdrasil │◄─────────────────────────►│Yggdrasil │     │
│   │  Daemon  │                            │  Daemon  │     │
│   └──────────┘                            └──────────┘     │
│        │                                            │        │
│        └────────────────┬───────────────────────────┘        │
│                         │                                    │
│                    ┌────▼────┐                              │
│                    │  Peer   │                              │
│                    │ Network │                              │
│                    └─────────┘                              │
└─────────────────────────────────────────────────────────────┘
```

1. **Yggdrasil** creates encrypted IPv6 tunnels
2. **TVCP** sends video over these tunnels using UDP
3. **Mesh routing** finds the best path automatically
4. **No servers** - all connections are peer-to-peer

## Security

### Encryption

- **Transport**: ChaCha20-Poly1305 or AES-GCM (crypto_box)
- **Key Exchange**: Curve25519
- **Signatures**: Ed25519
- **Perfect Forward Secrecy**: Yes

### Privacy

- **No Registration**: No accounts, usernames, or emails
- **No Metadata**: Yggdrasil doesn't log who talks to whom
- **Anonymous**: Share only your public IPv6 address
- **Local-First**: Works entirely within mesh network

### Threats Mitigated

✅ **Traffic Interception**: All traffic encrypted E2E
✅ **Man-in-the-Middle**: Public key cryptography
✅ **Server Compromise**: No servers to compromise
✅ **Surveillance**: No central authority
✅ **Censorship**: Mesh routing bypasses blocks

## Troubleshooting

### Yggdrasil Not Running

**Error:**
```
❌ Yggdrasil daemon is not running
```

**Fix:**
```bash
# Start Yggdrasil
sudo systemctl start yggdrasil

# Check status
sudo systemctl status yggdrasil

# View logs
sudo journalctl -u yggdrasil -f
```

### No Peers Connected

**Error:**
```
No connected peers
```

**Fix:**
1. Add public peers to `/etc/yggdrasil/yggdrasil.conf`
2. Check firewall allows outbound connections
3. Try different peers from the public list

### Cannot Reach Contact

**Symptoms:**
- Call hangs or times out
- No video received

**Debugging:**
```bash
# 1. Ping their Yggdrasil address
ping6 200:1234:5678:90ab:cdef:1234:5678:90ab

# 2. Check your own connectivity
tvcp yggdrasil

# 3. Verify peers
yggdrasilctl getPeers

# 4. Check routing
yggdrasilctl getRoutes
```

### Permission Denied

**Error:**
```
Error adding contact: permission denied
```

**Fix:**
```bash
# Check permissions on contacts file
ls -l ~/.tvcp/contacts.json

# Fix if needed
chmod 644 ~/.tvcp/contacts.json
```

## Advanced Features

### Local Mesh (No Internet)

Create a local mesh network without internet:

1. **Configure local peers:**
```json
{
  "Peers": [
    "tcp://192.168.1.10:9001",
    "tcp://192.168.1.11:9001"
  ],
  "Listen": ["tcp://0.0.0.0:9001"]
}
```

2. **Share addresses** via QR code or local file

3. **Call directly** using Yggdrasil addresses

### Multi-Hop Routing

Yggdrasil automatically routes through multiple peers if needed:

```
Alice → Peer1 → Peer2 → Peer3 → Bob
```

- Transparent to applications
- Optimizes for lowest latency
- Self-healing if peers disconnect

### Firewall Configuration

Yggdrasil works through NAT, but for best performance:

```bash
# Allow Yggdrasil traffic
sudo ufw allow 9001/tcp
sudo ufw allow 9001/udp

# For TVCP
sudo ufw allow 5000/udp
```

## Performance

| Metric | Value |
|--------|-------|
| Latency Overhead | +5-20ms (depending on hops) |
| Bandwidth | Full UDP speed (no overhead) |
| Max Peers | Unlimited |
| Connection Time | < 1 second (cached routes) |

## Comparison

| Feature | TVCP + Yggdrasil | Zoom | Skype | Jitsi |
|---------|------------------|------|-------|-------|
| P2P | ✅ Always | ❌ No | ⚠️ Sometimes | ⚠️ Optional |
| E2E Encrypted | ✅ Yes | ❌ No | ❌ No | ⚠️ Optional |
| No Servers | ✅ Yes | ❌ No | ❌ No | ❌ No |
| No Accounts | ✅ Yes | ❌ No | ❌ No | ⚠️ Optional |
| Works Offline | ✅ Yes | ❌ No | ❌ No | ❌ No |
| Anonymous | ✅ Yes | ❌ No | ❌ No | ❌ No |

## Resources

- **Yggdrasil Website**: https://yggdrasil-network.github.io/
- **Documentation**: https://yggdrasil-network.github.io/documentation.html
- **Public Peers**: https://publicpeers.neilalexander.dev/
- **GitHub**: https://github.com/yggdrasil-network/yggdrasil-go
- **Matrix Chat**: #yggdrasil:matrix.org

## Related Documentation

- [LOSS_RECOVERY.md](LOSS_RECOVERY.md) - Network resilience
- [NETWORK.md](NETWORK.md) - Transport protocol
- [README.md](README.md) - Project overview
