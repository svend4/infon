//go:build linux

package device

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// V4L2Camera implements Camera interface using Video4Linux2
//
// Read, Close and IsOpen are serialised by mu, so a Close at hang-up cannot unmap
// the buffers while a Read is between DQBUF and QBUF.
type V4L2Camera struct {
	mu         sync.Mutex
	devicePath string
	width      int
	height     int
	fps        float64
	fd         int
	isOpen     bool
	buffers    [][]byte
	pixelFormat uint32
}

// ioctl request encoding (asm-generic layout used by amd64, arm64, 386, arm,
// riscv64, ...): dir<<30 | size<<16 | type<<8 | nr, as built by the kernel's
// _IOR/_IOW/_IOWR macros.
const (
	iocWrite     = 1 << 30
	iocRead      = 2 << 30
	iocSizeShift = 16
	iocTypeV4L2  = 'V' << 8
)

// V4L2 ioctl constants. The size field is taken from the Go struct passed with
// each request, so the numbers follow the struct layouts on 32- and 64-bit.
const (
	VIDIOC_QUERYCAP  = iocRead | unsafe.Sizeof(v4l2Capability{})<<iocSizeShift | iocTypeV4L2 | 0
	VIDIOC_S_FMT     = iocRead | iocWrite | unsafe.Sizeof(v4l2Format{})<<iocSizeShift | iocTypeV4L2 | 5
	VIDIOC_G_FMT     = iocRead | iocWrite | unsafe.Sizeof(v4l2Format{})<<iocSizeShift | iocTypeV4L2 | 4
	VIDIOC_REQBUFS   = iocRead | iocWrite | unsafe.Sizeof(v4l2RequestBuffers{})<<iocSizeShift | iocTypeV4L2 | 8
	VIDIOC_QUERYBUF  = iocRead | iocWrite | unsafe.Sizeof(v4l2Buffer{})<<iocSizeShift | iocTypeV4L2 | 9
	VIDIOC_QBUF      = iocRead | iocWrite | unsafe.Sizeof(v4l2Buffer{})<<iocSizeShift | iocTypeV4L2 | 15
	VIDIOC_DQBUF     = iocRead | iocWrite | unsafe.Sizeof(v4l2Buffer{})<<iocSizeShift | iocTypeV4L2 | 17
	VIDIOC_STREAMON  = iocWrite | unsafe.Sizeof(int32(0))<<iocSizeShift | iocTypeV4L2 | 18
	VIDIOC_STREAMOFF = iocWrite | unsafe.Sizeof(int32(0))<<iocSizeShift | iocTypeV4L2 | 19

	V4L2_CAP_VIDEO_CAPTURE = 0x00000001
	V4L2_CAP_DEVICE_CAPS   = 0x80000000

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

// v4l2_format structure. The fmt union contains pointers (v4l2_window), so it
// is pointer-aligned: offset 8 and 208 bytes in total on 64-bit, offset 4 and
// 204 bytes on 32-bit.
type v4l2Format struct {
	typ  uint32
	_    [0]uintptr // align the fmt union like the C compiler does
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
	_         [0]uintptr                          // the m union is pointer-aligned...
	offset    uint32                              // m.offset
	_         [unsafe.Sizeof(uintptr(0)) - 4]byte // ...and pointer-sized
	length    uint32
	_         [2]uint32 // reserved2, request_fd
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
		c.release()
		return fmt.Errorf("failed to query capabilities: %w", err)
	}

	// Set format
	var v4l2Fmt v4l2Format
	v4l2Fmt.typ = V4L2_BUF_TYPE_VIDEO_CAPTURE

	// Get current format
	if err := c.ioctl(VIDIOC_G_FMT, unsafe.Pointer(&v4l2Fmt)); err != nil {
		c.release()
		return fmt.Errorf("failed to get format: %w", err)
	}

	// Set desired format
	pix := (*v4l2PixFormat)(unsafe.Pointer(&v4l2Fmt.data[0]))
	pix.width = uint32(c.width)
	pix.height = uint32(c.height)
	pix.pixelformat = V4L2_PIX_FMT_YUYV // Use YUYV format
	pix.field = V4L2_FIELD_NONE

	if err := c.ioctl(VIDIOC_S_FMT, unsafe.Pointer(&v4l2Fmt)); err != nil {
		c.release()
		return fmt.Errorf("failed to set format: %w", err)
	}

	c.pixelFormat = pix.pixelformat

	// Request buffers
	var req v4l2RequestBuffers
	req.count = 4
	req.typ = V4L2_BUF_TYPE_VIDEO_CAPTURE
	req.memory = V4L2_MEMORY_MMAP

	if err := c.ioctl(VIDIOC_REQBUFS, unsafe.Pointer(&req)); err != nil {
		c.release()
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
			_ = c.Close()
			return fmt.Errorf("failed to query buffer %d: %w", i, err)
		}

		// Memory map the buffer
		data, err := syscall.Mmap(c.fd, int64(buf.offset), int(buf.length),
			syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			_ = c.Close()
			return fmt.Errorf("failed to mmap buffer %d: %w", i, err)
		}
		c.buffers[i] = data

		// Queue buffer
		if err := c.ioctl(VIDIOC_QBUF, unsafe.Pointer(&buf)); err != nil {
			_ = c.Close()
			return fmt.Errorf("failed to queue buffer %d: %w", i, err)
		}
	}

	// Start streaming
	bufType := uint32(V4L2_BUF_TYPE_VIDEO_CAPTURE)
	if err := c.ioctl(VIDIOC_STREAMON, unsafe.Pointer(&bufType)); err != nil {
		_ = c.Close()
		return fmt.Errorf("failed to start streaming: %w", err)
	}

	c.isOpen = true
	return nil
}

// Read captures a frame from the camera
func (c *V4L2Camera) Read() (image.Image, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
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

	if int(buf.index) >= len(c.buffers) || int(buf.bytesused) > len(c.buffers[buf.index]) {
		_ = c.ioctl(VIDIOC_QBUF, unsafe.Pointer(&buf))
		return nil, fmt.Errorf("driver returned buffer %d (%d bytes) outside the mapped buffers", buf.index, buf.bytesused)
	}

	// Convert buffer to image
	img := c.decodeFrame(c.buffers[buf.index][:buf.bytesused])

	// Requeue buffer
	if err := c.ioctl(VIDIOC_QBUF, unsafe.Pointer(&buf)); err != nil {
		return nil, fmt.Errorf("failed to requeue buffer: %w", err)
	}

	return img, nil
}

// Close closes the camera. It also releases what a failed Open had acquired:
// Open's error paths call it before isOpen is set.
func (c *V4L2Camera) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isOpen {
		// Stop streaming
		bufType := uint32(V4L2_BUF_TYPE_VIDEO_CAPTURE)
		_ = c.ioctl(VIDIOC_STREAMOFF, unsafe.Pointer(&bufType))
	}

	c.release()
	c.isOpen = false
	return nil
}

// release unmaps the buffers and closes the device, if held.
func (c *V4L2Camera) release() {
	// Unmap buffers
	for _, buf := range c.buffers {
		if buf != nil {
			_ = syscall.Munmap(buf)
		}
	}
	c.buffers = nil

	// Close device
	if c.fd >= 0 {
		_ = syscall.Close(c.fd)
		c.fd = -1
	}
}

// IsOpen returns whether the camera is open
func (c *V4L2Camera) IsOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
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
	nodes, err := scanV4L2Nodes()
	if err != nil {
		return nil, err
	}

	var cameras []string
	for _, n := range nodes {
		cameras = append(cameras, fmt.Sprintf("%s: %s", n.path, n.card))
	}
	return cameras, nil
}

// v4l2Node is a /dev/videoN node that can capture video.
type v4l2Node struct {
	path string
	num  int // N in /dev/videoN: NewCamera(N) opens this node
	card string
}

// scanV4L2Nodes finds the video capture nodes among /dev/video*.
func scanV4L2Nodes() ([]v4l2Node, error) {
	matches, err := filepath.Glob("/dev/video*")
	if err != nil {
		return nil, err
	}
	return probeV4L2Nodes(matches, queryV4L2Cap), nil
}

// probeV4L2Nodes keeps the paths that are /dev/videoN capture nodes, ordered by
// N (the glob is lexical: video10 sorts before video2). query returns a node's
// capabilities, or false if it cannot be opened or is not a V4L2 device.
func probeV4L2Nodes(paths []string, query func(device string) (v4l2Capability, bool)) []v4l2Node {
	var nodes []v4l2Node
	for _, device := range paths {
		num, ok := videoNodeNumber(device)
		if !ok {
			continue
		}
		cap, ok := query(device)
		if !ok || !cap.isVideoCapture() {
			continue
		}
		nodes = append(nodes, v4l2Node{
			path: device,
			num:  num,
			card: strings.TrimRight(string(cap.card[:]), "\x00"),
		})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].num < nodes[j].num })
	return nodes
}

// queryV4L2Cap opens a device to verify it is a V4L2 device and returns its
// capabilities.
func queryV4L2Cap(device string) (v4l2Capability, bool) {
	var cap v4l2Capability
	fd, err := syscall.Open(device, syscall.O_RDONLY, 0)
	if err != nil {
		return cap, false
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), VIDIOC_QUERYCAP, uintptr(unsafe.Pointer(&cap)))
	_ = syscall.Close(fd)
	return cap, errno == 0
}

// isVideoCapture reports whether the node can capture single-planar video, the
// only buffer type V4L2Camera uses. A UVC webcam also has a metadata node that
// answers QUERYCAP but cannot capture.
func (cap *v4l2Capability) isVideoCapture() bool {
	caps := cap.capabilities
	if caps&V4L2_CAP_DEVICE_CAPS != 0 {
		caps = cap.device_caps
	}
	return caps&V4L2_CAP_VIDEO_CAPTURE != 0
}

// videoNodeNumber returns N for a path of the form /dev/videoN.
func videoNodeNumber(path string) (int, bool) {
	digits, ok := strings.CutPrefix(filepath.Base(path), "video")
	if !ok || digits == "" || strings.Trim(digits, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(digits)
	return n, err == nil
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
