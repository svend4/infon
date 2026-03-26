package screen

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// ScreenShare manages terminal output sharing
type ScreenShare struct {
	width      int
	height     int
	cmd        *exec.Cmd
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	buffer     [][]rune // Terminal buffer
	mutex      sync.RWMutex
	running    bool
	frameChan  chan *terminal.Frame
	command    string
	cursorRow  int
	cursorCol  int
	scrollback int // Number of lines in scrollback buffer
}

// NewScreenShare creates a new screen sharing session
func NewScreenShare(command string, width, height int) *ScreenShare {
	return &ScreenShare{
		width:      width,
		height:     height,
		command:    command,
		frameChan:  make(chan *terminal.Frame, 10),
		buffer:     make([][]rune, height),
		scrollback: 1000, // Keep 1000 lines of history
	}
}

// Start begins capturing terminal output
func (ss *ScreenShare) Start() error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if ss.running {
		return fmt.Errorf("screen share already running")
	}

	// Initialize buffer
	for i := 0; i < ss.height; i++ {
		ss.buffer[i] = make([]rune, ss.width)
		for j := 0; j < ss.width; j++ {
			ss.buffer[i][j] = ' '
		}
	}

	// Parse command
	parts := strings.Fields(ss.command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Create command
	ss.cmd = exec.Command(parts[0], parts[1:]...)
	ss.cmd.Env = append(os.Environ(),
		fmt.Sprintf("COLUMNS=%d", ss.width),
		fmt.Sprintf("LINES=%d", ss.height),
		"TERM=xterm-256color",
	)

	// Setup pipes
	var err error
	ss.stdout, err = ss.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	ss.stderr, err = ss.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := ss.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	ss.running = true

	// Start output readers
	go ss.readOutput(ss.stdout)
	go ss.readOutput(ss.stderr)

	// Start frame generator
	go ss.generateFrames()

	return nil
}

// readOutput reads and processes command output
func (ss *ScreenShare) readOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		ss.processLine(line)
	}
}

// processLine adds a line to the terminal buffer
func (ss *ScreenShare) processLine(line string) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	// Scroll buffer up
	copy(ss.buffer[0:], ss.buffer[1:])

	// Add new line at bottom
	lastRow := ss.height - 1
	ss.buffer[lastRow] = make([]rune, ss.width)

	runes := []rune(line)
	for i := 0; i < ss.width; i++ {
		if i < len(runes) {
			ss.buffer[lastRow][i] = runes[i]
		} else {
			ss.buffer[lastRow][i] = ' '
		}
	}
}

// generateFrames creates video frames from terminal buffer
func (ss *ScreenShare) generateFrames() {
	ticker := time.NewTicker(time.Second / 15) // 15 FPS
	defer ticker.Stop()

	for range ticker.C {
		ss.mutex.RLock()
		if !ss.running {
			ss.mutex.RUnlock()
			break
		}

		frame := ss.captureFrame()
		ss.mutex.RUnlock()

		select {
		case ss.frameChan <- frame:
		default:
			// Drop frame if channel full
		}
	}
}

// captureFrame creates a video frame from current buffer
func (ss *ScreenShare) captureFrame() *terminal.Frame {
	frame := terminal.NewFrame(ss.width, ss.height)

	for row := 0; row < ss.height; row++ {
		for col := 0; col < ss.width; col++ {
			char := ss.buffer[row][col]

			// Use different colors for better visibility
			var glyph rune
			if char != ' ' && char != 0 {
				glyph = char // Use actual character
			} else {
				glyph = ' ' // Space
			}

			frame.SetBlock(col, row, glyph,
				color.RGB{R: 200, G: 200, B: 200}, // Light gray text
				color.RGB{R: 0, G: 0, B: 0},       // Black background
			)
		}
	}

	return frame
}

// GetFrameChannel returns the frame channel
func (ss *ScreenShare) GetFrameChannel() <-chan *terminal.Frame {
	return ss.frameChan
}

// Stop stops the screen sharing session
func (ss *ScreenShare) Stop() error {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if !ss.running {
		return nil
	}

	ss.running = false

	// Kill command
	if ss.cmd != nil && ss.cmd.Process != nil {
		ss.cmd.Process.Kill()
	}

	close(ss.frameChan)

	return nil
}

// IsRunning returns whether screen sharing is active
func (ss *ScreenShare) IsRunning() bool {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	return ss.running
}

// GetBuffer returns current terminal buffer (for debugging)
func (ss *ScreenShare) GetBuffer() []string {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	lines := make([]string, ss.height)
	for i := 0; i < ss.height; i++ {
		lines[i] = string(ss.buffer[i])
	}
	return lines
}

// ScreenFrame represents a screen sharing frame message
type ScreenFrame struct {
	SessionID uint32         // Screen sharing session ID
	Width     uint16         // Frame width
	Height    uint16         // Frame height
	Lines     []string       // Terminal lines
	Timestamp time.Time      // Frame timestamp
}

// EncodeScreenFrame encodes a screen frame to bytes
func EncodeScreenFrame(frame *ScreenFrame) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write session ID (4 bytes)
	if err := binary.Write(buf, binary.BigEndian, frame.SessionID); err != nil {
		return nil, err
	}

	// Write dimensions (4 bytes)
	if err := binary.Write(buf, binary.BigEndian, frame.Width); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, frame.Height); err != nil {
		return nil, err
	}

	// Write timestamp (8 bytes)
	timestamp := uint64(frame.Timestamp.UnixMilli())
	if err := binary.Write(buf, binary.BigEndian, timestamp); err != nil {
		return nil, err
	}

	// Write number of lines (2 bytes)
	lineCount := uint16(len(frame.Lines))
	if err := binary.Write(buf, binary.BigEndian, lineCount); err != nil {
		return nil, err
	}

	// Write each line
	for _, line := range frame.Lines {
		lineBytes := []byte(line)
		lineLen := uint16(len(lineBytes))

		// Write line length (2 bytes)
		if err := binary.Write(buf, binary.BigEndian, lineLen); err != nil {
			return nil, err
		}

		// Write line content
		if _, err := buf.Write(lineBytes); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// DecodeScreenFrame decodes a screen frame from bytes
func DecodeScreenFrame(data []byte) (*ScreenFrame, error) {
	if len(data) < 18 { // Minimum size: 4+2+2+8+2 = 18 bytes
		return nil, fmt.Errorf("screen frame data too short")
	}

	buf := bytes.NewReader(data)
	frame := &ScreenFrame{}

	// Read session ID
	if err := binary.Read(buf, binary.BigEndian, &frame.SessionID); err != nil {
		return nil, err
	}

	// Read dimensions
	if err := binary.Read(buf, binary.BigEndian, &frame.Width); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.BigEndian, &frame.Height); err != nil {
		return nil, err
	}

	// Read timestamp
	var timestamp uint64
	if err := binary.Read(buf, binary.BigEndian, &timestamp); err != nil {
		return nil, err
	}
	frame.Timestamp = time.UnixMilli(int64(timestamp))

	// Read number of lines
	var lineCount uint16
	if err := binary.Read(buf, binary.BigEndian, &lineCount); err != nil {
		return nil, err
	}

	// Read each line
	frame.Lines = make([]string, lineCount)
	for i := uint16(0); i < lineCount; i++ {
		var lineLen uint16
		if err := binary.Read(buf, binary.BigEndian, &lineLen); err != nil {
			return nil, err
		}

		lineBytes := make([]byte, lineLen)
		if _, err := io.ReadFull(buf, lineBytes); err != nil {
			return nil, err
		}

		frame.Lines[i] = string(lineBytes)
	}

	return frame, nil
}

// FormatLines formats lines for terminal display
func (sf *ScreenFrame) FormatLines() string {
	var sb strings.Builder
	for _, line := range sf.Lines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}
