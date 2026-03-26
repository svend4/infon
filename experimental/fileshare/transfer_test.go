//go:build experimental

package fileshare

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func createTestFile(t *testing.T, size int) string {
	t.Helper()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile.bin")

	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	// Write test data
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	if _, err := file.Write(data); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	return filePath
}

func TestNewFileTransfer(t *testing.T) {
	filePath := createTestFile(t, 1024)

	ft, err := NewFileTransfer("test-1", filePath, "recipient")
	if err != nil {
		t.Fatalf("Failed to create file transfer: %v", err)
	}

	if ft.ID != "test-1" {
		t.Errorf("Expected ID test-1, got %s", ft.ID)
	}

	if ft.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", ft.Size)
	}

	if ft.Direction != DirectionSend {
		t.Errorf("Expected direction Send, got %v", ft.Direction)
	}

	if ft.Status != StatusPending {
		t.Errorf("Expected status Pending, got %v", ft.Status)
	}

	if len(ft.Checksum) == 0 {
		t.Error("Checksum should be calculated")
	}
}

func TestChunking(t *testing.T) {
	// Create file larger than one chunk
	filePath := createTestFile(t, ChunkSize*2+1000)

	ft, err := NewFileTransfer("test-2", filePath, "recipient")
	if err != nil {
		t.Fatalf("Failed to create file transfer: %v", err)
	}

	// Should have 3 chunks
	expectedChunks := 3
	if ft.ChunkCount != expectedChunks {
		t.Errorf("Expected %d chunks, got %d", expectedChunks, ft.ChunkCount)
	}

	if len(ft.Chunks) != expectedChunks {
		t.Errorf("Expected %d chunk objects, got %d", expectedChunks, len(ft.Chunks))
	}

	// Verify chunk sizes
	if ft.Chunks[0].Size != ChunkSize {
		t.Errorf("First chunk should be %d bytes, got %d", ChunkSize, ft.Chunks[0].Size)
	}

	if ft.Chunks[1].Size != ChunkSize {
		t.Errorf("Second chunk should be %d bytes, got %d", ChunkSize, ft.Chunks[1].Size)
	}

	if ft.Chunks[2].Size != 1000 {
		t.Errorf("Third chunk should be 1000 bytes, got %d", ft.Chunks[2].Size)
	}
}

func TestSendChunk(t *testing.T) {
	filePath := createTestFile(t, 1024)

	ft, err := NewFileTransfer("test-3", filePath, "recipient")
	if err != nil {
		t.Fatalf("Failed to create file transfer: %v", err)
	}

	chunk, err := ft.SendChunk(0)
	if err != nil {
		t.Fatalf("Failed to send chunk: %v", err)
	}

	if chunk.Index != 0 {
		t.Errorf("Expected index 0, got %d", chunk.Index)
	}

	if len(chunk.Data) != 1024 {
		t.Errorf("Expected 1024 bytes, got %d", len(chunk.Data))
	}

	if len(chunk.Checksum) == 0 {
		t.Error("Chunk checksum should be calculated")
	}
}

func TestReceiveChunk(t *testing.T) {
	filePath := createTestFile(t, 1024)

	// Create sender
	sender, err := NewFileTransfer("test-4", filePath, "recipient")
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}

	// Create receiver
	tmpDir := t.TempDir()
	receiver := NewReceiveTransfer("test-4", "testfile.bin", tmpDir, "sender", 1024, sender.Checksum)

	// Send chunk
	chunk, err := sender.SendChunk(0)
	if err != nil {
		t.Fatalf("Failed to send chunk: %v", err)
	}

	// Receive chunk
	err = receiver.ReceiveChunk(chunk)
	if err != nil {
		t.Fatalf("Failed to receive chunk: %v", err)
	}

	if !receiver.CompletedChunks[0] {
		t.Error("Chunk 0 should be marked as completed")
	}

	if receiver.BytesReceived != 1024 {
		t.Errorf("Expected 1024 bytes received, got %d", receiver.BytesReceived)
	}

	// Should be completed and file should exist
	if receiver.Status != StatusCompleted {
		t.Errorf("Expected status Completed, got %v", receiver.Status)
	}

	if _, err := os.Stat(receiver.FilePath); os.IsNotExist(err) {
		t.Error("Received file should exist")
	}
}

func TestChecksumVerification(t *testing.T) {
	filePath := createTestFile(t, 1024)

	sender, _ := NewFileTransfer("test-5", filePath, "recipient")
	tmpDir := t.TempDir()
	receiver := NewReceiveTransfer("test-5", "testfile.bin", tmpDir, "sender", 1024, sender.Checksum)

	chunk, _ := sender.SendChunk(0)

	// Corrupt the data
	chunk.Data[0] = ^chunk.Data[0]

	// Should fail checksum verification
	err := receiver.ReceiveChunk(chunk)
	if err == nil {
		t.Error("Expected error for corrupted chunk")
	}
}

func TestFileChecksumVerification(t *testing.T) {
	filePath := createTestFile(t, 1024)

	sender, _ := NewFileTransfer("test-6", filePath, "recipient")
	tmpDir := t.TempDir()

	// Wrong checksum
	receiver := NewReceiveTransfer("test-6", "testfile.bin", tmpDir, "sender", 1024, "wrongchecksum")

	chunk, _ := sender.SendChunk(0)
	receiver.ReceiveChunk(chunk)

	if receiver.Status != StatusFailed {
		t.Errorf("Expected status Failed, got %v", receiver.Status)
	}
}

func TestAcknowledgeChunk(t *testing.T) {
	filePath := createTestFile(t, 1024)

	ft, _ := NewFileTransfer("test-7", filePath, "recipient")

	err := ft.AcknowledgeChunk(0)
	if err != nil {
		t.Fatalf("Failed to acknowledge chunk: %v", err)
	}

	if !ft.Chunks[0].Acked {
		t.Error("Chunk should be marked as acknowledged")
	}

	if !ft.CompletedChunks[0] {
		t.Error("Chunk should be in completed map")
	}

	if ft.BytesSent == 0 {
		t.Error("BytesSent should be updated")
	}
}

func TestGetNextChunk(t *testing.T) {
	filePath := createTestFile(t, ChunkSize*3)

	ft, _ := NewFileTransfer("test-8", filePath, "recipient")

	// First call should return 0
	index, err := ft.GetNextChunk()
	if err != nil {
		t.Fatalf("Failed to get next chunk: %v", err)
	}

	if index != 0 {
		t.Errorf("Expected index 0, got %d", index)
	}

	// Acknowledge chunk 0
	ft.AcknowledgeChunk(0)

	// Next should be 1
	index, err = ft.GetNextChunk()
	if err != nil {
		t.Fatalf("Failed to get next chunk: %v", err)
	}

	if index != 1 {
		t.Errorf("Expected index 1, got %d", index)
	}
}

func TestGetMissingChunks(t *testing.T) {
	filePath := createTestFile(t, ChunkSize*5)

	ft, _ := NewFileTransfer("test-9", filePath, "recipient")

	// Initially all chunks are missing
	missing := ft.GetMissingChunks()
	if len(missing) != 5 {
		t.Errorf("Expected 5 missing chunks, got %d", len(missing))
	}

	// Acknowledge some chunks
	ft.AcknowledgeChunk(0)
	ft.AcknowledgeChunk(2)
	ft.AcknowledgeChunk(4)

	missing = ft.GetMissingChunks()
	if len(missing) != 2 {
		t.Errorf("Expected 2 missing chunks, got %d", len(missing))
	}

	// Should be chunks 1 and 3
	if missing[0] != 1 || missing[1] != 3 {
		t.Errorf("Expected [1, 3], got %v", missing)
	}
}

func TestStartTransfer(t *testing.T) {
	filePath := createTestFile(t, 1024)

	ft, _ := NewFileTransfer("test-10", filePath, "recipient")

	err := ft.Start()
	if err != nil {
		t.Fatalf("Failed to start transfer: %v", err)
	}

	if ft.Status != StatusTransferring {
		t.Errorf("Expected status Transferring, got %v", ft.Status)
	}

	if ft.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}

	// Try starting again - should fail
	err = ft.Start()
	if err == nil {
		t.Error("Expected error when starting already started transfer")
	}
}

func TestPauseResume(t *testing.T) {
	filePath := createTestFile(t, 1024)

	ft, _ := NewFileTransfer("test-11", filePath, "recipient")
	ft.Start()

	err := ft.Pause()
	if err != nil {
		t.Fatalf("Failed to pause: %v", err)
	}

	if ft.Status != StatusPaused {
		t.Errorf("Expected status Paused, got %v", ft.Status)
	}

	err = ft.Resume()
	if err != nil {
		t.Fatalf("Failed to resume: %v", err)
	}

	if ft.Status != StatusTransferring {
		t.Errorf("Expected status Transferring, got %v", ft.Status)
	}
}

func TestCancel(t *testing.T) {
	filePath := createTestFile(t, 1024)

	ft, _ := NewFileTransfer("test-12", filePath, "recipient")
	ft.Start()

	err := ft.Cancel()
	if err != nil {
		t.Fatalf("Failed to cancel: %v", err)
	}

	if ft.Status != StatusCanceled {
		t.Errorf("Expected status Canceled, got %v", ft.Status)
	}

	if ft.EndTime.IsZero() {
		t.Error("EndTime should be set")
	}
}

func TestProgressTracking(t *testing.T) {
	filePath := createTestFile(t, ChunkSize*4)

	ft, _ := NewFileTransfer("test-13", filePath, "recipient")

	progressCalled := false
	ft.OnProgress = func(progress float64) {
		progressCalled = true
	}

	ft.AcknowledgeChunk(0)

	if ft.Progress != 0.25 {
		t.Errorf("Expected progress 0.25, got %f", ft.Progress)
	}

	// Wait a bit for goroutine
	time.Sleep(10 * time.Millisecond)

	if !progressCalled {
		t.Error("OnProgress callback should be called")
	}

	ft.AcknowledgeChunk(1)
	if ft.Progress != 0.5 {
		t.Errorf("Expected progress 0.5, got %f", ft.Progress)
	}
}

func TestCompleteCallback(t *testing.T) {
	filePath := createTestFile(t, 1024)

	ft, _ := NewFileTransfer("test-14", filePath, "recipient")

	completeCalled := false
	var receivedChecksum string

	ft.OnComplete = func(checksum string) {
		completeCalled = true
		receivedChecksum = checksum
	}

	ft.AcknowledgeChunk(0)

	// Wait for goroutine
	time.Sleep(10 * time.Millisecond)

	if !completeCalled {
		t.Error("OnComplete callback should be called")
	}

	if receivedChecksum != ft.Checksum {
		t.Errorf("Expected checksum %s, got %s", ft.Checksum, receivedChecksum)
	}
}

func TestThrottling(t *testing.T) {
	filePath := createTestFile(t, ChunkSize)

	ft, _ := NewFileTransfer("test-15", filePath, "recipient")

	// Set throttle to 10KB/s
	ft.SetThrottle(10 * 1024)

	start := time.Now()

	// Send chunk (should be throttled)
	_, _ = ft.SendChunk(0)

	elapsed := time.Since(start)

	// Should take some time due to throttling
	// (This is a simplified test - real throttling would be tested with multiple chunks)
	if elapsed > 5*time.Second {
		t.Error("Throttling seems too aggressive")
	}
}

func TestGetStats(t *testing.T) {
	filePath := createTestFile(t, 1024)

	ft, _ := NewFileTransfer("test-16", filePath, "recipient")
	ft.Start()

	stats := ft.GetStats()

	if stats["id"] != "test-16" {
		t.Errorf("Expected id test-16, got %v", stats["id"])
	}

	if stats["size"] != int64(1024) {
		t.Errorf("Expected size 1024, got %v", stats["size"])
	}

	if stats["status"] != "Transferring" {
		t.Errorf("Expected status Transferring, got %v", stats["status"])
	}

	if stats["chunks_total"] != 1 {
		t.Errorf("Expected 1 chunk, got %v", stats["chunks_total"])
	}
}

func TestFormatProgress(t *testing.T) {
	filePath := createTestFile(t, ChunkSize*2)

	ft, _ := NewFileTransfer("test-17", filePath, "recipient")

	ft.AcknowledgeChunk(0)

	formatted := ft.FormatProgress()

	if len(formatted) == 0 {
		t.Error("Formatted progress should not be empty")
	}

	// Should contain progress bar
	if !contains(formatted, "█") && !contains(formatted, "░") {
		t.Error("Formatted progress should contain progress bar")
	}

	// Should contain percentage
	if !contains(formatted, "50.0%") {
		t.Error("Formatted progress should show percentage")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1024 * 1024, "1.0 MiB"},
		{1024 * 1024 * 1024, "1.0 GiB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestFormatSpeed(t *testing.T) {
	result := formatSpeed(0)
	if result != "0 B/s" {
		t.Errorf("Expected '0 B/s', got %s", result)
	}

	result = formatSpeed(1024)
	if result != "1.0 KiB/s" {
		t.Errorf("Expected '1.0 KiB/s', got %s", result)
	}
}

func TestDirectionString(t *testing.T) {
	if DirectionSend.String() != "Send" {
		t.Errorf("Expected 'Send', got %s", DirectionSend.String())
	}

	if DirectionReceive.String() != "Receive" {
		t.Errorf("Expected 'Receive', got %s", DirectionReceive.String())
	}
}

func TestStatusString(t *testing.T) {
	statuses := []TransferStatus{
		StatusPending, StatusTransferring, StatusPaused,
		StatusCompleted, StatusFailed, StatusCanceled,
	}

	expected := []string{
		"Pending", "Transferring", "Paused",
		"Completed", "Failed", "Canceled",
	}

	for i, status := range statuses {
		if status.String() != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], status.String())
		}
	}
}

func TestFullTransfer(t *testing.T) {
	// Create a test file
	filePath := createTestFile(t, ChunkSize*3+500)

	// Create sender
	sender, err := NewFileTransfer("full-1", filePath, "recipient")
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}

	// Create receiver
	tmpDir := t.TempDir()
	receiver := NewReceiveTransfer("full-1", "testfile.bin", tmpDir, "sender", sender.Size, sender.Checksum)

	// Start both
	sender.Start()
	receiver.Start()

	// Transfer all chunks
	for {
		index, err := sender.GetNextChunk()
		if err != nil {
			break
		}

		chunk, err := sender.SendChunk(index)
		if err != nil {
			t.Fatalf("Failed to send chunk %d: %v", index, err)
		}

		err = receiver.ReceiveChunk(chunk)
		if err != nil {
			t.Fatalf("Failed to receive chunk %d: %v", index, err)
		}

		sender.AcknowledgeChunk(index)
	}

	// Verify completion
	if sender.Status != StatusCompleted {
		t.Errorf("Sender should be completed, got %v", sender.Status)
	}

	if receiver.Status != StatusCompleted {
		t.Errorf("Receiver should be completed, got %v", receiver.Status)
	}

	// Verify file contents
	originalData, _ := os.ReadFile(filePath)
	receivedData, _ := os.ReadFile(receiver.FilePath)

	originalHash := sha256.Sum256(originalData)
	receivedHash := sha256.Sum256(receivedData)

	if hex.EncodeToString(originalHash[:]) != hex.EncodeToString(receivedHash[:]) {
		t.Error("File checksums don't match")
	}
}

// Benchmarks
func BenchmarkSendChunk(b *testing.B) {
	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "bench.bin")

	file, _ := os.Create(filePath)
	data := make([]byte, ChunkSize)
	file.Write(data)
	file.Close()

	ft, _ := NewFileTransfer("bench", filePath, "recipient")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ft.SendChunk(0)
	}
}

func BenchmarkReceiveChunk(b *testing.B) {
	tmpDir := b.TempDir()
	filePath := filepath.Join(tmpDir, "bench.bin")

	file, _ := os.Create(filePath)
	data := make([]byte, ChunkSize)
	file.Write(data)
	file.Close()

	sender, _ := NewFileTransfer("bench", filePath, "recipient")
	chunk, _ := sender.SendChunk(0)

	receiver := NewReceiveTransfer("bench", "out.bin", tmpDir, "sender", ChunkSize, sender.Checksum)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset for each iteration
		receiver.mu.Lock()
		receiver.CompletedChunks = make(map[int]bool)
		receiver.BytesReceived = 0
		receiver.Status = StatusPending
		receiver.mu.Unlock()

		receiver.ReceiveChunk(chunk)
	}
}

// Helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
