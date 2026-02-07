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

| Platform | Status | API |
|----------|--------|-----|
| Linux | 🚧 Planned | ALSA |
| macOS | 🚧 Planned | CoreAudio |
| Windows | 🚧 Planned | WASAPI |
| Test Mode | ✅ Ready | Synthetic tones |

Currently uses test audio sources. Real microphone/speaker support coming soon.

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
