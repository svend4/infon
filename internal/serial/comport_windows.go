//go:build windows
// +build windows

package serial

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	createFile  = kernel32.NewProc("CreateFileW")
	closeHandle = kernel32.NewProc("CloseHandle")
	readFile    = kernel32.NewProc("ReadFile")
	writeFile   = kernel32.NewProc("WriteFile")
	flushFileBuffers = kernel32.NewProc("FlushFileBuffers")
	getDCB      = kernel32.NewProc("GetCommState")
	setDCB      = kernel32.NewProc("SetCommState")
	getTimeouts = kernel32.NewProc("GetCommTimeouts")
	setTimeouts = kernel32.NewProc("SetCommTimeouts")
	purgeComm   = kernel32.NewProc("PurgeComm")
	setCommMask = kernel32.NewProc("SetCommMask")
	escapeCommFunction = kernel32.NewProc("EscapeCommFunction")
	getCommModemStatus = kernel32.NewProc("GetCommModemStatus")
)

const (
	// Generic access rights
	GENERIC_READ  = 0x80000000
	GENERIC_WRITE = 0x40000000

	// File sharing modes
	FILE_SHARE_READ  = 0x00000001
	FILE_SHARE_WRITE = 0x00000002

	// File creation disposition
	OPEN_EXISTING = 3

	// File attributes
	FILE_ATTRIBUTE_NORMAL = 0x00000080
	FILE_FLAG_OVERLAPPED  = 0x40000000

	// Purge flags
	PURGE_RXABORT = 0x0002
	PURGE_RXCLEAR = 0x0008
	PURGE_TXABORT = 0x0001
	PURGE_TXCLEAR = 0x0004

	// EscapeCommFunction commands
	SETDTR  = 5
	CLRDTR  = 6
	SETRTS  = 3
	CLRRTS  = 4

	// Modem status
	MS_CTS_ON  = 0x0010
	MS_DSR_ON  = 0x0020
	MS_RING_ON = 0x0040
	MS_RLSD_ON = 0x0080

	INVALID_HANDLE_VALUE = ^uintptr(0)
)

// DCB structure for Windows
type dcb struct {
	DCBlength uint32
	BaudRate  uint32
	Flags     uint32
	wReserved uint16
	XonLim    uint16
	XoffLim   uint16
	ByteSize  byte
	Parity    byte
	StopBits  byte
	XonChar   byte
	XoffChar  byte
	ErrorChar byte
	EofChar   byte
	EvtChar   byte
	wReserved1 uint16
}

// COMMTIMEOUTS structure
type commTimeouts struct {
	ReadIntervalTimeout         uint32
	ReadTotalTimeoutMultiplier  uint32
	ReadTotalTimeoutConstant    uint32
	WriteTotalTimeoutMultiplier uint32
	WriteTotalTimeoutConstant   uint32
}

// open opens the serial port (Windows implementation)
func (p *Port) open() error {
	// Convert port name to UTF-16
	portName := `\\.\` + p.config.Name
	portNameUTF16, err := syscall.UTF16PtrFromString(portName)
	if err != nil {
		return fmt.Errorf("invalid port name: %w", err)
	}

	// Open port
	r1, _, err := createFile.Call(
		uintptr(unsafe.Pointer(portNameUTF16)),
		GENERIC_READ|GENERIC_WRITE,
		0, // No sharing
		0, // No security
		OPEN_EXISTING,
		FILE_ATTRIBUTE_NORMAL,
		0,
	)

	if r1 == INVALID_HANDLE_VALUE {
		return fmt.Errorf("failed to open port %s: %w", p.config.Name, err)
	}

	p.handle = syscall.Handle(r1)

	// Configure DCB
	var d dcb
	d.DCBlength = uint32(unsafe.Sizeof(d))

	r1, _, err = getDCB.Call(uintptr(p.handle.(syscall.Handle)), uintptr(unsafe.Pointer(&d)))
	if r1 == 0 {
		closeHandle.Call(uintptr(p.handle.(syscall.Handle)))
		return fmt.Errorf("failed to get DCB: %w", err)
	}

	// Set baud rate
	d.BaudRate = uint32(p.config.BaudRate)

	// Set data bits
	d.ByteSize = byte(p.config.DataBits)

	// Set stop bits
	switch p.config.StopBits {
	case StopBits1:
		d.StopBits = 0 // ONESTOPBIT
	case StopBits1_5:
		d.StopBits = 1 // ONE5STOPBITS
	case StopBits2:
		d.StopBits = 2 // TWOSTOPBITS
	}

	// Set parity
	switch p.config.Parity {
	case ParityNone:
		d.Parity = 0 // NOPARITY
		d.Flags &^= 0x0002 // Clear parity enable bit
	case ParityOdd:
		d.Parity = 1 // ODDPARITY
		d.Flags |= 0x0002 // Set parity enable bit
	case ParityEven:
		d.Parity = 2 // EVENPARITY
		d.Flags |= 0x0002
	case ParityMark:
		d.Parity = 3 // MARKPARITY
		d.Flags |= 0x0002
	case ParitySpace:
		d.Parity = 4 // SPACEPARITY
		d.Flags |= 0x0002
	}

	// Set other flags
	d.Flags |= 0x0001 // Binary mode
	d.Flags |= 0x0010 // Enable DTR

	r1, _, err = setDCB.Call(uintptr(p.handle.(syscall.Handle)), uintptr(unsafe.Pointer(&d)))
	if r1 == 0 {
		closeHandle.Call(uintptr(p.handle.(syscall.Handle)))
		return fmt.Errorf("failed to set DCB: %w", err)
	}

	// Set timeouts
	var timeouts commTimeouts
	timeoutMs := uint32(p.config.ReadTimeout.Milliseconds())

	timeouts.ReadIntervalTimeout = 0
	timeouts.ReadTotalTimeoutMultiplier = 0
	timeouts.ReadTotalTimeoutConstant = timeoutMs
	timeouts.WriteTotalTimeoutMultiplier = 0
	timeouts.WriteTotalTimeoutConstant = 1000 // 1 second write timeout

	r1, _, err = setTimeouts.Call(
		uintptr(p.handle.(syscall.Handle)),
		uintptr(unsafe.Pointer(&timeouts)),
	)
	if r1 == 0 {
		closeHandle.Call(uintptr(p.handle.(syscall.Handle)))
		return fmt.Errorf("failed to set timeouts: %w", err)
	}

	return nil
}

// read reads data from the port (Windows implementation)
func (p *Port) read(buf []byte) (int, error) {
	var bytesRead uint32

	r1, _, err := readFile.Call(
		uintptr(p.handle.(syscall.Handle)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&bytesRead)),
		0,
	)

	if r1 == 0 {
		return 0, fmt.Errorf("read failed: %w", err)
	}

	return int(bytesRead), nil
}

// write writes data to the port (Windows implementation)
func (p *Port) write(buf []byte) (int, error) {
	var bytesWritten uint32

	r1, _, err := writeFile.Call(
		uintptr(p.handle.(syscall.Handle)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&bytesWritten)),
		0,
	)

	if r1 == 0 {
		return 0, fmt.Errorf("write failed: %w", err)
	}

	return int(bytesWritten), nil
}

// close closes the port (Windows implementation)
func (p *Port) close() error {
	if p.handle == nil {
		return nil
	}

	r1, _, err := closeHandle.Call(uintptr(p.handle.(syscall.Handle)))
	if r1 == 0 {
		return fmt.Errorf("close failed: %w", err)
	}

	p.handle = nil
	return nil
}

// flush flushes the port buffers (Windows implementation)
func (p *Port) flush() error {
	// Flush both RX and TX buffers
	r1, _, err := purgeComm.Call(
		uintptr(p.handle.(syscall.Handle)),
		PURGE_RXCLEAR|PURGE_TXCLEAR,
	)

	if r1 == 0 {
		return fmt.Errorf("flush failed: %w", err)
	}

	return nil
}

// available returns bytes available (Windows - simplified)
func (p *Port) available() (int, error) {
	// Windows doesn't have a direct equivalent
	// We'd need to use overlapped I/O for proper implementation
	return 0, fmt.Errorf("available() not implemented on Windows")
}

// setDTR sets the DTR line (Windows implementation)
func (p *Port) setDTR(state bool) error {
	var cmd uintptr
	if state {
		cmd = SETDTR
	} else {
		cmd = CLRDTR
	}

	r1, _, err := escapeCommFunction.Call(
		uintptr(p.handle.(syscall.Handle)),
		cmd,
	)

	if r1 == 0 {
		return fmt.Errorf("setDTR failed: %w", err)
	}

	return nil
}

// setRTS sets the RTS line (Windows implementation)
func (p *Port) setRTS(state bool) error {
	var cmd uintptr
	if state {
		cmd = SETRTS
	} else {
		cmd = CLRRTS
	}

	r1, _, err := escapeCommFunction.Call(
		uintptr(p.handle.(syscall.Handle)),
		cmd,
	)

	if r1 == 0 {
		return fmt.Errorf("setRTS failed: %w", err)
	}

	return nil
}

// getModemStatus returns modem status (Windows implementation)
func (p *Port) getModemStatus() (ModemStatus, error) {
	var status uint32

	r1, _, err := getCommModemStatus.Call(
		uintptr(p.handle.(syscall.Handle)),
		uintptr(unsafe.Pointer(&status)),
	)

	if r1 == 0 {
		return ModemStatus{}, fmt.Errorf("getModemStatus failed: %w", err)
	}

	return ModemStatus{
		CTS: (status & MS_CTS_ON) != 0,
		DSR: (status & MS_DSR_ON) != 0,
		RI:  (status & MS_RING_ON) != 0,
		DCD: (status & MS_RLSD_ON) != 0,
	}, nil
}

// listPorts returns available ports (Windows implementation)
func listPorts() ([]string, error) {
	ports := make([]string, 0)

	// Try COM1 through COM256
	for i := 1; i <= 256; i++ {
		portName := fmt.Sprintf("COM%d", i)
		portNameFull := `\\.\` + portName
		portNameUTF16, err := syscall.UTF16PtrFromString(portNameFull)
		if err != nil {
			continue
		}

		// Try to open the port
		r1, _, _ := createFile.Call(
			uintptr(unsafe.Pointer(portNameUTF16)),
			GENERIC_READ|GENERIC_WRITE,
			0,
			0,
			OPEN_EXISTING,
			FILE_ATTRIBUTE_NORMAL,
			0,
		)

		if r1 != INVALID_HANDLE_VALUE {
			// Port exists and can be opened
			closeHandle.Call(r1)
			ports = append(ports, portName)
		}
	}

	return ports, nil
}
