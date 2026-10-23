//go:build linux

package device

import (
	"testing"
)

// V4L2 capability bits from <linux/videodev2.h>.
const (
	testCapVideoCapture = 0x00000001
	testCapMetaCapture  = 0x00800000
	testCapStreaming    = 0x04000000
	testCapDeviceCaps   = 0x80000000
)

func testCap(card string, devCaps uint32) v4l2Capability {
	var c v4l2Capability
	copy(c.card[:], card)
	// A UVC driver advertises the union of its nodes in capabilities and the
	// node's own abilities in device_caps.
	c.capabilities = testCapVideoCapture | testCapMetaCapture | testCapStreaming | testCapDeviceCaps
	c.device_caps = devCaps
	return c
}

// ListCameras must describe each camera by its device number, not its position
// in the list: NewCamera(ID) opens /dev/video<ID>. /dev/video* globs lexically
// (video10 before video2), nodes that cannot be opened are skipped, and a UVC
// webcam's metadata node answers QUERYCAP but cannot capture.
func TestV4L2CameraListUsesDeviceNumbers(t *testing.T) {
	capture := uint32(testCapVideoCapture | testCapStreaming)
	meta := uint32(testCapMetaCapture | testCapStreaming)
	caps := map[string]v4l2Capability{
		"/dev/video0":  testCap("Integrated Camera", capture),
		"/dev/video1":  testCap("Integrated Camera", meta),
		"/dev/video10": testCap("USB Camera", capture),
		"/dev/video11": testCap("USB Camera", meta),
		"/dev/video2":  testCap("Capture Card", capture),
		// /dev/video3 cannot be opened (permissions) and is absent from caps.
	}
	glob := []string{"/dev/video0", "/dev/video1", "/dev/video10", "/dev/video11", "/dev/video2", "/dev/video3", "/dev/video-legacy"}
	query := func(path string) (v4l2Capability, bool) {
		c, ok := caps[path]
		return c, ok
	}

	cams := camerasFromNodes(probeV4L2Nodes(glob, query))

	want := []struct {
		id   int
		path string
	}{{0, "/dev/video0"}, {2, "/dev/video2"}, {10, "/dev/video10"}}
	if len(cams) != len(want) {
		t.Fatalf("got %d cameras %+v, want %d", len(cams), cams, len(want))
	}
	for i, w := range want {
		c := cams[i]
		if c.ID != w.id || c.Path != w.path {
			t.Errorf("camera %d = ID %d path %q, want ID %d path %q", i, c.ID, c.Path, w.id, w.path)
		}
		cam, err := newCameraImpl(c.ID)
		if err != nil {
			t.Fatalf("newCameraImpl(%d): %v", c.ID, err)
		}
		if got := cam.(*V4L2Camera).devicePath; got != c.Path {
			t.Errorf("NewCamera(%d) opens %q, listed as %q", c.ID, got, c.Path)
		}
	}
}

// Drivers without per-node caps (no V4L2_CAP_DEVICE_CAPS) report the node's
// abilities in capabilities.
func TestV4L2CaptureCheckWithoutDeviceCaps(t *testing.T) {
	var c v4l2Capability
	c.capabilities = testCapVideoCapture
	if !c.isVideoCapture() {
		t.Error("capture node without DEVICE_CAPS not recognised")
	}
	c.capabilities = testCapMetaCapture
	if c.isVideoCapture() {
		t.Error("metadata node without DEVICE_CAPS listed as a camera")
	}
}
