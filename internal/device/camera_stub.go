// +build !linux

package device

// Stub implementation for non-Linux platforms
// In production, we would have platform-specific implementations:
// - camera_darwin.go (AVFoundation)
// - camera_windows.go (DirectShow/Media Foundation)

func listCamerasImpl() ([]CameraInfo, error) {
	// Return empty list - real implementation would enumerate devices
	return []CameraInfo{}, nil
}

func newCameraImpl(deviceID int) (Camera, error) {
	// For now, return test camera
	// Real implementation would open actual camera device
	return NewTestCamera(640, 480, 30.0, "gradient"), nil
}
