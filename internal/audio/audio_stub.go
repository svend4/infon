//go:build (!linux && !darwin && !windows) || (darwin && !cgo)
// +build !linux,!darwin,!windows darwin,!cgo

package audio

// Default implementation using test audio sources
// Used on unsupported platforms or when CGO is not available
// TODO: Platform-specific implementations:
// - audio_linux.go (ALSA)
// - audio_darwin.go (CoreAudio with CGO)
// - audio_windows.go (WASAPI)

func listCaptureDevicesImpl() ([]DeviceInfo, error) {
	// Return empty list - real implementation would enumerate devices
	return []DeviceInfo{}, nil
}

func listPlaybackDevicesImpl() ([]DeviceInfo, error) {
	// Return empty list - real implementation would enumerate devices
	return []DeviceInfo{}, nil
}

func newCaptureImpl(deviceID int, format AudioFormat) (AudioCapture, error) {
	// For now, return test audio source
	return NewTestAudioSource(format), nil
}

func newPlaybackImpl(deviceID int, format AudioFormat) (AudioPlayback, error) {
	// For now, return test audio sink
	return NewTestAudioSink(format), nil
}

func newDefaultCaptureImpl() (AudioCapture, error) {
	return newCaptureImpl(0, DefaultFormat())
}

func newDefaultPlaybackImpl() (AudioPlayback, error) {
	return newPlaybackImpl(0, DefaultFormat())
}
