# Call Recording

TVCP supports recording video+audio calls to disk for later playback.

## Features

- **📹 Video recording**: Save all video frames
- **🎤 Audio recording**: Capture audio streams
- **⏱️ Synchronized playback**: Timestamps ensure perfect sync
- **💾 Custom format**: Efficient .tvcp recording format
- **📊 Metadata**: Duration, resolution, codec information
- **🎬 Playback**: Watch recorded calls in terminal

## Recording Format

TVCP uses a custom binary format (`.tvcp`) optimized for terminal video:

### File Structure

```
┌────────────────────────────────────────────┐
│ Header (34 bytes)                          │
│ - Magic: "TVCP" (4 bytes)                  │
│ - Version: 1 (2 bytes)                     │
│ - Start time: Unix timestamp (8 bytes)     │
│ - Duration: milliseconds (4 bytes)         │
│ - Frame width: blocks (2 bytes)            │
│ - Frame height: blocks (2 bytes)           │
│ - Audio rate: Hz (2 bytes)                 │
│ - Audio codec: PCM/Opus (1 byte)           │
│ - Video codec: .babe (1 byte)              │
│ - Frame count: total frames (4 bytes)      │
│ - Audio chunk count: total chunks (4 bytes)│
├────────────────────────────────────────────┤
│ Video Frames                               │
│ For each frame:                            │
│ - Timestamp: ms from start (8 bytes)       │
│ - Data length (4 bytes)                    │
│ - Frame data:                              │
│   - Width/Height (4 bytes)                 │
│   - Blocks: glyph + fg + bg (10 bytes ea) │
├────────────────────────────────────────────┤
│ Audio Chunks                               │
│ For each chunk:                            │
│ - Timestamp: ms from start (8 bytes)       │
│ - Sample count (4 bytes)                   │
│ - Samples: int16 PCM data (2 bytes ea)     │
└────────────────────────────────────────────┘
```

### Block Encoding (10 bytes per block)

```
- Glyph: Unicode character (4 bytes)
- Foreground RGB: (3 bytes)
  - R (1 byte)
  - G (1 byte)
  - B (1 byte)
- Background RGB: (3 bytes)
  - R (1 byte)
  - G (1 byte)
  - B (1 byte)
```

## Commands

### Playback

Play a recorded call:

```bash
tvcp playback recording.tvcp
```

**Example:**
```bash
$ tvcp playback recordings/call-2026-02-07.tvcp

🎬 TVCP Playback
File: recordings/call-2026-02-07.tvcp

Recording Information:
  Started: 2026-02-07 14:30:15
  Duration: 45.2 seconds
  Resolution: 40x30
  Video frames: 678 (15.0 FPS)
  Audio chunks: 2260 (16000 Hz)

▶️  Playing recording...
Duration: 45.2s | Frames: 678 | Audio chunks: 2260
Press Ctrl+C to stop

[Playback] 45.3% | 307/678 frames | 20.5s
```

### Recording (Planned)

Recording during calls will be available via flag:

```bash
# Record a call (coming soon)
tvcp call --record alice

# Specify output file (coming soon)
tvcp call --record --output my-call.tvcp alice
```

## File Size Estimation

### Video

- **Frame size**: 40×30 blocks × 10 bytes/block = 12,000 bytes = 12 KB
- **15 FPS**: 12 KB × 15 = 180 KB/s
- **1 minute**: 180 KB × 60 = 10.8 MB

### Audio

- **PCM 16 kHz mono**: 16000 samples/s × 2 bytes = 32 KB/s
- **1 minute**: 32 KB × 60 = 1.9 MB

### Total

- **Video + Audio**: ~212 KB/s
- **1 minute call**: ~12.7 MB
- **10 minute call**: ~127 MB
- **1 hour call**: ~762 MB

### With Opus Compression

When using Opus codec (optional):

- **Opus audio**: ~12 KB/s (62% reduction)
- **Total**: ~192 KB/s
- **1 minute**: ~11.5 MB
- **1 hour**: ~691 MB

## Use Cases

### 1. Call Logging

Record important calls for documentation:

```bash
# Business call
tvcp call --record --output contracts/client-meeting.tvcp alice

# Technical discussion
tvcp call --record --output debug/issue-123.tvcp bob
```

### 2. Quality Assurance

Review call quality and network performance:

```bash
# Record test call
tvcp call --record test-local.tvcp localhost:5000

# Analyze playback for issues
tvcp playback test-local.tvcp
```

### 3. Training & Education

Save example calls for training purposes:

```bash
# Record demo call
tvcp call --record demo/onboarding-call.tvcp demo-server

# Share recording with team
scp demo/onboarding-call.tvcp team-server:/recordings/
```

## Implementation Details

### Recorder API

```go
import "github.com/svend4/infon/internal/recorder"

// Create recorder
rec := recorder.NewRecorder(40, 30, 16000)

// Start recording
rec.Start("my-recording.tvcp")

// Record frames and audio
rec.RecordFrame(videoFrame)
rec.RecordAudio(audioSamples)

// Stop and save
rec.Stop()
```

### Player API

```go
import "github.com/svend4/infon/internal/recorder"

// Create player
player := recorder.NewPlayer()

// Load recording
player.Load("my-recording.tvcp")

// Play back
player.Play()
```

## Storage Recommendations

### Local Storage

```bash
# Create recordings directory
mkdir -p ~/.tvcp/recordings

# Save recordings there
tvcp call --record ~/.tvcp/recordings/call-$(date +%Y%m%d-%H%M%S).tvcp alice
```

### Automatic Cleanup

```bash
# Delete recordings older than 30 days
find ~/.tvcp/recordings -name "*.tvcp" -mtime +30 -delete

# Keep only recent 100 recordings
ls -t ~/.tvcp/recordings/*.tvcp | tail -n +101 | xargs rm
```

### Compression

TVCP recordings compress well with standard tools:

```bash
# gzip compression (~40-60% reduction)
gzip call-recording.tvcp
# Result: call-recording.tvcp.gz

# Playback compressed file
gunzip -c call-recording.tvcp.gz | tvcp playback -
```

## Current Status

**Implemented:**
- ✅ Recording format design
- ✅ Recorder infrastructure (record frames and audio)
- ✅ Player infrastructure (playback)
- ✅ Metadata handling
- ✅ File I/O

**Planned:**
- [ ] Integration with call command (--record flag)
- [ ] Automatic output filename generation
- [ ] Real-time recording indicator during calls
- [ ] Recording statistics display
- [ ] Pause/resume functionality
- [ ] Recording compression (on-the-fly)
- [ ] Playback controls (seek, speed, pause)
- [ ] Export to standard formats (MP4, WebM)

## Technical Specifications

| Aspect | Value |
|--------|-------|
| Format version | 1 |
| Magic header | 0x54564350 ("TVCP") |
| Header size | 34 bytes |
| Frame entry | 12 bytes + frame data |
| Audio entry | 12 bytes + sample data |
| Max file size | Limited only by disk |
| Endianness | Big-endian |

## Troubleshooting

### Cannot Open Recording

**Symptom:**
```
Error loading recording: invalid recording file: bad magic header
```

**Solutions:**
- Verify file is a valid .tvcp recording
- Check file is not corrupted (try `hexdump -C file.tvcp | head`)
- Ensure file was created with compatible version

### Playback Issues

**Symptom:**
```
Playback stutters or freezes
```

**Solutions:**
- Check CPU usage (playback requires real-time rendering)
- Verify terminal supports required frame rate
- Try smaller resolution recordings

### Large File Sizes

**Symptom:**
```
1-hour recording is 800+ MB
```

**Solutions:**
- Use Opus codec instead of PCM (saves ~20 KB/s)
- Reduce video frame rate (10 FPS instead of 15 FPS)
- Compress recordings with gzip after saving
- Consider selective recording (don't record entire call)

## Security & Privacy

### Encryption

Recordings are **not encrypted by default**. To protect sensitive calls:

```bash
# Encrypt recording
gpg -c call-recording.tvcp
# Creates: call-recording.tvcp.gpg

# Decrypt and play
gpg -d call-recording.tvcp.gpg | tvcp playback -
```

### Access Control

Set appropriate permissions:

```bash
# Owner read/write only
chmod 600 recordings/*.tvcp

# Prevent accidental deletion
chattr +i important-call.tvcp
```

### Auto-Delete

For privacy-sensitive calls:

```bash
# Record with auto-delete timer (1 hour)
tvcp call --record --delete-after 3600 alice
```

## Future Enhancements

- [ ] **Streaming**: Stream recordings over network
- [ ] **Highlights**: Bookmark interesting moments
- [ ] **Transcoding**: Convert to MP4, GIF, etc.
- [ ] **Cloud storage**: Upload to S3, Google Drive
- [ ] **Search**: Search recordings by metadata
- [ ] **Thumbnails**: Generate preview thumbnails
- [ ] **Annotations**: Add notes to recordings

## Examples

### Basic Recording Workflow (Planned)

```bash
# Make and record a call
tvcp call --record alice

# Recording saved to: ~/.tvcp/recordings/2026-02-07-143015.tvcp

# Play it back
tvcp playback ~/.tvcp/recordings/2026-02-07-143015.tvcp

# Share with colleague
scp ~/.tvcp/recordings/2026-02-07-143015.tvcp colleague@server:

# Colleague plays back
tvcp playback 2026-02-07-143015.tvcp
```

### Advanced Usage (Planned)

```bash
# Record with custom name
tvcp call --record --output client-demo.tvcp alice

# Record only video (no audio)
tvcp call --record --no-audio --output video-only.tvcp alice

# Record only audio (no video)
tvcp call --record --no-video --output audio-only.tvcp alice

# Record and compress on-the-fly
tvcp call --record --compress alice
```

## Format Versioning

The .tvcp format supports versioning for future compatibility:

- **Version 1** (current): Basic video+audio recording
- **Version 2** (planned): Opus compression support
- **Version 3** (planned): Multiple streams (screen share)
- **Version 4** (planned): Metadata tags and chapters

Older players can detect version and gracefully handle unsupported features.
