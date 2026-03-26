//go:build experimental

package interactive

import (
	"strings"
	"testing"
	"time"
)

// ============ POLL TESTS ============

func TestNewPoll(t *testing.T) {
	poll := NewPoll("poll-1", "What's your favorite color?",
		[]string{"Red", "Blue", "Green"}, "creator")

	if poll.ID != "poll-1" {
		t.Errorf("Expected ID poll-1, got %s", poll.ID)
	}

	if poll.Question != "What's your favorite color?" {
		t.Errorf("Expected question, got %s", poll.Question)
	}

	if len(poll.Options) != 3 {
		t.Errorf("Expected 3 options, got %d", len(poll.Options))
	}

	if !poll.Active {
		t.Error("Poll should be active")
	}
}

func TestPollVote(t *testing.T) {
	poll := NewPoll("poll-2", "Test?", []string{"A", "B", "C"}, "creator")

	err := poll.Vote("voter1", 0)
	if err != nil {
		t.Fatalf("Failed to vote: %v", err)
	}

	if poll.Votes["A"] != 1 {
		t.Errorf("Expected 1 vote for A, got %d", poll.Votes["A"])
	}

	total := poll.GetTotalVotes()
	if total != 1 {
		t.Errorf("Expected 1 total vote, got %d", total)
	}
}

func TestPollChangeVote(t *testing.T) {
	poll := NewPoll("poll-3", "Test?", []string{"A", "B", "C"}, "creator")

	poll.Vote("voter1", 0) // Vote for A
	poll.Vote("voter1", 1) // Change to B

	if poll.Votes["A"] != 0 {
		t.Errorf("Expected 0 votes for A, got %d", poll.Votes["A"])
	}

	if poll.Votes["B"] != 1 {
		t.Errorf("Expected 1 vote for B, got %d", poll.Votes["B"])
	}

	total := poll.GetTotalVotes()
	if total != 1 {
		t.Errorf("Expected 1 total vote, got %d", total)
	}
}

func TestPollInvalidOption(t *testing.T) {
	poll := NewPoll("poll-4", "Test?", []string{"A", "B"}, "creator")

	err := poll.Vote("voter1", 5)
	if err == nil {
		t.Error("Expected error for invalid option")
	}

	err = poll.Vote("voter1", -1)
	if err == nil {
		t.Error("Expected error for negative option")
	}
}

func TestPollClose(t *testing.T) {
	poll := NewPoll("poll-5", "Test?", []string{"A", "B"}, "creator")

	poll.Vote("voter1", 0)

	poll.Close()

	if poll.Active {
		t.Error("Poll should not be active after close")
	}

	err := poll.Vote("voter2", 1)
	if err == nil {
		t.Error("Expected error when voting on closed poll")
	}
}

func TestPollMultipleVotes(t *testing.T) {
	poll := NewPoll("poll-6", "Test?", []string{"A", "B", "C"}, "creator")

	poll.Vote("voter1", 0)
	poll.Vote("voter2", 0)
	poll.Vote("voter3", 1)

	if poll.Votes["A"] != 2 {
		t.Errorf("Expected 2 votes for A, got %d", poll.Votes["A"])
	}

	if poll.Votes["B"] != 1 {
		t.Errorf("Expected 1 vote for B, got %d", poll.Votes["B"])
	}

	total := poll.GetTotalVotes()
	if total != 3 {
		t.Errorf("Expected 3 total votes, got %d", total)
	}
}

func TestPollGetResults(t *testing.T) {
	poll := NewPoll("poll-7", "Test?", []string{"A", "B", "C"}, "creator")

	poll.Vote("voter1", 0)
	poll.Vote("voter2", 0)
	poll.Vote("voter3", 1)

	results := poll.GetResults()

	if results["A"] != 2 {
		t.Errorf("Expected 2 votes for A, got %d", results["A"])
	}

	if results["B"] != 1 {
		t.Errorf("Expected 1 vote for B, got %d", results["B"])
	}
}

func TestPollFormatResults(t *testing.T) {
	poll := NewPoll("poll-8", "Favorite language?",
		[]string{"Go", "Python", "JavaScript"}, "creator")

	poll.Vote("voter1", 0)
	poll.Vote("voter2", 0)
	poll.Vote("voter3", 1)

	formatted := poll.FormatResults()

	if !strings.Contains(formatted, "Favorite language?") {
		t.Error("Formatted results should contain question")
	}

	if !strings.Contains(formatted, "Go") {
		t.Error("Formatted results should contain options")
	}

	if !strings.Contains(formatted, "Total votes: 3") {
		t.Error("Formatted results should show total votes")
	}

	if !strings.Contains(formatted, "█") {
		t.Error("Formatted results should contain progress bars")
	}
}

// ============ REACTION TESTS ============

func TestNewReactionManager(t *testing.T) {
	rm := NewReactionManager(100, 3*time.Second)

	if rm.maxReactions != 100 {
		t.Errorf("Expected max 100 reactions, got %d", rm.maxReactions)
	}

	if rm.ttl != 3*time.Second {
		t.Errorf("Expected TTL 3s, got %v", rm.ttl)
	}
}

func TestAddReaction(t *testing.T) {
	rm := NewReactionManager(100, 3*time.Second)

	reaction := rm.AddReaction("👍", "user1", nil)

	if reaction == nil {
		t.Fatal("Reaction should not be nil")
	}

	if reaction.Type != "👍" {
		t.Errorf("Expected type 👍, got %s", reaction.Type)
	}

	if reaction.Sender != "user1" {
		t.Errorf("Expected sender user1, got %s", reaction.Sender)
	}
}

func TestGetActiveReactions(t *testing.T) {
	rm := NewReactionManager(100, 100*time.Millisecond)

	rm.AddReaction("👍", "user1", nil)
	rm.AddReaction("❤️", "user2", nil)

	active := rm.GetActiveReactions()

	if len(active) != 2 {
		t.Errorf("Expected 2 active reactions, got %d", len(active))
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	active = rm.GetActiveReactions()

	if len(active) != 0 {
		t.Errorf("Expected 0 active reactions after expiration, got %d", len(active))
	}
}

func TestReactionWithPosition(t *testing.T) {
	rm := NewReactionManager(100, 3*time.Second)

	pos := &Position{X: 0.5, Y: 0.5}
	reaction := rm.AddReaction("🎉", "user1", pos)

	if reaction.Position == nil {
		t.Fatal("Position should not be nil")
	}

	if reaction.Position.X != 0.5 || reaction.Position.Y != 0.5 {
		t.Errorf("Expected position (0.5, 0.5), got (%.1f, %.1f)",
			reaction.Position.X, reaction.Position.Y)
	}
}

func TestReactionCounts(t *testing.T) {
	rm := NewReactionManager(100, 3*time.Second)

	rm.AddReaction("👍", "user1", nil)
	rm.AddReaction("👍", "user2", nil)
	rm.AddReaction("❤️", "user3", nil)

	counts := rm.GetReactionCounts()

	if counts["👍"] != 2 {
		t.Errorf("Expected 2 👍 reactions, got %d", counts["👍"])
	}

	if counts["❤️"] != 1 {
		t.Errorf("Expected 1 ❤️ reaction, got %d", counts["❤️"])
	}
}

func TestReactionCleanup(t *testing.T) {
	rm := NewReactionManager(3, 3*time.Second)

	// Add more than max
	rm.AddReaction("👍", "user1", nil)
	rm.AddReaction("❤️", "user2", nil)
	rm.AddReaction("🎉", "user3", nil)
	rm.AddReaction("😂", "user4", nil)

	active := rm.GetActiveReactions()

	// Should keep only last 3
	if len(active) > 3 {
		t.Errorf("Expected max 3 reactions, got %d", len(active))
	}
}

// ============ Q&A TESTS ============

func TestNewQuestion(t *testing.T) {
	q := NewQuestion("q1", "What is Go?", "user1", "Alice")

	if q.ID != "q1" {
		t.Errorf("Expected ID q1, got %s", q.ID)
	}

	if q.Text != "What is Go?" {
		t.Errorf("Expected text, got %s", q.Text)
	}

	if q.AskerName != "Alice" {
		t.Errorf("Expected asker Alice, got %s", q.AskerName)
	}

	if q.Answered {
		t.Error("Question should not be answered initially")
	}
}

func TestQuestionUpvote(t *testing.T) {
	q := NewQuestion("q2", "Test?", "user1", "Alice")

	err := q.Upvote("user2")
	if err != nil {
		t.Fatalf("Failed to upvote: %v", err)
	}

	if q.Upvotes != 1 {
		t.Errorf("Expected 1 upvote, got %d", q.Upvotes)
	}

	// Try upvoting again
	err = q.Upvote("user2")
	if err == nil {
		t.Error("Expected error for duplicate upvote")
	}

	if q.Upvotes != 1 {
		t.Errorf("Expected still 1 upvote, got %d", q.Upvotes)
	}
}

func TestQuestionRemoveUpvote(t *testing.T) {
	q := NewQuestion("q3", "Test?", "user1", "Alice")

	q.Upvote("user2")

	err := q.RemoveUpvote("user2")
	if err != nil {
		t.Fatalf("Failed to remove upvote: %v", err)
	}

	if q.Upvotes != 0 {
		t.Errorf("Expected 0 upvotes, got %d", q.Upvotes)
	}

	// Try removing again
	err = q.RemoveUpvote("user2")
	if err == nil {
		t.Error("Expected error when removing non-existent upvote")
	}
}

func TestQuestionAnswer(t *testing.T) {
	q := NewQuestion("q4", "What is 2+2?", "user1", "Alice")

	q.Answer("It's 4", "moderator")

	if !q.Answered {
		t.Error("Question should be marked as answered")
	}

	if q.AnswerText != "It's 4" {
		t.Errorf("Expected answer 'It's 4', got %s", q.AnswerText)
	}

	if q.AnsweredBy != "moderator" {
		t.Errorf("Expected answered by moderator, got %s", q.AnsweredBy)
	}
}

func TestNewQAManager(t *testing.T) {
	qam := NewQAManager()

	if len(qam.questions) != 0 {
		t.Errorf("Expected 0 questions, got %d", len(qam.questions))
	}
}

func TestSubmitQuestion(t *testing.T) {
	qam := NewQAManager()

	q := qam.SubmitQuestion("q1", "Test question?", "user1", "Alice")

	if q == nil {
		t.Fatal("Question should not be nil")
	}

	if q.Text != "Test question?" {
		t.Errorf("Expected text, got %s", q.Text)
	}
}

func TestGetQuestion(t *testing.T) {
	qam := NewQAManager()

	qam.SubmitQuestion("q1", "Test?", "user1", "Alice")

	q, err := qam.GetQuestion("q1")
	if err != nil {
		t.Fatalf("Failed to get question: %v", err)
	}

	if q.Text != "Test?" {
		t.Errorf("Expected text 'Test?', got %s", q.Text)
	}

	// Try getting non-existent question
	_, err = qam.GetQuestion("q999")
	if err == nil {
		t.Error("Expected error for non-existent question")
	}
}

func TestGetAllQuestions(t *testing.T) {
	qam := NewQAManager()

	qam.SubmitQuestion("q1", "Question 1?", "user1", "Alice")
	qam.SubmitQuestion("q2", "Question 2?", "user2", "Bob")
	qam.SubmitQuestion("q3", "Question 3?", "user3", "Charlie")

	questions := qam.GetAllQuestions()

	if len(questions) != 3 {
		t.Errorf("Expected 3 questions, got %d", len(questions))
	}
}

func TestGetTopQuestions(t *testing.T) {
	qam := NewQAManager()

	q1 := qam.SubmitQuestion("q1", "Q1?", "user1", "Alice")
	q2 := qam.SubmitQuestion("q2", "Q2?", "user2", "Bob")
	q3 := qam.SubmitQuestion("q3", "Q3?", "user3", "Charlie")

	q1.Upvote("u1")
	q2.Upvote("u1")
	q2.Upvote("u2")
	q2.Upvote("u3")
	q3.Upvote("u1")
	q3.Upvote("u2")

	top := qam.GetTopQuestions(2)

	if len(top) != 2 {
		t.Errorf("Expected 2 questions, got %d", len(top))
	}

	// q2 should be first (3 upvotes)
	if top[0].ID != "q2" {
		t.Errorf("Expected q2 first, got %s", top[0].ID)
	}

	// q3 should be second (2 upvotes)
	if top[1].ID != "q3" {
		t.Errorf("Expected q3 second, got %s", top[1].ID)
	}
}

func TestGetUnansweredQuestions(t *testing.T) {
	qam := NewQAManager()

	q1 := qam.SubmitQuestion("q1", "Q1?", "user1", "Alice")
	qam.SubmitQuestion("q2", "Q2?", "user2", "Bob")
	q3 := qam.SubmitQuestion("q3", "Q3?", "user3", "Charlie")

	// Answer q1 and q3
	q1.Answer("Answer 1", "mod")
	q3.Answer("Answer 3", "mod")

	unanswered := qam.GetUnansweredQuestions()

	if len(unanswered) != 1 {
		t.Errorf("Expected 1 unanswered question, got %d", len(unanswered))
	}

	if unanswered[0].ID != "q2" {
		t.Errorf("Expected q2 to be unanswered, got %s", unanswered[0].ID)
	}
}

func TestFormatQuestions(t *testing.T) {
	qam := NewQAManager()

	q1 := qam.SubmitQuestion("q1", "What is Go?", "user1", "Alice")
	_ = qam.SubmitQuestion("q2", "What is Rust?", "user2", "Bob")

	q1.Upvote("u1")
	q1.Answer("A programming language", "moderator")

	formatted := qam.FormatQuestions()

	if !strings.Contains(formatted, "Questions & Answers") {
		t.Error("Formatted should contain title")
	}

	if !strings.Contains(formatted, "What is Go?") {
		t.Error("Formatted should contain question text")
	}

	if !strings.Contains(formatted, "✅") {
		t.Error("Formatted should show answered questions")
	}

	if !strings.Contains(formatted, "❔") {
		t.Error("Formatted should show unanswered questions")
	}
}

// ============ HAND RAISING TESTS ============

func TestNewHandRaising(t *testing.T) {
	hr := NewHandRaising()

	if len(hr.raised) != 0 {
		t.Errorf("Expected 0 raised hands, got %d", len(hr.raised))
	}
}

func TestRaiseHand(t *testing.T) {
	hr := NewHandRaising()

	hr.RaiseHand("user1")

	count := hr.GetCount()
	if count != 1 {
		t.Errorf("Expected 1 raised hand, got %d", count)
	}

	hands := hr.GetRaisedHands()
	if len(hands) != 1 || hands[0] != "user1" {
		t.Error("Expected user1 in raised hands")
	}
}

func TestRaiseHandDuplicate(t *testing.T) {
	hr := NewHandRaising()

	hr.RaiseHand("user1")
	hr.RaiseHand("user1") // Raise again

	count := hr.GetCount()
	if count != 1 {
		t.Errorf("Expected still 1 raised hand, got %d", count)
	}
}

func TestLowerHand(t *testing.T) {
	hr := NewHandRaising()

	hr.RaiseHand("user1")
	hr.LowerHand("user1")

	count := hr.GetCount()
	if count != 0 {
		t.Errorf("Expected 0 raised hands, got %d", count)
	}
}

func TestHandRaisingOrder(t *testing.T) {
	hr := NewHandRaising()

	hr.RaiseHand("user1")
	hr.RaiseHand("user2")
	hr.RaiseHand("user3")

	hands := hr.GetRaisedHands()

	if len(hands) != 3 {
		t.Errorf("Expected 3 hands, got %d", len(hands))
	}

	// Check order
	if hands[0] != "user1" || hands[1] != "user2" || hands[2] != "user3" {
		t.Errorf("Expected order [user1, user2, user3], got %v", hands)
	}
}

func TestClearHands(t *testing.T) {
	hr := NewHandRaising()

	hr.RaiseHand("user1")
	hr.RaiseHand("user2")
	hr.RaiseHand("user3")

	hr.Clear()

	count := hr.GetCount()
	if count != 0 {
		t.Errorf("Expected 0 hands after clear, got %d", count)
	}
}

// Benchmarks
func BenchmarkPollVote(b *testing.B) {
	poll := NewPoll("bench", "Test?", []string{"A", "B", "C"}, "creator")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		poll.Vote("voter", i%3)
	}
}

func BenchmarkAddReaction(b *testing.B) {
	rm := NewReactionManager(10000, 3*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.AddReaction("👍", "user", nil)
	}
}

func BenchmarkQuestionUpvote(b *testing.B) {
	q := NewQuestion("bench", "Test?", "user", "User")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset for each iteration
		q.mu.Lock()
		q.Upvoters = make(map[string]bool)
		q.Upvotes = 0
		q.mu.Unlock()

		q.Upvote("voter")
	}
}
