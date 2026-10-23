//go:build linux

package device

import (
	"fmt"
)

// listCamerasImpl lists available cameras on Linux (V4L2)
func listCamerasImpl() ([]CameraInfo, error) {
	nodes, err := scanV4L2Nodes()
	if err != nil {
		return nil, err
	}

	cameras := camerasFromNodes(nodes)
	if len(cameras) == 0 {
		return nil, ErrNoCamera
	}

	return cameras, nil
}

// camerasFromNodes describes V4L2 capture nodes as cameras. ID is the node's
// device number, so NewCamera(ID) opens the node at Path.
func camerasFromNodes(nodes []v4l2Node) []CameraInfo {
	var cameras []CameraInfo
	for _, n := range nodes {
		cameras = append(cameras, CameraInfo{
			ID:        n.num,
			Name:      fmt.Sprintf("%s: %s", n.path, n.card),
			Path:      fmt.Sprintf("/dev/video%d", n.num),
			Available: true,
			Resolutions: []Resolution{
				{Width: 640, Height: 480},
				{Width: 1280, Height: 720},
				{Width: 1920, Height: 1080},
			},
		})
	}
	return cameras
}

// newCameraImpl creates a new camera instance for Linux
func newCameraImpl(deviceID int) (Camera, error) {
	devicePath := fmt.Sprintf("/dev/video%d", deviceID)

	// Default resolution and FPS
	cam := NewV4L2Camera(devicePath, 640, 480, 15.0)

	return cam, nil
}
