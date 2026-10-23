//go:build linux

package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/yobert/alsa"
)

// alsaCaptureDevice is the part of *alsa.Device that ALSACapture uses
type alsaCaptureDevice interface {
	Read(buf []byte) error
	Close()
}

// ALSACapture implements AudioCapture using ALSA
type ALSACapture struct {
	device alsaCaptureDevice
	format AudioFormat
	open   bool
	mutex  sync.Mutex

	// yobert/alsa's Prepare sets the capture start threshold to the whole
	// buffer, and the kernel only starts a prepared capture stream on a read
	// of at least that many frames. The first Read therefore asks for
	// startFrames and keeps the surplus in pending for the next calls.
	startFrames int
	started     bool
	pending     []int16
}

// alsaPlaybackDevice is the part of *alsa.Device that ALSAPlayback uses
type alsaPlaybackDevice interface {
	Write(buf []byte, frames int) error
	Close()
}

// ALSAPlayback implements AudioPlayback using ALSA
type ALSAPlayback struct {
	device alsaPlaybackDevice
	format AudioFormat
	open   bool
	mutex  sync.Mutex
}

func listCaptureDevicesImpl() ([]DeviceInfo, error) {
	cards, err := alsa.OpenCards()
	if err != nil {
		return nil, fmt.Errorf("failed to open ALSA cards: %w", err)
	}
	defer alsa.CloseCards(cards)

	var devices []DeviceInfo
	deviceID := 0

	for _, card := range cards {
		cardDevices, err := card.Devices()
		if err != nil {
			continue
		}

		for _, dev := range cardDevices {
			if dev.Type != alsa.PCM || !dev.Record {
				continue
			}

			devices = append(devices, DeviceInfo{
				ID:          deviceID,
				Name:        dev.Title,
				Type:        "capture",
				IsDefault:   deviceID == 0,
				SampleRates: []int{8000, 16000, 44100, 48000},
				Channels:    []int{1, 2},
			})
			deviceID++
		}
	}

	return devices, nil
}

func listPlaybackDevicesImpl() ([]DeviceInfo, error) {
	cards, err := alsa.OpenCards()
	if err != nil {
		return nil, fmt.Errorf("failed to open ALSA cards: %w", err)
	}
	defer alsa.CloseCards(cards)

	var devices []DeviceInfo
	deviceID := 0

	for _, card := range cards {
		cardDevices, err := card.Devices()
		if err != nil {
			continue
		}

		for _, dev := range cardDevices {
			if dev.Type != alsa.PCM || !dev.Play {
				continue
			}

			devices = append(devices, DeviceInfo{
				ID:          deviceID,
				Name:        dev.Title,
				Type:        "playback",
				IsDefault:   deviceID == 0,
				SampleRates: []int{8000, 16000, 44100, 48000},
				Channels:    []int{1, 2},
			})
			deviceID++
		}
	}

	return devices, nil
}

func newCaptureImpl(deviceID int, format AudioFormat) (AudioCapture, error) {
	return &ALSACapture{
		format: format,
		open:   false,
	}, nil
}

func newPlaybackImpl(deviceID int, format AudioFormat) (AudioPlayback, error) {
	return &ALSAPlayback{
		format: format,
		open:   false,
	}, nil
}

//nolint:unused // Used in cross-platform tests
func newDefaultCaptureImpl() (AudioCapture, error) {
	return newCaptureImpl(0, DefaultFormat())
}

//nolint:unused // Used in cross-platform tests
func newDefaultPlaybackImpl() (AudioPlayback, error) {
	return newPlaybackImpl(0, DefaultFormat())
}

// ALSACapture implementation

func (a *ALSACapture) Open() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.open {
		return ErrDeviceBusy
	}

	cards, err := alsa.OpenCards()
	if err != nil {
		return fmt.Errorf("failed to open ALSA cards: %w", err)
	}
	defer alsa.CloseCards(cards)

	if len(cards) == 0 {
		return ErrNoDevice
	}

	// Find the first capture device
	var captureDevice *alsa.Device
	for _, card := range cards {
		devices, err := card.Devices()
		if err != nil {
			continue
		}

		for _, dev := range devices {
			if dev.Type == alsa.PCM && dev.Record {
				captureDevice = dev
				break
			}
		}
		if captureDevice != nil {
			break
		}
	}

	if captureDevice == nil {
		return ErrNoDevice
	}

	// Open device
	if err := captureDevice.Open(); err != nil {
		return fmt.Errorf("failed to open capture device: %w", err)
	}

	// Negotiate parameters
	_, err = captureDevice.NegotiateChannels(a.format.Channels)
	if err != nil {
		captureDevice.Close()
		return fmt.Errorf("failed to set channels: %w", err)
	}

	_, err = captureDevice.NegotiateRate(a.format.SampleRate)
	if err != nil {
		captureDevice.Close()
		return fmt.Errorf("failed to set sample rate: %w", err)
	}

	_, err = captureDevice.NegotiateFormat(alsa.S16_LE)
	if err != nil {
		captureDevice.Close()
		return fmt.Errorf("failed to set format: %w", err)
	}

	bufferSize := a.format.SampleRate / 50 // 20ms buffer
	_, err = captureDevice.NegotiatePeriodSize(bufferSize)
	if err != nil {
		captureDevice.Close()
		return fmt.Errorf("failed to set period size: %w", err)
	}

	bufferFrames, err := captureDevice.NegotiateBufferSize(bufferSize * 2)
	if err != nil {
		captureDevice.Close()
		return fmt.Errorf("failed to set buffer size: %w", err)
	}

	if err := captureDevice.Prepare(); err != nil {
		captureDevice.Close()
		return fmt.Errorf("failed to prepare device: %w", err)
	}

	a.device = captureDevice
	a.startFrames = bufferFrames
	a.started = false
	a.pending = nil
	a.open = true

	return nil
}

func (a *ALSACapture) Close() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.open {
		return nil
	}

	if a.device != nil {
		a.device.Close()
		a.device = nil
	}

	a.open = false
	return nil
}

func (a *ALSACapture) Read(buffer []int16) (int, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.open {
		return 0, ErrDeviceNotOpen
	}

	// Serve samples left over from the read that started the stream
	n := copy(buffer, a.pending)
	a.pending = a.pending[n:]
	if n == len(buffer) {
		return n, nil
	}

	// Until the stream runs, read at least the start threshold
	count := len(buffer) - n
	if startSamples := a.startFrames * a.format.Channels; !a.started && count < startSamples {
		count = startSamples
	}

	// Read from ALSA device
	samples := make([]byte, count*2) // 2 bytes per int16
	err := a.device.Read(samples)
	if err != nil {
		return 0, fmt.Errorf("failed to read from device: %w", err)
	}
	a.started = true

	// Convert bytes to int16 (little-endian)
	captured := make([]int16, count)
	for i := range captured {
		captured[i] = int16(samples[i*2]) | (int16(samples[i*2+1]) << 8)
	}
	a.pending = captured[copy(buffer[n:], captured):]

	return len(buffer), nil
}

func (a *ALSACapture) IsOpen() bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.open
}

func (a *ALSACapture) GetFormat() AudioFormat {
	return a.format
}

// ALSAPlayback implementation

func (a *ALSAPlayback) Open() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.open {
		return ErrDeviceBusy
	}

	cards, err := alsa.OpenCards()
	if err != nil {
		return fmt.Errorf("failed to open ALSA cards: %w", err)
	}
	defer alsa.CloseCards(cards)

	if len(cards) == 0 {
		return ErrNoDevice
	}

	// Find the first playback device
	var playbackDevice *alsa.Device
	for _, card := range cards {
		devices, err := card.Devices()
		if err != nil {
			continue
		}

		for _, dev := range devices {
			if dev.Type == alsa.PCM && dev.Play {
				playbackDevice = dev
				break
			}
		}
		if playbackDevice != nil {
			break
		}
	}

	if playbackDevice == nil {
		return ErrNoDevice
	}

	// Open device
	if err := playbackDevice.Open(); err != nil {
		return fmt.Errorf("failed to open playback device: %w", err)
	}

	// Negotiate parameters
	_, err = playbackDevice.NegotiateChannels(a.format.Channels)
	if err != nil {
		playbackDevice.Close()
		return fmt.Errorf("failed to set channels: %w", err)
	}

	_, err = playbackDevice.NegotiateRate(a.format.SampleRate)
	if err != nil {
		playbackDevice.Close()
		return fmt.Errorf("failed to set sample rate: %w", err)
	}

	_, err = playbackDevice.NegotiateFormat(alsa.S16_LE)
	if err != nil {
		playbackDevice.Close()
		return fmt.Errorf("failed to set format: %w", err)
	}

	bufferSize := a.format.SampleRate / 50 // 20ms buffer
	_, err = playbackDevice.NegotiatePeriodSize(bufferSize)
	if err != nil {
		playbackDevice.Close()
		return fmt.Errorf("failed to set period size: %w", err)
	}

	_, err = playbackDevice.NegotiateBufferSize(bufferSize * 2)
	if err != nil {
		playbackDevice.Close()
		return fmt.Errorf("failed to set buffer size: %w", err)
	}

	if err := playbackDevice.Prepare(); err != nil {
		playbackDevice.Close()
		return fmt.Errorf("failed to prepare device: %w", err)
	}

	a.device = playbackDevice
	a.open = true

	return nil
}

func (a *ALSAPlayback) Close() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.open {
		return nil
	}

	if a.device != nil {
		a.device.Close()
		a.device = nil
	}

	a.open = false
	return nil
}

func (a *ALSAPlayback) Write(buffer []int16) (int, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.open {
		return 0, ErrDeviceNotOpen
	}

	// The device counts frames (one sample per channel), not samples, and
	// reads frames*channels samples from the buffer
	channels := a.format.Channels
	if channels < 1 {
		channels = 1
	}
	frames := len(buffer) / channels
	if frames == 0 {
		return 0, nil
	}
	samples := frames * channels

	// Convert int16 to bytes (little-endian) using bytes.Buffer
	var buf bytes.Buffer
	for _, sample := range buffer[:samples] {
		_ = binary.Write(&buf, binary.LittleEndian, sample)
	}

	// Write to ALSA device
	err := a.device.Write(buf.Bytes(), frames)
	if err != nil {
		return 0, fmt.Errorf("failed to write to device: %w", err)
	}

	return samples, nil
}

func (a *ALSAPlayback) IsOpen() bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.open
}

func (a *ALSAPlayback) GetFormat() AudioFormat {
	return a.format
}
