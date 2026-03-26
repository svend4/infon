//go:build experimental

package recording

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RecordingState represents the state of a recording
type RecordingState int

const (
	StateIdle RecordingState = iota
	StateRecording
	StatePaused
	StateFinished
)

func (s RecordingState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateRecording:
		return "Recording"
	case StatePaused:
		return "Paused"
	case StateFinished:
		return "Finished"
	default:
		return "Unknown"
	}
}

// RecordingFormat represents supported recording formats
type RecordingFormat int

const (
	FormatWebM RecordingFormat = iota
	FormatMP4
	FormatMKV
	FormatAudioOnly // Audio-only (WebM/Opus)
)

func (f RecordingFormat) String() string {
	switch f {
	case FormatWebM:
		return "WebM"
	case FormatMP4:
		return "MP4"
	case FormatMKV:
		return "MKV"
	case FormatAudioOnly:
		return "Audio (WebM/Opus)"
	default:
		return "Unknown"
	}
}

func (f RecordingFormat) Extension() string {
	switch f {
	case FormatWebM, FormatAudioOnly:
		return ".webm"
	case FormatMP4:
		return ".mp4"
	case FormatMKV:
		return ".mkv"
	default:
		return ".bin"
	}
}

// RecordingQuality represents quality settings
type RecordingQuality int

const (
	QualityLow RecordingQuality = iota
	QualityMedium
	QualityHigh
	QualityUltra
)

func (q RecordingQuality) String() string {
	switch q {
	case QualityLow:
		return "Low (360p)"
	case QualityMedium:
		return "Medium (720p)"
	case QualityHigh:
		return "High (1080p)"
	case QualityUltra:
		return "Ultra (4K)"
	default:
		return "Unknown"
	}
}

func (q RecordingQuality) VideoBitrate() int {
	switch q {
	case QualityLow:
		return 500_000 // 500 kbps
	case QualityMedium:
		return 2_000_000 // 2 Mbps
	case QualityHigh:
		return 5_000_000 // 5 Mbps
	case QualityUltra:
		return 15_000_000 // 15 Mbps
	default:
		return 2_000_000
	}
}

func (q RecordingQuality) AudioBitrate() int {
	switch q {
	case QualityLow:
		return 64_000 // 64 kbps
	case QualityMedium:
		return 128_000 // 128 kbps
	case QualityHigh, QualityUltra:
		return 192_000 // 192 kbps
	default:
		return 128_000
	}
}

// RecordingConfig holds recording configuration
type RecordingConfig struct {
	Format          RecordingFormat
	Quality         RecordingQuality
	OutputDir       string
	MaxDuration     time.Duration // 0 = unlimited
	MaxFileSize     int64         // bytes, 0 = unlimited
	RecordAudio     bool
	RecordVideo     bool
	RecordScreen    bool
	SeparateTracks  bool // Record each participant separately
	AutoStop        bool // Auto-stop when call ends
}

// DefaultConfig returns default recording configuration
func DefaultConfig() *RecordingConfig {
	return &RecordingConfig{
		Format:         FormatWebM,
		Quality:        QualityMedium,
		OutputDir:      "./recordings",
		MaxDuration:    4 * time.Hour,
		MaxFileSize:    10 * 1024 * 1024 * 1024, // 10 GB
		RecordAudio:    true,
		RecordVideo:    true,
		RecordScreen:   false,
		SeparateTracks: false,
		AutoStop:       true,
	}
}

// Recorder manages call recording
type Recorder struct {
	mu sync.RWMutex

	ID            string
	CallID        string
	Config        *RecordingConfig
	State         RecordingState
	StartTime     time.Time
	EndTime       time.Time
	PausedAt      time.Time
	TotalPaused   time.Duration
	OutputPath    string
	FileSize      int64
	Error         error

	// Tracks
	audioWriter io.WriteCloser
	videoWriter io.WriteCloser
	screenWriter io.WriteCloser

	// Metadata
	Participants []string
	Metadata     map[string]interface{}

	// Callbacks
	OnStart      func()
	OnStop       func(outputPath string)
	OnPause      func()
	OnResume     func()
	OnError      func(err error)
	OnSizeLimit  func()
	OnTimeLimit  func()
}

// NewRecorder creates a new recorder
func NewRecorder(id, callID string, config *RecordingConfig) *Recorder {
	if config == nil {
		config = DefaultConfig()
	}

	return &Recorder{
		ID:           id,
		CallID:       callID,
		Config:       config,
		State:        StateIdle,
		Participants: make([]string, 0),
		Metadata:     make(map[string]interface{}),
	}
}

// Start starts the recording
func (r *Recorder) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != StateIdle {
		return errors.New("recording already started")
	}

	// Create output directory
	if err := os.MkdirAll(r.Config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate output filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("recording_%s_%s%s", r.CallID, timestamp, r.Config.Format.Extension())
	r.OutputPath = filepath.Join(r.Config.OutputDir, filename)

	// Open output file
	file, err := os.Create(r.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	// For now, use a single writer for the combined stream
	// In production, this would open separate audio/video/screen writers
	r.audioWriter = file

	r.State = StateRecording
	r.StartTime = time.Now()
	r.FileSize = 0

	// Start monitoring goroutine
	go r.monitorLimits()

	if r.OnStart != nil {
		go r.OnStart()
	}

	return nil
}

// Stop stops the recording
func (r *Recorder) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != StateRecording && r.State != StatePaused {
		return errors.New("recording not active")
	}

	// Close writers
	if r.audioWriter != nil {
		r.audioWriter.Close()
	}
	if r.videoWriter != nil {
		r.videoWriter.Close()
	}
	if r.screenWriter != nil {
		r.screenWriter.Close()
	}

	r.State = StateFinished
	r.EndTime = time.Now()

	if r.OnStop != nil {
		go r.OnStop(r.OutputPath)
	}

	return nil
}

// Pause pauses the recording
func (r *Recorder) Pause() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != StateRecording {
		return errors.New("not recording")
	}

	r.State = StatePaused
	r.PausedAt = time.Now()

	if r.OnPause != nil {
		go r.OnPause()
	}

	return nil
}

// Resume resumes the recording
func (r *Recorder) Resume() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != StatePaused {
		return errors.New("not paused")
	}

	r.TotalPaused += time.Since(r.PausedAt)
	r.State = StateRecording

	if r.OnResume != nil {
		go r.OnResume()
	}

	return nil
}

// WriteAudio writes audio data
func (r *Recorder) WriteAudio(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != StateRecording {
		return 0, errors.New("not recording")
	}

	if !r.Config.RecordAudio {
		return 0, nil
	}

	if r.audioWriter == nil {
		return 0, errors.New("audio writer not initialized")
	}

	n, err := r.audioWriter.Write(data)
	if err != nil {
		r.handleError(fmt.Errorf("audio write error: %w", err))
		return n, err
	}

	r.FileSize += int64(n)
	return n, nil
}

// WriteVideo writes video data
func (r *Recorder) WriteVideo(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != StateRecording {
		return 0, errors.New("not recording")
	}

	if !r.Config.RecordVideo {
		return 0, nil
	}

	if r.videoWriter == nil {
		// For single-stream recording, write to audio writer
		if r.audioWriter != nil {
			n, err := r.audioWriter.Write(data)
			if err != nil {
				r.handleError(fmt.Errorf("video write error: %w", err))
				return n, err
			}
			r.FileSize += int64(n)
			return n, nil
		}
		return 0, errors.New("video writer not initialized")
	}

	n, err := r.videoWriter.Write(data)
	if err != nil {
		r.handleError(fmt.Errorf("video write error: %w", err))
		return n, err
	}

	r.FileSize += int64(n)
	return n, nil
}

// WriteScreen writes screen share data
func (r *Recorder) WriteScreen(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != StateRecording {
		return 0, errors.New("not recording")
	}

	if !r.Config.RecordScreen {
		return 0, nil
	}

	if r.screenWriter == nil {
		return 0, errors.New("screen writer not initialized")
	}

	n, err := r.screenWriter.Write(data)
	if err != nil {
		r.handleError(fmt.Errorf("screen write error: %w", err))
		return n, err
	}

	r.FileSize += int64(n)
	return n, nil
}

// AddParticipant adds a participant to the recording metadata
func (r *Recorder) AddParticipant(participantID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Participants = append(r.Participants, participantID)
}

// GetDuration returns the current recording duration
func (r *Recorder) GetDuration() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.State == StateIdle {
		return 0
	}

	if r.State == StateFinished {
		return r.EndTime.Sub(r.StartTime) - r.TotalPaused
	}

	duration := time.Since(r.StartTime) - r.TotalPaused
	if r.State == StatePaused {
		duration -= time.Since(r.PausedAt)
	}

	return duration
}

// GetFileSize returns the current file size
func (r *Recorder) GetFileSize() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.FileSize
}

// GetStats returns recording statistics
func (r *Recorder) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"id":             r.ID,
		"call_id":        r.CallID,
		"state":          r.State.String(),
		"duration":       r.GetDuration().String(),
		"file_size":      r.FileSize,
		"format":         r.Config.Format.String(),
		"quality":        r.Config.Quality.String(),
		"output_path":    r.OutputPath,
		"participants":   len(r.Participants),
		"started_at":     r.StartTime,
		"record_audio":   r.Config.RecordAudio,
		"record_video":   r.Config.RecordVideo,
		"record_screen":  r.Config.RecordScreen,
	}
}

// monitorLimits monitors recording limits
func (r *Recorder) monitorLimits() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.RLock()
		state := r.State
		duration := r.GetDuration()
		fileSize := r.FileSize
		maxDuration := r.Config.MaxDuration
		maxFileSize := r.Config.MaxFileSize
		r.mu.RUnlock()

		if state != StateRecording {
			return
		}

		// Check duration limit
		if maxDuration > 0 && duration >= maxDuration {
			if r.OnTimeLimit != nil {
				r.OnTimeLimit()
			}
			r.Stop()
			return
		}

		// Check file size limit
		if maxFileSize > 0 && fileSize >= maxFileSize {
			if r.OnSizeLimit != nil {
				r.OnSizeLimit()
			}
			r.Stop()
			return
		}
	}
}

// handleError handles recording errors
func (r *Recorder) handleError(err error) {
	r.Error = err

	if r.OnError != nil {
		go r.OnError(err)
	}
}

// RecordingManager manages multiple recordings
type RecordingManager struct {
	mu sync.RWMutex

	recordings map[string]*Recorder
	config     *RecordingConfig
}

// NewRecordingManager creates a new recording manager
func NewRecordingManager(config *RecordingConfig) *RecordingManager {
	if config == nil {
		config = DefaultConfig()
	}

	return &RecordingManager{
		recordings: make(map[string]*Recorder),
		config:     config,
	}
}

// StartRecording starts a new recording
func (rm *RecordingManager) StartRecording(callID string) (*Recorder, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Check if already recording this call
	if _, exists := rm.recordings[callID]; exists {
		return nil, errors.New("call already being recorded")
	}

	// Create recorder
	id := fmt.Sprintf("rec-%d", time.Now().UnixNano())
	recorder := NewRecorder(id, callID, rm.config)

	// Start recording
	if err := recorder.Start(); err != nil {
		return nil, err
	}

	rm.recordings[callID] = recorder
	return recorder, nil
}

// StopRecording stops a recording
func (rm *RecordingManager) StopRecording(callID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	recorder, exists := rm.recordings[callID]
	if !exists {
		return errors.New("recording not found")
	}

	if err := recorder.Stop(); err != nil {
		return err
	}

	delete(rm.recordings, callID)
	return nil
}

// GetRecording gets a recording by call ID
func (rm *RecordingManager) GetRecording(callID string) (*Recorder, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	recorder, exists := rm.recordings[callID]
	if !exists {
		return nil, errors.New("recording not found")
	}

	return recorder, nil
}

// GetAllRecordings returns all active recordings
func (rm *RecordingManager) GetAllRecordings() []*Recorder {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	recordings := make([]*Recorder, 0, len(rm.recordings))
	for _, recorder := range rm.recordings {
		recordings = append(recordings, recorder)
	}

	return recordings
}

// GetActiveCount returns the number of active recordings
func (rm *RecordingManager) GetActiveCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return len(rm.recordings)
}

// StopAll stops all active recordings
func (rm *RecordingManager) StopAll() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var lastErr error
	for callID, recorder := range rm.recordings {
		if err := recorder.Stop(); err != nil {
			lastErr = err
		}
		delete(rm.recordings, callID)
	}

	return lastErr
}
