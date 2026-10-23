package main

import (
	"errors"
	"testing"

	"github.com/svend4/infon/internal/audio"
	"github.com/svend4/infon/internal/device"
)

// stubGroupDevices makes openGroupDevices see the given cameras and no sound
// card, so the tests never open the hardware of the machine they run on.
func stubGroupDevices(t *testing.T, cameras []device.CameraInfo, newCamera func(int) (device.Camera, error)) {
	t.Helper()
	l, c, cp, pb := groupListCameras, groupNewCamera, groupNewCapture, groupNewPlayback
	t.Cleanup(func() { groupListCameras, groupNewCamera, groupNewCapture, groupNewPlayback = l, c, cp, pb })
	noSound := errors.New("no sound card")
	groupListCameras = func() ([]device.CameraInfo, error) { return cameras, nil }
	groupNewCamera = newCamera
	groupNewCapture = func() (audio.AudioCapture, error) { return nil, noSound }
	groupNewPlayback = func() (audio.AudioPlayback, error) { return nil, noSound }
}

// `tvcp group` must start on a host with no camera and no sound card, as `tvcp
// call` does: it falls back to a test camera and runs without audio, instead of
// failing with "failed to open camera" / "failed to initialize microphone".
func TestGroupCallDevicesFallBackOnHeadlessHost(t *testing.T) {
	stubGroupDevices(t, nil, func(int) (device.Camera, error) {
		t.Error("opened a camera although none was listed")
		return nil, errors.New("no camera")
	})

	cam, mic, speaker, err := openGroupDevices()
	if err != nil {
		t.Fatalf("openGroupDevices: %v", err)
	}
	defer func() { _ = cam.Close() }()
	if mic != nil || speaker != nil {
		t.Errorf("mic, speaker = %v, %v; want nil without a sound card", mic, speaker)
	}

	if !cam.IsOpen() {
		t.Fatal("camera is not open")
	}
	img, err := cam.Read()
	if err != nil || img == nil {
		t.Fatalf("camera Read: %v, %v", img, err)
	}
}

// A camera that is listed but fails to open also falls back to the test camera.
func TestGroupCallFallsBackWhenCameraFailsToOpen(t *testing.T) {
	stubGroupDevices(t, []device.CameraInfo{{Name: "busy cam"}}, func(int) (device.Camera, error) {
		return nil, errors.New("device busy")
	})

	cam, _, _, err := openGroupDevices()
	if err != nil {
		t.Fatalf("openGroupDevices: %v", err)
	}
	defer func() { _ = cam.Close() }()
	if _, ok := cam.(*device.TestCamera); !ok {
		t.Errorf("camera = %T, want the test camera fallback", cam)
	}
}
