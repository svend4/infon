# Windows WASAPI Audio Implementation Guide

**Status**: 🚧 Infrastructure prepared, COM calls need implementation
**Platform**: Windows
**API**: WASAPI (Windows Audio Session API)

---

## Overview

WASAPI (Windows Audio Session API) is the low-level audio API in Windows Vista and later. It provides:
- Low-latency audio capture and playback
- Shared and exclusive mode
- Event-driven or polling-based operation
- Direct access to audio hardware

---

## Current Implementation Status

### ✅ Completed

1. **Basic structure** (`internal/audio/audio_windows.go`)
   - WASAPICapture and WASAPIPlayback structs
   - Ring buffer for audio data transfer
   - Device enumeration framework
   - COM initialization

2. **Platform integration**
   - Build tags (`//go:build windows`)
   - Correct LDFLAGS (`-lole32 -loleaut32`)
   - GUID definitions for WASAPI interfaces

3. **Interfaces**
   - AudioCapture and AudioPlayback interfaces implemented
   - GetFormat(), IsOpen() methods complete
   - Thread-safe with mutexes

### ⏳ Needs Implementation

1. **COM Interface Calls**
   - IMMDeviceEnumerator::GetDefaultAudioEndpoint
   - IMMDevice::Activate
   - IAudioClient::GetMixFormat
   - IAudioClient::Initialize
   - IAudioClient::GetBufferSize
   - IAudioClient::GetService
   - IAudioClient::Start/Stop
   - IAudioCaptureClient::GetBuffer/ReleaseBuffer
   - IAudioRenderClient::GetBuffer/ReleaseBuffer

2. **Audio Processing**
   - Capture loop (reading from capture client)
   - Playback loop (writing to render client)
   - Format conversion (if needed)
   - Error handling

---

## Architecture

### Data Flow

```
Capture Path:
  Microphone
      ↓
  Windows Audio Engine
      ↓
  WASAPI IAudioCaptureClient
      ↓
  Capture Loop (goroutine)
      ↓
  Ring Buffer (wasapiRingBuffer)
      ↓
  Read() method
      ↓
  Application

Playback Path:
  Application
      ↓
  Write() method
      ↓
  Ring Buffer (wasapiRingBuffer)
      ↓
  Playback Loop (goroutine)
      ↓
  WASAPI IAudioRenderClient
      ↓
  Windows Audio Engine
      ↓
  Speakers
```

### COM Interface Hierarchy

```
IUnknown
  └─ IMMDeviceEnumerator
       └─ GetDefaultAudioEndpoint() → IMMDevice
            └─ Activate() → IAudioClient
                 ├─ GetMixFormat() → WAVEFORMATEX
                 ├─ Initialize()
                 ├─ GetBufferSize()
                 ├─ GetService() → IAudioCaptureClient or IAudioRenderClient
                 └─ Start() / Stop()
```

---

## Implementation Guide

### Step 1: COM Initialization

```go
func initCOM() error {
	ret, _, _ := coInitializeEx.Call(
		0,
		0, // COINIT_MULTITHREADED
	)
	if ret != 0 && ret != 1 { // S_OK or S_FALSE
		return fmt.Errorf("failed to initialize COM: %x", ret)
	}
	return nil
}
```

✅ Already implemented

### Step 2: Get Default Audio Device

```go
// Create device enumerator
var enumerator unsafe.Pointer
ret := coCreateInstance.Call(
	uintptr(unsafe.Pointer(&CLSID_MMDeviceEnumerator)),
	0,
	1, // CLSCTX_INPROC_SERVER
	uintptr(unsafe.Pointer(&IID_IMMDeviceEnumerator)),
	uintptr(unsafe.Pointer(&enumerator)),
)

// Call IMMDeviceEnumerator::GetDefaultAudioEndpoint
// VTable offset: 3 (0=QueryInterface, 1=AddRef, 2=Release, 3=EnumAudioEndpoints, 4=GetDefaultAudioEndpoint)
vtable := *(**uintptr)(enumerator)
getDefaultAudioEndpoint := *(*func(uintptr, uint32, uint32, *unsafe.Pointer) uintptr)(unsafe.Pointer(vtable + 4*unsafe.Sizeof(uintptr(0))))

var device unsafe.Pointer
ret = getDefaultAudioEndpoint(
	uintptr(enumerator),
	0, // eRender or eCapture
	1, // eConsole
	&device,
)
```

⏳ Needs implementation

### Step 3: Activate Audio Client

```go
// Call IMMDevice::Activate
// VTable offset: 3
vtable := *(**uintptr)(device)
activate := *(*func(uintptr, unsafe.Pointer, uint32, unsafe.Pointer, *unsafe.Pointer) uintptr)(unsafe.Pointer(vtable + 3*unsafe.Sizeof(uintptr(0))))

var audioClient unsafe.Pointer
ret = activate(
	uintptr(device),
	unsafe.Pointer(&IID_IAudioClient),
	1, // CLSCTX_INPROC_SERVER
	nil,
	&audioClient,
)
```

⏳ Needs implementation

### Step 4: Get Mix Format

```go
// Call IAudioClient::GetMixFormat
// VTable offset: 8
vtable := *(**uintptr)(audioClient)
getMixFormat := *(*func(uintptr, *unsafe.Pointer) uintptr)(unsafe.Pointer(vtable + 8*unsafe.Sizeof(uintptr(0))))

var waveFormat unsafe.Pointer
ret = getMixFormat(uintptr(audioClient), &waveFormat)

// Read WAVEFORMATEX structure
wfx := (*C.WAVEFORMATEX_T)(waveFormat)
// Use wfx.nSamplesPerSec, wfx.nChannels, etc.

// Free format
coTaskMemFree.Call(uintptr(waveFormat))
```

⏳ Needs implementation

### Step 5: Initialize Audio Client

```go
// Call IAudioClient::Initialize
// VTable offset: 3
vtable := *(**uintptr)(audioClient)
initialize := *(*func(uintptr, uint32, uint32, int64, int64, unsafe.Pointer, unsafe.Pointer) uintptr)(unsafe.Pointer(vtable + 3*unsafe.Sizeof(uintptr(0))))

// Buffer duration (100ms)
bufferDuration := int64(100 * C.REFTIMES_PER_MILLISEC)

ret = initialize(
	uintptr(audioClient),
	C.AUDCLNT_SHAREMODE_SHARED,
	0, // Stream flags
	bufferDuration,
	0,
	waveFormat,
	nil,
)
```

⏳ Needs implementation

### Step 6: Get Buffer Size

```go
// Call IAudioClient::GetBufferSize
// VTable offset: 4
vtable := *(**uintptr)(audioClient)
getBufferSize := *(*func(uintptr, *uint32) uintptr)(unsafe.Pointer(vtable + 4*unsafe.Sizeof(uintptr(0))))

var bufferFrames uint32
ret = getBufferSize(uintptr(audioClient), &bufferFrames)
```

⏳ Needs implementation

### Step 7: Get Capture/Render Client

```go
// Call IAudioClient::GetService
// VTable offset: 14
vtable := *(**uintptr)(audioClient)
getService := *(*func(uintptr, unsafe.Pointer, *unsafe.Pointer) uintptr)(unsafe.Pointer(vtable + 14*unsafe.Sizeof(uintptr(0))))

var captureClient unsafe.Pointer
ret = getService(
	uintptr(audioClient),
	unsafe.Pointer(&IID_IAudioCaptureClient),
	&captureClient,
)

// Or for playback:
var renderClient unsafe.Pointer
ret = getService(
	uintptr(audioClient),
	unsafe.Pointer(&IID_IAudioRenderClient),
	&renderClient,
)
```

⏳ Needs implementation

### Step 8: Start Audio Client

```go
// Call IAudioClient::Start
// VTable offset: 5
vtable := *(**uintptr)(audioClient)
start := *(*func(uintptr) uintptr)(unsafe.Pointer(vtable + 5*unsafe.Sizeof(uintptr(0))))

ret = start(uintptr(audioClient))
```

⏳ Needs implementation

### Step 9: Capture Loop

```go
func (c *WASAPICapture) captureLoop() {
	defer close(c.doneChan)

	for {
		select {
		case <-c.stopChan:
			return
		default:
		}

		// Call IAudioCaptureClient::GetNextPacketSize
		// VTable offset: 4
		vtable := *(**uintptr)(c.captureClient)
		getNextPacketSize := *(*func(uintptr, *uint32) uintptr)(unsafe.Pointer(vtable + 4*unsafe.Sizeof(uintptr(0))))

		var packetSize uint32
		ret := getNextPacketSize(uintptr(c.captureClient), &packetSize)
		if ret != 0 || packetSize == 0 {
			time.Sleep(time.Millisecond)
			continue
		}

		// Call IAudioCaptureClient::GetBuffer
		// VTable offset: 3
		getBuffer := *(*func(uintptr, *unsafe.Pointer, *uint32, *uint32, *uint64, *uint64) uintptr)(unsafe.Pointer(vtable + 3*unsafe.Sizeof(uintptr(0))))

		var data unsafe.Pointer
		var numFrames uint32
		var flags uint32
		var devicePosition uint64
		var qpcPosition uint64

		ret = getBuffer(
			uintptr(c.captureClient),
			&data,
			&numFrames,
			&flags,
			&devicePosition,
			&qpcPosition,
		)

		if ret == 0 && data != nil {
			// Convert to int16 slice
			samples := (*[1 << 30]int16)(data)[:numFrames*uint32(c.format.Channels):numFrames*uint32(c.format.Channels)]

			// Write to ring buffer
			c.ringBuffer.Write(samples)
		}

		// Call IAudioCaptureClient::ReleaseBuffer
		// VTable offset: 5
		releaseBuffer := *(*func(uintptr, uint32) uintptr)(unsafe.Pointer(vtable + 5*unsafe.Sizeof(uintptr(0))))
		releaseBuffer(uintptr(c.captureClient), numFrames)
	}
}
```

⏳ Needs implementation

### Step 10: Playback Loop

```go
func (p *WASAPIPlayback) playbackLoop() {
	defer close(p.doneChan)

	for {
		select {
		case <-p.stopChan:
			return
		default:
		}

		// Call IAudioClient::GetCurrentPadding
		// VTable offset: 6
		vtable := *(**uintptr)(p.audioClient)
		getCurrentPadding := *(*func(uintptr, *uint32) uintptr)(unsafe.Pointer(vtable + 6*unsafe.Sizeof(uintptr(0))))

		var padding uint32
		ret := getCurrentPadding(uintptr(p.audioClient), &padding)
		if ret != 0 {
			time.Sleep(time.Millisecond)
			continue
		}

		available := p.bufferFrames - padding
		if available == 0 {
			time.Sleep(time.Millisecond)
			continue
		}

		// Call IAudioRenderClient::GetBuffer
		// VTable offset: 3
		vtable = *(**uintptr)(p.renderClient)
		getBuffer := *(*func(uintptr, uint32, *unsafe.Pointer) uintptr)(unsafe.Pointer(vtable + 3*unsafe.Sizeof(uintptr(0))))

		var data unsafe.Pointer
		ret = getBuffer(uintptr(p.renderClient), available, &data)
		if ret != 0 || data == nil {
			time.Sleep(time.Millisecond)
			continue
		}

		// Convert to int16 slice
		samples := (*[1 << 30]int16)(data)[:available*uint32(p.format.Channels):available*uint32(p.format.Channels)]

		// Read from ring buffer
		n := p.ringBuffer.Read(samples)

		// Fill rest with silence if not enough data
		for i := n; i < len(samples); i++ {
			samples[i] = 0
		}

		// Call IAudioRenderClient::ReleaseBuffer
		// VTable offset: 4
		releaseBuffer := *(*func(uintptr, uint32, uint32) uintptr)(unsafe.Pointer(vtable + 4*unsafe.Sizeof(uintptr(0))))
		releaseBuffer(uintptr(p.renderClient), available, 0)
	}
}
```

⏳ Needs implementation

---

## COM VTable Reference

### IMMDeviceEnumerator

| Offset | Method | Parameters |
|--------|--------|------------|
| 0 | QueryInterface | (riid, ppvObject) |
| 1 | AddRef | () |
| 2 | Release | () |
| 3 | EnumAudioEndpoints | (dataFlow, stateMask, devices) |
| 4 | GetDefaultAudioEndpoint | (dataFlow, role, device) |

### IMMDevice

| Offset | Method | Parameters |
|--------|--------|------------|
| 0 | QueryInterface | (riid, ppvObject) |
| 1 | AddRef | () |
| 2 | Release | () |
| 3 | Activate | (iid, clsCtx, activationParams, interface) |
| 4 | OpenPropertyStore | (access, properties) |
| 5 | GetId | (id) |
| 6 | GetState | (state) |

### IAudioClient

| Offset | Method | Parameters |
|--------|--------|------------|
| 0 | QueryInterface | (riid, ppvObject) |
| 1 | AddRef | () |
| 2 | Release | () |
| 3 | Initialize | (shareMode, streamFlags, bufferDuration, periodicity, format, sessionGuid) |
| 4 | GetBufferSize | (bufferFrames) |
| 5 | GetStreamLatency | (latency) |
| 6 | GetCurrentPadding | (padding) |
| 7 | IsFormatSupported | (shareMode, format, closestMatch) |
| 8 | GetMixFormat | (format) |
| 9 | GetDevicePeriod | (defaultPeriod, minimumPeriod) |
| 10 | Start | () |
| 11 | Stop | () |
| 12 | Reset | () |
| 13 | SetEventHandle | (eventHandle) |
| 14 | GetService | (riid, service) |

### IAudioCaptureClient

| Offset | Method | Parameters |
|--------|--------|------------|
| 0 | QueryInterface | (riid, ppvObject) |
| 1 | AddRef | () |
| 2 | Release | () |
| 3 | GetBuffer | (data, numFrames, flags, devicePosition, qpcPosition) |
| 4 | ReleaseBuffer | (numFrames) |
| 5 | GetNextPacketSize | (packetSize) |

### IAudioRenderClient

| Offset | Method | Parameters |
|--------|--------|------------|
| 0 | QueryInterface | (riid, ppvObject) |
| 1 | AddRef | () |
| 2 | Release | () |
| 3 | GetBuffer | (numFrames, data) |
| 4 | ReleaseBuffer | (numFrames, flags) |

---

## Testing

Once implemented, test with:

```bash
# Build on Windows
go build -tags windows -o tvcp.exe ./cmd/tvcp

# List devices
tvcp.exe list-devices

# Test call
tvcp.exe call localhost:5000
```

---

## References

- [WASAPI Documentation](https://docs.microsoft.com/en-us/windows/win32/coreaudio/wasapi)
- [IAudioClient](https://docs.microsoft.com/en-us/windows/win32/api/audioclient/nn-audioclient-iaudioclient)
- [IAudioCaptureClient](https://docs.microsoft.com/en-us/windows/win32/api/audioclient/nn-audioclient-iaudiocaptureclient)
- [IAudioRenderClient](https://docs.microsoft.com/en-us/windows/win32/api/audioclient/nn-audioclient-iaudiorenderclient)
- [Example: Capturing Audio Stream](https://docs.microsoft.com/en-us/windows/win32/coreaudio/capturing-a-stream)

---

## Next Steps

1. Implement COM vtable calls (Steps 2-8)
2. Implement capture/playback loops (Steps 9-10)
3. Test on Windows with real audio devices
4. Add error handling and recovery
5. Optimize buffer sizes and latency

**Total Estimated Lines**: ~400-500 lines of additional COM interface code

---

**Created**: 2026-02-07
**Status**: Infrastructure ready, COM calls pending
**Version**: 0.3.0-alpha
