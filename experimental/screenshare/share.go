//go:build experimental

package screenshare

import (
	"errors"
	"sync"
	"time"
)

// ShareState represents screen sharing state
type ShareState int

const (
	StateIdle ShareState = iota
	StateSharing
	StatePaused
	StateStopped
)

func (s ShareState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateSharing:
		return "Sharing"
	case StatePaused:
		return "Paused"
	case StateStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

// ShareType represents what is being shared
type ShareType int

const (
	TypeFullScreen ShareType = iota
	TypeWindow
	TypeApplication
	TypeRegion
)

func (t ShareType) String() string {
	switch t {
	case TypeFullScreen:
		return "Full Screen"
	case TypeWindow:
		return "Window"
	case TypeApplication:
		return "Application"
	case TypeRegion:
		return "Region"
	default:
		return "Unknown"
	}
}

// Quality represents streaming quality
type Quality int

const (
	QualityLow Quality = iota
	QualityMedium
	QualityHigh
	QualityAuto
)

func (q Quality) String() string {
	switch q {
	case QualityLow:
		return "Low (720p@15fps)"
	case QualityMedium:
		return "Medium (1080p@30fps)"
	case QualityHigh:
		return "High (1080p@60fps)"
	case QualityAuto:
		return "Auto"
	default:
		return "Unknown"
	}
}

func (q Quality) FPS() int {
	switch q {
	case QualityLow:
		return 15
	case QualityMedium:
		return 30
	case QualityHigh:
		return 60
	case QualityAuto:
		return 30
	default:
		return 30
	}
}

func (q Quality) Bitrate() int {
	switch q {
	case QualityLow:
		return 1_000_000 // 1 Mbps
	case QualityMedium:
		return 2_500_000 // 2.5 Mbps
	case QualityHigh:
		return 5_000_000 // 5 Mbps
	case QualityAuto:
		return 2_500_000
	default:
		return 2_500_000
	}
}

// Frame represents a video frame
type Frame struct {
	Data      []byte
	Width     int
	Height    int
	Timestamp time.Time
	FrameNum  int64
}

// Config holds screen sharing configuration
type Config struct {
	Type              ShareType
	Quality           Quality
	ShareAudio        bool
	ShareSystemAudio  bool
	AllowRemoteControl bool
	MaxViewers        int
	Width             int
	Height            int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Type:              TypeFullScreen,
		Quality:           QualityMedium,
		ShareAudio:        false,
		ShareSystemAudio:  false,
		AllowRemoteControl: false,
		MaxViewers:        50,
		Width:             1920,
		Height:            1080,
	}
}

// ScreenShare manages screen sharing
type ScreenShare struct {
	mu sync.RWMutex

	ID            string
	UserID        string
	UserName      string
	CallID        string
	Config        *Config
	State         ShareState
	StartTime     time.Time
	PausedAt      time.Time
	TotalPaused   time.Duration
	FrameCount    int64
	BytesSent     int64
	Viewers       map[string]*Viewer
	CurrentFrame  *Frame

	// Callbacks
	OnStart       func()
	OnStop        func()
	OnPause       func()
	OnResume      func()
	OnFrameSent   func(frame *Frame)
	OnViewerJoin  func(viewer *Viewer)
	OnViewerLeave func(viewerID string)
	OnError       func(err error)
}

// Viewer represents someone viewing the share
type Viewer struct {
	ID           string
	UserID       string
	UserName     string
	JoinedAt     time.Time
	LastFrameAt  time.Time
	FramesReceived int64
	BytesReceived int64
	IsActive     bool
}

// NewScreenShare creates a new screen share
func NewScreenShare(id, userID, userName, callID string, config *Config) *ScreenShare {
	if config == nil {
		config = DefaultConfig()
	}

	return &ScreenShare{
		ID:       id,
		UserID:   userID,
		UserName: userName,
		CallID:   callID,
		Config:   config,
		State:    StateIdle,
		Viewers:  make(map[string]*Viewer),
	}
}

// Start starts screen sharing
func (ss *ScreenShare) Start() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.State != StateIdle {
		return errors.New("already sharing")
	}

	ss.State = StateSharing
	ss.StartTime = time.Now()
	ss.FrameCount = 0
	ss.BytesSent = 0

	if ss.OnStart != nil {
		go ss.OnStart()
	}

	return nil
}

// Stop stops screen sharing
func (ss *ScreenShare) Stop() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.State != StateSharing && ss.State != StatePaused {
		return errors.New("not sharing")
	}

	ss.State = StateStopped

	if ss.OnStop != nil {
		go ss.OnStop()
	}

	return nil
}

// Pause pauses screen sharing
func (ss *ScreenShare) Pause() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.State != StateSharing {
		return errors.New("not sharing")
	}

	ss.State = StatePaused
	ss.PausedAt = time.Now()

	if ss.OnPause != nil {
		go ss.OnPause()
	}

	return nil
}

// Resume resumes screen sharing
func (ss *ScreenShare) Resume() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.State != StatePaused {
		return errors.New("not paused")
	}

	ss.TotalPaused += time.Since(ss.PausedAt)
	ss.State = StateSharing

	if ss.OnResume != nil {
		go ss.OnResume()
	}

	return nil
}

// SendFrame sends a video frame to viewers
func (ss *ScreenShare) SendFrame(frame *Frame) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.State != StateSharing {
		return errors.New("not sharing")
	}

	frame.FrameNum = ss.FrameCount
	frame.Timestamp = time.Now()

	ss.CurrentFrame = frame
	ss.FrameCount++
	ss.BytesSent += int64(len(frame.Data))

	// Update viewers
	for _, viewer := range ss.Viewers {
		if viewer.IsActive {
			viewer.LastFrameAt = time.Now()
			viewer.FramesReceived++
			viewer.BytesReceived += int64(len(frame.Data))
		}
	}

	if ss.OnFrameSent != nil {
		go ss.OnFrameSent(frame)
	}

	return nil
}

// AddViewer adds a viewer
func (ss *ScreenShare) AddViewer(userID, userName string) (*Viewer, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.Config.MaxViewers > 0 && len(ss.Viewers) >= ss.Config.MaxViewers {
		return nil, errors.New("max viewers reached")
	}

	if _, exists := ss.Viewers[userID]; exists {
		return nil, errors.New("viewer already exists")
	}

	viewer := &Viewer{
		ID:       userID,
		UserID:   userID,
		UserName: userName,
		JoinedAt: time.Now(),
		IsActive: true,
	}

	ss.Viewers[userID] = viewer

	if ss.OnViewerJoin != nil {
		go ss.OnViewerJoin(viewer)
	}

	return viewer, nil
}

// RemoveViewer removes a viewer
func (ss *ScreenShare) RemoveViewer(userID string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if _, exists := ss.Viewers[userID]; !exists {
		return errors.New("viewer not found")
	}

	delete(ss.Viewers, userID)

	if ss.OnViewerLeave != nil {
		go ss.OnViewerLeave(userID)
	}

	return nil
}

// GetViewer gets a viewer
func (ss *ScreenShare) GetViewer(userID string) (*Viewer, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	viewer, exists := ss.Viewers[userID]
	if !exists {
		return nil, errors.New("viewer not found")
	}

	return viewer, nil
}

// GetViewers returns all viewers
func (ss *ScreenShare) GetViewers() []*Viewer {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	viewers := make([]*Viewer, 0, len(ss.Viewers))
	for _, viewer := range ss.Viewers {
		viewers = append(viewers, viewer)
	}

	return viewers
}

// GetViewerCount returns the number of viewers
func (ss *ScreenShare) GetViewerCount() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	return len(ss.Viewers)
}

// GetActiveViewerCount returns the number of active viewers
func (ss *ScreenShare) GetActiveViewerCount() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	count := 0
	for _, viewer := range ss.Viewers {
		if viewer.IsActive {
			count++
		}
	}

	return count
}

// GetDuration returns sharing duration
func (ss *ScreenShare) GetDuration() time.Duration {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if ss.State == StateIdle {
		return 0
	}

	duration := time.Since(ss.StartTime) - ss.TotalPaused
	if ss.State == StatePaused {
		duration -= time.Since(ss.PausedAt)
	}

	return duration
}

// GetFPS calculates current FPS
func (ss *ScreenShare) GetFPS() float64 {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	duration := ss.GetDuration()
	if duration == 0 {
		return 0
	}

	return float64(ss.FrameCount) / duration.Seconds()
}

// GetBitrate calculates current bitrate in bps
func (ss *ScreenShare) GetBitrate() float64 {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	duration := ss.GetDuration()
	if duration == 0 {
		return 0
	}

	return float64(ss.BytesSent*8) / duration.Seconds()
}

// GetStats returns sharing statistics
func (ss *ScreenShare) GetStats() map[string]interface{} {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	return map[string]interface{}{
		"id":             ss.ID,
		"user_id":        ss.UserID,
		"user_name":      ss.UserName,
		"call_id":        ss.CallID,
		"state":          ss.State.String(),
		"type":           ss.Config.Type.String(),
		"quality":        ss.Config.Quality.String(),
		"duration":       ss.GetDuration().String(),
		"frame_count":    ss.FrameCount,
		"bytes_sent":     ss.BytesSent,
		"viewer_count":   len(ss.Viewers),
		"active_viewers": ss.GetActiveViewerCount(),
		"fps":            ss.GetFPS(),
		"bitrate":        ss.GetBitrate(),
		"started_at":     ss.StartTime,
	}
}

// ShareManager manages multiple screen shares
type ShareManager struct {
	mu sync.RWMutex

	shares map[string]*ScreenShare
}

// NewShareManager creates a new share manager
func NewShareManager() *ShareManager {
	return &ShareManager{
		shares: make(map[string]*ScreenShare),
	}
}

// CreateShare creates a new screen share
func (sm *ShareManager) CreateShare(id, userID, userName, callID string, config *Config) (*ScreenShare, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.shares[id]; exists {
		return nil, errors.New("share already exists")
	}

	share := NewScreenShare(id, userID, userName, callID, config)
	sm.shares[id] = share

	return share, nil
}

// GetShare gets a share by ID
func (sm *ShareManager) GetShare(id string) (*ScreenShare, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	share, exists := sm.shares[id]
	if !exists {
		return nil, errors.New("share not found")
	}

	return share, nil
}

// GetShareByUser gets shares by user ID
func (sm *ShareManager) GetShareByUser(userID string) []*ScreenShare {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shares := make([]*ScreenShare, 0)
	for _, share := range sm.shares {
		if share.UserID == userID {
			shares = append(shares, share)
		}
	}

	return shares
}

// GetSharesByCall gets shares by call ID
func (sm *ShareManager) GetSharesByCall(callID string) []*ScreenShare {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shares := make([]*ScreenShare, 0)
	for _, share := range sm.shares {
		if share.CallID == callID {
			shares = append(shares, share)
		}
	}

	return shares
}

// DeleteShare deletes a share
func (sm *ShareManager) DeleteShare(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.shares[id]; !exists {
		return errors.New("share not found")
	}

	delete(sm.shares, id)
	return nil
}

// GetAllShares returns all shares
func (sm *ShareManager) GetAllShares() []*ScreenShare {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shares := make([]*ScreenShare, 0, len(sm.shares))
	for _, share := range sm.shares {
		shares = append(shares, share)
	}

	return shares
}

// GetActiveShares returns all active shares
func (sm *ShareManager) GetActiveShares() []*ScreenShare {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	shares := make([]*ScreenShare, 0)
	for _, share := range sm.shares {
		if share.State == StateSharing {
			shares = append(shares, share)
		}
	}

	return shares
}

// GetShareCount returns the number of shares
func (sm *ShareManager) GetShareCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.shares)
}

// GetActiveShareCount returns the number of active shares
func (sm *ShareManager) GetActiveShareCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, share := range sm.shares {
		if share.State == StateSharing {
			count++
		}
	}

	return count
}

// StopAllSharesForCall stops all shares for a call
func (sm *ShareManager) StopAllSharesForCall(callID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var lastErr error
	for _, share := range sm.shares {
		if share.CallID == callID && share.State == StateSharing {
			if err := share.Stop(); err != nil {
				lastErr = err
			}
		}
	}

	return lastErr
}
