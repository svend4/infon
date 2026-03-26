// +build linux

package device

import (
	"fmt"
)

// listCamerasImpl lists available cameras on Linux (V4L2)
func listCamerasImpl() ([]CameraInfo, error) {
	devices, err := ListV4L2Cameras()
	if err != nil {
		return nil, err
	}

	var cameras []CameraInfo
	for i, device := range devices {
		cameras = append(cameras, CameraInfo{
			ID:        i,
			Name:      device,
			Path:      fmt.Sprintf("/dev/video%d", i),
			Available: true,
			Resolutions: []Resolution{
				{Width: 640, Height: 480},
				{Width: 1280, Height: 720},
				{Width: 1920, Height: 1080},
			},
		})
	}

	if len(cameras) == 0 {
		return nil, ErrNoCamera
	}

	return cameras, nil
}

// newCameraImpl creates a new camera instance for Linux
func newCameraImpl(deviceID int) (Camera, error) {
	devicePath := fmt.Sprintf("/dev/video%d", deviceID)

	// Default resolution and FPS
	cam := NewV4L2Camera(devicePath, 640, 480, 15.0)

	return cam, nil
}
