//go:build linux

package audio

import (
	"encoding/binary"
	"syscall"
	"testing"
)

// fakeCaptureDevice models the kernel's SNDRV_PCM_IOCTL_READI_FRAMES rule
// (sound/core/pcm_lib.c): a prepared capture stream only starts when a read
// asks for at least start_threshold frames; a smaller read waits for a start
// that never comes and fails with EIO once wait_for_avail times out.
type fakeCaptureDevice struct {
	startThreshold int // frames; yobert/alsa Prepare sets this to the buffer size
	started        bool
	next           int16
}

func (d *fakeCaptureDevice) Read(buf []byte) error {
	frames := len(buf) / 2 // mono S16_LE
	if !d.started && frames < d.startThreshold {
		return syscall.EIO
	}
	d.started = true
	for i := 0; i < frames; i++ {
		binary.LittleEndian.PutUint16(buf[2*i:], uint16(d.next))
		d.next++
	}
	return nil
}

func (d *fakeCaptureDevice) Close() {}

// Reading one 20ms period at a time from a freshly prepared capture device
// (buffer = two periods = start threshold) must deliver a continuous stream.
func TestALSACaptureReadPeriodBelowStartThreshold(t *testing.T) {
	format := DefaultFormat()
	period := format.SampleRate / 50
	dev := &fakeCaptureDevice{startThreshold: 2 * period}
	c := &ALSACapture{device: dev, format: format, open: true, startFrames: 2 * period}

	var want int16
	buf := make([]int16, period)
	for call := 0; call < 5; call++ {
		n, err := c.Read(buf)
		if err != nil {
			t.Fatalf("read %d: %v", call, err)
		}
		if n != period {
			t.Fatalf("read %d: n = %d, want %d", call, n, period)
		}
		for i, s := range buf[:n] {
			if s != want {
				t.Fatalf("read %d sample %d = %d, want %d", call, i, s, want)
			}
			want++
		}
	}
}
