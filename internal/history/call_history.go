package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// CallType represents the type of call
type CallType string

const (
	CallTypeOutgoing CallType = "outgoing"
	CallTypeIncoming CallType = "incoming"
	CallTypeMissed   CallType = "missed"
)

// CallDirection represents whether call was peer-to-peer or group
type CallDirection string

const (
	CallDirectionP2P   CallDirection = "p2p"
	CallDirectionGroup CallDirection = "group"
)

// CallEntry represents a single call in history
type CallEntry struct {
	ID             string        `json:"id"`              // Unique call ID
	Type           CallType      `json:"type"`            // outgoing, incoming, missed
	Direction      CallDirection `json:"direction"`       // p2p or group
	RemoteAddress  string        `json:"remote_address"`  // Peer IP:port or group ID
	RemoteName     string        `json:"remote_name"`     // Contact name if available
	StartTime      time.Time     `json:"start_time"`      // When call started
	EndTime        time.Time     `json:"end_time"`        // When call ended
	Duration       time.Duration `json:"duration"`        // Call duration
	VideoEnabled   bool          `json:"video_enabled"`   // Was video enabled
	AudioEnabled   bool          `json:"audio_enabled"`   // Was audio enabled
	ScreenSharing  bool          `json:"screen_sharing"`  // Was screen sharing active
	Recorded       bool          `json:"recorded"`        // Was call recorded
	RecordingPath  string        `json:"recording_path"`  // Path to recording if available
	Participants   []string      `json:"participants"`    // List of participants (for group calls)
	Quality        string        `json:"quality"`         // Call quality: "excellent", "good", "poor"
	AvgBandwidth   float64       `json:"avg_bandwidth"`   // Average bandwidth in KB/s
	PacketLoss     float64       `json:"packet_loss"`     // Packet loss percentage
	AvgLatency     int           `json:"avg_latency"`     // Average latency in ms
	Notes          string        `json:"notes"`           // Optional notes
}

// CallHistory manages call history
type CallHistory struct {
	entries    []*CallEntry
	maxEntries int
	filePath   string
}

// NewCallHistory creates a new call history manager
func NewCallHistory() (*CallHistory, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	historyDir := filepath.Join(homeDir, ".tvcp")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	filePath := filepath.Join(historyDir, "call_history.json")

	ch := &CallHistory{
		entries:    make([]*CallEntry, 0),
		maxEntries: 1000, // Keep last 1000 calls
		filePath:   filePath,
	}

	// Load existing history
	if err := ch.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return ch, nil
}

// Add adds a new call entry to history
func (ch *CallHistory) Add(entry *CallEntry) error {
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("call_%d", time.Now().UnixNano())
	}

	ch.entries = append(ch.entries, entry)

	// Trim if exceeds max
	if len(ch.entries) > ch.maxEntries {
		ch.entries = ch.entries[len(ch.entries)-ch.maxEntries:]
	}

	return ch.Save()
}

// GetAll returns all call entries, sorted by start time (newest first)
func (ch *CallHistory) GetAll() []*CallEntry {
	// Create a copy and sort
	entries := make([]*CallEntry, len(ch.entries))
	copy(entries, ch.entries)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].StartTime.After(entries[j].StartTime)
	})

	return entries
}

// GetRecent returns the N most recent calls
func (ch *CallHistory) GetRecent(n int) []*CallEntry {
	all := ch.GetAll()
	if len(all) < n {
		return all
	}
	return all[:n]
}

// GetByPeer returns all calls with a specific peer
func (ch *CallHistory) GetByPeer(remoteAddress string) []*CallEntry {
	var result []*CallEntry
	for _, entry := range ch.entries {
		if entry.RemoteAddress == remoteAddress {
			result = append(result, entry)
		}
	}

	// Sort by time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime.After(result[j].StartTime)
	})

	return result
}

// GetByType returns all calls of a specific type
func (ch *CallHistory) GetByType(callType CallType) []*CallEntry {
	var result []*CallEntry
	for _, entry := range ch.entries {
		if entry.Type == callType {
			result = append(result, entry)
		}
	}

	// Sort by time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime.After(result[j].StartTime)
	})

	return result
}

// GetByDateRange returns calls within a date range
func (ch *CallHistory) GetByDateRange(start, end time.Time) []*CallEntry {
	var result []*CallEntry
	for _, entry := range ch.entries {
		if entry.StartTime.After(start) && entry.StartTime.Before(end) {
			result = append(result, entry)
		}
	}

	// Sort by time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime.After(result[j].StartTime)
	})

	return result
}

// Delete removes a call entry by ID
func (ch *CallHistory) Delete(id string) error {
	for i, entry := range ch.entries {
		if entry.ID == id {
			ch.entries = append(ch.entries[:i], ch.entries[i+1:]...)
			return ch.Save()
		}
	}
	return fmt.Errorf("call entry not found: %s", id)
}

// Clear removes all call history
func (ch *CallHistory) Clear() error {
	ch.entries = make([]*CallEntry, 0)
	return ch.Save()
}

// GetStatistics returns call statistics
func (ch *CallHistory) GetStatistics() Statistics {
	stats := Statistics{
		TotalCalls:     len(ch.entries),
		OutgoingCalls:  0,
		IncomingCalls:  0,
		MissedCalls:    0,
		TotalDuration:  0,
		AverageDuration: 0,
		VideoCallCount: 0,
		AudioCallCount: 0,
		GroupCallCount: 0,
	}

	for _, entry := range ch.entries {
		switch entry.Type {
		case CallTypeOutgoing:
			stats.OutgoingCalls++
		case CallTypeIncoming:
			stats.IncomingCalls++
		case CallTypeMissed:
			stats.MissedCalls++
		}

		stats.TotalDuration += entry.Duration

		if entry.VideoEnabled {
			stats.VideoCallCount++
		}
		if entry.AudioEnabled {
			stats.AudioCallCount++
		}
		if entry.Direction == CallDirectionGroup {
			stats.GroupCallCount++
		}
	}

	if stats.TotalCalls > 0 {
		stats.AverageDuration = stats.TotalDuration / time.Duration(stats.TotalCalls)
	}

	return stats
}

// Statistics represents call history statistics
type Statistics struct {
	TotalCalls      int
	OutgoingCalls   int
	IncomingCalls   int
	MissedCalls     int
	TotalDuration   time.Duration
	AverageDuration time.Duration
	VideoCallCount  int
	AudioCallCount  int
	GroupCallCount  int
}

// Save saves call history to disk
func (ch *CallHistory) Save() error {
	data, err := json.MarshalIndent(ch.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(ch.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history: %w", err)
	}

	return nil
}

// Load loads call history from disk
func (ch *CallHistory) Load() error {
	data, err := os.ReadFile(ch.filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &ch.entries); err != nil {
		return fmt.Errorf("failed to unmarshal history: %w", err)
	}

	return nil
}

// Export exports call history to CSV format
func (ch *CallHistory) Export(path string) error {
	entries := ch.GetAll()

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer f.Close()

	// Write CSV header
	header := "ID,Type,Direction,Remote Address,Remote Name,Start Time,End Time,Duration,Video,Audio,Screen Sharing,Recorded,Participants,Quality,Bandwidth,Packet Loss,Latency\n"
	if _, err := f.WriteString(header); err != nil {
		return err
	}

	// Write entries
	for _, entry := range entries {
		participants := ""
		if len(entry.Participants) > 0 {
			for i, p := range entry.Participants {
				if i > 0 {
					participants += ";"
				}
				participants += p
			}
		}

		line := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%t,%t,%t,%t,%s,%s,%.2f,%.2f,%d\n",
			entry.ID,
			entry.Type,
			entry.Direction,
			entry.RemoteAddress,
			entry.RemoteName,
			entry.StartTime.Format(time.RFC3339),
			entry.EndTime.Format(time.RFC3339),
			entry.Duration,
			entry.VideoEnabled,
			entry.AudioEnabled,
			entry.ScreenSharing,
			entry.Recorded,
			participants,
			entry.Quality,
			entry.AvgBandwidth,
			entry.PacketLoss,
			entry.AvgLatency,
		)

		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}
