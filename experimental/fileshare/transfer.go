//go:build experimental

package fileshare

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// ChunkSize is the size of each file chunk (64KB)
	ChunkSize = 64 * 1024

	// MaxRetries is the maximum number of retries for a chunk
	MaxRetries = 3

	// AckTimeout is the timeout waiting for chunk acknowledgment
	AckTimeout = 5 * time.Second
)

// FileTransfer represents an ongoing file transfer
type FileTransfer struct {
	mu sync.RWMutex

	ID           string
	Filename     string
	FilePath     string
	Size         int64
	MimeType     string
	Sender       string
	Recipient    string
	Direction    Direction // Send or Receive
	Status       TransferStatus
	Progress     float64
	BytesSent    int64
	BytesReceived int64
	Checksum     string // SHA256

	Chunks       []*Chunk
	ChunkCount   int
	CompletedChunks map[int]bool

	StartTime    time.Time
	EndTime      time.Time
	Speed        int64 // Bytes per second

	Throttle     int64 // Max bytes per second, 0 = unlimited
	lastSendTime time.Time
	sentBytes    int64

	OnProgress   func(progress float64)
	OnComplete   func(checksum string)
	OnError      func(err error)
}

// Direction represents transfer direction
type Direction int

const (
	DirectionSend Direction = iota
	DirectionReceive
)

func (d Direction) String() string {
	switch d {
	case DirectionSend:
		return "Send"
	case DirectionReceive:
		return "Receive"
	default:
		return "Unknown"
	}
}

// TransferStatus represents transfer status
type TransferStatus int

const (
	StatusPending TransferStatus = iota
	StatusTransferring
	StatusPaused
	StatusCompleted
	StatusFailed
	StatusCanceled
)

func (s TransferStatus) String() string {
	switch s {
	case StatusPending:
		return "Pending"
	case StatusTransferring:
		return "Transferring"
	case StatusPaused:
		return "Paused"
	case StatusCompleted:
		return "Completed"
	case StatusFailed:
		return "Failed"
	case StatusCanceled:
		return "Canceled"
	default:
		return "Unknown"
	}
}

// Chunk represents a file chunk
type Chunk struct {
	Index    int
	Offset   int64
	Size     int
	Data     []byte
	Checksum string // SHA256 of chunk
	Retries  int
	Acked    bool
}

// NewFileTransfer creates a new file transfer for sending
func NewFileTransfer(id, filePath, recipient string) (*FileTransfer, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Calculate checksum
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}
	checksum := hex.EncodeToString(hasher.Sum(nil))

	chunkCount := int((stat.Size() + ChunkSize - 1) / ChunkSize)

	ft := &FileTransfer{
		ID:              id,
		Filename:        filepath.Base(filePath),
		FilePath:        filePath,
		Size:            stat.Size(),
		Recipient:       recipient,
		Direction:       DirectionSend,
		Status:          StatusPending,
		Checksum:        checksum,
		ChunkCount:      chunkCount,
		CompletedChunks: make(map[int]bool),
		Throttle:        0, // Unlimited by default
	}

	// Create chunks
	ft.Chunks = make([]*Chunk, chunkCount)
	for i := 0; i < chunkCount; i++ {
		offset := int64(i) * ChunkSize
		size := ChunkSize
		if offset+ChunkSize > stat.Size() {
			size = int(stat.Size() - offset)
		}

		ft.Chunks[i] = &Chunk{
			Index:  i,
			Offset: offset,
			Size:   size,
		}
	}

	return ft, nil
}

// NewReceiveTransfer creates a new file transfer for receiving
func NewReceiveTransfer(id, filename, savePath, sender string, size int64, checksum string) *FileTransfer {
	chunkCount := int((size + ChunkSize - 1) / ChunkSize)

	ft := &FileTransfer{
		ID:              id,
		Filename:        filename,
		FilePath:        filepath.Join(savePath, filename),
		Size:            size,
		Sender:          sender,
		Direction:       DirectionReceive,
		Status:          StatusPending,
		Checksum:        checksum,
		ChunkCount:      chunkCount,
		CompletedChunks: make(map[int]bool),
	}

	ft.Chunks = make([]*Chunk, chunkCount)
	for i := 0; i < chunkCount; i++ {
		offset := int64(i) * ChunkSize
		chunkSize := ChunkSize
		if offset+ChunkSize > size {
			chunkSize = int(size - offset)
		}

		ft.Chunks[i] = &Chunk{
			Index:  i,
			Offset: offset,
			Size:   chunkSize,
		}
	}

	return ft
}

// Start starts the transfer
func (ft *FileTransfer) Start() error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if ft.Status != StatusPending && ft.Status != StatusPaused {
		return errors.New("transfer already started or finished")
	}

	ft.Status = StatusTransferring
	ft.StartTime = time.Now()

	return nil
}

// SendChunk reads and prepares a chunk for sending
func (ft *FileTransfer) SendChunk(index int) (*Chunk, error) {
	ft.mu.RLock()
	if index < 0 || index >= len(ft.Chunks) {
		ft.mu.RUnlock()
		return nil, errors.New("invalid chunk index")
	}

	chunk := ft.Chunks[index]
	ft.mu.RUnlock()

	// Read chunk data from file
	file, err := os.Open(ft.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data := make([]byte, chunk.Size)
	n, err := file.ReadAt(data, chunk.Offset)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read chunk: %w", err)
	}

	chunk.Data = data[:n]

	// Calculate chunk checksum
	hasher := sha256.New()
	hasher.Write(chunk.Data)
	chunk.Checksum = hex.EncodeToString(hasher.Sum(nil))

	// Apply throttling
	ft.throttle()

	return chunk, nil
}

// ReceiveChunk processes a received chunk
func (ft *FileTransfer) ReceiveChunk(chunk *Chunk) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if chunk.Index < 0 || chunk.Index >= len(ft.Chunks) {
		return errors.New("invalid chunk index")
	}

	// Verify chunk checksum
	hasher := sha256.New()
	hasher.Write(chunk.Data)
	checksum := hex.EncodeToString(hasher.Sum(nil))

	if checksum != chunk.Checksum {
		return errors.New("chunk checksum mismatch")
	}

	// Store chunk
	ft.Chunks[chunk.Index].Data = chunk.Data
	ft.Chunks[chunk.Index].Checksum = chunk.Checksum
	ft.Chunks[chunk.Index].Acked = true
	ft.CompletedChunks[chunk.Index] = true
	ft.BytesReceived += int64(len(chunk.Data))

	// Update progress
	ft.updateProgressLocked()

	// Check if complete
	if len(ft.CompletedChunks) == ft.ChunkCount {
		return ft.finalizeLocked()
	}

	return nil
}

// AcknowledgeChunk marks a chunk as acknowledged
func (ft *FileTransfer) AcknowledgeChunk(index int) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if index < 0 || index >= len(ft.Chunks) {
		return errors.New("invalid chunk index")
	}

	ft.Chunks[index].Acked = true
	ft.CompletedChunks[index] = true
	ft.BytesSent += int64(ft.Chunks[index].Size)

	// Update progress
	ft.updateProgressLocked()

	// Check if complete
	if len(ft.CompletedChunks) == ft.ChunkCount {
		ft.Status = StatusCompleted
		ft.EndTime = time.Now()

		if ft.OnComplete != nil {
			go ft.OnComplete(ft.Checksum)
		}
	}

	return nil
}

// GetNextChunk returns the next chunk to send (unacknowledged)
func (ft *FileTransfer) GetNextChunk() (int, error) {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	for i := 0; i < ft.ChunkCount; i++ {
		if !ft.Chunks[i].Acked {
			return i, nil
		}
	}

	return -1, errors.New("no chunks to send")
}

// GetMissingChunks returns indices of missing chunks
func (ft *FileTransfer) GetMissingChunks() []int {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	missing := make([]int, 0)
	for i := 0; i < ft.ChunkCount; i++ {
		if !ft.CompletedChunks[i] {
			missing = append(missing, i)
		}
	}

	return missing
}

// Pause pauses the transfer
func (ft *FileTransfer) Pause() error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if ft.Status != StatusTransferring {
		return errors.New("transfer not in progress")
	}

	ft.Status = StatusPaused

	return nil
}

// Resume resumes the transfer
func (ft *FileTransfer) Resume() error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if ft.Status != StatusPaused {
		return errors.New("transfer not paused")
	}

	ft.Status = StatusTransferring

	return nil
}

// Cancel cancels the transfer
func (ft *FileTransfer) Cancel() error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if ft.Status == StatusCompleted {
		return errors.New("transfer already completed")
	}

	ft.Status = StatusCanceled
	ft.EndTime = time.Now()

	return nil
}

// SetThrottle sets the transfer speed limit (bytes per second)
func (ft *FileTransfer) SetThrottle(bytesPerSecond int64) {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	ft.Throttle = bytesPerSecond
}

// throttle implements bandwidth throttling
func (ft *FileTransfer) throttle() {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if ft.Throttle <= 0 {
		return // No throttling
	}

	now := time.Now()

	if ft.lastSendTime.IsZero() {
		ft.lastSendTime = now
		ft.sentBytes = 0
		return
	}

	elapsed := now.Sub(ft.lastSendTime)
	if elapsed < time.Second {
		allowedBytes := int64(float64(ft.Throttle) * elapsed.Seconds())
		if ft.sentBytes >= allowedBytes {
			// Need to wait
			waitTime := time.Second - elapsed
			time.Sleep(waitTime)
			ft.lastSendTime = time.Now()
			ft.sentBytes = 0
		}
	} else {
		ft.lastSendTime = now
		ft.sentBytes = 0
	}
}

// updateProgressLocked updates progress percentage (must hold lock)
func (ft *FileTransfer) updateProgressLocked() {
	if ft.ChunkCount == 0 {
		ft.Progress = 0
		return
	}

	ft.Progress = float64(len(ft.CompletedChunks)) / float64(ft.ChunkCount)

	// Calculate speed
	if !ft.StartTime.IsZero() {
		elapsed := time.Since(ft.StartTime).Seconds()
		if elapsed > 0 {
			ft.Speed = int64(float64(ft.BytesSent+ft.BytesReceived) / elapsed)
		}
	}

	if ft.OnProgress != nil {
		go ft.OnProgress(ft.Progress)
	}
}

// finalizeLocked finalizes the received file (must hold lock)
func (ft *FileTransfer) finalizeLocked() error {
	// Create output file
	file, err := os.Create(ft.FilePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write all chunks in order
	for i := 0; i < ft.ChunkCount; i++ {
		chunk := ft.Chunks[i]
		if _, err := file.Write(chunk.Data); err != nil {
			return fmt.Errorf("failed to write chunk: %w", err)
		}
	}

	// Verify file checksum
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	finalChecksum := hex.EncodeToString(hasher.Sum(nil))
	if finalChecksum != ft.Checksum {
		ft.Status = StatusFailed
		return errors.New("file checksum mismatch")
	}

	ft.Status = StatusCompleted
	ft.EndTime = time.Now()

	if ft.OnComplete != nil {
		go ft.OnComplete(finalChecksum)
	}

	return nil
}

// GetStats returns transfer statistics
func (ft *FileTransfer) GetStats() map[string]interface{} {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["id"] = ft.ID
	stats["filename"] = ft.Filename
	stats["size"] = ft.Size
	stats["direction"] = ft.Direction.String()
	stats["status"] = ft.Status.String()
	stats["progress"] = fmt.Sprintf("%.1f%%", ft.Progress*100)
	stats["chunks_total"] = ft.ChunkCount
	stats["chunks_completed"] = len(ft.CompletedChunks)
	stats["speed"] = formatSpeed(ft.Speed)

	if !ft.StartTime.IsZero() {
		if ft.EndTime.IsZero() {
			stats["duration"] = time.Since(ft.StartTime).String()
		} else {
			stats["duration"] = ft.EndTime.Sub(ft.StartTime).String()
		}
	}

	if ft.Speed > 0 && ft.Progress < 1.0 {
		remaining := float64(ft.Size) * (1.0 - ft.Progress)
		eta := time.Duration(remaining/float64(ft.Speed)) * time.Second
		stats["eta"] = eta.String()
	}

	return stats
}

// FormatProgress returns a formatted progress string
func (ft *FileTransfer) FormatProgress() string {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	barWidth := 30
	completed := int(ft.Progress * float64(barWidth))
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < completed {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return fmt.Sprintf("[%s] %.1f%% | %s/%s | %s",
		bar,
		ft.Progress*100,
		formatBytes(ft.BytesSent+ft.BytesReceived),
		formatBytes(ft.Size),
		formatSpeed(ft.Speed))
}

// Helper functions
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatSpeed(bytesPerSecond int64) string {
	if bytesPerSecond == 0 {
		return "0 B/s"
	}

	return formatBytes(bytesPerSecond) + "/s"
}
