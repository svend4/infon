//go:build darwin

package audio

/*
#cgo LDFLAGS: -framework CoreAudio -framework AudioToolbox
#include <CoreAudio/CoreAudio.h>
#include <AudioToolbox/AudioToolbox.h>
#include <stdlib.h>
#include <string.h>

// External Go callback functions (implemented in Go)
extern OSStatus goInputCallback(void *inRefCon, AudioUnitRenderActionFlags *ioActionFlags, const AudioTimeStamp *inTimeStamp, UInt32 inBusNumber, UInt32 inNumberFrames, AudioBufferList *ioData, AudioUnit audioUnit);
extern OSStatus goOutputCallback(void *inRefCon, AudioUnitRenderActionFlags *ioActionFlags, const AudioTimeStamp *inTimeStamp, UInt32 inBusNumber, UInt32 inNumberFrames, AudioBufferList *ioData);

// C wrapper for input callback
static OSStatus inputCallbackWrapper(
	void *inRefCon,
	AudioUnitRenderActionFlags *ioActionFlags,
	const AudioTimeStamp *inTimeStamp,
	UInt32 inBusNumber,
	UInt32 inNumberFrames,
	AudioBufferList *ioData) {

	// Get audio unit from refcon
	AudioUnit audioUnit = *(AudioUnit*)inRefCon;

	// Call Go callback
	return goInputCallback(inRefCon, ioActionFlags, inTimeStamp, inBusNumber, inNumberFrames, ioData, audioUnit);
}

// C wrapper for output callback
static OSStatus outputCallbackWrapper(
	void *inRefCon,
	AudioUnitRenderActionFlags *ioActionFlags,
	const AudioTimeStamp *inTimeStamp,
	UInt32 inBusNumber,
	UInt32 inNumberFrames,
	AudioBufferList *ioData) {

	return goOutputCallback(inRefCon, ioActionFlags, inTimeStamp, inBusNumber, inNumberFrames, ioData);
}

// Helper to set input callback
static OSStatus setInputCallback(AudioUnit audioUnit, void *refCon) {
	AURenderCallbackStruct callback;
	callback.inputProc = inputCallbackWrapper;
	callback.inputProcRefCon = refCon;

	return AudioUnitSetProperty(
		audioUnit,
		kAudioOutputUnitProperty_SetInputCallback,
		kAudioUnitScope_Global,
		0,
		&callback,
		sizeof(callback)
	);
}

// Helper to set output callback
static OSStatus setOutputCallback(AudioUnit audioUnit, void *refCon) {
	AURenderCallbackStruct callback;
	callback.inputProc = outputCallbackWrapper;
	callback.inputProcRefCon = refCon;

	return AudioUnitSetProperty(
		audioUnit,
		kAudioUnitProperty_SetRenderCallback,
		kAudioUnitScope_Input,
		0,
		&callback,
		sizeof(callback)
	);
}
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

// Ring buffer for audio data
type ringBuffer struct {
	data     []int16
	size     int
	readPos  int
	writePos int
	count    int
	mutex    sync.Mutex
	cond     *sync.Cond
}

func newRingBuffer(size int) *ringBuffer {
	rb := &ringBuffer{
		data: make([]int16, size),
		size: size,
	}
	rb.cond = sync.NewCond(&rb.mutex)
	return rb
}

func (rb *ringBuffer) Write(samples []int16) int {
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

func (rb *ringBuffer) Read(samples []int16) int {
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

	return read
}

func (rb *ringBuffer) Available() int {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()
	return rb.count
}

func (rb *ringBuffer) WaitForData(minSamples int) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	for rb.count < minSamples {
		rb.cond.Wait()
	}
}

// CoreAudioCapture implements AudioCapture using CoreAudio
type CoreAudioCapture struct {
	audioUnit   C.AudioUnit
	format      AudioFormat
	ringBuffer  *ringBuffer
	bufferList  *C.AudioBufferList
	open        bool
	mutex       sync.Mutex
}

// CoreAudioPlayback implements AudioPlayback using CoreAudio
type CoreAudioPlayback struct {
	audioUnit  C.AudioUnit
	format     AudioFormat
	ringBuffer *ringBuffer
	open       bool
	mutex      sync.Mutex
}

// Global maps to access capture/playback from C callbacks
var (
	captureMap  = make(map[uintptr]*CoreAudioCapture)
	playbackMap = make(map[uintptr]*CoreAudioPlayback)
	captureMu   sync.RWMutex
	playbackMu  sync.RWMutex
)

func listCaptureDevicesImpl() ([]DeviceInfo, error) {
	var devices []DeviceInfo

	// Get default input device
	var deviceID C.AudioDeviceID
	var size C.UInt32 = C.UInt32(unsafe.Sizeof(deviceID))

	var address C.AudioObjectPropertyAddress
	address.mSelector = C.kAudioHardwarePropertyDefaultInputDevice
	address.mScope = C.kAudioObjectPropertyScopeGlobal
	address.mElement = C.kAudioObjectPropertyElementMain

	status := C.AudioObjectGetPropertyData(
		C.kAudioObjectSystemObject,
		&address,
		0,
		nil,
		&size,
		unsafe.Pointer(&deviceID),
	)

	if status != 0 {
		return nil, fmt.Errorf("failed to get default input device: %d", status)
	}

	// Get device name
	var nameSize C.UInt32 = 256
	name := make([]byte, nameSize)

	var nameAddress C.AudioObjectPropertyAddress
	nameAddress.mSelector = C.kAudioDevicePropertyDeviceNameCFString
	nameAddress.mScope = C.kAudioObjectPropertyScopeGlobal
	nameAddress.mElement = C.kAudioObjectPropertyElementMain

	var cfName C.CFStringRef
	nameSize = C.UInt32(unsafe.Sizeof(cfName))

	status = C.AudioObjectGetPropertyData(
		C.AudioObjectID(deviceID),
		&nameAddress,
		0,
		nil,
		&nameSize,
		unsafe.Pointer(&cfName),
	)

	deviceName := "Default Input Device"
	if status == 0 && cfName != 0 {
		// Convert CFString to Go string
		length := C.CFStringGetLength(cfName)
		if length > 0 {
			cStr := C.CFStringGetCStringPtr(cfName, C.kCFStringEncodingUTF8)
			if cStr != nil {
				deviceName = C.GoString(cStr)
			}
		}
		C.CFRelease(C.CFTypeRef(cfName))
	}

	devices = append(devices, DeviceInfo{
		ID:          0,
		Name:        deviceName,
		Type:        "capture",
		IsDefault:   true,
		SampleRates: []int{8000, 16000, 44100, 48000},
		Channels:    []int{1, 2},
	})

	return devices, nil
}

func listPlaybackDevicesImpl() ([]DeviceInfo, error) {
	var devices []DeviceInfo

	// Get default output device
	var deviceID C.AudioDeviceID
	var size C.UInt32 = C.UInt32(unsafe.Sizeof(deviceID))

	var address C.AudioObjectPropertyAddress
	address.mSelector = C.kAudioHardwarePropertyDefaultOutputDevice
	address.mScope = C.kAudioObjectPropertyScopeGlobal
	address.mElement = C.kAudioObjectPropertyElementMain

	status := C.AudioObjectGetPropertyData(
		C.kAudioObjectSystemObject,
		&address,
		0,
		nil,
		&size,
		unsafe.Pointer(&deviceID),
	)

	if status != 0 {
		return nil, fmt.Errorf("failed to get default output device: %d", status)
	}

	// Get device name
	var cfName C.CFStringRef
	var nameSize C.UInt32 = C.UInt32(unsafe.Sizeof(cfName))

	var nameAddress C.AudioObjectPropertyAddress
	nameAddress.mSelector = C.kAudioDevicePropertyDeviceNameCFString
	nameAddress.mScope = C.kAudioObjectPropertyScopeGlobal
	nameAddress.mElement = C.kAudioObjectPropertyElementMain

	status = C.AudioObjectGetPropertyData(
		C.AudioObjectID(deviceID),
		&nameAddress,
		0,
		nil,
		&nameSize,
		unsafe.Pointer(&cfName),
	)

	deviceName := "Default Output Device"
	if status == 0 && cfName != 0 {
		length := C.CFStringGetLength(cfName)
		if length > 0 {
			cStr := C.CFStringGetCStringPtr(cfName, C.kCFStringEncodingUTF8)
			if cStr != nil {
				deviceName = C.GoString(cStr)
			}
		}
		C.CFRelease(C.CFTypeRef(cfName))
	}

	devices = append(devices, DeviceInfo{
		ID:          0,
		Name:        deviceName,
		Type:        "playback",
		IsDefault:   true,
		SampleRates: []int{8000, 16000, 44100, 48000},
		Channels:    []int{1, 2},
	})

	return devices, nil
}

func newCaptureImpl(deviceID int, format AudioFormat) (AudioCapture, error) {
	// Create ring buffer (1 second capacity)
	bufferSize := format.SampleRate * format.Channels
	capture := &CoreAudioCapture{
		format:     format,
		ringBuffer: newRingBuffer(bufferSize),
	}

	// Create Audio Component Description
	var desc C.AudioComponentDescription
	desc.componentType = C.kAudioUnitType_Output
	desc.componentSubType = C.kAudioUnitSubType_HALOutput
	desc.componentManufacturer = C.kAudioUnitManufacturer_Apple
	desc.componentFlags = 0
	desc.componentFlagsMask = 0

	// Find component
	component := C.AudioComponentFindNext(nil, &desc)
	if component == nil {
		return nil, fmt.Errorf("failed to find audio component")
	}

	// Create audio unit
	status := C.AudioComponentInstanceNew(component, &capture.audioUnit)
	if status != 0 {
		return nil, fmt.Errorf("failed to create audio unit: %d", status)
	}

	// Enable input
	var one C.UInt32 = 1
	status = C.AudioUnitSetProperty(
		capture.audioUnit,
		C.kAudioOutputUnitProperty_EnableIO,
		C.kAudioUnitScope_Input,
		1, // Input bus
		unsafe.Pointer(&one),
		C.UInt32(unsafe.Sizeof(one)),
	)
	if status != 0 {
		C.AudioComponentInstanceDispose(capture.audioUnit)
		return nil, fmt.Errorf("failed to enable input: %d", status)
	}

	// Disable output
	var zero C.UInt32 = 0
	status = C.AudioUnitSetProperty(
		capture.audioUnit,
		C.kAudioOutputUnitProperty_EnableIO,
		C.kAudioUnitScope_Output,
		0, // Output bus
		unsafe.Pointer(&zero),
		C.UInt32(unsafe.Sizeof(zero)),
	)
	if status != 0 {
		C.AudioComponentInstanceDispose(capture.audioUnit)
		return nil, fmt.Errorf("failed to disable output: %d", status)
	}

	// Set format
	var streamFormat C.AudioStreamBasicDescription
	streamFormat.mSampleRate = C.Float64(format.SampleRate)
	streamFormat.mFormatID = C.kAudioFormatLinearPCM
	streamFormat.mFormatFlags = C.kAudioFormatFlagIsSignedInteger | C.kAudioFormatFlagIsPacked
	streamFormat.mBytesPerPacket = C.UInt32(format.FrameSize())
	streamFormat.mFramesPerPacket = 1
	streamFormat.mBytesPerFrame = C.UInt32(format.FrameSize())
	streamFormat.mChannelsPerFrame = C.UInt32(format.Channels)
	streamFormat.mBitsPerChannel = C.UInt32(format.BitDepth)

	status = C.AudioUnitSetProperty(
		capture.audioUnit,
		C.kAudioUnitProperty_StreamFormat,
		C.kAudioUnitScope_Output,
		1, // Input bus
		unsafe.Pointer(&streamFormat),
		C.UInt32(unsafe.Sizeof(streamFormat)),
	)
	if status != 0 {
		C.AudioComponentInstanceDispose(capture.audioUnit)
		return nil, fmt.Errorf("failed to set stream format: %d", status)
	}

	// Allocate buffer list for audio unit render
	bufferListSize := C.sizeof_AudioBufferList + C.sizeof_AudioBuffer
	capture.bufferList = (*C.AudioBufferList)(C.malloc(C.size_t(bufferListSize)))
	capture.bufferList.mNumberBuffers = 1
	capture.bufferList.mBuffers[0].mNumberChannels = C.UInt32(format.Channels)
	capture.bufferList.mBuffers[0].mDataByteSize = C.UInt32(4096 * format.FrameSize()) // 4096 frames buffer
	capture.bufferList.mBuffers[0].mData = C.malloc(C.size_t(capture.bufferList.mBuffers[0].mDataByteSize))

	// Register in global map
	captureMu.Lock()
	captureMap[uintptr(unsafe.Pointer(&capture.audioUnit))] = capture
	captureMu.Unlock()

	// Set input callback
	status = C.setInputCallback(capture.audioUnit, unsafe.Pointer(&capture.audioUnit))
	if status != 0 {
		C.free(capture.bufferList.mBuffers[0].mData)
		C.free(unsafe.Pointer(capture.bufferList))
		C.AudioComponentInstanceDispose(capture.audioUnit)
		captureMu.Lock()
		delete(captureMap, uintptr(unsafe.Pointer(&capture.audioUnit)))
		captureMu.Unlock()
		return nil, fmt.Errorf("failed to set input callback: %d", status)
	}

	return capture, nil
}

func newPlaybackImpl(deviceID int, format AudioFormat) (AudioPlayback, error) {
	// Create ring buffer (1 second capacity)
	bufferSize := format.SampleRate * format.Channels
	playback := &CoreAudioPlayback{
		format:     format,
		ringBuffer: newRingBuffer(bufferSize),
	}

	// Create Audio Component Description
	var desc C.AudioComponentDescription
	desc.componentType = C.kAudioUnitType_Output
	desc.componentSubType = C.kAudioUnitSubType_DefaultOutput
	desc.componentManufacturer = C.kAudioUnitManufacturer_Apple
	desc.componentFlags = 0
	desc.componentFlagsMask = 0

	// Find component
	component := C.AudioComponentFindNext(nil, &desc)
	if component == nil {
		return nil, fmt.Errorf("failed to find audio component")
	}

	// Create audio unit
	status := C.AudioComponentInstanceNew(component, &playback.audioUnit)
	if status != 0 {
		return nil, fmt.Errorf("failed to create audio unit: %d", status)
	}

	// Set format
	var streamFormat C.AudioStreamBasicDescription
	streamFormat.mSampleRate = C.Float64(format.SampleRate)
	streamFormat.mFormatID = C.kAudioFormatLinearPCM
	streamFormat.mFormatFlags = C.kAudioFormatFlagIsSignedInteger | C.kAudioFormatFlagIsPacked
	streamFormat.mBytesPerPacket = C.UInt32(format.FrameSize())
	streamFormat.mFramesPerPacket = 1
	streamFormat.mBytesPerFrame = C.UInt32(format.FrameSize())
	streamFormat.mChannelsPerFrame = C.UInt32(format.Channels)
	streamFormat.mBitsPerChannel = C.UInt32(format.BitDepth)

	status = C.AudioUnitSetProperty(
		playback.audioUnit,
		C.kAudioUnitProperty_StreamFormat,
		C.kAudioUnitScope_Input,
		0, // Output bus
		unsafe.Pointer(&streamFormat),
		C.UInt32(unsafe.Sizeof(streamFormat)),
	)
	if status != 0 {
		C.AudioComponentInstanceDispose(playback.audioUnit)
		return nil, fmt.Errorf("failed to set stream format: %d", status)
	}

	// Register in global map
	playbackMu.Lock()
	playbackMap[uintptr(unsafe.Pointer(&playback.audioUnit))] = playback
	playbackMu.Unlock()

	// Set output callback
	status = C.setOutputCallback(playback.audioUnit, unsafe.Pointer(&playback.audioUnit))
	if status != 0 {
		C.AudioComponentInstanceDispose(playback.audioUnit)
		playbackMu.Lock()
		delete(playbackMap, uintptr(unsafe.Pointer(&playback.audioUnit)))
		playbackMu.Unlock()
		return nil, fmt.Errorf("failed to set output callback: %d", status)
	}

	return playback, nil
}

func newDefaultCaptureImpl() (AudioCapture, error) {
	return newCaptureImpl(0, DefaultFormat())
}

func newDefaultPlaybackImpl() (AudioPlayback, error) {
	return newPlaybackImpl(0, DefaultFormat())
}

// CoreAudioCapture methods

func (c *CoreAudioCapture) Open() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.open {
		return nil
	}

	// Initialize audio unit
	status := C.AudioUnitInitialize(c.audioUnit)
	if status != 0 {
		return fmt.Errorf("failed to initialize audio unit: %d", status)
	}

	// Start audio unit
	status = C.AudioOutputUnitStart(c.audioUnit)
	if status != 0 {
		C.AudioUnitUninitialize(c.audioUnit)
		return fmt.Errorf("failed to start audio unit: %d", status)
	}

	c.open = true
	return nil
}

func (c *CoreAudioCapture) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.open {
		return nil
	}

	C.AudioOutputUnitStop(c.audioUnit)
	C.AudioUnitUninitialize(c.audioUnit)
	C.AudioComponentInstanceDispose(c.audioUnit)

	// Free buffer list
	if c.bufferList != nil {
		if c.bufferList.mBuffers[0].mData != nil {
			C.free(c.bufferList.mBuffers[0].mData)
		}
		C.free(unsafe.Pointer(c.bufferList))
		c.bufferList = nil
	}

	// Remove from global map
	captureMu.Lock()
	delete(captureMap, uintptr(unsafe.Pointer(&c.audioUnit)))
	captureMu.Unlock()

	c.open = false
	c.ringBuffer.cond.Broadcast()
	return nil
}

func (c *CoreAudioCapture) Read(buffer []int16) (int, error) {
	c.mutex.Lock()
	open := c.open
	c.mutex.Unlock()

	if !open {
		return 0, ErrDeviceNotOpen
	}

	// Wait for enough data in ring buffer
	c.ringBuffer.WaitForData(len(buffer))

	// Read from ring buffer
	n := c.ringBuffer.Read(buffer)

	// If not enough data was available, fill rest with silence
	for i := n; i < len(buffer); i++ {
		buffer[i] = 0
	}

	return len(buffer), nil
}

func (c *CoreAudioCapture) IsOpen() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.open
}

func (c *CoreAudioCapture) GetFormat() AudioFormat {
	return c.format
}

// CoreAudioPlayback methods

func (p *CoreAudioPlayback) Open() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.open {
		return nil
	}

	// Initialize audio unit
	status := C.AudioUnitInitialize(p.audioUnit)
	if status != 0 {
		return fmt.Errorf("failed to initialize audio unit: %d", status)
	}

	// Start audio unit
	status = C.AudioOutputUnitStart(p.audioUnit)
	if status != 0 {
		C.AudioUnitUninitialize(p.audioUnit)
		return fmt.Errorf("failed to start audio unit: %d", status)
	}

	p.open = true
	return nil
}

func (p *CoreAudioPlayback) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.open {
		return nil
	}

	C.AudioOutputUnitStop(p.audioUnit)
	C.AudioUnitUninitialize(p.audioUnit)
	C.AudioComponentInstanceDispose(p.audioUnit)

	// Remove from global map
	playbackMu.Lock()
	delete(playbackMap, uintptr(unsafe.Pointer(&p.audioUnit)))
	playbackMu.Unlock()

	p.open = false
	p.ringBuffer.cond.Broadcast()
	return nil
}

func (p *CoreAudioPlayback) Write(buffer []int16) (int, error) {
	p.mutex.Lock()
	open := p.open
	p.mutex.Unlock()

	if !open {
		return 0, ErrDeviceNotOpen
	}

	// Write to ring buffer
	written := 0
	for written < len(buffer) {
		n := p.ringBuffer.Write(buffer[written:])
		written += n

		// If buffer is full, wait a bit
		if n == 0 {
			p.ringBuffer.cond.L.Lock()
			p.ringBuffer.cond.Wait()
			p.ringBuffer.cond.L.Unlock()
		}
	}

	return written, nil
}

func (p *CoreAudioPlayback) IsOpen() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.open
}

func (p *CoreAudioPlayback) GetFormat() AudioFormat {
	return p.format
}

// Go callbacks exported to C

//export goInputCallback
func goInputCallback(
	inRefCon unsafe.Pointer,
	ioActionFlags *C.AudioUnitRenderActionFlags,
	inTimeStamp *C.AudioTimeStamp,
	inBusNumber C.UInt32,
	inNumberFrames C.UInt32,
	ioData *C.AudioBufferList,
	audioUnit C.AudioUnit,
) C.OSStatus {
	// Find capture from global map
	captureMu.RLock()
	capture, ok := captureMap[uintptr(inRefCon)]
	captureMu.RUnlock()

	if !ok || capture == nil || capture.bufferList == nil {
		return C.OSStatus(-1)
	}

	// Render audio into our buffer
	status := C.AudioUnitRender(
		audioUnit,
		ioActionFlags,
		inTimeStamp,
		inBusNumber,
		inNumberFrames,
		capture.bufferList,
	)

	if status != 0 {
		return status
	}

	// Copy rendered data to ring buffer
	buffer := capture.bufferList.mBuffers[0]
	if buffer.mData != nil && buffer.mDataByteSize > 0 {
		// Convert to int16 slice
		numSamples := int(buffer.mDataByteSize) / 2 // 2 bytes per int16
		samples := (*[1 << 30]int16)(buffer.mData)[:numSamples:numSamples]

		// Write to ring buffer
		capture.ringBuffer.Write(samples)
	}

	return 0
}

//export goOutputCallback
func goOutputCallback(
	inRefCon unsafe.Pointer,
	ioActionFlags *C.AudioUnitRenderActionFlags,
	inTimeStamp *C.AudioTimeStamp,
	inBusNumber C.UInt32,
	inNumberFrames C.UInt32,
	ioData *C.AudioBufferList,
) C.OSStatus {
	// Find playback from global map
	playbackMu.RLock()
	playback, ok := playbackMap[uintptr(inRefCon)]
	playbackMu.RUnlock()

	if !ok || playback == nil || ioData == nil {
		return C.OSStatus(-1)
	}

	// Get output buffer
	buffer := &ioData.mBuffers[0]
	if buffer.mData == nil || buffer.mDataByteSize == 0 {
		return C.OSStatus(-1)
	}

	// Convert to int16 slice
	numSamples := int(buffer.mDataByteSize) / 2 // 2 bytes per int16
	outSamples := (*[1 << 30]int16)(buffer.mData)[:numSamples:numSamples]

	// Read from ring buffer
	n := playback.ringBuffer.Read(outSamples)

	// Fill rest with silence if not enough data
	for i := n; i < numSamples; i++ {
		outSamples[i] = 0
	}

	return 0
}
