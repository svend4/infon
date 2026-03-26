//go:build linux || darwin || freebsd || openbsd
// +build linux darwin freebsd openbsd

package serial

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	// Baud rates
	B0      = 0
	B50     = 50
	B75     = 75
	B110    = 110
	B134    = 134
	B150    = 150
	B200    = 200
	B300    = 300
	B600    = 600
	B1200   = 1200
	B1800   = 1800
	B2400   = 2400
	B4800   = 4800
	B9600   = 9600
	B19200  = 19200
	B38400  = 38400
	B57600  = 57600
	B115200 = 115200
	B230400 = 230400

	// Character size
	CS5 = 0x00000000
	CS6 = 0x00000010
	CS7 = 0x00000020
	CS8 = 0x00000030

	// Stop bits
	CSTOPB = 0x00000040

	// Parity
	PARENB = 0x00000100
	PARODD = 0x00000200

	// Control flags
	CREAD  = 0x00000080
	CLOCAL = 0x00000800

	// Input flags
	IGNPAR = 0x00000004
	ICRNL  = 0x00000100

	// Output flags
	OPOST = 0x00000001

	// Local flags
	ICANON = 0x00000002
	ECHO   = 0x00000008
	ECHOE  = 0x00000010
	ISIG   = 0x00000001

	// Control characters
	VMIN  = 6
	VTIME = 5

	// Modem control
	TIOCM_DTR = 0x002
	TIOCM_RTS = 0x004
	TIOCM_CTS = 0x020
	TIOCM_DSR = 0x100
	TIOCM_RI  = 0x080
	TIOCM_CD  = 0x040
)

// Termios structure for Unix
type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Line   uint8
	Cc     [32]uint8
	Ispeed uint32
	Ospeed uint32
}

// open opens the serial port (Unix implementation)
func (p *Port) open() error {
	// Open the port
	file, err := os.OpenFile(p.config.Name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return fmt.Errorf("failed to open port %s: %w", p.config.Name, err)
	}

	p.handle = file

	// Get current termios
	var t termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(getTermios()),
		uintptr(unsafe.Pointer(&t)))

	if errno != 0 {
		file.Close()
		return fmt.Errorf("failed to get termios: %s", errno.Error())
	}

	// Set raw mode
	t.Iflag &^= (IGNPAR | ICRNL)
	t.Oflag &^= OPOST
	t.Lflag &^= (ICANON | ECHO | ECHOE | ISIG)
	t.Cflag |= (CREAD | CLOCAL)

	// Set data bits
	t.Cflag &^= 0x00000030 // Clear CS bits
	switch p.config.DataBits {
	case 5:
		t.Cflag |= CS5
	case 6:
		t.Cflag |= CS6
	case 7:
		t.Cflag |= CS7
	case 8:
		t.Cflag |= CS8
	default:
		file.Close()
		return fmt.Errorf("invalid data bits: %d", p.config.DataBits)
	}

	// Set stop bits
	if p.config.StopBits == StopBits2 {
		t.Cflag |= CSTOPB
	} else {
		t.Cflag &^= CSTOPB
	}

	// Set parity
	switch p.config.Parity {
	case ParityNone:
		t.Cflag &^= PARENB
	case ParityOdd:
		t.Cflag |= (PARENB | PARODD)
	case ParityEven:
		t.Cflag |= PARENB
		t.Cflag &^= PARODD
	default:
		file.Close()
		return fmt.Errorf("unsupported parity mode on Unix")
	}

	// Set baud rate
	baudRate := getBaudRate(p.config.BaudRate)
	if baudRate == 0 {
		file.Close()
		return fmt.Errorf("unsupported baud rate: %d", p.config.BaudRate)
	}
	t.Ispeed = baudRate
	t.Ospeed = baudRate

	// Set timeout (in deciseconds)
	timeoutDeciseconds := uint8(p.config.ReadTimeout.Milliseconds() / 100)
	if timeoutDeciseconds == 0 {
		timeoutDeciseconds = 1
	}
	t.Cc[VMIN] = 0
	t.Cc[VTIME] = timeoutDeciseconds

	// Apply settings
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(setTermios()),
		uintptr(unsafe.Pointer(&t)))

	if errno != 0 {
		file.Close()
		return fmt.Errorf("failed to set termios: %s", errno.Error())
	}

	// Make file descriptor blocking
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, file.Fd(), syscall.F_GETFL, 0)
	if errno != 0 {
		file.Close()
		return fmt.Errorf("failed to get file flags: %s", errno.Error())
	}

	flags &^= uintptr(syscall.O_NONBLOCK)
	_, _, errno = syscall.Syscall(syscall.SYS_FCNTL, file.Fd(), syscall.F_SETFL, flags)
	if errno != 0 {
		file.Close()
		return fmt.Errorf("failed to set file flags: %s", errno.Error())
	}

	return nil
}

// read reads data from the port (Unix implementation)
func (p *Port) read(buf []byte) (int, error) {
	file := p.handle.(*os.File)
	return file.Read(buf)
}

// write writes data to the port (Unix implementation)
func (p *Port) write(buf []byte) (int, error) {
	file := p.handle.(*os.File)
	return file.Write(buf)
}

// close closes the port (Unix implementation)
func (p *Port) close() error {
	if p.handle == nil {
		return nil
	}

	file := p.handle.(*os.File)
	err := file.Close()
	p.handle = nil
	return err
}

// flush flushes the port buffers (Unix implementation)
func (p *Port) flush() error {
	file := p.handle.(*os.File)

	// TCIOFLUSH flushes both input and output
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(getTCIOFLUSH()),
		0)

	if errno != 0 {
		return fmt.Errorf("flush failed: %s", errno.Error())
	}

	return nil
}

// available returns bytes available (Unix implementation)
func (p *Port) available() (int, error) {
	file := p.handle.(*os.File)

	var n int
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(getFIONREAD()),
		uintptr(unsafe.Pointer(&n)))

	if errno != 0 {
		return 0, fmt.Errorf("available failed: %s", errno.Error())
	}

	return n, nil
}

// setDTR sets the DTR line (Unix implementation)
func (p *Port) setDTR(state bool) error {
	file := p.handle.(*os.File)

	var flag int
	if state {
		flag = TIOCM_DTR
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(getTIOCMBIS()),
		uintptr(unsafe.Pointer(&flag)))

	if errno != 0 {
		return fmt.Errorf("setDTR failed: %s", errno.Error())
	}

	return nil
}

// setRTS sets the RTS line (Unix implementation)
func (p *Port) setRTS(state bool) error {
	file := p.handle.(*os.File)

	var flag int
	if state {
		flag = TIOCM_RTS
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(getTIOCMBIS()),
		uintptr(unsafe.Pointer(&flag)))

	if errno != 0 {
		return fmt.Errorf("setRTS failed: %s", errno.Error())
	}

	return nil
}

// getModemStatus returns modem status (Unix implementation)
func (p *Port) getModemStatus() (ModemStatus, error) {
	file := p.handle.(*os.File)

	var status int
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(getTIOCMGET()),
		uintptr(unsafe.Pointer(&status)))

	if errno != 0 {
		return ModemStatus{}, fmt.Errorf("getModemStatus failed: %s", errno.Error())
	}

	return ModemStatus{
		CTS: (status & TIOCM_CTS) != 0,
		DSR: (status & TIOCM_DSR) != 0,
		RI:  (status & TIOCM_RI) != 0,
		DCD: (status & TIOCM_CD) != 0,
	}, nil
}

// listPorts returns available ports (Unix implementation)
func listPorts() ([]string, error) {
	ports := make([]string, 0)

	// Common serial port patterns
	patterns := []string{
		"/dev/ttyUSB*",
		"/dev/ttyACM*",
		"/dev/ttyS*",
		"/dev/tty.usb*",     // macOS
		"/dev/cu.usb*",      // macOS
		"/dev/tty.serial*",  // macOS
		"/dev/cu.serial*",   // macOS
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		ports = append(ports, matches...)
	}

	return ports, nil
}

// getBaudRate converts integer baud rate to system constant
func getBaudRate(rate int) uint32 {
	switch rate {
	case 50:
		return B50
	case 75:
		return B75
	case 110:
		return B110
	case 134:
		return B134
	case 150:
		return B150
	case 200:
		return B200
	case 300:
		return B300
	case 600:
		return B600
	case 1200:
		return B1200
	case 1800:
		return B1800
	case 2400:
		return B2400
	case 4800:
		return B4800
	case 9600:
		return B9600
	case 19200:
		return B19200
	case 38400:
		return B38400
	case 57600:
		return B57600
	case 115200:
		return B115200
	case 230400:
		return B230400
	default:
		return 0
	}
}
