//go:build windows

package audio

/*
#cgo LDFLAGS: -lole32 -loleaut32 -luuid

#include <windows.h>
#include <mmdeviceapi.h>
#include <audioclient.h>
#include <functiondiscoverykeys_devpkey.h>

// WASAPI GUIDs and constants
static const GUID CLSID_MMDeviceEnumerator = {0xBCDE0395, 0xE52F, 0x467C, {0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}};
static const GUID IID_IMMDeviceEnumerator = {0xA95664D2, 0x9614, 0x4F35, {0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}};
static const GUID IID_IAudioClient = {0x1CB9AD4C, 0xDBFA, 0x4c32, {0xB1, 0x78, 0xC2, 0xF5, 0x68, 0xA7, 0x03, 0xB2}};
static const GUID IID_IAudioCaptureClient = {0xC8ADBD64, 0xE71E, 0x48a0, {0xA4, 0xDE, 0x18, 0x5C, 0x39, 0x5C, 0xD3, 0x17}};
static const GUID IID_IAudioRenderClient = {0xF294ACFC, 0x3146, 0x4483, {0xA7, 0xBF, 0xAD, 0xDC, 0xA7, 0xC2, 0x60, 0xE2}};

// WAVEFORMATEX structure
typedef struct {
	WORD  wFormatTag;
	WORD  nChannels;
	DWORD nSamplesPerSec;
	DWORD nAvgBytesPerSec;
	WORD  nBlockAlign;
	WORD  wBitsPerSample;
	WORD  cbSize;
} WAVEFORMATEX_T;

#define WAVE_FORMAT_PCM 0x0001
#define AUDCLNT_SHAREMODE_SHARED 0
#define AUDCLNT_STREAMFLAGS_EVENTCALLBACK 0x00040000
#define AUDCLNT_STREAMFLAGS_AUTOCONVERTPCM 0x80000000
#define AUDCLNT_STREAMFLAGS_SRC_DEFAULT_QUALITY 0x08000000
#define REFTIMES_PER_SEC 10000000LL
#define REFTIMES_PER_MILLISEC 10000LL

// Helper function to call COM methods through vtable
static HRESULT CallMethod0(void* obj, int idx) {
	typedef HRESULT (*Method)(void*);
	Method method = (Method)(((void***)obj)[0][idx]);
	return method(obj);
}

static HRESULT CallMethod1(void* obj, int idx, void* arg1) {
	typedef HRESULT (*Method)(void*, void*);
	Method method = (Method)(((void***)obj)[0][idx]);
	return method(obj, arg1);
}

static HRESULT CallMethod2(void* obj, int idx, void* arg1, void* arg2) {
	typedef HRESULT (*Method)(void*, void*, void*);
	Method method = (Method)(((void***)obj)[0][idx]);
	return method(obj, arg1, arg2);
}

static HRESULT CallMethod3(void* obj, int idx, void* arg1, void* arg2, void* arg3) {
	typedef HRESULT (*Method)(void*, void*, void*, void*);
	Method method = (Method)(((void***)obj)[0][idx]);
	return method(obj, arg1, arg2, arg3);
}

static HRESULT CallMethod4(void* obj, int idx, void* arg1, void* arg2, void* arg3, void* arg4) {
	typedef HRESULT (*Method)(void*, void*, void*, void*, void*);
	Method method = (Method)(((void***)obj)[0][idx]);
	return method(obj, arg1, arg2, arg3, arg4);
}

static HRESULT CallMethod5(void* obj, int idx, void* arg1, void* arg2, void* arg3, void* arg4, void* arg5) {
	typedef HRESULT (*Method)(void*, void*, void*, void*, void*, void*);
	Method method = (Method)(((void***)obj)[0][idx]);
	return method(obj, arg1, arg2, arg3, arg4, arg5);
}

static HRESULT CallMethod6(void* obj, int idx, void* arg1, void* arg2, void* arg3, void* arg4, void* arg5, void* arg6) {
	typedef HRESULT (*Method)(void*, void*, void*, void*, void*, void*, void*);
	Method method = (Method)(((void***)obj)[0][idx]);
	return method(obj, arg1, arg2, arg3, arg4, arg5, arg6);
}

// COM Release helper
static ULONG ReleaseInterface(void* obj) {
	if (obj == NULL) return 0;
	typedef ULONG (*ReleaseMethod)(void*);
	ReleaseMethod release = (ReleaseMethod)(((void***)obj)[0][2]); // IUnknown::Release is at index 2
	return release(obj);
}
*/
import "C"

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	ole32            = syscall.NewLazyDLL("ole32.dll")
	coInitializeEx   = ole32.NewProc("CoInitializeEx")
	coCreateInstance = ole32.NewProc("CoCreateInstance")
	coUninitialize   = ole32.NewProc("CoUninitialize")
	coTaskMemFree    = ole32.NewProc("CoTaskMemFree")
)

const (
	COINIT_MULTITHREADED = 0x0
	CLSCTX_INPROC_SERVER = 0x1

	// EDataFlow
	eRender  = 0
	eCapture = 1

	// ERole
	eConsole = 0

	// AUDCLNT_SHAREMODE
	AUDCLNT_SHAREMODE_SHARED = 0

	// Buffer duration in 100-nanosecond units (20ms)
	BUFFER_DURATION = 20 * C.REFTIMES_PER_MILLISEC
)

// Ring buffer for audio data
type wasapiRingBuffer struct {
	data     []int16
	size     int
	readPos  int
	writePos int
	count    int
	mutex    sync.Mutex
	cond     *sync.Cond
}

func newWASAPIRingBuffer(size int) *wasapiRingBuffer {
	rb := &wasapiRingBuffer{
		data: make([]int16, size),
		size: size,
	}
	rb.cond = sync.NewCond(&rb.mutex)
	return rb
}

func (rb *wasapiRingBuffer) Write(samples []int16) int {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	written := 0
	for _, sample := range samples {
		if rb.count >= rb.size {
			break // Buffer full
		}
		rb.data[rb.writePos] = sample
		rb.writePos = (rb.writePos + 1) % rb.size
		rb.count++
		written++
	}

	if written > 0 {
		rb.cond.Signal()
	}

	return written
}

func (rb *wasapiRingBuffer) Read(samples []int16) int {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	read := 0
	for i := range samples {
		if rb.count == 0 {
			break // Buffer empty
		}
		samples[i] = rb.data[rb.readPos]
		rb.readPos = (rb.readPos + 1) % rb.size
		rb.count--
		read++
	}

	if read > 0 {
		rb.cond.Signal()
	}

	return read
}

func (rb *wasapiRingBuffer) Available() int {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()
	return rb.count
}

func (rb *wasapiRingBuffer) WaitForData(minSamples int) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	for rb.count < minSamples {
		rb.cond.Wait()
	}
}

// WASAPICapture implements AudioCapture using WASAPI
type WASAPICapture struct {
	device        unsafe.Pointer
	audioClient   unsafe.Pointer
	captureClient unsafe.Pointer
	format        AudioFormat
	bufferFrames  uint32
	ringBuffer    *wasapiRingBuffer
	stopChan      chan struct{}
	doneChan      chan struct{}
	captureThread chan struct{}
	open          bool
	mutex         sync.Mutex
}

// WASAPIPlayback implements AudioPlayback using WASAPI
type WASAPIPlayback struct {
	device       unsafe.Pointer
	audioClient  unsafe.Pointer
	renderClient unsafe.Pointer
	format       AudioFormat
	bufferFrames uint32
	ringBuffer   *wasapiRingBuffer
	stopChan     chan struct{}
	doneChan     chan struct{}
	renderThread chan struct{}
	open         bool
	mutex        sync.Mutex
}

func initCOM() error {
	// Initialize COM library
	ret, _, _ := coInitializeEx.Call(
		0,
		COINIT_MULTITHREADED,
	)
	// S_OK = 0, S_FALSE = 1 (already initialized)
	if ret != 0 && ret != 1 {
		return fmt.Errorf("failed to initialize COM: %x", ret)
	}
	return nil
}

func listCaptureDevicesImpl() ([]DeviceInfo, error) {
	if err := initCOM(); err != nil {
		return nil, err
	}
	defer coUninitialize.Call()

	var devices []DeviceInfo

	// Return default device
	devices = append(devices, DeviceInfo{
		ID:          0,
		Name:        "Default Capture Device",
		Type:        "capture",
		IsDefault:   true,
		SampleRates: []int{8000, 16000, 44100, 48000},
		Channels:    []int{1, 2},
	})

	return devices, nil
}

func listPlaybackDevicesImpl() ([]DeviceInfo, error) {
	if err := initCOM(); err != nil {
		return nil, err
	}
	defer coUninitialize.Call()

	var devices []DeviceInfo

	devices = append(devices, DeviceInfo{
		ID:          0,
		Name:        "Default Playback Device",
		Type:        "playback",
		IsDefault:   true,
		SampleRates: []int{8000, 16000, 44100, 48000},
		Channels:    []int{1, 2},
	})

	return devices, nil
}

func newCaptureImpl(deviceID int, format AudioFormat) (AudioCapture, error) {
	capture := &WASAPICapture{
		format:     format,
		ringBuffer: newWASAPIRingBuffer(format.SampleRate * 2), // 1 second buffer
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
	}

	return capture, nil
}

func newPlaybackImpl(deviceID int, format AudioFormat) (AudioPlayback, error) {
	playback := &WASAPIPlayback{
		format:     format,
		ringBuffer: newWASAPIRingBuffer(format.SampleRate * 2), // 1 second buffer
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
	}

	return playback, nil
}

func newDefaultCaptureImpl() (AudioCapture, error) {
	return newCaptureImpl(0, DefaultFormat())
}

func newDefaultPlaybackImpl() (AudioPlayback, error) {
	return newPlaybackImpl(0, DefaultFormat())
}

// WASAPICapture methods

func (c *WASAPICapture) Open() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.open {
		return nil
	}

	if err := initCOM(); err != nil {
		return err
	}

	// Step 1: Create device enumerator
	var enumerator unsafe.Pointer
	ret, _, _ := coCreateInstance.Call(
		uintptr(unsafe.Pointer(&C.CLSID_MMDeviceEnumerator)),
		0,
		CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(&C.IID_IMMDeviceEnumerator)),
		uintptr(unsafe.Pointer(&enumerator)),
	)
	if ret != 0 {
		return fmt.Errorf("failed to create device enumerator: %x", ret)
	}
	defer C.ReleaseInterface(enumerator)

	// Step 2: Get default audio endpoint (IMMDeviceEnumerator::GetDefaultAudioEndpoint)
	var device unsafe.Pointer
	hr := C.CallMethod2(enumerator, 4, // GetDefaultAudioEndpoint is method 4
		unsafe.Pointer(uintptr(eCapture)),
		unsafe.Pointer(uintptr(eConsole)))

	// Get the device pointer from method call
	hr = C.CallMethod3(enumerator, 4,
		unsafe.Pointer(uintptr(eCapture)),
		unsafe.Pointer(uintptr(eConsole)),
		unsafe.Pointer(&device))

	if hr != 0 {
		return fmt.Errorf("failed to get default capture device: %x", hr)
	}
	c.device = device

	// Step 3: Activate IAudioClient (IMMDevice::Activate)
	var audioClient unsafe.Pointer
	hr = C.CallMethod4(device, 3, // Activate is method 3
		unsafe.Pointer(&C.IID_IAudioClient),
		unsafe.Pointer(uintptr(CLSCTX_INPROC_SERVER)),
		nil,
		unsafe.Pointer(&audioClient))

	if hr != 0 {
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to activate audio client: %x", hr)
	}
	c.audioClient = audioClient

	// Step 4: Get mix format (IAudioClient::GetMixFormat)
	var pwfx unsafe.Pointer
	hr = C.CallMethod1(audioClient, 8, // GetMixFormat is method 8
		unsafe.Pointer(&pwfx))

	if hr != 0 {
		C.ReleaseInterface(audioClient)
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to get mix format: %x", hr)
	}

	// Use the mix format or create our own
	wfx := (*C.WAVEFORMATEX_T)(pwfx)

	// Set desired format (16kHz mono for TVCP)
	wfx.wFormatTag = C.WAVE_FORMAT_PCM
	wfx.nChannels = C.WORD(c.format.Channels)
	wfx.nSamplesPerSec = C.DWORD(c.format.SampleRate)
	wfx.wBitsPerSample = C.WORD(c.format.BitDepth)
	wfx.nBlockAlign = wfx.nChannels * (wfx.wBitsPerSample / 8)
	wfx.nAvgBytesPerSec = wfx.nSamplesPerSec * C.DWORD(wfx.nBlockAlign)
	wfx.cbSize = 0

	// Step 5: Initialize audio client (IAudioClient::Initialize)
	hr = C.CallMethod6(audioClient, 3, // Initialize is method 3
		unsafe.Pointer(uintptr(AUDCLNT_SHAREMODE_SHARED)),
		unsafe.Pointer(uintptr(C.AUDCLNT_STREAMFLAGS_AUTOCONVERTPCM|C.AUDCLNT_STREAMFLAGS_SRC_DEFAULT_QUALITY)),
		unsafe.Pointer(uintptr(BUFFER_DURATION)),
		unsafe.Pointer(uintptr(0)),
		pwfx,
		nil)

	if hr != 0 {
		coTaskMemFree.Call(uintptr(pwfx))
		C.ReleaseInterface(audioClient)
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to initialize audio client: %x", hr)
	}

	coTaskMemFree.Call(uintptr(pwfx))

	// Step 6: Get buffer size (IAudioClient::GetBufferSize)
	var bufferFrames C.UINT32
	hr = C.CallMethod1(audioClient, 6, // GetBufferSize is method 6
		unsafe.Pointer(&bufferFrames))

	if hr != 0 {
		C.ReleaseInterface(audioClient)
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to get buffer size: %x", hr)
	}
	c.bufferFrames = uint32(bufferFrames)

	// Step 7: Get capture client (IAudioClient::GetService)
	var captureClient unsafe.Pointer
	hr = C.CallMethod2(audioClient, 14, // GetService is method 14
		unsafe.Pointer(&C.IID_IAudioCaptureClient),
		unsafe.Pointer(&captureClient))

	if hr != 0 {
		C.ReleaseInterface(audioClient)
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to get capture client: %x", hr)
	}
	c.captureClient = captureClient

	// Step 8: Start audio client (IAudioClient::Start)
	hr = C.CallMethod0(audioClient, 5) // Start is method 5
	if hr != 0 {
		C.ReleaseInterface(captureClient)
		C.ReleaseInterface(audioClient)
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to start audio client: %x", hr)
	}

	c.open = true

	// Start capture thread
	c.captureThread = make(chan struct{})
	go c.captureLoop()

	return nil
}

func (c *WASAPICapture) captureLoop() {
	defer close(c.captureThread)

	for {
		select {
		case <-c.stopChan:
			return
		default:
			// Capture audio data
			c.captureAudio()
			time.Sleep(5 * time.Millisecond) // Sleep briefly
		}
	}
}

func (c *WASAPICapture) captureAudio() {
	// Get next packet size (IAudioCaptureClient::GetNextPacketSize)
	var packetSize C.UINT32
	hr := C.CallMethod1(c.captureClient, 5, // GetNextPacketSize is method 5
		unsafe.Pointer(&packetSize))

	if hr != 0 || packetSize == 0 {
		return
	}

	// Get buffer (IAudioCaptureClient::GetBuffer)
	var pData unsafe.Pointer
	var numFrames C.UINT32
	var flags C.DWORD

	hr = C.CallMethod5(c.captureClient, 3, // GetBuffer is method 3
		unsafe.Pointer(&pData),
		unsafe.Pointer(&numFrames),
		unsafe.Pointer(&flags),
		nil,
		nil)

	if hr != 0 {
		return
	}

	// Copy data to ring buffer
	if pData != nil && numFrames > 0 {
		samples := (*[1 << 20]int16)(pData)[:int(numFrames)*c.format.Channels]
		c.ringBuffer.Write(samples)
	}

	// Release buffer (IAudioCaptureClient::ReleaseBuffer)
	C.CallMethod1(c.captureClient, 4, // ReleaseBuffer is method 4
		unsafe.Pointer(uintptr(numFrames)))
}

func (c *WASAPICapture) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.open {
		return nil
	}

	// Stop capture thread
	close(c.stopChan)
	<-c.captureThread

	// Stop audio client
	if c.audioClient != nil {
		C.CallMethod0(c.audioClient, 4) // Stop is method 4
		C.ReleaseInterface(c.audioClient)
		c.audioClient = nil
	}

	// Release capture client
	if c.captureClient != nil {
		C.ReleaseInterface(c.captureClient)
		c.captureClient = nil
	}

	// Release device
	if c.device != nil {
		C.ReleaseInterface(c.device)
		c.device = nil
	}

	coUninitialize.Call()

	c.open = false
	return nil
}

func (c *WASAPICapture) Read(buffer []int16) (int, error) {
	if !c.IsOpen() {
		return 0, ErrDeviceNotOpen
	}

	return c.ringBuffer.Read(buffer), nil
}

func (c *WASAPICapture) IsOpen() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.open
}

func (c *WASAPICapture) GetFormat() AudioFormat {
	return c.format
}

// WASAPIPlayback methods

func (p *WASAPIPlayback) Open() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.open {
		return nil
	}

	if err := initCOM(); err != nil {
		return err
	}

	// Similar to capture: create enumerator, get device, activate client, etc.
	var enumerator unsafe.Pointer
	ret, _, _ := coCreateInstance.Call(
		uintptr(unsafe.Pointer(&C.CLSID_MMDeviceEnumerator)),
		0,
		CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(&C.IID_IMMDeviceEnumerator)),
		uintptr(unsafe.Pointer(&enumerator)),
	)
	if ret != 0 {
		return fmt.Errorf("failed to create device enumerator: %x", ret)
	}
	defer C.ReleaseInterface(enumerator)

	// Get default render endpoint
	var device unsafe.Pointer
	hr := C.CallMethod3(enumerator, 4,
		unsafe.Pointer(uintptr(eRender)),
		unsafe.Pointer(uintptr(eConsole)),
		unsafe.Pointer(&device))

	if hr != 0 {
		return fmt.Errorf("failed to get default render device: %x", hr)
	}
	p.device = device

	// Activate IAudioClient
	var audioClient unsafe.Pointer
	hr = C.CallMethod4(device, 3,
		unsafe.Pointer(&C.IID_IAudioClient),
		unsafe.Pointer(uintptr(CLSCTX_INPROC_SERVER)),
		nil,
		unsafe.Pointer(&audioClient))

	if hr != 0 {
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to activate audio client: %x", hr)
	}
	p.audioClient = audioClient

	// Get and set format
	var pwfx unsafe.Pointer
	hr = C.CallMethod1(audioClient, 8, unsafe.Pointer(&pwfx))
	if hr != 0 {
		C.ReleaseInterface(audioClient)
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to get mix format: %x", hr)
	}

	wfx := (*C.WAVEFORMATEX_T)(pwfx)
	wfx.wFormatTag = C.WAVE_FORMAT_PCM
	wfx.nChannels = C.WORD(p.format.Channels)
	wfx.nSamplesPerSec = C.DWORD(p.format.SampleRate)
	wfx.wBitsPerSample = C.WORD(p.format.BitDepth)
	wfx.nBlockAlign = wfx.nChannels * (wfx.wBitsPerSample / 8)
	wfx.nAvgBytesPerSec = wfx.nSamplesPerSec * C.DWORD(wfx.nBlockAlign)
	wfx.cbSize = 0

	// Initialize
	hr = C.CallMethod6(audioClient, 3,
		unsafe.Pointer(uintptr(AUDCLNT_SHAREMODE_SHARED)),
		unsafe.Pointer(uintptr(C.AUDCLNT_STREAMFLAGS_AUTOCONVERTPCM|C.AUDCLNT_STREAMFLAGS_SRC_DEFAULT_QUALITY)),
		unsafe.Pointer(uintptr(BUFFER_DURATION)),
		unsafe.Pointer(uintptr(0)),
		pwfx,
		nil)

	if hr != 0 {
		coTaskMemFree.Call(uintptr(pwfx))
		C.ReleaseInterface(audioClient)
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to initialize audio client: %x", hr)
	}

	coTaskMemFree.Call(uintptr(pwfx))

	// Get buffer size
	var bufferFrames C.UINT32
	hr = C.CallMethod1(audioClient, 6, unsafe.Pointer(&bufferFrames))
	if hr != 0 {
		C.ReleaseInterface(audioClient)
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to get buffer size: %x", hr)
	}
	p.bufferFrames = uint32(bufferFrames)

	// Get render client
	var renderClient unsafe.Pointer
	hr = C.CallMethod2(audioClient, 14,
		unsafe.Pointer(&C.IID_IAudioRenderClient),
		unsafe.Pointer(&renderClient))

	if hr != 0 {
		C.ReleaseInterface(audioClient)
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to get render client: %x", hr)
	}
	p.renderClient = renderClient

	// Start
	hr = C.CallMethod0(audioClient, 5)
	if hr != 0 {
		C.ReleaseInterface(renderClient)
		C.ReleaseInterface(audioClient)
		C.ReleaseInterface(device)
		return fmt.Errorf("failed to start audio client: %x", hr)
	}

	p.open = true

	// Start render thread
	p.renderThread = make(chan struct{})
	go p.renderLoop()

	return nil
}

func (p *WASAPIPlayback) renderLoop() {
	defer close(p.renderThread)

	for {
		select {
		case <-p.stopChan:
			return
		default:
			p.renderAudio()
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (p *WASAPIPlayback) renderAudio() {
	// Get current padding (IAudioClient::GetCurrentPadding)
	var numPadding C.UINT32
	hr := C.CallMethod1(p.audioClient, 7, // GetCurrentPadding is method 7
		unsafe.Pointer(&numPadding))

	if hr != 0 {
		return
	}

	// Calculate available frames
	available := int(p.bufferFrames) - int(numPadding)
	if available <= 0 {
		return
	}

	// Get buffer (IAudioRenderClient::GetBuffer)
	var pData unsafe.Pointer
	hr = C.CallMethod2(p.renderClient, 3, // GetBuffer is method 3
		unsafe.Pointer(uintptr(available)),
		unsafe.Pointer(&pData))

	if hr != 0 {
		return
	}

	// Fill buffer from ring buffer
	if pData != nil {
		samples := (*[1 << 20]int16)(pData)[:available*p.format.Channels]
		read := p.ringBuffer.Read(samples)

		// Fill remaining with silence
		for i := read; i < len(samples); i++ {
			samples[i] = 0
		}
	}

	// Release buffer (IAudioRenderClient::ReleaseBuffer)
	C.CallMethod2(p.renderClient, 4, // ReleaseBuffer is method 4
		unsafe.Pointer(uintptr(available)),
		unsafe.Pointer(uintptr(0)))
}

func (p *WASAPIPlayback) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.open {
		return nil
	}

	// Stop render thread
	close(p.stopChan)
	<-p.renderThread

	// Stop audio client
	if p.audioClient != nil {
		C.CallMethod0(p.audioClient, 4)
		C.ReleaseInterface(p.audioClient)
		p.audioClient = nil
	}

	// Release render client
	if p.renderClient != nil {
		C.ReleaseInterface(p.renderClient)
		p.renderClient = nil
	}

	// Release device
	if p.device != nil {
		C.ReleaseInterface(p.device)
		p.device = nil
	}

	coUninitialize.Call()

	p.open = false
	return nil
}

func (p *WASAPIPlayback) Write(buffer []int16) (int, error) {
	if !p.IsOpen() {
		return 0, ErrDeviceNotOpen
	}

	return p.ringBuffer.Write(buffer), nil
}

func (p *WASAPIPlayback) IsOpen() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.open
}

func (p *WASAPIPlayback) GetFormat() AudioFormat {
	return p.format
}
