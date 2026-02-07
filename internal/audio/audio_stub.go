//go:build !linux && !darwin && !windows

package audio

// Default implementation using test audio sources
// TODO: Platform-specific implementations:
// - audio_linux.go (ALSA)
// - audio_darwin.go (CoreAudio)
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
