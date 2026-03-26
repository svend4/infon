//go:build experimental

package security

import (
	"strings"
	"testing"
	"time"
)

func TestNewWaitingRoom(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	if wr.ID != "wr1" {
		t.Errorf("Expected ID wr1, got %s", wr.ID)
	}

	if wr.RoomID != "room1" {
		t.Errorf("Expected RoomID room1, got %s", wr.RoomID)
	}

	if !wr.Enabled {
		t.Error("Waiting room should be enabled by default")
	}

	if wr.AutoAdmit {
		t.Error("Auto-admit should be false by default")
	}
}

func TestRequestJoin(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	err := wr.RequestJoin("user1", "Alice", "alice@example.com", "Hello")
	if err != nil {
		t.Fatalf("Failed to request join: %v", err)
	}

	count := wr.GetWaitingCount()
	if count != 1 {
		t.Errorf("Expected 1 waiting participant, got %d", count)
	}

	participant, err := wr.GetParticipant("user1")
	if err != nil {
		t.Fatalf("Failed to get participant: %v", err)
	}

	if participant.Name != "Alice" {
		t.Errorf("Expected name Alice, got %s", participant.Name)
	}
}

func TestRequestJoinDuplicate(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	wr.RequestJoin("user1", "Alice", "", "")

	err := wr.RequestJoin("user1", "Alice", "", "")
	if err == nil {
		t.Error("Expected error for duplicate join request")
	}
}

func TestAdmit(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	wr.RequestJoin("user1", "Alice", "", "")

	admitted := false
	wr.OnAdmitted = func(participantID string) {
		admitted = true
	}

	err := wr.Admit("user1")
	if err != nil {
		t.Fatalf("Failed to admit: %v", err)
	}

	// Wait for goroutine
	time.Sleep(10 * time.Millisecond)

	if !admitted {
		t.Error("OnAdmitted callback should be called")
	}

	if !wr.IsAdmitted("user1") {
		t.Error("User should be admitted")
	}

	count := wr.GetWaitingCount()
	if count != 0 {
		t.Errorf("Expected 0 waiting, got %d", count)
	}
}

func TestReject(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	wr.RequestJoin("user1", "Alice", "", "")

	rejected := false
	wr.OnRejected = func(participantID string) {
		rejected = true
	}

	err := wr.Reject("user1", "Not allowed")
	if err != nil {
		t.Fatalf("Failed to reject: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if !rejected {
		t.Error("OnRejected callback should be called")
	}

	if !wr.IsRejected("user1") {
		t.Error("User should be rejected")
	}
}

func TestAutoAdmit(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")
	wr.AutoAdmit = true

	admitted := false
	wr.OnAdmitted = func(participantID string) {
		admitted = true
	}

	wr.RequestJoin("user1", "Alice", "", "")

	time.Sleep(10 * time.Millisecond)

	if !admitted {
		t.Error("Should be auto-admitted")
	}

	if !wr.IsAdmitted("user1") {
		t.Error("User should be admitted")
	}

	count := wr.GetWaitingCount()
	if count != 0 {
		t.Errorf("Expected 0 waiting with auto-admit, got %d", count)
	}
}

func TestDisabledWaitingRoom(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")
	wr.Disable()

	wr.RequestJoin("user1", "Alice", "", "")

	if !wr.IsAdmitted("user1") {
		t.Error("Should be admitted when waiting room disabled")
	}

	count := wr.GetWaitingCount()
	if count != 0 {
		t.Errorf("Expected 0 waiting when disabled, got %d", count)
	}
}

func TestEnableDisable(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	// Add some waiting participants
	wr.RequestJoin("user1", "Alice", "", "")
	wr.RequestJoin("user2", "Bob", "", "")

	if wr.GetWaitingCount() != 2 {
		t.Error("Expected 2 waiting participants")
	}

	// Disable - should admit everyone
	wr.Disable()

	time.Sleep(10 * time.Millisecond)

	if wr.GetWaitingCount() != 0 {
		t.Error("Expected 0 waiting after disable")
	}

	if !wr.IsAdmitted("user1") || !wr.IsAdmitted("user2") {
		t.Error("All should be admitted after disable")
	}

	// Re-enable
	wr.Enable()

	if !wr.Enabled {
		t.Error("Should be enabled")
	}
}

func TestCapacity(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")
	wr.Capacity = 2

	wr.RequestJoin("user1", "Alice", "", "")
	wr.RequestJoin("user2", "Bob", "", "")

	err := wr.RequestJoin("user3", "Charlie", "", "")
	if err == nil {
		t.Error("Expected error when capacity reached")
	}
}

func TestTimeout(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")
	wr.Timeout = 100 * time.Millisecond

	timedOut := false
	wr.OnTimeout = func(participantID string) {
		timedOut = true
	}

	wr.RequestJoin("user1", "Alice", "", "")

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	if !timedOut {
		t.Error("OnTimeout callback should be called")
	}

	if !wr.IsRejected("user1") {
		t.Error("User should be rejected after timeout")
	}

	count := wr.GetWaitingCount()
	if count != 0 {
		t.Errorf("Expected 0 waiting after timeout, got %d", count)
	}
}

func TestAdmitAll(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	wr.RequestJoin("user1", "Alice", "", "")
	wr.RequestJoin("user2", "Bob", "", "")
	wr.RequestJoin("user3", "Charlie", "", "")

	count := wr.AdmitAll()

	if count != 3 {
		t.Errorf("Expected 3 admitted, got %d", count)
	}

	if wr.GetWaitingCount() != 0 {
		t.Error("Expected 0 waiting after admit all")
	}

	if !wr.IsAdmitted("user1") || !wr.IsAdmitted("user2") || !wr.IsAdmitted("user3") {
		t.Error("All users should be admitted")
	}
}

func TestRejectAll(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	wr.RequestJoin("user1", "Alice", "", "")
	wr.RequestJoin("user2", "Bob", "", "")
	wr.RequestJoin("user3", "Charlie", "", "")

	count := wr.RejectAll("Meeting cancelled")

	if count != 3 {
		t.Errorf("Expected 3 rejected, got %d", count)
	}

	if wr.GetWaitingCount() != 0 {
		t.Error("Expected 0 waiting after reject all")
	}

	if !wr.IsRejected("user1") || !wr.IsRejected("user2") || !wr.IsRejected("user3") {
		t.Error("All users should be rejected")
	}
}

func TestGetWaitingParticipants(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	wr.RequestJoin("user1", "Alice", "", "")
	wr.RequestJoin("user2", "Bob", "", "")
	wr.RequestJoin("user3", "Charlie", "", "")

	participants := wr.GetWaitingParticipants()

	if len(participants) != 3 {
		t.Errorf("Expected 3 participants, got %d", len(participants))
	}

	// Check order (FIFO)
	if participants[0].ID != "user1" {
		t.Errorf("Expected user1 first, got %s", participants[0].ID)
	}

	if participants[1].ID != "user2" {
		t.Errorf("Expected user2 second, got %s", participants[1].ID)
	}
}

func TestGetStats(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	wr.RequestJoin("user1", "Alice", "", "")
	wr.RequestJoin("user2", "Bob", "", "")
	wr.Admit("user1")
	wr.Reject("user2", "Test")

	stats := wr.GetStats()

	if stats["enabled"] != true {
		t.Error("Expected enabled to be true")
	}

	if stats["waiting_count"] != 0 {
		t.Errorf("Expected 0 waiting, got %v", stats["waiting_count"])
	}

	if stats["admitted_count"] != 1 {
		t.Errorf("Expected 1 admitted, got %v", stats["admitted_count"])
	}

	if stats["rejected_count"] != 1 {
		t.Errorf("Expected 1 rejected, got %v", stats["rejected_count"])
	}
}

func TestFormatWaitingList(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	wr.RequestJoin("user1", "Alice", "alice@test.com", "Please let me in")
	wr.RequestJoin("user2", "Bob", "", "")

	formatted := wr.FormatWaitingList()

	if !strings.Contains(formatted, "Waiting Room") {
		t.Error("Formatted should contain title")
	}

	if !strings.Contains(formatted, "Alice") {
		t.Error("Formatted should contain participant name")
	}

	if !strings.Contains(formatted, "alice@test.com") {
		t.Error("Formatted should contain email")
	}

	if !strings.Contains(formatted, "Please let me in") {
		t.Error("Formatted should contain message")
	}
}

func TestFormatWaitingListEmpty(t *testing.T) {
	wr := NewWaitingRoom("wr1", "room1")

	formatted := wr.FormatWaitingList()

	if !strings.Contains(formatted, "No participants") {
		t.Error("Should indicate no participants")
	}
}

// ============ PASSWORD PROTECTION TESTS ============

func TestNewPasswordProtection(t *testing.T) {
	pp := NewPasswordProtection("secret123")

	if pp.Password != "secret123" {
		t.Errorf("Expected password secret123, got %s", pp.Password)
	}

	if !pp.Enabled {
		t.Error("Should be enabled by default")
	}
}

func TestPasswordVerifyCorrect(t *testing.T) {
	pp := NewPasswordProtection("secret123")

	valid, err := pp.Verify("secret123", "user1")
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	if !valid {
		t.Error("Password should be valid")
	}
}

func TestPasswordVerifyIncorrect(t *testing.T) {
	pp := NewPasswordProtection("secret123")

	valid, err := pp.Verify("wrong", "user1")
	if err == nil {
		t.Error("Expected error for wrong password")
	}

	if valid {
		t.Error("Password should not be valid")
	}
}

func TestPasswordLockout(t *testing.T) {
	pp := NewPasswordProtection("secret123")

	// Try wrong password 3 times
	for i := 0; i < 3; i++ {
		pp.Verify("wrong", "user1")
	}

	// Should be locked out now
	valid, err := pp.Verify("secret123", "user1")
	if err == nil {
		t.Error("Expected lockout error")
	}

	if valid {
		t.Error("Should not be valid during lockout")
	}

	if !strings.Contains(err.Error(), "locked out") {
		t.Errorf("Expected lockout error, got: %v", err)
	}
}

func TestPasswordLockoutExpiry(t *testing.T) {
	pp := NewPasswordProtection("secret123")
	pp.lockoutDuration = 50 * time.Millisecond

	// Trigger lockout
	for i := 0; i < 3; i++ {
		pp.Verify("wrong", "user1")
	}

	// Wait for lockout to expire
	time.Sleep(60 * time.Millisecond)

	// Should be able to try again
	valid, err := pp.Verify("secret123", "user1")
	if err != nil {
		t.Fatalf("Should work after lockout expiry: %v", err)
	}

	if !valid {
		t.Error("Password should be valid after lockout expiry")
	}
}

func TestPasswordDisabled(t *testing.T) {
	pp := NewPasswordProtection("secret123")
	pp.Enabled = false

	valid, err := pp.Verify("anythingworks", "user1")
	if err != nil {
		t.Fatalf("Should work when disabled: %v", err)
	}

	if !valid {
		t.Error("Should be valid when password protection disabled")
	}
}

func TestSetPassword(t *testing.T) {
	pp := NewPasswordProtection("secret123")

	pp.SetPassword("newsecret")

	// Old password should not work
	valid, _ := pp.Verify("secret123", "user1")
	if valid {
		t.Error("Old password should not work")
	}

	// New password should work
	valid, err := pp.Verify("newsecret", "user1")
	if err != nil {
		t.Fatalf("New password should work: %v", err)
	}

	if !valid {
		t.Error("New password should be valid")
	}
}

func TestClearAttempts(t *testing.T) {
	pp := NewPasswordProtection("secret123")

	// Wrong password twice
	pp.Verify("wrong", "user1")
	pp.Verify("wrong", "user1")

	// Clear attempts
	pp.ClearAttempts("user1")

	// Should have full attempts again
	pp.Verify("wrong", "user1")
	pp.Verify("wrong", "user1")
	valid, err := pp.Verify("wrong", "user1")

	// Should be locked out after 3 attempts
	if err == nil {
		t.Error("Expected lockout error")
	}

	if valid {
		t.Error("Should be locked out")
	}
}

func TestMultipleIdentifiers(t *testing.T) {
	pp := NewPasswordProtection("secret123")

	// Wrong password for user1
	pp.Verify("wrong", "user1")
	pp.Verify("wrong", "user1")

	// Wrong password for user2
	pp.Verify("wrong", "user2")

	// user1 should be close to lockout (2 attempts)
	// user2 should be fine (1 attempt)

	// user1 locks out
	_, err := pp.Verify("wrong", "user1")
	if !strings.Contains(err.Error(), "locked out") {
		t.Error("user1 should be locked out")
	}

	// user2 should still work
	valid, err := pp.Verify("secret123", "user2")
	if err != nil {
		t.Errorf("user2 should not be locked out: %v", err)
	}

	if !valid {
		t.Error("user2 password should be valid")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		contains string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m"},
		{2*time.Minute + 15*time.Second, "2m"},
		{75 * time.Minute, "1h"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("formatDuration(%v) = %s, should contain %s", tt.duration, result, tt.contains)
		}
	}
}

// Benchmarks
func BenchmarkRequestJoin(b *testing.B) {
	wr := NewWaitingRoom("bench", "room")
	wr.AutoAdmit = true // Avoid filling up

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wr.RequestJoin(string(rune(i)), "User", "", "")
	}
}

func BenchmarkAdmit(b *testing.B) {
	wr := NewWaitingRoom("bench", "room")

	// Pre-populate
	for i := 0; i < b.N; i++ {
		wr.RequestJoin(string(rune(i)), "User", "", "")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wr.Admit(string(rune(i)))
	}
}

func BenchmarkPasswordVerify(b *testing.B) {
	pp := NewPasswordProtection("secret123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pp.Verify("secret123", "user")
		pp.ClearAttempts("user") // Reset for next iteration
	}
}
