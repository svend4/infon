package device

import (
	"errors"
	"image"
)

var (
	// ErrNoCamera indicates no camera device found
	ErrNoCamera = errors.New("no camera device found")
	// ErrCameraNotOpen indicates camera is not open
	ErrCameraNotOpen = errors.New("camera is not open")
	// ErrCameraBusy indicates camera is already in use
	ErrCameraBusy = errors.New("camera is busy or in use by another process")
)

// Camera represents a video capture device
type Camera interface {
	// Open opens the camera device
	Open() error

	// Close closes the camera device
	Close() error

	// Read reads a frame from the camera
	Read() (image.Image, error)

	// IsOpen returns true if camera is open
	IsOpen() bool

	// GetWidth returns the frame width
	GetWidth() int

	// GetHeight returns the frame height
	GetHeight() int

	// GetFPS returns the camera FPS
	GetFPS() float64

	// SetResolution sets the desired resolution
	SetResolution(width, height int) error

	// SetFPS sets the desired FPS
	SetFPS(fps float64) error
}

// CameraInfo contains camera device information
type CameraInfo struct {
	ID          int
	Name        string
	Path        string
	Available   bool
	Resolutions []Resolution
}

// Resolution represents a supported video resolution
type Resolution struct {
	Width  int
	Height int
}

// ListCameras returns a list of available camera devices
func ListCameras() ([]CameraInfo, error) {
	// Platform-specific implementation
	return listCamerasImpl()
}

// NewCamera creates a new camera instance for the specified device
func NewCamera(deviceID int) (Camera, error) {
	return newCameraImpl(deviceID)
}

// NewDefaultCamera creates a camera instance for the default device (usually /dev/video0 or device 0)
func NewDefaultCamera() (Camera, error) {
	return NewCamera(0)
}
