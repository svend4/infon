# Audio Support

TVCP includes audio capture and transmission for complete voice+video calls.

## Features

- **🎤 Audio Capture**: Microphone input support
- **🔊 Audio Playback**: Speaker output
- **📦 PCM Encoding**: Uncompressed 16-bit audio
- **🌐 Network Transmission**: UDP audio packets
- **⚡ Low Latency**: Optimized for real-time communication
- **🔬 Test Audio**: Built-in tone generator for testing

## Audio Format

TVCP uses a voice-optimized audio format:

| Parameter | Value | Notes |
|-----------|-------|-------|
| Sample Rate | 16000 Hz | Optimized for voice (bandwidth efficient) |
| Channels | 1 (Mono) | Sufficient for voice calls |
| Bit Depth | 16 bits | CD-quality samples |
| Bandwidth | 32 KB/s | ~256 kbps (uncompressed PCM) |

### Why 16 kHz?

- **Voice Frequency**: Human voice is typically 300-3400 Hz
- **Nyquist**: 16 kHz covers up to 8 kHz (more than enough)
- **Bandwidth**: 50% less than 48 kHz while maintaining voice quality
- **Processing**: Lower CPU usage on embedded devices

## Commands

### List Audio Devices

```bash
tvcp list-audio
```

Shows available microphones and speakers.

### Test Audio Generation

```bash
tvcp audio-test
```

Generates test tones:
- **Sine wave**: 440 Hz (A4 note)
- **Beep pattern**: 800 Hz beeping
- **Silence**: No audio

## Audio Packet Format

Audio is transmitted in UDP packets with the following structure:

```
┌────────────────┬──────────┬──────────┬───────┬─────────────┐
│ Timestamp (8)  │ Rate (2) │ Ch (1)   │ Codec │ Samples...  │
│ milliseconds   │ Hz       │ count    │ (1)   │ 16-bit PCM  │
└────────────────┴──────────┴──────────┴───────┴─────────────┘
```

**Header:**
- Timestamp: 8 bytes (uint64) - Packet timestamp in milliseconds
- Sample Rate: 2 bytes (uint16) - Samples per second
- Channels: 1 byte (uint8) - Number of audio channels
- Codec: 1 byte (uint8) - Codec type (0=PCM, 1=Opus)
- Sample Count: 2 bytes (uint16) - Number of samples following

**Payload:**
- Samples: N × 2 bytes (int16) - Audio samples

### Packet Size

For 20ms audio chunks at 16 kHz:
- Samples: 16000 Hz × 0.02s = 320 samples
- Size: 14 bytes (header) + 640 bytes (samples) = 654 bytes
- Well under MTU (1400 bytes)

## Technical Details

### Test Audio Source

The `TestAudioSource` generates synthetic audio for testing:

```go
source := audio.NewTestAudioSource(audio.DefaultFormat())
source.Open()
source.SetTone("sine") // or "beep", "silence"

buffer := make([]int16, 320) // 20ms at 16kHz
n, _ := source.Read(buffer)
```

### Sine Wave Generation

```
sample = amplitude × sin(2π × frequency × position / sampleRate)
```

- Amplitude: 10000 (~30% of max 32767)
- Frequency: 440 Hz (A4 note)
- Position: Current sample number

### Integration with Calls

Audio packets are sent alongside video frames:

```
Video Packet (every 67ms @ 15 FPS)
Audio Packet (every 20ms @ 50 Hz)
Audio Packet (every 20ms)
Audio Packet (every 20ms)
Video Packet (every 67ms)
...
```

Audio has higher packet rate but smaller size compared to video.

## Platform Support

| Platform | Status | API | Library |
|----------|--------|-----|---------|
| Linux | ✅ **Implemented** | ALSA | github.com/yobert/alsa |
| macOS | 🚧 Planned | CoreAudio | - |
| Windows | 🚧 Planned | WASAPI | - |
| Test Mode | ✅ Ready | Synthetic tones | Built-in |

### Linux (ALSA)

ALSA (Advanced Linux Sound Architecture) support is fully implemented:

```bash
# List available audio devices
tvcp list-audio

# Make a call with real microphone/speakers
tvcp call <address>
```

**Features:**
- ✅ Device enumeration (capture and playback)
- ✅ Automatic device selection (uses first available)
- ✅ 16 kHz mono capture/playback
- ✅ 20ms buffer size (low latency)
- ✅ Thread-safe operations

**Requirements:**
- ALSA-compatible sound card
- `/dev/snd/` devices available
- Permissions to access audio devices

**Implementation Details:**
- Pure Go implementation (no CGO required)
- Uses `github.com/yobert/alsa` library
- Negotiates format: S16_LE (16-bit little-endian)
- Buffer: 320 samples period (20ms @ 16kHz)
- Automatic parameter negotiation

**Note:** On systems without ALSA devices (containers, minimal environments), the system will fall back to test audio sources.

## Bandwidth Comparison

| Codec | Sample Rate | Bandwidth | Quality |
|-------|-------------|-----------|---------|
| **TVCP PCM** | 16 kHz | 32 KB/s | Voice |
| Opus (VoIP) | 16 kHz | 12 KB/s | Voice |
| Opus (Music) | 48 kHz | 24 KB/s | Music |
| Zoom Audio | 16 kHz | ~15 KB/s | Voice |
| Skype | 16 kHz | ~10 KB/s | Voice |

TVCP currently uses uncompressed PCM. Future Opus integration will reduce bandwidth by 60%.

## Future Enhancements

- [ ] **ALSA Support** (Linux microphone/speaker)
- [ ] **Opus Codec** (60% bandwidth reduction)
- [ ] **Noise Suppression** (WebRTC NS)
- [ ] **Automatic Gain Control** (WebRTC AGC)
- [ ] **Echo Cancellation** (WebRTC AEC)
- [ ] **VAD** (Voice Activity Detection)
- [ ] **Jitter Buffer** (Audio-specific buffering)
- [ ] **CoreAudio** (macOS support)
- [ ] **WASAPI** (Windows support)

## Example Usage

### Standalone Audio Test

```bash
# Test audio generation
tvcp audio-test

# List audio devices
tvcp list-audio
```

### With Video Calls

```bash
# Full audio+video call (coming soon)
tvcp call alice --audio

# Disable audio (video only)
tvcp call bob --no-audio
```

## Troubleshooting

### No Audio Devices

**Symptom:**
```
No capture devices found.
```

**Solution:**
- Currently using test audio (real devices not yet supported)
- Real ALSA/CoreAudio/WASAPI support coming soon
- Test audio works for development and demo

### Audio Quality Issues

**Problem**: Audio sounds distorted or choppy

**Solutions:**
1. Check network packet loss: `tvcp call ... | grep Loss`
2. Reduce video quality to save bandwidth
3. Use wired connection instead of Wi-Fi
4. Check CPU usage (audio encoding is CPU-intensive)

## Related Documentation

- [YGGDRASIL.md](YGGDRASIL.md) - P2P networking
- [LOSS_RECOVERY.md](LOSS_RECOVERY.md) - Packet loss handling
- [NETWORK.md](NETWORK.md) - Transport protocol
- [README.md](README.md) - Project overview

## Integration with Video Calls

Audio is now fully integrated with the `call` command for complete voice+video communication.

### Usage

```bash
# Audio+Video call (automatic)
tvcp call alice
tvcp call localhost:5001

# Audio is enabled by default
```

### How It Works

**Parallel Transmission:**
- Video: 15 FPS (1 frame every ~67ms)
- Audio: 50 chunks/s (1 chunk every 20ms)

```
Time:   0ms    20ms   40ms   60ms   67ms   80ms
        │      │      │      │      │      │
Audio:  ▓─────►▓─────►▓─────►▓─────►      ▓─────►
Video:                        █────────────►
```

**Bandwidth:**
- Video: ~350 KB/s (compressed .babe)
- Audio: 32 KB/s (PCM)
- Total: ~382 KB/s

**Statistics:**
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
  Retransmissions: 8
```

### Technical Details

**Audio Goroutine:**
- Runs in parallel with video transmission
- 20ms chunks (320 samples @ 16 kHz)
- ~50 packets per second
- Independent from video timing

**Packet Priority:**
Audio packets are small (~654 bytes) and sent frequently, ensuring low latency voice transmission even if some packets are lost.

## Current Limitations

- **Test Audio Only**: Currently uses test tone generator (sine wave)
- **No Real Microphones**: ALSA/CoreAudio not yet implemented
- **PCM Only**: No compression (Opus coming soon)
- **No Audio Processing**: No noise suppression, AGC, or AEC

These limitations will be addressed in future releases.
