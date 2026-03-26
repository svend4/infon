//go:build experimental

package recording

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewRecorder tests recorder creation
func TestNewRecorder(t *testing.T) {
	config := DefaultConfig()
	recorder := NewRecorder("rec1", "call1", config)

	if recorder.ID != "rec1" {
		t.Errorf("Expected ID 'rec1', got '%s'", recorder.ID)
	}

	if recorder.CallID != "call1" {
		t.Errorf("Expected CallID 'call1', got '%s'", recorder.CallID)
	}

	if recorder.State != StateIdle {
		t.Errorf("Expected state Idle, got %v", recorder.State)
	}

	if recorder.Config == nil {
		t.Error("Config should not be nil")
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Format != FormatWebM {
		t.Errorf("Expected format WebM, got %v", config.Format)
	}

	if config.Quality != QualityMedium {
		t.Errorf("Expected quality Medium, got %v", config.Quality)
	}

	if !config.RecordAudio {
		t.Error("RecordAudio should be true by default")
	}

	if !config.RecordVideo {
		t.Error("RecordVideo should be true by default")
	}
}

// TestRecorderStart tests starting a recording
func TestRecorderStart(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	err := recorder.Start()
	if err != nil {
		t.Fatalf("Failed to start recording: %v", err)
	}

	if recorder.State != StateRecording {
		t.Errorf("Expected state Recording, got %v", recorder.State)
	}

	if recorder.OutputPath == "" {
		t.Error("OutputPath should be set")
	}

	if !recorder.StartTime.After(time.Time{}) {
		t.Error("StartTime should be set")
	}

	// Cleanup
	recorder.Stop()
}

// TestRecorderStop tests stopping a recording
func TestRecorderStop(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	recorder.Start()

	err := recorder.Stop()
	if err != nil {
		t.Fatalf("Failed to stop recording: %v", err)
	}

	if recorder.State != StateFinished {
		t.Errorf("Expected state Finished, got %v", recorder.State)
	}

	if !recorder.EndTime.After(recorder.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
}

// TestRecorderPauseResume tests pausing and resuming
func TestRecorderPauseResume(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	recorder.Start()

	// Pause
	err := recorder.Pause()
	if err != nil {
		t.Fatalf("Failed to pause recording: %v", err)
	}

	if recorder.State != StatePaused {
		t.Errorf("Expected state Paused, got %v", recorder.State)
	}

	// Resume
	time.Sleep(100 * time.Millisecond)
	err = recorder.Resume()
	if err != nil {
		t.Fatalf("Failed to resume recording: %v", err)
	}

	if recorder.State != StateRecording {
		t.Errorf("Expected state Recording, got %v", recorder.State)
	}

	if recorder.TotalPaused == 0 {
		t.Error("TotalPaused should be greater than 0")
	}

	// Cleanup
	recorder.Stop()
}

// TestRecorderWriteAudio tests writing audio data
func TestRecorderWriteAudio(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	recorder.Start()

	data := []byte("audio data")
	n, err := recorder.WriteAudio(data)
	if err != nil {
		t.Fatalf("Failed to write audio: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	if recorder.GetFileSize() != int64(len(data)) {
		t.Errorf("Expected file size %d, got %d", len(data), recorder.GetFileSize())
	}

	// Cleanup
	recorder.Stop()
}

// TestRecorderWriteVideo tests writing video data
func TestRecorderWriteVideo(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	recorder.Start()

	data := []byte("video data")
	n, err := recorder.WriteVideo(data)
	if err != nil {
		t.Fatalf("Failed to write video: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Cleanup
	recorder.Stop()
}

// TestRecorderDuration tests duration calculation
func TestRecorderDuration(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	// Before start
	if recorder.GetDuration() != 0 {
		t.Error("Duration should be 0 before start")
	}

	recorder.Start()
	time.Sleep(100 * time.Millisecond)

	duration := recorder.GetDuration()
	if duration < 50*time.Millisecond {
		t.Errorf("Duration should be at least 50ms, got %v", duration)
	}

	// Cleanup
	recorder.Stop()
}

// TestRecorderPauseSubtractsFromDuration tests that pause time is subtracted
func TestRecorderPauseSubtractsFromDuration(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	recorder.Start()
	time.Sleep(100 * time.Millisecond)

	recorder.Pause()
	time.Sleep(100 * time.Millisecond)

	recorder.Resume()
	time.Sleep(50 * time.Millisecond)

	recorder.Stop()

	duration := recorder.GetDuration()
	// Duration should be around 150ms (100ms + 50ms), not 250ms
	if duration > 200*time.Millisecond {
		t.Errorf("Duration should not include paused time, got %v", duration)
	}
}

// TestRecorderAddParticipant tests adding participants
func TestRecorderAddParticipant(t *testing.T) {
	recorder := NewRecorder("rec1", "call1", DefaultConfig())

	recorder.AddParticipant("user1")
	recorder.AddParticipant("user2")

	if len(recorder.Participants) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(recorder.Participants))
	}

	if recorder.Participants[0] != "user1" {
		t.Errorf("Expected first participant 'user1', got '%s'", recorder.Participants[0])
	}
}

// TestRecorderGetStats tests statistics retrieval
func TestRecorderGetStats(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	recorder.Start()
	recorder.WriteAudio([]byte("test data"))

	stats := recorder.GetStats()

	if stats["id"] != recorder.ID {
		t.Error("Stats should include ID")
	}

	if stats["state"] != recorder.State.String() {
		t.Error("Stats should include state")
	}

	if stats["file_size"] != recorder.FileSize {
		t.Error("Stats should include file size")
	}

	// Cleanup
	recorder.Stop()
}

// TestRecorderCallbacks tests callback functions
func TestRecorderCallbacks(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	startCalled := false
	stopCalled := false
	pauseCalled := false
	resumeCalled := false

	recorder.OnStart = func() { startCalled = true }
	recorder.OnStop = func(path string) { stopCalled = true }
	recorder.OnPause = func() { pauseCalled = true }
	recorder.OnResume = func() { resumeCalled = true }

	recorder.Start()
	time.Sleep(50 * time.Millisecond)

	if !startCalled {
		t.Error("OnStart callback should be called")
	}

	recorder.Pause()
	time.Sleep(50 * time.Millisecond)

	if !pauseCalled {
		t.Error("OnPause callback should be called")
	}

	recorder.Resume()
	time.Sleep(50 * time.Millisecond)

	if !resumeCalled {
		t.Error("OnResume callback should be called")
	}

	recorder.Stop()
	time.Sleep(50 * time.Millisecond)

	if !stopCalled {
		t.Error("OnStop callback should be called")
	}
}

// TestRecorderInvalidStateTransitions tests invalid state transitions
func TestRecorderInvalidStateTransitions(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	// Can't stop before starting
	err := recorder.Stop()
	if err == nil {
		t.Error("Should not be able to stop before starting")
	}

	// Can't pause before starting
	err = recorder.Pause()
	if err == nil {
		t.Error("Should not be able to pause before starting")
	}

	// Can't resume before pausing
	err = recorder.Resume()
	if err == nil {
		t.Error("Should not be able to resume before pausing")
	}

	recorder.Start()

	// Can't start again
	err = recorder.Start()
	if err == nil {
		t.Error("Should not be able to start again")
	}

	recorder.Stop()

	// Can't pause after stopping
	err = recorder.Pause()
	if err == nil {
		t.Error("Should not be able to pause after stopping")
	}
}

// TestRecorderMaxDuration tests maximum duration limit
func TestRecorderMaxDuration(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	config.MaxDuration = 200 * time.Millisecond
	recorder := NewRecorder("rec1", "call1", config)

	limitReached := false
	recorder.OnTimeLimit = func() { limitReached = true }

	recorder.Start()

	// Wait for limit to be reached, checking periodically
	maxWait := 2 * time.Second
	checkInterval := 100 * time.Millisecond
	elapsed := time.Duration(0)

	for elapsed < maxWait {
		time.Sleep(checkInterval)
		elapsed += checkInterval

		if limitReached && recorder.State == StateFinished {
			return // Test passed
		}
	}

	if !limitReached {
		t.Error("Time limit callback should be called")
	}

	if recorder.State != StateFinished {
		t.Errorf("Recording should be finished, got state %v", recorder.State)
	}
}

// TestRecorderOutputFileCreation tests output file creation
func TestRecorderOutputFileCreation(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	recorder := NewRecorder("rec1", "call1", config)

	recorder.Start()
	recorder.WriteAudio([]byte("test audio data"))
	recorder.Stop()

	// Check if file exists
	if _, err := os.Stat(recorder.OutputPath); os.IsNotExist(err) {
		t.Errorf("Output file should exist at %s", recorder.OutputPath)
	}

	// Check if file has correct extension
	ext := filepath.Ext(recorder.OutputPath)
	expectedExt := config.Format.Extension()
	if ext != expectedExt {
		t.Errorf("Expected extension %s, got %s", expectedExt, ext)
	}
}

// TestRecordingFormats tests different recording formats
func TestRecordingFormats(t *testing.T) {
	formats := []RecordingFormat{
		FormatWebM,
		FormatMP4,
		FormatMKV,
		FormatAudioOnly,
	}

	expectedExtensions := []string{
		".webm",
		".mp4",
		".mkv",
		".webm",
	}

	for i, format := range formats {
		ext := format.Extension()
		if ext != expectedExtensions[i] {
			t.Errorf("Format %v: expected extension %s, got %s", format, expectedExtensions[i], ext)
		}

		str := format.String()
		if str == "Unknown" {
			t.Errorf("Format %d should have a name", format)
		}
	}
}

// TestRecordingQualitySettings tests quality settings
func TestRecordingQualitySettings(t *testing.T) {
	qualities := []RecordingQuality{
		QualityLow,
		QualityMedium,
		QualityHigh,
		QualityUltra,
	}

	for _, quality := range qualities {
		videoBitrate := quality.VideoBitrate()
		if videoBitrate <= 0 {
			t.Errorf("Quality %v: video bitrate should be positive, got %d", quality, videoBitrate)
		}

		audioBitrate := quality.AudioBitrate()
		if audioBitrate <= 0 {
			t.Errorf("Quality %v: audio bitrate should be positive, got %d", quality, audioBitrate)
		}

		str := quality.String()
		if str == "Unknown" {
			t.Errorf("Quality %d should have a name", quality)
		}
	}
}

// TestRecordingManager tests recording manager
func TestRecordingManager(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	manager := NewRecordingManager(config)

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	if manager.GetActiveCount() != 0 {
		t.Error("Active count should be 0 initially")
	}
}

// TestRecordingManagerStartStop tests starting and stopping recordings
func TestRecordingManagerStartStop(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	manager := NewRecordingManager(config)

	// Start recording
	recorder, err := manager.StartRecording("call1")
	if err != nil {
		t.Fatalf("Failed to start recording: %v", err)
	}

	if recorder == nil {
		t.Fatal("Recorder should not be nil")
	}

	if manager.GetActiveCount() != 1 {
		t.Errorf("Expected 1 active recording, got %d", manager.GetActiveCount())
	}

	// Stop recording
	err = manager.StopRecording("call1")
	if err != nil {
		t.Fatalf("Failed to stop recording: %v", err)
	}

	if manager.GetActiveCount() != 0 {
		t.Errorf("Expected 0 active recordings after stop, got %d", manager.GetActiveCount())
	}
}

// TestRecordingManagerDuplicateStart tests preventing duplicate recordings
func TestRecordingManagerDuplicateStart(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	manager := NewRecordingManager(config)

	// Start first recording
	_, err := manager.StartRecording("call1")
	if err != nil {
		t.Fatalf("Failed to start first recording: %v", err)
	}

	// Try to start duplicate
	_, err = manager.StartRecording("call1")
	if err == nil {
		t.Error("Should not allow duplicate recording for same call")
	}

	// Cleanup
	manager.StopRecording("call1")
}

// TestRecordingManagerGetRecording tests retrieving recordings
func TestRecordingManagerGetRecording(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	manager := NewRecordingManager(config)

	// Start recording
	original, _ := manager.StartRecording("call1")

	// Get recording
	retrieved, err := manager.GetRecording("call1")
	if err != nil {
		t.Fatalf("Failed to get recording: %v", err)
	}

	if retrieved.ID != original.ID {
		t.Error("Retrieved recorder should match original")
	}

	// Try to get non-existent recording
	_, err = manager.GetRecording("call2")
	if err == nil {
		t.Error("Should return error for non-existent recording")
	}

	// Cleanup
	manager.StopRecording("call1")
}

// TestRecordingManagerGetAllRecordings tests getting all recordings
func TestRecordingManagerGetAllRecordings(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	manager := NewRecordingManager(config)

	// Start multiple recordings
	manager.StartRecording("call1")
	manager.StartRecording("call2")
	manager.StartRecording("call3")

	recordings := manager.GetAllRecordings()

	if len(recordings) != 3 {
		t.Errorf("Expected 3 recordings, got %d", len(recordings))
	}

	// Cleanup
	manager.StopAll()
}

// TestRecordingManagerStopAll tests stopping all recordings
func TestRecordingManagerStopAll(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.OutputDir = tmpDir
	manager := NewRecordingManager(config)

	// Start multiple recordings
	manager.StartRecording("call1")
	manager.StartRecording("call2")

	// Stop all
	err := manager.StopAll()
	if err != nil {
		t.Fatalf("Failed to stop all recordings: %v", err)
	}

	if manager.GetActiveCount() != 0 {
		t.Errorf("Expected 0 active recordings after StopAll, got %d", manager.GetActiveCount())
	}
}

// BenchmarkRecorderStart benchmarks starting a recording
func BenchmarkRecorderStart(b *testing.B) {
	tmpDir := b.TempDir()
	config := DefaultConfig()
	config.OutputDir = tmpDir

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := NewRecorder("rec1", "call1", config)
		recorder.Start()
		recorder.Stop()
	}
}

// BenchmarkRecorderWriteAudio benchmarks writing audio
func BenchmarkRecorderWriteAudio(b *testing.B) {
	tmpDir := b.TempDir()
	config := DefaultConfig()
	config.OutputDir = tmpDir

	recorder := NewRecorder("rec1", "call1", config)
	recorder.Start()
	defer recorder.Stop()

	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder.WriteAudio(data)
	}
}

// BenchmarkRecordingManager benchmarks recording manager operations
func BenchmarkRecordingManager(b *testing.B) {
	tmpDir := b.TempDir()
	config := DefaultConfig()
	config.OutputDir = tmpDir

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager := NewRecordingManager(config)
		manager.StartRecording("call1")
		manager.StopRecording("call1")
	}
}
