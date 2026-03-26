// +build linux

package device

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// V4L2Camera implements Camera interface using Video4Linux2
type V4L2Camera struct {
	devicePath string
	width      int
	height     int
	fps        float64
	fd         int
	isOpen     bool
	buffers    [][]byte
	pixelFormat uint32
}

// V4L2 ioctl constants
const (
	VIDIOC_QUERYCAP  = 0x80685600
	VIDIOC_S_FMT     = 0xc0d05605
	VIDIOC_G_FMT     = 0xc0d05604
	VIDIOC_REQBUFS   = 0xc0145608
	VIDIOC_QUERYBUF  = 0xc0445609
	VIDIOC_QBUF      = 0xc044560f
	VIDIOC_DQBUF     = 0xc0445611
	VIDIOC_STREAMON  = 0x40045612
	VIDIOC_STREAMOFF = 0x40045613

	V4L2_BUF_TYPE_VIDEO_CAPTURE = 1
	V4L2_MEMORY_MMAP           = 1
	V4L2_FIELD_NONE            = 1

	// Pixel formats
	V4L2_PIX_FMT_YUYV = 0x56595559 // YUYV 4:2:2
	V4L2_PIX_FMT_MJPEG = 0x47504a4d // Motion-JPEG
)

// v4l2_capability structure
type v4l2Capability struct {
	driver       [16]byte
	card         [32]byte
	bus_info     [32]byte
	version      uint32
	capabilities uint32
	device_caps  uint32
	reserved     [3]uint32
}

// v4l2_format structure
type v4l2Format struct {
	typ  uint32
	data [200]byte
}

// v4l2_pix_format structure
type v4l2PixFormat struct {
	width        uint32
	height       uint32
	pixelformat  uint32
	field        uint32
	bytesperline uint32
	sizeimage    uint32
	colorspace   uint32
	priv         uint32
}

// v4l2_requestbuffers structure
type v4l2RequestBuffers struct {
	count  uint32
	typ    uint32
	memory uint32
	_      [2]uint32
}

// v4l2_buffer structure
type v4l2Buffer struct {
	index     uint32
	typ       uint32
	bytesused uint32
	flags     uint32
	field     uint32
	timestamp syscall.Timeval
	timecode  [4]uint32
	sequence  uint32
	memory    uint32
	offset    uint32
	length    uint32
	_         [2]uint32
}

// NewV4L2Camera creates a new V4L2 camera instance
func NewV4L2Camera(devicePath string, width, height int, fps float64) *V4L2Camera {
	return &V4L2Camera{
		devicePath: devicePath,
		width:      width,
		height:     height,
		fps:        fps,
		fd:         -1,
		isOpen:     false,
	}
}

// Open opens the camera device
func (c *V4L2Camera) Open() error {
	if c.isOpen {
		return errors.New("camera already open")
	}

	// Open device
	fd, err := syscall.Open(c.devicePath, syscall.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open device %s: %w", c.devicePath, err)
	}
	c.fd = fd

	// Query capabilities
	var cap v4l2Capability
	if err := c.ioctl(VIDIOC_QUERYCAP, unsafe.Pointer(&cap)); err != nil {
		syscall.Close(c.fd)
		return fmt.Errorf("failed to query capabilities: %w", err)
	}

	// Set format
	var v4l2Fmt v4l2Format
	v4l2Fmt.typ = V4L2_BUF_TYPE_VIDEO_CAPTURE

	// Get current format
	if err := c.ioctl(VIDIOC_G_FMT, unsafe.Pointer(&v4l2Fmt)); err != nil {
		syscall.Close(c.fd)
		return fmt.Errorf("failed to get format: %w", err)
	}

	// Set desired format
	pix := (*v4l2PixFormat)(unsafe.Pointer(&v4l2Fmt.data[0]))
	pix.width = uint32(c.width)
	pix.height = uint32(c.height)
	pix.pixelformat = V4L2_PIX_FMT_YUYV // Use YUYV format
	pix.field = V4L2_FIELD_NONE

	if err := c.ioctl(VIDIOC_S_FMT, unsafe.Pointer(&v4l2Fmt)); err != nil {
		syscall.Close(c.fd)
		return fmt.Errorf("failed to set format: %w", err)
	}

	c.pixelFormat = pix.pixelformat

	// Request buffers
	var req v4l2RequestBuffers
	req.count = 4
	req.typ = V4L2_BUF_TYPE_VIDEO_CAPTURE
	req.memory = V4L2_MEMORY_MMAP

	if err := c.ioctl(VIDIOC_REQBUFS, unsafe.Pointer(&req)); err != nil {
		syscall.Close(c.fd)
		return fmt.Errorf("failed to request buffers: %w", err)
	}

	// Map buffers
	c.buffers = make([][]byte, req.count)
	for i := uint32(0); i < req.count; i++ {
		var buf v4l2Buffer
		buf.index = i
		buf.typ = V4L2_BUF_TYPE_VIDEO_CAPTURE
		buf.memory = V4L2_MEMORY_MMAP

		if err := c.ioctl(VIDIOC_QUERYBUF, unsafe.Pointer(&buf)); err != nil {
			c.Close()
			return fmt.Errorf("failed to query buffer %d: %w", i, err)
		}

		// Memory map the buffer
		data, err := syscall.Mmap(c.fd, int64(buf.offset), int(buf.length),
			syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			c.Close()
			return fmt.Errorf("failed to mmap buffer %d: %w", i, err)
		}
		c.buffers[i] = data

		// Queue buffer
		if err := c.ioctl(VIDIOC_QBUF, unsafe.Pointer(&buf)); err != nil {
			c.Close()
			return fmt.Errorf("failed to queue buffer %d: %w", i, err)
		}
	}

	// Start streaming
	bufType := uint32(V4L2_BUF_TYPE_VIDEO_CAPTURE)
	if err := c.ioctl(VIDIOC_STREAMON, unsafe.Pointer(&bufType)); err != nil {
		c.Close()
		return fmt.Errorf("failed to start streaming: %w", err)
	}

	c.isOpen = true
	return nil
}

// Read captures a frame from the camera
func (c *V4L2Camera) Read() (image.Image, error) {
	if !c.isOpen {
		return nil, errors.New("camera not open")
	}

	// Dequeue buffer
	var buf v4l2Buffer
	buf.typ = V4L2_BUF_TYPE_VIDEO_CAPTURE
	buf.memory = V4L2_MEMORY_MMAP

	if err := c.ioctl(VIDIOC_DQBUF, unsafe.Pointer(&buf)); err != nil {
		return nil, fmt.Errorf("failed to dequeue buffer: %w", err)
	}

	// Convert buffer to image
	img := c.decodeFrame(c.buffers[buf.index][:buf.bytesused])

	// Requeue buffer
	if err := c.ioctl(VIDIOC_QBUF, unsafe.Pointer(&buf)); err != nil {
		return nil, fmt.Errorf("failed to requeue buffer: %w", err)
	}

	return img, nil
}

// Close closes the camera
func (c *V4L2Camera) Close() error {
	if !c.isOpen {
		return nil
	}

	// Stop streaming
	bufType := uint32(V4L2_BUF_TYPE_VIDEO_CAPTURE)
	c.ioctl(VIDIOC_STREAMOFF, unsafe.Pointer(&bufType))

	// Unmap buffers
	for _, buf := range c.buffers {
		if buf != nil {
			syscall.Munmap(buf)
		}
	}
	c.buffers = nil

	// Close device
	if c.fd >= 0 {
		syscall.Close(c.fd)
		c.fd = -1
	}

	c.isOpen = false
	return nil
}

// IsOpen returns whether the camera is open
func (c *V4L2Camera) IsOpen() bool {
	return c.isOpen
}

// GetWidth returns camera width
func (c *V4L2Camera) GetWidth() int {
	return c.width
}

// GetHeight returns camera height
func (c *V4L2Camera) GetHeight() int {
	return c.height
}

// GetFPS returns camera FPS
func (c *V4L2Camera) GetFPS() float64 {
	return c.fps
}

// SetResolution sets the camera resolution
func (c *V4L2Camera) SetResolution(width, height int) error {
	if c.isOpen {
		return errors.New("cannot change resolution while camera is open")
	}
	c.width = width
	c.height = height
	return nil
}

// SetFPS sets the camera FPS
func (c *V4L2Camera) SetFPS(fps float64) error {
	if c.isOpen {
		return errors.New("cannot change FPS while camera is open")
	}
	c.fps = fps
	return nil
}

// ioctl performs ioctl system call
func (c *V4L2Camera) ioctl(request uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(c.fd), request, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

// decodeFrame converts YUYV frame to RGB image
func (c *V4L2Camera) decodeFrame(data []byte) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, c.width, c.height))

	// YUYV format: Y0 U0 Y1 V0 (2 pixels per 4 bytes)
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x += 2 {
			offset := (y*c.width + x) * 2

			if offset+3 >= len(data) {
				break
			}

			y0 := int(data[offset+0])
			u := int(data[offset+1])
			y1 := int(data[offset+2])
			v := int(data[offset+3])

			// Convert YUV to RGB for pixel 0
			r0, g0, b0 := yuv2rgb(y0, u, v)
			img.SetRGBA(x, y, color.RGBA{R: r0, G: g0, B: b0, A: 255})

			// Convert YUV to RGB for pixel 1
			if x+1 < c.width {
				r1, g1, b1 := yuv2rgb(y1, u, v)
				img.SetRGBA(x+1, y, color.RGBA{R: r1, G: g1, B: b1, A: 255})
			}
		}
	}

	return img
}

// yuv2rgb converts YUV color to RGB
func yuv2rgb(y, u, v int) (uint8, uint8, uint8) {
	// YUV to RGB conversion
	c := y - 16
	d := u - 128
	e := v - 128

	r := clamp((298*c + 409*e + 128) >> 8)
	g := clamp((298*c - 100*d - 208*e + 128) >> 8)
	b := clamp((298*c + 516*d + 128) >> 8)

	return uint8(r), uint8(g), uint8(b)
}

// clamp restricts value to 0-255 range
func clamp(val int) int {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return val
}

// ListV4L2Cameras lists available V4L2 camera devices
func ListV4L2Cameras() ([]string, error) {
	var cameras []string

	// Check /dev/video* devices
	matches, err := filepath.Glob("/dev/video*")
	if err != nil {
		return nil, err
	}

	for _, device := range matches {
		// Try to open device to verify it's a valid camera
		fd, err := syscall.Open(device, syscall.O_RDONLY, 0)
		if err != nil {
			continue
		}

		var cap v4l2Capability
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QUERYCAP, uintptr(unsafe.Pointer(&cap)))
		syscall.Close(fd)

		if errno == 0 {
			// Device is a valid V4L2 device
			cardName := strings.TrimRight(string(cap.card[:]), "\x00")
			cameras = append(cameras, fmt.Sprintf("%s: %s", device, cardName))
		}
	}

	return cameras, nil
}

// GetDefaultV4L2Camera returns the first available camera device
func GetDefaultV4L2Camera() (string, error) {
	// Check for /dev/video0 first
	if _, err := os.Stat("/dev/video0"); err == nil {
		return "/dev/video0", nil
	}

	// Try to find any /dev/video* device
	matches, err := filepath.Glob("/dev/video*")
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", errors.New("no camera devices found")
	}

	return matches[0], nil
}
