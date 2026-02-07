//go:build experimental

package breakout

import (
	"fmt"
	"testing"
	"time"
)

func TestNewBreakoutRoom(t *testing.T) {
	room := NewBreakoutRoom("room1", "Room 1", "session1", 5)

	if room.ID != "room1" {
		t.Errorf("Expected ID 'room1', got '%s'", room.ID)
	}

	if room.State != StateSetup {
		t.Errorf("Expected state Setup, got %v", room.State)
	}
}

func TestBreakoutRoomAddParticipant(t *testing.T) {
	room := NewBreakoutRoom("room1", "Room 1", "session1", 5)

	participant := &Participant{
		ID:       "p1",
		UserID:   "user1",
		UserName: "Alice",
	}

	err := room.AddParticipant(participant)
	if err != nil {
		t.Fatalf("Failed to add participant: %v", err)
	}

	if room.GetParticipantCount() != 1 {
		t.Errorf("Expected 1 participant, got %d", room.GetParticipantCount())
	}
}

func TestBreakoutRoomCapacity(t *testing.T) {
	room := NewBreakoutRoom("room1", "Room 1", "session1", 2)

	room.AddParticipant(&Participant{ID: "p1", UserID: "user1", UserName: "Alice"})
	room.AddParticipant(&Participant{ID: "p2", UserID: "user2", UserName: "Bob"})

	err := room.AddParticipant(&Participant{ID: "p3", UserID: "user3", UserName: "Charlie"})
	if err == nil {
		t.Error("Should not allow exceeding capacity")
	}

	if !room.IsFull() {
		t.Error("Room should be full")
	}
}

func TestBreakoutRoomOpenClose(t *testing.T) {
	room := NewBreakoutRoom("room1", "Room 1", "session1", 5)

	err := room.Open()
	if err != nil {
		t.Fatalf("Failed to open room: %v", err)
	}

	if room.State != StateOpen {
		t.Errorf("Expected state Open, got %v", room.State)
	}

	err = room.Close()
	if err != nil {
		t.Fatalf("Failed to close room: %v", err)
	}

	if room.State != StateClosed {
		t.Errorf("Expected state Closed, got %v", room.State)
	}
}

func TestNewBreakoutSession(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")

	if session.ID != "session1" {
		t.Errorf("Expected ID 'session1', got '%s'", session.ID)
	}

	if session.State != StateSetup {
		t.Errorf("Expected state Setup, got %v", session.State)
	}
}

func TestBreakoutSessionCreateRoom(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")

	room, err := session.CreateRoom("Room 1", 5)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	if room.Name != "Room 1" {
		t.Error("Room name should match")
	}

	if session.GetRoomCount() != 1 {
		t.Errorf("Expected 1 room, got %d", session.GetRoomCount())
	}
}

func TestBreakoutSessionCreateRooms(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")

	rooms, err := session.CreateRooms(3, 5)
	if err != nil {
		t.Fatalf("Failed to create rooms: %v", err)
	}

	if len(rooms) != 3 {
		t.Errorf("Expected 3 rooms, got %d", len(rooms))
	}

	if session.GetRoomCount() != 3 {
		t.Errorf("Expected 3 rooms in session, got %d", session.GetRoomCount())
	}
}

func TestBreakoutSessionDeleteRoom(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")
	room, _ := session.CreateRoom("Room 1", 5)

	err := session.DeleteRoom(room.ID)
	if err != nil {
		t.Fatalf("Failed to delete room: %v", err)
	}

	if session.GetRoomCount() != 0 {
		t.Errorf("Expected 0 rooms, got %d", session.GetRoomCount())
	}
}

func TestBreakoutSessionAssignParticipant(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")
	room, _ := session.CreateRoom("Room 1", 5)

	participant := &Participant{
		ID:       "p1",
		UserID:   "user1",
		UserName: "Alice",
	}
	session.MainRoom[participant.ID] = participant

	err := session.AssignParticipant(participant.ID, room.ID)
	if err != nil {
		t.Fatalf("Failed to assign participant: %v", err)
	}

	if room.GetParticipantCount() != 1 {
		t.Errorf("Expected 1 participant in room, got %d", room.GetParticipantCount())
	}

	if len(session.MainRoom) != 0 {
		t.Errorf("Expected 0 participants in main room, got %d", len(session.MainRoom))
	}
}

func TestBreakoutSessionMoveParticipant(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")
	room1, _ := session.CreateRoom("Room 1", 5)
	room2, _ := session.CreateRoom("Room 2", 5)

	participant := &Participant{
		ID:       "p1",
		UserID:   "user1",
		UserName: "Alice",
	}
	room1.AddParticipant(participant)

	err := session.MoveParticipant(participant.ID, room1.ID, room2.ID)
	if err != nil {
		t.Fatalf("Failed to move participant: %v", err)
	}

	if room1.GetParticipantCount() != 0 {
		t.Errorf("Expected 0 participants in room1, got %d", room1.GetParticipantCount())
	}

	if room2.GetParticipantCount() != 1 {
		t.Errorf("Expected 1 participant in room2, got %d", room2.GetParticipantCount())
	}
}

func TestBreakoutSessionReturnToMain(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")
	room, _ := session.CreateRoom("Room 1", 5)

	participant := &Participant{
		ID:       "p1",
		UserID:   "user1",
		UserName: "Alice",
	}
	room.AddParticipant(participant)

	err := session.ReturnParticipantToMain(participant.ID)
	if err != nil {
		t.Fatalf("Failed to return participant: %v", err)
	}

	if room.GetParticipantCount() != 0 {
		t.Error("Room should be empty")
	}

	if len(session.MainRoom) != 1 {
		t.Errorf("Expected 1 participant in main room, got %d", len(session.MainRoom))
	}
}

func TestBreakoutSessionAutoAssignRandom(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")
	session.CreateRooms(3, 5)

	// Add participants to main room
	for i := 1; i <= 9; i++ {
		participant := &Participant{
			ID:       fmt.Sprintf("p%d", i),
			UserID:   fmt.Sprintf("user%d", i),
			UserName: fmt.Sprintf("User%d", i),
		}
		session.MainRoom[participant.ID] = participant
	}

	err := session.AutoAssignParticipants(ModeRandom)
	if err != nil {
		t.Fatalf("Failed to auto-assign: %v", err)
	}

	// All participants should be assigned
	if len(session.MainRoom) != 0 {
		t.Errorf("Expected 0 participants in main room, got %d", len(session.MainRoom))
	}

	// Check distribution
	total := 0
	for _, room := range session.Rooms {
		total += room.GetParticipantCount()
	}

	if total != 9 {
		t.Errorf("Expected 9 total participants in rooms, got %d", total)
	}
}

func TestBreakoutSessionAutoAssignCountBased(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")
	session.CreateRooms(3, 5)

	// Add participants
	for i := 1; i <= 9; i++ {
		participant := &Participant{
			ID:       fmt.Sprintf("p%d", i),
			UserID:   fmt.Sprintf("user%d", i),
			UserName: fmt.Sprintf("User%d", i),
		}
		session.MainRoom[participant.ID] = participant
	}

	err := session.AutoAssignParticipants(ModeCountBased)
	if err != nil {
		t.Fatalf("Failed to auto-assign: %v", err)
	}

	// Check balanced distribution (3 participants per room)
	for _, room := range session.Rooms {
		count := room.GetParticipantCount()
		if count != 3 {
			t.Errorf("Expected balanced distribution of 3 per room, got %d", count)
		}
	}
}

func TestBreakoutSessionOpenAllRooms(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")
	session.CreateRooms(3, 5)

	err := session.OpenAllRooms()
	if err != nil {
		t.Fatalf("Failed to open all rooms: %v", err)
	}

	if session.GetOpenRoomCount() != 3 {
		t.Errorf("Expected 3 open rooms, got %d", session.GetOpenRoomCount())
	}

	if session.State != StateOpen {
		t.Errorf("Expected session state Open, got %v", session.State)
	}
}

func TestBreakoutSessionCloseAllRooms(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")
	session.CreateRooms(2, 5)
	session.OpenAllRooms()

	// Add participants to rooms
	participantNum := 0
	for _, room := range session.Rooms {
		participantNum++
		room.AddParticipant(&Participant{
			ID:       fmt.Sprintf("p%d", participantNum),
			UserID:   fmt.Sprintf("user%d", participantNum),
			UserName: "User",
		})
	}

	err := session.CloseAllRooms()
	if err != nil {
		t.Fatalf("Failed to close all rooms: %v", err)
	}

	// All participants should be back in main room
	if len(session.MainRoom) != 2 {
		t.Errorf("Expected 2 participants in main room, got %d", len(session.MainRoom))
	}

	if session.State != StateClosed {
		t.Errorf("Expected session state Closed, got %v", session.State)
	}
}

func TestBreakoutSessionGetStats(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")
	session.CreateRooms(2, 5)

	stats := session.GetStats()

	if stats["id"] != "session1" {
		t.Error("Stats should include ID")
	}

	if stats["room_count"] != 2 {
		t.Error("Stats should include room count")
	}
}

func TestBreakoutSessionCallbacks(t *testing.T) {
	session := NewBreakoutSession("session1", "call1")

	roomCreatedChan := make(chan bool, 1)
	roomOpenedChan := make(chan bool, 1)

	session.OnRoomCreated = func(room *BreakoutRoom) { roomCreatedChan <- true }
	session.OnRoomOpened = func(room *BreakoutRoom) { roomOpenedChan <- true }

	session.CreateRoom("Room 1", 5)

	select {
	case <-roomCreatedChan:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("OnRoomCreated should be called")
	}

	session.OpenAllRooms()

	select {
	case <-roomOpenedChan:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("OnRoomOpened should be called")
	}
}

func TestBreakoutManager(t *testing.T) {
	manager := NewBreakoutManager()

	if manager.GetSessionCount() != 0 {
		t.Error("Manager should start with 0 sessions")
	}
}

func TestBreakoutManagerCreateSession(t *testing.T) {
	manager := NewBreakoutManager()

	session, err := manager.CreateSession("session1", "call1")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session == nil {
		t.Fatal("Session should not be nil")
	}

	if manager.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session, got %d", manager.GetSessionCount())
	}
}

func TestBreakoutManagerGetSession(t *testing.T) {
	manager := NewBreakoutManager()
	manager.CreateSession("session1", "call1")

	session, err := manager.GetSession("session1")
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if session.ID != "session1" {
		t.Error("Session should match")
	}
}

func TestBreakoutManagerGetSessionByCall(t *testing.T) {
	manager := NewBreakoutManager()
	manager.CreateSession("session1", "call1")

	session, err := manager.GetSessionByCall("call1")
	if err != nil {
		t.Fatalf("Failed to get session by call: %v", err)
	}

	if session.CallID != "call1" {
		t.Error("Session should match call ID")
	}
}

func TestBreakoutManagerDeleteSession(t *testing.T) {
	manager := NewBreakoutManager()
	manager.CreateSession("session1", "call1")

	err := manager.DeleteSession("session1")
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	if manager.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions, got %d", manager.GetSessionCount())
	}
}

func TestBreakoutManagerGetAllSessions(t *testing.T) {
	manager := NewBreakoutManager()
	manager.CreateSession("session1", "call1")
	manager.CreateSession("session2", "call2")

	sessions := manager.GetAllSessions()

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func BenchmarkCreateRoom(b *testing.B) {
	session := NewBreakoutSession("session1", "call1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.CreateRoom(fmt.Sprintf("Room %d", i), 5)
	}
}

func BenchmarkAssignParticipant(b *testing.B) {
	session := NewBreakoutSession("session1", "call1")
	room, _ := session.CreateRoom("Room 1", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		participant := &Participant{
			ID:       fmt.Sprintf("p%d", i),
			UserID:   fmt.Sprintf("user%d", i),
			UserName: "User",
		}
		session.MainRoom[participant.ID] = participant
		session.AssignParticipant(participant.ID, room.ID)
	}
}

func BenchmarkAutoAssign(b *testing.B) {
	session := NewBreakoutSession("session1", "call1")
	session.CreateRooms(10, 10)

	for i := 0; i < 100; i++ {
		participant := &Participant{
			ID:       fmt.Sprintf("p%d", i),
			UserID:   fmt.Sprintf("user%d", i),
			UserName: "User",
		}
		session.MainRoom[participant.ID] = participant
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.AutoAssignParticipants(ModeCountBased)
	}
}
