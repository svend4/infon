//go:build linux

package device

import (
	"runtime"
	"testing"
	"unsafe"
)

// Expected layouts of the kernel structs, measured with sizeof/offsetof on
// <linux/videodev2.h>. The unions in v4l2_format and v4l2_buffer contain
// pointers, so their alignment (and the struct sizes) follow the pointer size.
type v4l2ABI struct {
	formatSize, formatData                    uintptr
	bufSize, bufTimestamp, bufTimecode        uintptr
	bufSequence, bufMemory, bufOffset, bufLen uintptr
	querycap, gfmt, sfmt, reqbufs             uintptr
	querybuf, qbuf, dqbuf, streamon, streamof uintptr
}

var v4l2Want = map[uintptr]v4l2ABI{
	// linux/amd64, linux/arm64, ... (gcc, /usr/include/linux/videodev2.h)
	8: {
		formatSize: 208, formatData: 8,
		bufSize: 88, bufTimestamp: 24, bufTimecode: 40,
		bufSequence: 56, bufMemory: 60, bufOffset: 64, bufLen: 72,
		querycap: 0x80685600, gfmt: 0xc0d05604, sfmt: 0xc0d05605, reqbufs: 0xc0145608,
		querybuf: 0xc0585609, qbuf: 0xc058560f, dqbuf: 0xc0585611,
		streamon: 0x40045612, streamof: 0x40045613,
	},
	// linux/386, linux/arm: 32-bit pointers and struct timeval (clang -m32
	// static asserts against the same header)
	4: {
		formatSize: 204, formatData: 4,
		bufSize: 68, bufTimestamp: 20, bufTimecode: 28,
		bufSequence: 44, bufMemory: 48, bufOffset: 52, bufLen: 56,
		querycap: 0x80685600, gfmt: 0xc0cc5604, sfmt: 0xc0cc5605, reqbufs: 0xc0145608,
		querybuf: 0xc0445609, qbuf: 0xc044560f, dqbuf: 0xc0445611,
		streamon: 0x40045612, streamof: 0x40045613,
	},
}

func TestV4L2StructLayoutMatchesKernel(t *testing.T) {
	want, ok := v4l2Want[unsafe.Sizeof(uintptr(0))]
	if !ok {
		t.Skip("no reference layout for this pointer size")
	}
	var f v4l2Format
	var b v4l2Buffer
	checks := []struct {
		name      string
		got, want uintptr
	}{
		{"sizeof(v4l2_capability)", unsafe.Sizeof(v4l2Capability{}), 104},
		{"sizeof(v4l2_requestbuffers)", unsafe.Sizeof(v4l2RequestBuffers{}), 20},
		{"sizeof(v4l2_format)", unsafe.Sizeof(f), want.formatSize},
		{"offsetof(v4l2_format, fmt)", unsafe.Offsetof(f.data), want.formatData},
		{"sizeof(v4l2_buffer)", unsafe.Sizeof(b), want.bufSize},
		{"offsetof(v4l2_buffer, timestamp)", unsafe.Offsetof(b.timestamp), want.bufTimestamp},
		{"offsetof(v4l2_buffer, timecode)", unsafe.Offsetof(b.timecode), want.bufTimecode},
		{"offsetof(v4l2_buffer, sequence)", unsafe.Offsetof(b.sequence), want.bufSequence},
		{"offsetof(v4l2_buffer, memory)", unsafe.Offsetof(b.memory), want.bufMemory},
		{"offsetof(v4l2_buffer, m.offset)", unsafe.Offsetof(b.offset), want.bufOffset},
		{"offsetof(v4l2_buffer, length)", unsafe.Offsetof(b.length), want.bufLen},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestV4L2IoctlNumbersMatchKernel(t *testing.T) {
	switch runtime.GOARCH {
	case "mips", "mipsle", "mips64", "mips64le", "ppc64", "ppc64le":
		t.Skip("these kernels use a different ioctl direction/size encoding")
	}
	want, ok := v4l2Want[unsafe.Sizeof(uintptr(0))]
	if !ok {
		t.Skip("no reference layout for this pointer size")
	}
	checks := []struct {
		name      string
		got, want uintptr
	}{
		{"VIDIOC_QUERYCAP", VIDIOC_QUERYCAP, want.querycap},
		{"VIDIOC_G_FMT", VIDIOC_G_FMT, want.gfmt},
		{"VIDIOC_S_FMT", VIDIOC_S_FMT, want.sfmt},
		{"VIDIOC_REQBUFS", VIDIOC_REQBUFS, want.reqbufs},
		{"VIDIOC_QUERYBUF", VIDIOC_QUERYBUF, want.querybuf},
		{"VIDIOC_QBUF", VIDIOC_QBUF, want.qbuf},
		{"VIDIOC_DQBUF", VIDIOC_DQBUF, want.dqbuf},
		{"VIDIOC_STREAMON", VIDIOC_STREAMON, want.streamon},
		{"VIDIOC_STREAMOFF", VIDIOC_STREAMOFF, want.streamof},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %#x, want %#x", c.name, c.got, c.want)
		}
	}

	// The size field of each ioctl number must equal the Go struct the call
	// passes, or the kernel copies more or fewer bytes than the buffer holds.
	size := func(req uintptr) uintptr { return (req >> 16) & 0x3fff }
	sizes := []struct {
		name      string
		req, size uintptr
	}{
		{"VIDIOC_QUERYCAP", VIDIOC_QUERYCAP, unsafe.Sizeof(v4l2Capability{})},
		{"VIDIOC_G_FMT", VIDIOC_G_FMT, unsafe.Sizeof(v4l2Format{})},
		{"VIDIOC_S_FMT", VIDIOC_S_FMT, unsafe.Sizeof(v4l2Format{})},
		{"VIDIOC_REQBUFS", VIDIOC_REQBUFS, unsafe.Sizeof(v4l2RequestBuffers{})},
		{"VIDIOC_QUERYBUF", VIDIOC_QUERYBUF, unsafe.Sizeof(v4l2Buffer{})},
		{"VIDIOC_QBUF", VIDIOC_QBUF, unsafe.Sizeof(v4l2Buffer{})},
		{"VIDIOC_DQBUF", VIDIOC_DQBUF, unsafe.Sizeof(v4l2Buffer{})},
		{"VIDIOC_STREAMON", VIDIOC_STREAMON, unsafe.Sizeof(uint32(0))},
		{"VIDIOC_STREAMOFF", VIDIOC_STREAMOFF, unsafe.Sizeof(uint32(0))},
	}
	for _, s := range sizes {
		if got := size(s.req); got != s.size {
			t.Errorf("%s encodes size %d, Go argument is %d bytes", s.name, got, s.size)
		}
	}
}
