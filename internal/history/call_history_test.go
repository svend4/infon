package history

import (
	"os"
	"testing"
	"time"
)

func TestNewCallHistory(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ch, err := NewCallHistory()
	if err != nil {
		t.Fatalf("NewCallHistory() failed: %v", err)
	}

	if ch == nil {
		t.Fatal("NewCallHistory() returned nil")
	}

	if len(ch.entries) != 0 {
		t.Errorf("New history should have 0 entries, got %d", len(ch.entries))
	}
}

func TestCallHistory_Add(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ch, _ := NewCallHistory()

	entry := &CallEntry{
		Type:          CallTypeOutgoing,
		Direction:     CallDirectionP2P,
		RemoteAddress: "192.168.1.100:5000",
		RemoteName:    "Alice",
		StartTime:     time.Now().Add(-10 * time.Minute),
		EndTime:       time.Now(),
		Duration:      10 * time.Minute,
		VideoEnabled:  true,
		AudioEnabled:  true,
		Quality:       "excellent",
	}

	err := ch.Add(entry)
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if len(ch.entries) != 1 {
		t.Errorf("After Add(), entries = %d, expected 1", len(ch.entries))
	}

	if entry.ID == "" {
		t.Error("Add() should set ID if empty")
	}
}

func TestCallHistory_GetAll(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ch, _ := NewCallHistory()

	// Add multiple entries
	now := time.Now()
	for i := 0; i < 5; i++ {
		entry := &CallEntry{
			Type:          CallTypeOutgoing,
			RemoteAddress: "test",
			StartTime:     now.Add(-time.Duration(i) * time.Hour),
			Duration:      time.Minute,
		}
		ch.Add(entry)
	}

	all := ch.GetAll()

	if len(all) != 5 {
		t.Errorf("GetAll() = %d entries, expected 5", len(all))
	}

	// Verify sorted by time (newest first)
	for i := 0; i < len(all)-1; i++ {
		if all[i].StartTime.Before(all[i+1].StartTime) {
			t.Error("GetAll() should return entries sorted by time (newest first)")
		}
	}
}

func TestCallHistory_GetRecent(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ch, _ := NewCallHistory()

	// Add 10 entries
	for i := 0; i < 10; i++ {
		entry := &CallEntry{
			Type:          CallTypeOutgoing,
			RemoteAddress: "test",
			StartTime:     time.Now(),
			Duration:      time.Minute,
		}
		ch.Add(entry)
	}

	recent := ch.GetRecent(3)

	if len(recent) != 3 {
		t.Errorf("GetRecent(3) = %d entries, expected 3", len(recent))
	}
}

func TestCallHistory_GetByPeer(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ch, _ := NewCallHistory()

	// Add entries for different peers
	for i := 0; i < 3; i++ {
		ch.Add(&CallEntry{
			RemoteAddress: "peer1",
			StartTime:     time.Now(),
			Duration:      time.Minute,
		})
	}

	for i := 0; i < 2; i++ {
		ch.Add(&CallEntry{
			RemoteAddress: "peer2",
			StartTime:     time.Now(),
			Duration:      time.Minute,
		})
	}

	peer1Calls := ch.GetByPeer("peer1")
	if len(peer1Calls) != 3 {
		t.Errorf("GetByPeer(peer1) = %d, expected 3", len(peer1Calls))
	}

	peer2Calls := ch.GetByPeer("peer2")
	if len(peer2Calls) != 2 {
		t.Errorf("GetByPeer(peer2) = %d, expected 2", len(peer2Calls))
	}
}

func TestCallHistory_GetByType(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ch, _ := NewCallHistory()

	// Add different types
	ch.Add(&CallEntry{Type: CallTypeOutgoing, StartTime: time.Now(), Duration: time.Minute})
	ch.Add(&CallEntry{Type: CallTypeIncoming, StartTime: time.Now(), Duration: time.Minute})
	ch.Add(&CallEntry{Type: CallTypeMissed, StartTime: time.Now(), Duration: 0})
	ch.Add(&CallEntry{Type: CallTypeOutgoing, StartTime: time.Now(), Duration: time.Minute})

	outgoing := ch.GetByType(CallTypeOutgoing)
	if len(outgoing) != 2 {
		t.Errorf("GetByType(outgoing) = %d, expected 2", len(outgoing))
	}

	incoming := ch.GetByType(CallTypeIncoming)
	if len(incoming) != 1 {
		t.Errorf("GetByType(incoming) = %d, expected 1", len(incoming))
	}

	missed := ch.GetByType(CallTypeMissed)
	if len(missed) != 1 {
		t.Errorf("GetByType(missed) = %d, expected 1", len(missed))
	}
}

func TestCallHistory_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ch, _ := NewCallHistory()

	entry := &CallEntry{
		ID:        "test123",
		StartTime: time.Now(),
		Duration:  time.Minute,
	}
	ch.Add(entry)

	if len(ch.entries) != 1 {
		t.Fatal("Failed to add entry")
	}

	err := ch.Delete("test123")
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	if len(ch.entries) != 0 {
		t.Errorf("After Delete(), entries = %d, expected 0", len(ch.entries))
	}
}

func TestCallHistory_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ch, _ := NewCallHistory()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		ch.Add(&CallEntry{
			StartTime: time.Now(),
			Duration:  time.Minute,
		})
	}

	err := ch.Clear()
	if err != nil {
		t.Fatalf("Clear() failed: %v", err)
	}

	if len(ch.entries) != 0 {
		t.Errorf("After Clear(), entries = %d, expected 0", len(ch.entries))
	}
}

func TestCallHistory_GetStatistics(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ch, _ := NewCallHistory()

	// Add various types of calls
	ch.Add(&CallEntry{
		Type:         CallTypeOutgoing,
		Direction:    CallDirectionP2P,
		StartTime:    time.Now(),
		Duration:     10 * time.Minute,
		VideoEnabled: true,
		AudioEnabled: true,
	})

	ch.Add(&CallEntry{
		Type:         CallTypeIncoming,
		Direction:    CallDirectionP2P,
		StartTime:    time.Now(),
		Duration:     5 * time.Minute,
		VideoEnabled: false,
		AudioEnabled: true,
	})

	ch.Add(&CallEntry{
		Type:         CallTypeMissed,
		Direction:    CallDirectionP2P,
		StartTime:    time.Now(),
		Duration:     0,
		VideoEnabled: false,
		AudioEnabled: false,
	})

	ch.Add(&CallEntry{
		Type:         CallTypeOutgoing,
		Direction:    CallDirectionGroup,
		StartTime:    time.Now(),
		Duration:     15 * time.Minute,
		VideoEnabled: true,
		AudioEnabled: true,
	})

	stats := ch.GetStatistics()

	if stats.TotalCalls != 4 {
		t.Errorf("TotalCalls = %d, expected 4", stats.TotalCalls)
	}

	if stats.OutgoingCalls != 2 {
		t.Errorf("OutgoingCalls = %d, expected 2", stats.OutgoingCalls)
	}

	if stats.IncomingCalls != 1 {
		t.Errorf("IncomingCalls = %d, expected 1", stats.IncomingCalls)
	}

	if stats.MissedCalls != 1 {
		t.Errorf("MissedCalls = %d, expected 1", stats.MissedCalls)
	}

	if stats.VideoCallCount != 2 {
		t.Errorf("VideoCallCount = %d, expected 2", stats.VideoCallCount)
	}

	if stats.AudioCallCount != 3 {
		t.Errorf("AudioCallCount = %d, expected 3", stats.AudioCallCount)
	}

	if stats.GroupCallCount != 1 {
		t.Errorf("GroupCallCount = %d, expected 1", stats.GroupCallCount)
	}

	expectedTotal := 30 * time.Minute
	if stats.TotalDuration != expectedTotal {
		t.Errorf("TotalDuration = %v, expected %v", stats.TotalDuration, expectedTotal)
	}

	expectedAvg := 7*time.Minute + 30*time.Second
	if stats.AverageDuration != expectedAvg {
		t.Errorf("AverageDuration = %v, expected %v", stats.AverageDuration, expectedAvg)
	}
}

func TestCallHistory_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ch1, _ := NewCallHistory()

	entry := &CallEntry{
		ID:            "test123",
		Type:          CallTypeOutgoing,
		RemoteAddress: "192.168.1.100:5000",
		RemoteName:    "Alice",
		StartTime:     time.Now(),
		Duration:      10 * time.Minute,
	}
	ch1.Add(entry)

	// Create new instance and load
	ch2, err := NewCallHistory()
	if err != nil {
		t.Fatalf("NewCallHistory() failed: %v", err)
	}

	if len(ch2.entries) != 1 {
		t.Errorf("After load, entries = %d, expected 1", len(ch2.entries))
	}

	if ch2.entries[0].ID != "test123" {
		t.Errorf("Loaded entry ID = %s, expected 'test123'", ch2.entries[0].ID)
	}

	if ch2.entries[0].RemoteName != "Alice" {
		t.Errorf("Loaded entry RemoteName = %s, expected 'Alice'", ch2.entries[0].RemoteName)
	}
}
