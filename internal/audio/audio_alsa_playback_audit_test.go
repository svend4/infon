//go:build linux

package audio

import (
	"encoding/binary"
	"fmt"
	"testing"
)

// fakePlaybackDevice models yobert/alsa Device.Write: frames goes straight
// into SNDRV_PCM_IOCTL_WRITEI_FRAMES with Buf=&buf[0], and the kernel copies
// frames*channels*2 bytes (S16_LE) from that address.
type fakePlaybackDevice struct {
	channels int
	written  []int16
}

func (d *fakePlaybackDevice) Write(buf []byte, frames int) error {
	_ = &buf[0] // as the real device does
	n := frames * d.channels * 2
	if n > len(buf) {
		return fmt.Errorf("kernel would read %d bytes from a %d-byte buffer", n, len(buf))
	}
	for i := 0; i < n; i += 2 {
		d.written = append(d.written, int16(binary.LittleEndian.Uint16(buf[i:])))
	}
	return nil
}

func (d *fakePlaybackDevice) Close() {}

// Write must pass the frame count (samples / channels) to the device, so
// a stereo buffer is written exactly, without reading past its end.
func TestALSAPlaybackAuditWriteFrames(t *testing.T) {
	for _, channels := range []int{1, 2} {
		t.Run(fmt.Sprintf("channels%d", channels), func(t *testing.T) {
			format := AudioFormat{SampleRate: 16000, Channels: channels, BitDepth: 16}
			dev := &fakePlaybackDevice{channels: channels}
			p := &ALSAPlayback{device: dev, format: format, open: true}

			buffer := make([]int16, 320*channels)
			for i := range buffer {
				buffer[i] = int16(i - 300)
			}
			n, err := p.Write(buffer)
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
			if n != len(buffer) {
				t.Fatalf("Write returned %d, want %d samples", n, len(buffer))
			}
			if len(dev.written) != len(buffer) {
				t.Fatalf("device received %d samples, want %d", len(dev.written), len(buffer))
			}
			for i := range buffer {
				if dev.written[i] != buffer[i] {
					t.Fatalf("sample %d = %d, want %d", i, dev.written[i], buffer[i])
				}
			}

			// An empty buffer writes nothing instead of panicking.
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("Write(empty) panicked: %v", r)
					}
				}()
				if n, err := p.Write(nil); n != 0 || err != nil {
					t.Fatalf("Write(empty) = %d, %v; want 0, nil", n, err)
				}
			}()
		})
	}
}
