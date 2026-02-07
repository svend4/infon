//go:build experimental

package security

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// WaitingRoom manages participant admission to calls
type WaitingRoom struct {
	mu sync.RWMutex

	ID           string
	RoomID       string
	Enabled      bool
	AutoAdmit    bool
	Capacity     int
	Timeout      time.Duration

	waiting      map[string]*WaitingParticipant
	admitted     map[string]bool
	rejected     map[string]bool
	waitingQueue []string // Order of join requests

	OnJoinRequest func(participant *WaitingParticipant)
	OnAdmitted    func(participantID string)
	OnRejected    func(participantID string)
	OnTimeout     func(participantID string)
}

// WaitingParticipant represents a participant waiting for admission
type WaitingParticipant struct {
	ID         string
	Name       string
	Email      string
	JoinedAt   time.Time
	Status     WaitingStatus
	Message    string // Optional message from participant
}

// WaitingStatus represents the status of a waiting participant
type WaitingStatus int

const (
	StatusWaiting WaitingStatus = iota
	StatusAdmitted
	StatusRejected
	StatusTimedOut
)

func (s WaitingStatus) String() string {
	switch s {
	case StatusWaiting:
		return "Waiting"
	case StatusAdmitted:
		return "Admitted"
	case StatusRejected:
		return "Rejected"
	case StatusTimedOut:
		return "Timed Out"
	default:
		return "Unknown"
	}
}

// NewWaitingRoom creates a new waiting room
func NewWaitingRoom(id, roomID string) *WaitingRoom {
	return &WaitingRoom{
		ID:           id,
		RoomID:       roomID,
		Enabled:      true,
		AutoAdmit:    false,
		Capacity:     50,
		Timeout:      5 * time.Minute,
		waiting:      make(map[string]*WaitingParticipant),
		admitted:     make(map[string]bool),
		rejected:     make(map[string]bool),
		waitingQueue: make([]string, 0),
	}
}

// Enable enables the waiting room
func (wr *WaitingRoom) Enable() {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	wr.Enabled = true
}

// Disable disables the waiting room (admit everyone automatically)
func (wr *WaitingRoom) Disable() {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	wr.Enabled = false

	// Auto-admit everyone waiting
	for _, participantID := range wr.waitingQueue {
		if participant, exists := wr.waiting[participantID]; exists {
			participant.Status = StatusAdmitted
			wr.admitted[participantID] = true

			if wr.OnAdmitted != nil {
				go wr.OnAdmitted(participantID)
			}
		}
	}

	wr.waiting = make(map[string]*WaitingParticipant)
	wr.waitingQueue = make([]string, 0)
}

// RequestJoin requests to join the room
func (wr *WaitingRoom) RequestJoin(id, name, email, message string) error {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	// Check if already processed
	if wr.admitted[id] {
		return errors.New("already admitted")
	}

	if wr.rejected[id] {
		return errors.New("previously rejected")
	}

	// Check if already waiting
	if _, waiting := wr.waiting[id]; waiting {
		return errors.New("already in waiting room")
	}

	// Check capacity
	if len(wr.waiting) >= wr.Capacity {
		return errors.New("waiting room full")
	}

	participant := &WaitingParticipant{
		ID:       id,
		Name:     name,
		Email:    email,
		JoinedAt: time.Now(),
		Status:   StatusWaiting,
		Message:  message,
	}

	// If waiting room disabled or auto-admit, admit immediately
	if !wr.Enabled || wr.AutoAdmit {
		participant.Status = StatusAdmitted
		wr.admitted[id] = true

		if wr.OnAdmitted != nil {
			go wr.OnAdmitted(id)
		}

		return nil
	}

	wr.waiting[id] = participant
	wr.waitingQueue = append(wr.waitingQueue, id)

	if wr.OnJoinRequest != nil {
		go wr.OnJoinRequest(participant)
	}

	// Start timeout timer
	if wr.Timeout > 0 {
		go wr.timeoutParticipant(id, wr.Timeout)
	}

	return nil
}

// timeoutParticipant automatically rejects after timeout
func (wr *WaitingRoom) timeoutParticipant(participantID string, timeout time.Duration) {
	time.Sleep(timeout)

	wr.mu.Lock()
	defer wr.mu.Unlock()

	participant, exists := wr.waiting[participantID]
	if !exists || participant.Status != StatusWaiting {
		return // Already processed
	}

	participant.Status = StatusTimedOut
	wr.rejected[participantID] = true
	delete(wr.waiting, participantID)

	// Remove from queue
	wr.removeFromQueueLocked(participantID)

	if wr.OnTimeout != nil {
		go wr.OnTimeout(participantID)
	}
}

// Admit admits a participant
func (wr *WaitingRoom) Admit(participantID string) error {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	participant, exists := wr.waiting[participantID]
	if !exists {
		return errors.New("participant not in waiting room")
	}

	if participant.Status != StatusWaiting {
		return fmt.Errorf("participant already %s", participant.Status.String())
	}

	participant.Status = StatusAdmitted
	wr.admitted[participantID] = true
	delete(wr.waiting, participantID)

	// Remove from queue
	wr.removeFromQueueLocked(participantID)

	if wr.OnAdmitted != nil {
		go wr.OnAdmitted(participantID)
	}

	return nil
}

// Reject rejects a participant
func (wr *WaitingRoom) Reject(participantID string, reason string) error {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	participant, exists := wr.waiting[participantID]
	if !exists {
		return errors.New("participant not in waiting room")
	}

	if participant.Status != StatusWaiting {
		return fmt.Errorf("participant already %s", participant.Status.String())
	}

	participant.Status = StatusRejected
	participant.Message = reason
	wr.rejected[participantID] = true
	delete(wr.waiting, participantID)

	// Remove from queue
	wr.removeFromQueueLocked(participantID)

	if wr.OnRejected != nil {
		go wr.OnRejected(participantID)
	}

	return nil
}

// AdmitAll admits all waiting participants
func (wr *WaitingRoom) AdmitAll() int {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	count := 0
	for _, participantID := range wr.waitingQueue {
		if participant, exists := wr.waiting[participantID]; exists && participant.Status == StatusWaiting {
			participant.Status = StatusAdmitted
			wr.admitted[participantID] = true

			if wr.OnAdmitted != nil {
				go wr.OnAdmitted(participantID)
			}

			count++
		}
	}

	wr.waiting = make(map[string]*WaitingParticipant)
	wr.waitingQueue = make([]string, 0)

	return count
}

// RejectAll rejects all waiting participants
func (wr *WaitingRoom) RejectAll(reason string) int {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	count := 0
	for _, participantID := range wr.waitingQueue {
		if participant, exists := wr.waiting[participantID]; exists && participant.Status == StatusWaiting {
			participant.Status = StatusRejected
			participant.Message = reason
			wr.rejected[participantID] = true

			if wr.OnRejected != nil {
				go wr.OnRejected(participantID)
			}

			count++
		}
	}

	wr.waiting = make(map[string]*WaitingParticipant)
	wr.waitingQueue = make([]string, 0)

	return count
}

// GetWaitingParticipants returns all waiting participants in order
func (wr *WaitingRoom) GetWaitingParticipants() []*WaitingParticipant {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	participants := make([]*WaitingParticipant, 0)
	for _, id := range wr.waitingQueue {
		if participant, exists := wr.waiting[id]; exists {
			participants = append(participants, participant)
		}
	}

	return participants
}

// GetParticipant gets a specific participant
func (wr *WaitingRoom) GetParticipant(participantID string) (*WaitingParticipant, error) {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	participant, exists := wr.waiting[participantID]
	if !exists {
		return nil, errors.New("participant not found")
	}

	return participant, nil
}

// GetWaitingCount returns the number of waiting participants
func (wr *WaitingRoom) GetWaitingCount() int {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	return len(wr.waiting)
}

// IsAdmitted checks if a participant is admitted
func (wr *WaitingRoom) IsAdmitted(participantID string) bool {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	return wr.admitted[participantID]
}

// IsRejected checks if a participant is rejected
func (wr *WaitingRoom) IsRejected(participantID string) bool {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	return wr.rejected[participantID]
}

// removeFromQueueLocked removes participant from queue (must hold lock)
func (wr *WaitingRoom) removeFromQueueLocked(participantID string) {
	newQueue := make([]string, 0)
	for _, id := range wr.waitingQueue {
		if id != participantID {
			newQueue = append(newQueue, id)
		}
	}
	wr.waitingQueue = newQueue
}

// GetStats returns waiting room statistics
func (wr *WaitingRoom) GetStats() map[string]interface{} {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["enabled"] = wr.Enabled
	stats["auto_admit"] = wr.AutoAdmit
	stats["capacity"] = wr.Capacity
	stats["waiting_count"] = len(wr.waiting)
	stats["admitted_count"] = len(wr.admitted)
	stats["rejected_count"] = len(wr.rejected)
	stats["timeout"] = wr.Timeout.String()

	return stats
}

// FormatWaitingList formats the waiting list for display
func (wr *WaitingRoom) FormatWaitingList() string {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	if len(wr.waiting) == 0 {
		return "No participants waiting.\n"
	}

	result := fmt.Sprintf("🚪 Waiting Room (%d waiting)\n", len(wr.waiting))
	result += "━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"

	for i, id := range wr.waitingQueue {
		participant := wr.waiting[id]
		waitTime := time.Since(participant.JoinedAt)

		result += fmt.Sprintf("%d. %s\n", i+1, participant.Name)
		if participant.Email != "" {
			result += fmt.Sprintf("   Email: %s\n", participant.Email)
		}
		if participant.Message != "" {
			result += fmt.Sprintf("   Message: %s\n", participant.Message)
		}
		result += fmt.Sprintf("   Waiting: %s\n", formatDuration(waitTime))
		result += "\n"
	}

	return result
}

// formatDuration formats duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	if minutes < 60 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	hours := minutes / 60
	minutes = minutes % 60

	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// PasswordProtection represents password-based room protection
type PasswordProtection struct {
	mu sync.RWMutex

	Enabled  bool
	Password string
	attempts map[string]int // IP/UserID -> failed attempts
	maxAttempts int
	lockoutDuration time.Duration
	lockedOut map[string]time.Time
}

// NewPasswordProtection creates password protection
func NewPasswordProtection(password string) *PasswordProtection {
	return &PasswordProtection{
		Enabled:         true,
		Password:        password,
		attempts:        make(map[string]int),
		maxAttempts:     3,
		lockoutDuration: 5 * time.Minute,
		lockedOut:       make(map[string]time.Time),
	}
}

// Verify verifies the password
func (pp *PasswordProtection) Verify(password, identifier string) (bool, error) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if !pp.Enabled {
		return true, nil
	}

	// Check if locked out
	if lockTime, locked := pp.lockedOut[identifier]; locked {
		if time.Since(lockTime) < pp.lockoutDuration {
			remaining := pp.lockoutDuration - time.Since(lockTime)
			return false, fmt.Errorf("locked out for %s", formatDuration(remaining))
		}
		// Lockout expired
		delete(pp.lockedOut, identifier)
		pp.attempts[identifier] = 0
	}

	if password == pp.Password {
		pp.attempts[identifier] = 0
		return true, nil
	}

	// Wrong password
	pp.attempts[identifier]++

	if pp.attempts[identifier] >= pp.maxAttempts {
		pp.lockedOut[identifier] = time.Now()
		return false, fmt.Errorf("too many failed attempts, locked out for %s", pp.lockoutDuration.String())
	}

	remaining := pp.maxAttempts - pp.attempts[identifier]
	return false, fmt.Errorf("incorrect password, %d attempts remaining", remaining)
}

// SetPassword sets a new password
func (pp *PasswordProtection) SetPassword(newPassword string) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	pp.Password = newPassword
}

// ClearAttempts clears failed attempts for an identifier
func (pp *PasswordProtection) ClearAttempts(identifier string) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	delete(pp.attempts, identifier)
	delete(pp.lockedOut, identifier)
}
