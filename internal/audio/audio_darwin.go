//go:build darwin

package audio

/*
#cgo LDFLAGS: -framework CoreAudio -framework AudioToolbox
#include <CoreAudio/CoreAudio.h>
#include <AudioToolbox/AudioToolbox.h>
#include <stdlib.h>

// Callback function for audio input
OSStatus inputCallback(
	void *inRefCon,
	AudioUnitRenderActionFlags *ioActionFlags,
	const AudioTimeStamp *inTimeStamp,
	UInt32 inBusNumber,
	UInt32 inNumberFrames,
	AudioBufferList *ioData);

// Callback function for audio output
OSStatus outputCallback(
	void *inRefCon,
	AudioUnitRenderActionFlags *ioActionFlags,
	const AudioTimeStamp *inTimeStamp,
	UInt32 inBusNumber,
	UInt32 inNumberFrames,
	AudioBufferList *ioData);
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

// CoreAudioCapture implements AudioCapture using CoreAudio
type CoreAudioCapture struct {
	audioUnit C.AudioUnit
	format    AudioFormat
	buffer    []int16
	bufferPos int
	open      bool
	mutex     sync.Mutex
	cond      *sync.Cond
}

// CoreAudioPlayback implements AudioPlayback using CoreAudio
type CoreAudioPlayback struct {
	audioUnit C.AudioUnit
	format    AudioFormat
	buffer    []int16
	bufferPos int
	open      bool
	mutex     sync.Mutex
	cond      *sync.Cond
}

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
	capture := &CoreAudioCapture{
		format: format,
		buffer: make([]int16, format.SampleRate), // 1 second buffer
	}
	capture.cond = sync.NewCond(&capture.mutex)

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

	return capture, nil
}

func newPlaybackImpl(deviceID int, format AudioFormat) (AudioPlayback, error) {
	playback := &CoreAudioPlayback{
		format: format,
		buffer: make([]int16, format.SampleRate), // 1 second buffer
	}
	playback.cond = sync.NewCond(&playback.mutex)

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

	c.open = false
	c.cond.Broadcast()
	return nil
}

func (c *CoreAudioCapture) Read(buffer []int16) (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.open {
		return 0, ErrDeviceNotOpen
	}

	// For now, return silence - full implementation would use callback
	// This is a simplified version
	for i := range buffer {
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

	p.open = false
	p.cond.Broadcast()
	return nil
}

func (p *CoreAudioPlayback) Write(buffer []int16) (int, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.open {
		return 0, ErrDeviceNotOpen
	}

	// For now, just consume the buffer - full implementation would use callback
	// This is a simplified version
	return len(buffer), nil
}

func (p *CoreAudioPlayback) IsOpen() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.open
}

func (p *CoreAudioPlayback) GetFormat() AudioFormat {
	return p.format
}
