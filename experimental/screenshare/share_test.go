//go:build experimental

package screenshare

import (
	"testing"
	"time"
)

func TestNewScreenShare(t *testing.T) {
	config := DefaultConfig()
	share := NewScreenShare("share1", "user1", "Alice", "call1", config)

	if share.ID != "share1" {
		t.Errorf("Expected ID 'share1', got '%s'", share.ID)
	}

	if share.State != StateIdle {
		t.Errorf("Expected state Idle, got %v", share.State)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Type != TypeFullScreen {
		t.Error("Default type should be FullScreen")
	}

	if config.Quality != QualityMedium {
		t.Error("Default quality should be Medium")
	}
}

func TestStartShare(t *testing.T) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)

	err := share.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	if share.State != StateSharing {
		t.Errorf("Expected state Sharing, got %v", share.State)
	}
}

func TestStopShare(t *testing.T) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)
	share.Start()

	err := share.Stop()
	if err != nil {
		t.Fatalf("Failed to stop: %v", err)
	}

	if share.State != StateStopped {
		t.Errorf("Expected state Stopped, got %v", share.State)
	}
}

func TestPauseResume(t *testing.T) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)
	share.Start()

	err := share.Pause()
	if err != nil {
		t.Fatalf("Failed to pause: %v", err)
	}

	if share.State != StatePaused {
		t.Errorf("Expected state Paused, got %v", share.State)
	}

	time.Sleep(100 * time.Millisecond)

	err = share.Resume()
	if err != nil {
		t.Fatalf("Failed to resume: %v", err)
	}

	if share.State != StateSharing {
		t.Errorf("Expected state Sharing, got %v", share.State)
	}
}

func TestSendFrame(t *testing.T) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)
	share.Start()

	frame := &Frame{
		Data:   make([]byte, 1920*1080*3),
		Width:  1920,
		Height: 1080,
	}

	err := share.SendFrame(frame)
	if err != nil {
		t.Fatalf("Failed to send frame: %v", err)
	}

	if share.FrameCount != 1 {
		t.Errorf("Expected frame count 1, got %d", share.FrameCount)
	}
}

func TestAddViewer(t *testing.T) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)

	viewer, err := share.AddViewer("user2", "Bob")
	if err != nil {
		t.Fatalf("Failed to add viewer: %v", err)
	}

	if viewer.UserID != "user2" {
		t.Error("Viewer should match")
	}

	if share.GetViewerCount() != 1 {
		t.Errorf("Expected 1 viewer, got %d", share.GetViewerCount())
	}
}

func TestRemoveViewer(t *testing.T) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)
	share.AddViewer("user2", "Bob")

	err := share.RemoveViewer("user2")
	if err != nil {
		t.Fatalf("Failed to remove viewer: %v", err)
	}

	if share.GetViewerCount() != 0 {
		t.Errorf("Expected 0 viewers, got %d", share.GetViewerCount())
	}
}

func TestMaxViewers(t *testing.T) {
	config := DefaultConfig()
	config.MaxViewers = 2
	share := NewScreenShare("share1", "user1", "Alice", "call1", config)

	share.AddViewer("user2", "Bob")
	share.AddViewer("user3", "Charlie")

	_, err := share.AddViewer("user4", "David")
	if err == nil {
		t.Error("Should not allow exceeding max viewers")
	}
}

func TestGetDuration(t *testing.T) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)

	if share.GetDuration() != 0 {
		t.Error("Duration should be 0 before start")
	}

	share.Start()
	time.Sleep(100 * time.Millisecond)

	duration := share.GetDuration()
	if duration < 50*time.Millisecond {
		t.Errorf("Duration should be at least 50ms, got %v", duration)
	}
}

func TestGetStats(t *testing.T) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)
	share.Start()
	share.AddViewer("user2", "Bob")

	frame := &Frame{
		Data:   make([]byte, 1000),
		Width:  1920,
		Height: 1080,
	}
	share.SendFrame(frame)

	stats := share.GetStats()

	if stats["id"] != "share1" {
		t.Error("Stats should include ID")
	}

	if stats["frame_count"] != int64(1) {
		t.Error("Stats should include frame count")
	}
}

func TestCallbacks(t *testing.T) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)

	startCalled := false
	stopCalled := false
	share.OnStart = func() { startCalled = true }
	share.OnStop = func() { stopCalled = true }

	share.Start()
	time.Sleep(50 * time.Millisecond)

	if !startCalled {
		t.Error("OnStart should be called")
	}

	share.Stop()
	time.Sleep(50 * time.Millisecond)

	if !stopCalled {
		t.Error("OnStop should be called")
	}
}

func TestInvalidStateTransitions(t *testing.T) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)

	err := share.Stop()
	if err == nil {
		t.Error("Should not allow stop before start")
	}

	err = share.Pause()
	if err == nil {
		t.Error("Should not allow pause before start")
	}

	share.Start()

	err = share.Start()
	if err == nil {
		t.Error("Should not allow start twice")
	}
}

func TestQualitySettings(t *testing.T) {
	qualities := []Quality{QualityLow, QualityMedium, QualityHigh, QualityAuto}

	for _, quality := range qualities {
		if quality.FPS() <= 0 {
			t.Errorf("Quality %v should have positive FPS", quality)
		}

		if quality.Bitrate() <= 0 {
			t.Errorf("Quality %v should have positive bitrate", quality)
		}
	}
}

func TestShareTypes(t *testing.T) {
	types := []ShareType{TypeFullScreen, TypeWindow, TypeApplication, TypeRegion}

	for _, typ := range types {
		str := typ.String()
		if str == "Unknown" {
			t.Errorf("Type %d should have a name", typ)
		}
	}
}

func TestShareManager(t *testing.T) {
	manager := NewShareManager()

	if manager.GetShareCount() != 0 {
		t.Error("Manager should start with 0 shares")
	}
}

func TestShareManagerCreate(t *testing.T) {
	manager := NewShareManager()

	share, err := manager.CreateShare("share1", "user1", "Alice", "call1", nil)
	if err != nil {
		t.Fatalf("Failed to create share: %v", err)
	}

	if share == nil {
		t.Fatal("Share should not be nil")
	}

	if manager.GetShareCount() != 1 {
		t.Errorf("Expected 1 share, got %d", manager.GetShareCount())
	}
}

func TestShareManagerGet(t *testing.T) {
	manager := NewShareManager()
	manager.CreateShare("share1", "user1", "Alice", "call1", nil)

	share, err := manager.GetShare("share1")
	if err != nil {
		t.Fatalf("Failed to get share: %v", err)
	}

	if share.ID != "share1" {
		t.Error("Share should match")
	}
}

func TestShareManagerDelete(t *testing.T) {
	manager := NewShareManager()
	manager.CreateShare("share1", "user1", "Alice", "call1", nil)

	err := manager.DeleteShare("share1")
	if err != nil {
		t.Fatalf("Failed to delete share: %v", err)
	}

	if manager.GetShareCount() != 0 {
		t.Errorf("Expected 0 shares, got %d", manager.GetShareCount())
	}
}

func TestShareManagerGetByUser(t *testing.T) {
	manager := NewShareManager()
	manager.CreateShare("share1", "user1", "Alice", "call1", nil)
	manager.CreateShare("share2", "user1", "Alice", "call2", nil)
	manager.CreateShare("share3", "user2", "Bob", "call1", nil)

	shares := manager.GetShareByUser("user1")

	if len(shares) != 2 {
		t.Errorf("Expected 2 shares for user1, got %d", len(shares))
	}
}

func TestShareManagerGetByCall(t *testing.T) {
	manager := NewShareManager()
	manager.CreateShare("share1", "user1", "Alice", "call1", nil)
	manager.CreateShare("share2", "user2", "Bob", "call1", nil)
	manager.CreateShare("share3", "user3", "Charlie", "call2", nil)

	shares := manager.GetSharesByCall("call1")

	if len(shares) != 2 {
		t.Errorf("Expected 2 shares for call1, got %d", len(shares))
	}
}

func TestShareManagerGetActiveShares(t *testing.T) {
	manager := NewShareManager()
	share1, _ := manager.CreateShare("share1", "user1", "Alice", "call1", nil)
	share2, _ := manager.CreateShare("share2", "user2", "Bob", "call1", nil)

	share1.Start()

	activeShares := manager.GetActiveShares()

	if len(activeShares) != 1 {
		t.Errorf("Expected 1 active share, got %d", len(activeShares))
	}

	share2.Start()

	activeShares = manager.GetActiveShares()

	if len(activeShares) != 2 {
		t.Errorf("Expected 2 active shares, got %d", len(activeShares))
	}
}

func TestShareManagerStopAllForCall(t *testing.T) {
	manager := NewShareManager()
	share1, _ := manager.CreateShare("share1", "user1", "Alice", "call1", nil)
	share2, _ := manager.CreateShare("share2", "user2", "Bob", "call1", nil)

	share1.Start()
	share2.Start()

	err := manager.StopAllSharesForCall("call1")
	if err != nil {
		t.Fatalf("Failed to stop shares: %v", err)
	}

	if share1.State != StateStopped {
		t.Error("Share1 should be stopped")
	}

	if share2.State != StateStopped {
		t.Error("Share2 should be stopped")
	}
}

func BenchmarkSendFrame(b *testing.B) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)
	share.Start()

	frame := &Frame{
		Data:   make([]byte, 1920*1080*3),
		Width:  1920,
		Height: 1080,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		share.SendFrame(frame)
	}
}

func BenchmarkAddViewer(b *testing.B) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		share.AddViewer("user"+string(rune(i)), "User")
	}
}

func BenchmarkGetStats(b *testing.B) {
	share := NewScreenShare("share1", "user1", "Alice", "call1", nil)
	share.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		share.GetStats()
	}
}
