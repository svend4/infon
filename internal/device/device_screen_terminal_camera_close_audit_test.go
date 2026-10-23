//go:build linux

package device

import (
	"syscall"
	"testing"
)

// fdOpen reports whether fd refers to an open file description.
func fdOpen(fd int) bool {
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0)
	return errno == 0
}

// Open's error paths after REQBUFS (QUERYBUF, mmap, QBUF, STREAMON) clean up by
// calling Close while isOpen is still false. Close must then still unmap the
// buffers and close the device; otherwise every failed Open leaks an fd and the
// mmapped driver buffers. Reaching those paths needs a V4L2 device, so this
// test puts the camera into the state Open leaves it in at that point.
func TestV4L2CloseReleasesPartiallyOpenedCamera(t *testing.T) {
	fd, err := syscall.Open("/dev/zero", syscall.O_RDWR, 0)
	if err != nil {
		t.Skipf("cannot open /dev/zero: %v", err)
	}
	buf, err := syscall.Mmap(-1, 0, 4096, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		_ = syscall.Close(fd)
		t.Fatalf("mmap: %v", err)
	}

	c := NewV4L2Camera("/dev/zero", 2, 2, 1)
	c.fd = fd
	c.buffers = [][]byte{buf, nil, nil, nil} // failed while mapping buffer 1

	_ = c.Close()

	if fdOpen(fd) {
		_ = syscall.Close(fd)
		t.Errorf("fd %d still open after Close of a camera whose Open did not finish", fd)
	}
	if c.fd != -1 {
		t.Errorf("c.fd = %d after Close, want -1", c.fd)
	}
	// syscall.Munmap rejects a slice it no longer tracks, so success here means
	// Close left the buffer mapped.
	if err := syscall.Munmap(buf); err == nil {
		t.Error("buffer still mapped after Close of a camera whose Open did not finish")
	}
	if c.buffers != nil {
		t.Errorf("c.buffers = %d entries after Close, want nil", len(c.buffers))
	}
}

// A failed Open must not leave a stale descriptor number behind for a later
// Close to act on: by then the number may belong to an unrelated file.
func TestV4L2FailedOpenLeavesNoDescriptor(t *testing.T) {
	c := NewV4L2Camera("/dev/null", 2, 2, 1) // not a V4L2 device: QUERYCAP fails
	if err := c.Open(); err == nil {
		_ = c.Close()
		t.Fatal("Open of /dev/null succeeded")
	}
	if c.fd != -1 {
		t.Errorf("c.fd = %d after a failed Open, want -1", c.fd)
	}

	// Whatever the camera's state, Close must not close a file it does not own.
	other, err := syscall.Open("/dev/null", syscall.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = syscall.Close(other) }()
	_ = c.Close()
	if !fdOpen(other) {
		t.Errorf("Close after a failed Open closed fd %d, which the camera does not own", other)
	}
}
