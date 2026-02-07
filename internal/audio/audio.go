package audio

import "errors"

var (
	// ErrNoDevice indicates no audio device found
	ErrNoDevice = errors.New("no audio device found")
	// ErrDeviceNotOpen indicates device is not open
	ErrDeviceNotOpen = errors.New("audio device is not open")
	// ErrDeviceBusy indicates device is already in use
	ErrDeviceBusy = errors.New("audio device is busy or in use")
)

// AudioFormat represents audio format parameters
type AudioFormat struct {
	SampleRate int // Samples per second (e.g., 48000, 16000, 8000)
	Channels   int // Number of channels (1=mono, 2=stereo)
	BitDepth   int // Bits per sample (8, 16, 24, 32)
}

// DefaultFormat returns a sensible default audio format for VoIP
func DefaultFormat() AudioFormat {
	return AudioFormat{
		SampleRate: 16000, // 16 kHz (good for voice, low bandwidth)
		Channels:   1,     // Mono
		BitDepth:   16,    // 16-bit
	}
}

// FrameSize returns the size of one audio frame in bytes
func (f AudioFormat) FrameSize() int {
	return f.Channels * (f.BitDepth / 8)
}

// BytesPerSecond returns bandwidth in bytes per second
func (f AudioFormat) BytesPerSecond() int {
	return f.SampleRate * f.FrameSize()
}

// AudioCapture represents an audio input device
type AudioCapture interface {
	// Open opens the audio device for capture
	Open() error

	// Close closes the audio device
	Close() error

	// Read reads audio samples into buffer
	// Returns number of frames read
	Read(buffer []int16) (int, error)

	// IsOpen returns true if device is open
	IsOpen() bool

	// GetFormat returns the audio format
	GetFormat() AudioFormat
}

// AudioPlayback represents an audio output device
type AudioPlayback interface {
	// Open opens the audio device for playback
	Open() error

	// Close closes the audio device
	Close() error

	// Write writes audio samples from buffer
	// Returns number of frames written
	Write(buffer []int16) (int, error)

	// IsOpen returns true if device is open
	IsOpen() bool

	// GetFormat returns the audio format
	GetFormat() AudioFormat
}

// DeviceInfo contains audio device information
type DeviceInfo struct {
	ID          int
	Name        string
	Type        string // "capture" or "playback"
	IsDefault   bool
	SampleRates []int
	Channels    []int
}

// ListCaptureDevices returns a list of available capture devices
func ListCaptureDevices() ([]DeviceInfo, error) {
	return listCaptureDevicesImpl()
}

// ListPlaybackDevices returns a list of available playback devices
func ListPlaybackDevices() ([]DeviceInfo, error) {
	return listPlaybackDevicesImpl()
}

// NewCapture creates a new audio capture device
func NewCapture(deviceID int, format AudioFormat) (AudioCapture, error) {
	return newCaptureImpl(deviceID, format)
}

// NewPlayback creates a new audio playback device
func NewPlayback(deviceID int, format AudioFormat) (AudioPlayback, error) {
	return newPlaybackImpl(deviceID, format)
}

// NewDefaultCapture creates a capture device with default settings
func NewDefaultCapture() (AudioCapture, error) {
	return NewCapture(0, DefaultFormat())
}

// NewDefaultPlayback creates a playback device with default settings
func NewDefaultPlayback() (AudioPlayback, error) {
	return NewPlayback(0, DefaultFormat())
}
