//go:build experimental

package breakout

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// RoomState represents breakout room state
type RoomState int

const (
	StateSetup RoomState = iota
	StateOpen
	StateClosed
)

func (s RoomState) String() string {
	switch s {
	case StateSetup:
		return "Setup"
	case StateOpen:
		return "Open"
	case StateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// AssignmentMode determines how participants are assigned
type AssignmentMode int

const (
	ModeManual AssignmentMode = iota
	ModeAutomatic
	ModeRandom
	ModeCountBased
)

func (m AssignmentMode) String() string {
	switch m {
	case ModeManual:
		return "Manual"
	case ModeAutomatic:
		return "Automatic"
	case ModeRandom:
		return "Random"
	case ModeCountBased:
		return "Count-Based"
	default:
		return "Unknown"
	}
}

// Participant represents a participant
type Participant struct {
	ID           string
	UserID       string
	UserName     string
	RoomID       string
	JoinedAt     time.Time
	IsHost       bool
	IsMuted      bool
	VideoEnabled bool
}

// BreakoutRoom represents a single breakout room
type BreakoutRoom struct {
	ID           string
	Name         string
	SessionID    string
	Capacity     int
	Participants map[string]*Participant
	State        RoomState
	CreatedAt    time.Time
	OpenedAt     time.Time
	ClosedAt     time.Time
	Duration     time.Duration
	Topic        string
	HostID       string
}

// NewBreakoutRoom creates a new breakout room
func NewBreakoutRoom(id, name, sessionID string, capacity int) *BreakoutRoom {
	return &BreakoutRoom{
		ID:           id,
		Name:         name,
		SessionID:    sessionID,
		Capacity:     capacity,
		Participants: make(map[string]*Participant),
		State:        StateSetup,
		CreatedAt:    time.Now(),
	}
}

// AddParticipant adds a participant to the room
func (br *BreakoutRoom) AddParticipant(participant *Participant) error {
	if br.State == StateClosed {
		return errors.New("room is closed")
	}

	if br.Capacity > 0 && len(br.Participants) >= br.Capacity {
		return errors.New("room is full")
	}

	if _, exists := br.Participants[participant.ID]; exists {
		return errors.New("participant already in room")
	}

	participant.RoomID = br.ID
	participant.JoinedAt = time.Now()
	br.Participants[participant.ID] = participant

	return nil
}

// RemoveParticipant removes a participant from the room
func (br *BreakoutRoom) RemoveParticipant(participantID string) error {
	if _, exists := br.Participants[participantID]; !exists {
		return errors.New("participant not found")
	}

	delete(br.Participants, participantID)
	return nil
}

// GetParticipantCount returns the number of participants
func (br *BreakoutRoom) GetParticipantCount() int {
	return len(br.Participants)
}

// IsFull checks if the room is full
func (br *BreakoutRoom) IsFull() bool {
	return br.Capacity > 0 && len(br.Participants) >= br.Capacity
}

// Open opens the room
func (br *BreakoutRoom) Open() error {
	if br.State != StateSetup {
		return errors.New("room already opened or closed")
	}

	br.State = StateOpen
	br.OpenedAt = time.Now()
	return nil
}

// Close closes the room
func (br *BreakoutRoom) Close() error {
	if br.State != StateOpen {
		return errors.New("room not open")
	}

	br.State = StateClosed
	br.ClosedAt = time.Now()
	return nil
}

// BreakoutSession manages breakout rooms for a call
type BreakoutSession struct {
	mu sync.RWMutex

	ID              string
	CallID          string
	Rooms           map[string]*BreakoutRoom
	RoomOrder       []string
	MainRoom        map[string]*Participant
	AssignmentMode  AssignmentMode
	AutoReturnTime  time.Duration
	AllowSelfSelect bool
	State           RoomState
	CreatedAt       time.Time
	OpenedAt        time.Time

	// Callbacks
	OnRoomCreated      func(room *BreakoutRoom)
	OnRoomOpened       func(room *BreakoutRoom)
	OnRoomClosed       func(room *BreakoutRoom)
	OnParticipantMoved func(participant *Participant, fromRoom, toRoom string)
	OnSessionClosed    func()
}

// NewBreakoutSession creates a new breakout session
func NewBreakoutSession(id, callID string) *BreakoutSession {
	return &BreakoutSession{
		ID:              id,
		CallID:          callID,
		Rooms:           make(map[string]*BreakoutRoom),
		RoomOrder:       make([]string, 0),
		MainRoom:        make(map[string]*Participant),
		AssignmentMode:  ModeManual,
		AutoReturnTime:  0,
		AllowSelfSelect: false,
		State:           StateSetup,
		CreatedAt:       time.Now(),
	}
}

// CreateRoom creates a new breakout room
func (bs *BreakoutSession) CreateRoom(name string, capacity int) (*BreakoutRoom, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	roomID := fmt.Sprintf("%s-room-%d", bs.ID, len(bs.Rooms)+1)
	room := NewBreakoutRoom(roomID, name, bs.ID, capacity)

	bs.Rooms[roomID] = room
	bs.RoomOrder = append(bs.RoomOrder, roomID)

	if bs.OnRoomCreated != nil {
		go bs.OnRoomCreated(room)
	}

	return room, nil
}

// CreateRooms creates multiple rooms at once
func (bs *BreakoutSession) CreateRooms(count int, capacity int) ([]*BreakoutRoom, error) {
	rooms := make([]*BreakoutRoom, 0, count)

	for i := 1; i <= count; i++ {
		room, err := bs.CreateRoom(fmt.Sprintf("Room %d", i), capacity)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// GetRoom gets a room by ID
func (bs *BreakoutSession) GetRoom(roomID string) (*BreakoutRoom, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	room, exists := bs.Rooms[roomID]
	if !exists {
		return nil, errors.New("room not found")
	}

	return room, nil
}

// DeleteRoom deletes a room
func (bs *BreakoutSession) DeleteRoom(roomID string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	room, exists := bs.Rooms[roomID]
	if !exists {
		return errors.New("room not found")
	}

	if room.State == StateOpen {
		return errors.New("cannot delete open room")
	}

	// Move participants back to main room
	for _, participant := range room.Participants {
		bs.MainRoom[participant.ID] = participant
	}

	delete(bs.Rooms, roomID)

	// Remove from order
	newOrder := make([]string, 0)
	for _, id := range bs.RoomOrder {
		if id != roomID {
			newOrder = append(newOrder, id)
		}
	}
	bs.RoomOrder = newOrder

	return nil
}

// AssignParticipant assigns a participant to a room
func (bs *BreakoutSession) AssignParticipant(participantID, roomID string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Get participant from main room
	participant, exists := bs.MainRoom[participantID]
	if !exists {
		return errors.New("participant not found in main room")
	}

	// Get target room
	room, exists := bs.Rooms[roomID]
	if !exists {
		return errors.New("room not found")
	}

	// Add to room
	if err := room.AddParticipant(participant); err != nil {
		return err
	}

	// Remove from main room
	delete(bs.MainRoom, participantID)

	if bs.OnParticipantMoved != nil {
		go bs.OnParticipantMoved(participant, "main", roomID)
	}

	return nil
}

// MoveParticipant moves a participant between rooms
func (bs *BreakoutSession) MoveParticipant(participantID, fromRoomID, toRoomID string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Get source room
	fromRoom, exists := bs.Rooms[fromRoomID]
	if !exists {
		return errors.New("source room not found")
	}

	// Get participant
	participant, exists := fromRoom.Participants[participantID]
	if !exists {
		return errors.New("participant not in source room")
	}

	// Get target room
	toRoom, exists := bs.Rooms[toRoomID]
	if !exists {
		return errors.New("target room not found")
	}

	// Move participant
	if err := fromRoom.RemoveParticipant(participantID); err != nil {
		return err
	}

	if err := toRoom.AddParticipant(participant); err != nil {
		// Rollback
		fromRoom.AddParticipant(participant)
		return err
	}

	if bs.OnParticipantMoved != nil {
		go bs.OnParticipantMoved(participant, fromRoomID, toRoomID)
	}

	return nil
}

// ReturnParticipantToMain returns a participant to main room
func (bs *BreakoutSession) ReturnParticipantToMain(participantID string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Find participant in rooms
	var participant *Participant
	var sourceRoomID string

	for roomID, room := range bs.Rooms {
		if p, exists := room.Participants[participantID]; exists {
			participant = p
			sourceRoomID = roomID
			break
		}
	}

	if participant == nil {
		return errors.New("participant not found in any room")
	}

	// Remove from current room
	bs.Rooms[sourceRoomID].RemoveParticipant(participantID)

	// Add to main room
	bs.MainRoom[participantID] = participant
	participant.RoomID = "main"

	if bs.OnParticipantMoved != nil {
		go bs.OnParticipantMoved(participant, sourceRoomID, "main")
	}

	return nil
}

// AutoAssignParticipants automatically assigns participants to rooms
func (bs *BreakoutSession) AutoAssignParticipants(mode AssignmentMode) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if len(bs.Rooms) == 0 {
		return errors.New("no rooms available")
	}

	participants := make([]*Participant, 0, len(bs.MainRoom))
	for _, p := range bs.MainRoom {
		participants = append(participants, p)
	}

	switch mode {
	case ModeRandom:
		return bs.autoAssignRandom(participants)
	case ModeCountBased:
		return bs.autoAssignCountBased(participants)
	default:
		return errors.New("unsupported assignment mode")
	}
}

// autoAssignRandom assigns participants randomly
func (bs *BreakoutSession) autoAssignRandom(participants []*Participant) error {
	rooms := make([]*BreakoutRoom, 0, len(bs.RoomOrder))
	for _, roomID := range bs.RoomOrder {
		rooms = append(rooms, bs.Rooms[roomID])
	}

	roomIndex := 0
	for _, participant := range participants {
		room := rooms[roomIndex]

		if err := room.AddParticipant(participant); err == nil {
			delete(bs.MainRoom, participant.ID)
		}

		roomIndex = (roomIndex + 1) % len(rooms)
	}

	return nil
}

// autoAssignCountBased assigns to balance room sizes
func (bs *BreakoutSession) autoAssignCountBased(participants []*Participant) error {
	rooms := make([]*BreakoutRoom, 0, len(bs.RoomOrder))
	for _, roomID := range bs.RoomOrder {
		rooms = append(rooms, bs.Rooms[roomID])
	}

	for _, participant := range participants {
		// Find room with fewest participants
		var targetRoom *BreakoutRoom
		minCount := int(^uint(0) >> 1) // Max int

		for _, room := range rooms {
			if !room.IsFull() && room.GetParticipantCount() < minCount {
				targetRoom = room
				minCount = room.GetParticipantCount()
			}
		}

		if targetRoom != nil {
			if err := targetRoom.AddParticipant(participant); err == nil {
				delete(bs.MainRoom, participant.ID)
			}
		}
	}

	return nil
}

// OpenAllRooms opens all rooms
func (bs *BreakoutSession) OpenAllRooms() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	for _, room := range bs.Rooms {
		if room.State == StateSetup {
			room.Open()

			if bs.OnRoomOpened != nil {
				go bs.OnRoomOpened(room)
			}
		}
	}

	bs.State = StateOpen
	bs.OpenedAt = time.Now()

	return nil
}

// CloseAllRooms closes all rooms and returns participants to main
func (bs *BreakoutSession) CloseAllRooms() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	for _, room := range bs.Rooms {
		if room.State == StateOpen {
			// Move participants back to main
			for _, participant := range room.Participants {
				bs.MainRoom[participant.ID] = participant
				participant.RoomID = "main"
			}

			room.Close()

			if bs.OnRoomClosed != nil {
				go bs.OnRoomClosed(room)
			}
		}
	}

	bs.State = StateClosed

	if bs.OnSessionClosed != nil {
		go bs.OnSessionClosed()
	}

	return nil
}

// GetRoomCount returns the number of rooms
func (bs *BreakoutSession) GetRoomCount() int {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	return len(bs.Rooms)
}

// GetOpenRoomCount returns the number of open rooms
func (bs *BreakoutSession) GetOpenRoomCount() int {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	count := 0
	for _, room := range bs.Rooms {
		if room.State == StateOpen {
			count++
		}
	}

	return count
}

// GetTotalParticipantCount returns total participants across all rooms
func (bs *BreakoutSession) GetTotalParticipantCount() int {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	count := len(bs.MainRoom)
	for _, room := range bs.Rooms {
		count += room.GetParticipantCount()
	}

	return count
}

// GetStats returns session statistics
func (bs *BreakoutSession) GetStats() map[string]interface{} {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	return map[string]interface{}{
		"id":                   bs.ID,
		"call_id":              bs.CallID,
		"state":                bs.State.String(),
		"room_count":           len(bs.Rooms),
		"open_room_count":      bs.GetOpenRoomCount(),
		"total_participants":   bs.GetTotalParticipantCount(),
		"main_room_count":      len(bs.MainRoom),
		"assignment_mode":      bs.AssignmentMode.String(),
		"allow_self_select":    bs.AllowSelfSelect,
		"auto_return_time":     bs.AutoReturnTime,
		"created_at":           bs.CreatedAt,
	}
}

// BreakoutManager manages breakout sessions
type BreakoutManager struct {
	mu sync.RWMutex

	sessions map[string]*BreakoutSession
}

// NewBreakoutManager creates a new breakout manager
func NewBreakoutManager() *BreakoutManager {
	return &BreakoutManager{
		sessions: make(map[string]*BreakoutSession),
	}
}

// CreateSession creates a new breakout session
func (bm *BreakoutManager) CreateSession(id, callID string) (*BreakoutSession, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.sessions[id]; exists {
		return nil, errors.New("session already exists")
	}

	session := NewBreakoutSession(id, callID)
	bm.sessions[id] = session

	return session, nil
}

// GetSession gets a session by ID
func (bm *BreakoutManager) GetSession(id string) (*BreakoutSession, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	session, exists := bm.sessions[id]
	if !exists {
		return nil, errors.New("session not found")
	}

	return session, nil
}

// GetSessionByCall gets session by call ID
func (bm *BreakoutManager) GetSessionByCall(callID string) (*BreakoutSession, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	for _, session := range bm.sessions {
		if session.CallID == callID {
			return session, nil
		}
	}

	return nil, errors.New("session not found")
}

// DeleteSession deletes a session
func (bm *BreakoutManager) DeleteSession(id string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.sessions[id]; !exists {
		return errors.New("session not found")
	}

	delete(bm.sessions, id)
	return nil
}

// GetAllSessions returns all sessions
func (bm *BreakoutManager) GetAllSessions() []*BreakoutSession {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	sessions := make([]*BreakoutSession, 0, len(bm.sessions))
	for _, session := range bm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// GetSessionCount returns the number of sessions
func (bm *BreakoutManager) GetSessionCount() int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	return len(bm.sessions)
}
