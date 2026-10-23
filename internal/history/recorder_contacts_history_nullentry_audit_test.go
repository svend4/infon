package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A call_history.json with a null element must load into a usable history:
// the null is not a call, and reading or changing the history afterwards must
// not dereference it.
func TestCallHistoryLoadNullEntryAudit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	dir := filepath.Join(home, ".tvcp")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	data := `[null, {"id": "c1", "type": "outgoing", "direction": "p2p", "remote_address": "10.0.0.1:5000",
		"start_time": "2026-01-02T03:04:05Z", "duration": 1000000000, "video_enabled": true}, null]`
	if err := os.WriteFile(filepath.Join(dir, "call_history.json"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("history with a null entry panicked: %v", r)
		}
	}()

	ch, err := NewCallHistory()
	if err != nil {
		t.Fatalf("NewCallHistory: %v", err)
	}

	stats := ch.GetStatistics()
	if stats.TotalCalls != 1 || stats.OutgoingCalls != 1 || stats.VideoCallCount != 1 || stats.AverageDuration != time.Second {
		t.Errorf("statistics = %+v, want the one outgoing video call", stats)
	}
	all := ch.GetAll()
	if len(all) != 1 || all[0] == nil || all[0].ID != "c1" {
		t.Fatalf("GetAll = %v, want just c1", all)
	}
	if got := ch.GetByPeer("10.0.0.1:5000"); len(got) != 1 {
		t.Errorf("GetByPeer = %d entries, want 1", len(got))
	}
	_ = ch.GetByType(CallTypeMissed)
	_ = ch.GetByDateRange(time.Time{}, time.Now())
	_ = ch.GetRecent(5)
	if err := ch.Export(filepath.Join(home, "history.csv")); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if err := ch.Delete("missing"); err == nil {
		t.Errorf("Delete(missing) succeeded")
	}
	if err := ch.Add(&CallEntry{Type: CallTypeIncoming, RemoteAddress: "10.0.0.2:5000"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if got := ch.GetStatistics().TotalCalls; got != 2 {
		t.Errorf("TotalCalls after Add = %d, want 2", got)
	}
}
