package serial

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Config represents serial port configuration
type Config struct {
	Name        string        // Port name (e.g., "COM1", "/dev/ttyUSB0")
	BaudRate    int           // Baud rate (9600, 115200, etc.)
	DataBits    int           // Data bits (5, 6, 7, 8)
	StopBits    StopBits      // Stop bits
	Parity      Parity        // Parity
	ReadTimeout time.Duration // Read timeout
}

// StopBits represents the number of stop bits
type StopBits int

const (
	StopBits1   StopBits = 1
	StopBits1_5 StopBits = 15 // 1.5 stop bits
	StopBits2   StopBits = 2
)

// Parity represents the parity mode
type Parity int

const (
	ParityNone  Parity = 0
	ParityOdd   Parity = 1
	ParityEven  Parity = 2
	ParityMark  Parity = 3
	ParitySpace Parity = 4
)

// Port represents a serial port
type Port struct {
	mu sync.RWMutex

	config Config
	handle interface{} // Platform-specific handle

	isOpen bool

	// Statistics
	bytesRead    uint64
	bytesWritten uint64
	errors       uint64
	lastError    error
}

// OpenPort opens a serial port with the given configuration
func OpenPort(config Config) (*Port, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("port name cannot be empty")
	}

	// Set defaults
	if config.BaudRate == 0 {
		config.BaudRate = 9600
	}
	if config.DataBits == 0 {
		config.DataBits = 8
	}
	if config.StopBits == 0 {
		config.StopBits = StopBits1
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 1 * time.Second
	}

	port := &Port{
		config: config,
	}

	// Platform-specific opening
	if err := port.open(); err != nil {
		return nil, err
	}

	port.isOpen = true

	return port, nil
}

// Read reads data from the serial port
func (p *Port) Read(buf []byte) (int, error) {
	p.mu.RLock()
	if !p.isOpen {
		p.mu.RUnlock()
		return 0, fmt.Errorf("port is not open")
	}
	p.mu.RUnlock()

	n, err := p.read(buf)

	p.mu.Lock()
	if err != nil {
		p.errors++
		p.lastError = err
	} else {
		p.bytesRead += uint64(n)
	}
	p.mu.Unlock()

	return n, err
}

// Write writes data to the serial port
func (p *Port) Write(buf []byte) (int, error) {
	p.mu.RLock()
	if !p.isOpen {
		p.mu.RUnlock()
		return 0, fmt.Errorf("port is not open")
	}
	p.mu.RUnlock()

	n, err := p.write(buf)

	p.mu.Lock()
	if err != nil {
		p.errors++
		p.lastError = err
	} else {
		p.bytesWritten += uint64(n)
	}
	p.mu.Unlock()

	return n, err
}

// Close closes the serial port
func (p *Port) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isOpen {
		return nil
	}

	err := p.close()
	p.isOpen = false

	return err
}

// Flush flushes both input and output buffers
func (p *Port) Flush() error {
	p.mu.RLock()
	if !p.isOpen {
		p.mu.RUnlock()
		return fmt.Errorf("port is not open")
	}
	p.mu.RUnlock()

	return p.flush()
}

// SetReadTimeout sets the read timeout
func (p *Port) SetReadTimeout(timeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config.ReadTimeout = timeout
}

// GetConfig returns the current configuration
func (p *Port) GetConfig() Config {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.config
}

// IsOpen returns whether the port is open
func (p *Port) IsOpen() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.isOpen
}

// GetStatistics returns port statistics
func (p *Port) GetStatistics() Statistics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return Statistics{
		BytesRead:    p.bytesRead,
		BytesWritten: p.bytesWritten,
		Errors:       p.errors,
		LastError:    p.lastError,
	}
}

// Statistics represents port statistics
type Statistics struct {
	BytesRead    uint64
	BytesWritten uint64
	Errors       uint64
	LastError    error
}

// ReadLine reads a line from the serial port (until \n or \r\n)
func (p *Port) ReadLine() (string, error) {
	buf := make([]byte, 1)
	line := make([]byte, 0, 256)

	for {
		n, err := p.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		if n == 0 {
			continue
		}

		// Check for line endings
		if buf[0] == '\n' {
			break
		}
		if buf[0] == '\r' {
			// Check for \r\n
			n, err := p.Read(buf)
			if err == nil && n > 0 && buf[0] == '\n' {
				// Found \r\n
			}
			break
		}

		line = append(line, buf[0])

		// Prevent infinite loops
		if len(line) > 4096 {
			return "", fmt.Errorf("line too long")
		}
	}

	return string(line), nil
}

// WriteLine writes a line to the serial port (adds \r\n)
func (p *Port) WriteLine(line string) error {
	data := []byte(line + "\r\n")
	n, err := p.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("incomplete write: wrote %d of %d bytes", n, len(data))
	}
	return nil
}

// Available returns the number of bytes available to read
func (p *Port) Available() (int, error) {
	p.mu.RLock()
	if !p.isOpen {
		p.mu.RUnlock()
		return 0, fmt.Errorf("port is not open")
	}
	p.mu.RUnlock()

	return p.available()
}

// SetDTR sets the DTR (Data Terminal Ready) line
func (p *Port) SetDTR(state bool) error {
	p.mu.RLock()
	if !p.isOpen {
		p.mu.RUnlock()
		return fmt.Errorf("port is not open")
	}
	p.mu.RUnlock()

	return p.setDTR(state)
}

// SetRTS sets the RTS (Request To Send) line
func (p *Port) SetRTS(state bool) error {
	p.mu.RLock()
	if !p.isOpen {
		p.mu.RUnlock()
		return fmt.Errorf("port is not open")
	}
	p.mu.RUnlock()

	return p.setRTS(state)
}

// GetModemStatus returns the modem status lines
func (p *Port) GetModemStatus() (ModemStatus, error) {
	p.mu.RLock()
	if !p.isOpen {
		p.mu.RUnlock()
		return ModemStatus{}, fmt.Errorf("port is not open")
	}
	p.mu.RUnlock()

	return p.getModemStatus()
}

// ModemStatus represents modem control line status
type ModemStatus struct {
	CTS bool // Clear To Send
	DSR bool // Data Set Ready
	RI  bool // Ring Indicator
	DCD bool // Data Carrier Detect
}

// ListPorts returns a list of available serial ports
func ListPorts() ([]string, error) {
	return listPorts()
}

// Reconfigure reconfigures the port with new settings
func (p *Port) Reconfigure(config Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isOpen {
		return fmt.Errorf("port is not open")
	}

	// Close and reopen with new config
	if err := p.close(); err != nil {
		return err
	}

	p.config = config

	if err := p.open(); err != nil {
		p.isOpen = false
		return err
	}

	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig(portName string) Config {
	return Config{
		Name:        portName,
		BaudRate:    9600,
		DataBits:    8,
		StopBits:    StopBits1,
		Parity:      ParityNone,
		ReadTimeout: 1 * time.Second,
	}
}

// HighSpeedConfig returns a high-speed configuration (115200 baud)
func HighSpeedConfig(portName string) Config {
	return Config{
		Name:        portName,
		BaudRate:    115200,
		DataBits:    8,
		StopBits:    StopBits1,
		Parity:      ParityNone,
		ReadTimeout: 100 * time.Millisecond,
	}
}

// String returns a string representation of the configuration
func (c Config) String() string {
	return fmt.Sprintf("%s: %d baud, %d%s%d",
		c.Name,
		c.BaudRate,
		c.DataBits,
		parityString(c.Parity),
		c.StopBits,
	)
}

func parityString(p Parity) string {
	switch p {
	case ParityNone:
		return "N"
	case ParityOdd:
		return "O"
	case ParityEven:
		return "E"
	case ParityMark:
		return "M"
	case ParitySpace:
		return "S"
	default:
		return "?"
	}
}
