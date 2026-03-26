//go:build experimental

package trivia

import (
	"encoding/json"
	"testing"
	"time"
)

func createTestQuestions() []*Question {
	return []*Question{
		{
			ID:         "q1",
			Text:       "What is the capital of France?",
			Options:    []string{"London", "Berlin", "Paris", "Madrid"},
			Correct:    2,
			Category:   "Geography",
			Difficulty: 1,
			Points:     100,
		},
		{
			ID:         "q2",
			Text:       "What is 2 + 2?",
			Options:    []string{"3", "4", "5", "6"},
			Correct:    1,
			Category:   "Math",
			Difficulty: 1,
			Points:     100,
		},
		{
			ID:         "q3",
			Text:       "Who wrote Romeo and Juliet?",
			Options:    []string{"Charles Dickens", "William Shakespeare", "Mark Twain", "Jane Austen"},
			Correct:    1,
			Category:   "Literature",
			Difficulty: 2,
			Points:     200,
		},
	}
}

func TestNewTriviaGame(t *testing.T) {
	game := NewTriviaGame("test-game", 30*time.Second)

	if game.ID != "test-game" {
		t.Errorf("Expected game ID 'test-game', got '%s'", game.ID)
	}

	if game.TimeLimit != 30*time.Second {
		t.Errorf("Expected time limit 30s, got %v", game.TimeLimit)
	}

	if game.State != StateWaiting {
		t.Errorf("Expected state Waiting, got %v", game.State)
	}

	if len(game.Players) != 0 {
		t.Errorf("Expected 0 players, got %d", len(game.Players))
	}
}

func TestAddQuestion(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)

	q := &Question{
		ID:         "q1",
		Text:       "Test question?",
		Options:    []string{"A", "B", "C"},
		Correct:    1,
		Category:   "Test",
		Difficulty: 3,
	}

	game.AddQuestion(q)

	if len(game.Questions) != 1 {
		t.Errorf("Expected 1 question, got %d", len(game.Questions))
	}

	// Check auto-assigned points
	if q.Points != 300 {
		t.Errorf("Expected points 300 (difficulty 3 * 100), got %d", q.Points)
	}
}

func TestAddQuestions(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	questions := createTestQuestions()

	game.AddQuestions(questions)

	if len(game.Questions) != 3 {
		t.Errorf("Expected 3 questions, got %d", len(game.Questions))
	}
}

func TestLoadQuestionsFromJSON(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)

	jsonData := `[
		{
			"id": "q1",
			"text": "Test?",
			"options": ["A", "B"],
			"correct": 0,
			"category": "Test",
			"difficulty": 1,
			"points": 100
		}
	]`

	err := game.LoadQuestionsFromJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to load questions: %v", err)
	}

	if len(game.Questions) != 1 {
		t.Errorf("Expected 1 question, got %d", len(game.Questions))
	}
}

func TestShuffleQuestions(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())

	originalFirst := game.Questions[0].ID

	game.ShuffleQuestions()

	// There's a chance it stays the same, but very unlikely with 3 questions
	// Just verify the count is the same
	if len(game.Questions) != 3 {
		t.Errorf("Expected 3 questions after shuffle, got %d", len(game.Questions))
	}

	// Verify all questions still exist
	ids := make(map[string]bool)
	for _, q := range game.Questions {
		ids[q.ID] = true
	}

	if !ids["q1"] || !ids["q2"] || !ids["q3"] {
		t.Error("Some questions missing after shuffle")
	}

	_ = originalFirst // Avoid unused warning
}

func TestAddPlayer(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)

	err := game.AddPlayer("player1", "Alice")
	if err != nil {
		t.Fatalf("Failed to add player: %v", err)
	}

	if len(game.Players) != 1 {
		t.Errorf("Expected 1 player, got %d", len(game.Players))
	}

	player := game.Players["player1"]
	if player.Name != "Alice" {
		t.Errorf("Expected player name 'Alice', got '%s'", player.Name)
	}

	// Try adding same player again
	err = game.AddPlayer("player1", "Alice")
	if err == nil {
		t.Error("Expected error when adding duplicate player")
	}
}

func TestRemovePlayer(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)

	game.AddPlayer("player1", "Alice")
	game.AddPlayer("player2", "Bob")

	if len(game.Players) != 2 {
		t.Fatalf("Expected 2 players")
	}

	game.RemovePlayer("player1")

	if len(game.Players) != 1 {
		t.Errorf("Expected 1 player after removal, got %d", len(game.Players))
	}

	if _, exists := game.Players["player1"]; exists {
		t.Error("Player1 should be removed")
	}
}

func TestStartGame(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")

	err := game.Start()
	if err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	if game.State != StatePlaying {
		t.Errorf("Expected state Playing, got %v", game.State)
	}

	// Try starting again
	err = game.Start()
	if err == nil {
		t.Error("Expected error when starting already started game")
	}
}

func TestStartGameNoPlayers(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())

	err := game.Start()
	if err == nil {
		t.Error("Expected error when starting game with no players")
	}
}

func TestStartGameNoQuestions(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddPlayer("player1", "Alice")

	err := game.Start()
	if err == nil {
		t.Error("Expected error when starting game with no questions")
	}
}

func TestNextQuestion(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.Start()

	q, err := game.NextQuestion()
	if err != nil {
		t.Fatalf("Failed to get next question: %v", err)
	}

	if q.ID != "q1" {
		t.Errorf("Expected first question, got %s", q.ID)
	}

	if game.State != StateQuestion {
		t.Errorf("Expected state Question, got %v", game.State)
	}

	if game.CurrentQ != 0 {
		t.Errorf("Expected CurrentQ = 0, got %d", game.CurrentQ)
	}
}

func TestSubmitAnswer(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.Start()
	game.NextQuestion()

	// Submit correct answer (Paris = index 2)
	err := game.SubmitAnswer("player1", 2)
	if err != nil {
		t.Fatalf("Failed to submit answer: %v", err)
	}

	player := game.Players["player1"]
	if len(player.Answers) != 1 {
		t.Errorf("Expected 1 answer, got %d", len(player.Answers))
	}

	answer := player.Answers[0]
	if !answer.Correct {
		t.Error("Expected answer to be correct")
	}

	if answer.Points <= 0 {
		t.Errorf("Expected positive points, got %d", answer.Points)
	}

	if player.Score <= 0 {
		t.Errorf("Expected positive score, got %d", player.Score)
	}
}

func TestSubmitWrongAnswer(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.Start()
	game.NextQuestion()

	// Submit wrong answer
	err := game.SubmitAnswer("player1", 0)
	if err != nil {
		t.Fatalf("Failed to submit answer: %v", err)
	}

	player := game.Players["player1"]
	answer := player.Answers[0]

	if answer.Correct {
		t.Error("Expected answer to be incorrect")
	}

	if answer.Points != 0 {
		t.Errorf("Expected 0 points for wrong answer, got %d", answer.Points)
	}

	if player.Score != 0 {
		t.Errorf("Expected 0 score, got %d", player.Score)
	}
}

func TestSubmitDuplicateAnswer(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.Start()
	game.NextQuestion()

	game.SubmitAnswer("player1", 2)

	// Try to answer again
	err := game.SubmitAnswer("player1", 1)
	if err == nil {
		t.Error("Expected error when submitting duplicate answer")
	}
}

func TestSubmitInvalidOption(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.Start()
	game.NextQuestion()

	err := game.SubmitAnswer("player1", 999)
	if err == nil {
		t.Error("Expected error for invalid option")
	}
}

func TestRevealAnswer(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.Start()
	game.NextQuestion()

	err := game.RevealAnswer()
	if err != nil {
		t.Fatalf("Failed to reveal answer: %v", err)
	}

	if game.State != StateRevealing {
		t.Errorf("Expected state Revealing, got %v", game.State)
	}
}

func TestContinueToNext(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.Start()
	game.NextQuestion()
	game.RevealAnswer()

	err := game.ContinueToNext()
	if err != nil {
		t.Fatalf("Failed to continue: %v", err)
	}

	// Should be on question 2 now
	if game.CurrentQ != 1 {
		t.Errorf("Expected CurrentQ = 1, got %d", game.CurrentQ)
	}
}

func TestGameCompletion(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.Start()

	// Go through first question
	_, err := game.NextQuestion()
	if err != nil {
		t.Fatalf("Failed at question 0: %v", err)
	}
	game.RevealAnswer()

	// ContinueToNext advances to question 1
	err = game.ContinueToNext()
	if err != nil {
		t.Fatalf("Failed to continue to question 1: %v", err)
	}
	game.RevealAnswer()

	// ContinueToNext advances to question 2
	err = game.ContinueToNext()
	if err != nil {
		t.Fatalf("Failed to continue to question 2: %v", err)
	}
	game.RevealAnswer()

	// ContinueToNext should fail and set state to Finished
	err = game.ContinueToNext()
	if err == nil {
		t.Error("Expected error when no more questions")
	}

	if game.State != StateFinished {
		t.Errorf("Expected state Finished, got %v", game.State)
	}
}

func TestGetScoreboard(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.AddPlayer("player2", "Bob")
	game.AddPlayer("player3", "Charlie")
	game.Start()

	// Give different scores
	game.Players["player1"].Score = 300
	game.Players["player2"].Score = 500
	game.Players["player3"].Score = 200

	scoreboard := game.GetScoreboard()

	if len(scoreboard) != 3 {
		t.Errorf("Expected 3 players in scoreboard, got %d", len(scoreboard))
	}

	// Check sorted order (highest first)
	if scoreboard[0].Name != "Bob" {
		t.Errorf("Expected Bob first, got %s", scoreboard[0].Name)
	}

	if scoreboard[1].Name != "Alice" {
		t.Errorf("Expected Alice second, got %s", scoreboard[1].Name)
	}

	if scoreboard[2].Name != "Charlie" {
		t.Errorf("Expected Charlie third, got %s", scoreboard[2].Name)
	}
}

func TestGetWinner(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddPlayer("player1", "Alice")
	game.AddPlayer("player2", "Bob")

	game.Players["player1"].Score = 300
	game.Players["player2"].Score = 500

	winner := game.GetWinner()

	if winner.Name != "Bob" {
		t.Errorf("Expected Bob to win, got %s", winner.Name)
	}
}

func TestGetStats(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.Start()
	game.NextQuestion()

	stats := game.GetStats()

	if stats["total_questions"] != 3 {
		t.Errorf("Expected 3 total questions, got %v", stats["total_questions"])
	}

	if stats["current_question"] != 1 {
		t.Errorf("Expected current question 1, got %v", stats["current_question"])
	}

	if stats["total_players"] != 1 {
		t.Errorf("Expected 1 player, got %v", stats["total_players"])
	}

	if stats["state"] != "Question" {
		t.Errorf("Expected state Question, got %v", stats["state"])
	}
}

func TestQuestionFormatting(t *testing.T) {
	q := &Question{
		ID:         "q1",
		Text:       "What is the capital of France?",
		Options:    []string{"London", "Berlin", "Paris", "Madrid"},
		Correct:    2,
		Category:   "Geography",
		Difficulty: 1,
		Points:     100,
	}

	formatted := q.FormatQuestion()

	if len(formatted) == 0 {
		t.Error("Formatted question is empty")
	}

	// Should contain question text
	if !contains(formatted, "What is the capital of France?") {
		t.Error("Formatted question missing question text")
	}

	// Should contain all options
	for _, opt := range q.Options {
		if !contains(formatted, opt) {
			t.Errorf("Formatted question missing option: %s", opt)
		}
	}
}

func TestScoreboardFormatting(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddPlayer("player1", "Alice")
	game.AddPlayer("player2", "Bob")

	game.Players["player1"].Score = 300
	game.Players["player2"].Score = 500

	formatted := game.FormatScoreboard()

	if len(formatted) == 0 {
		t.Error("Formatted scoreboard is empty")
	}

	// Should contain player names and scores
	if !contains(formatted, "Alice") || !contains(formatted, "Bob") {
		t.Error("Formatted scoreboard missing player names")
	}

	// Should show medals
	if !contains(formatted, "🥇") {
		t.Error("Formatted scoreboard missing gold medal")
	}
}

func TestCallbacks(t *testing.T) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")

	gameStarted := false
	questionAsked := false
	answerSubmitted := false
	questionEnded := false

	game.Callbacks.OnGameStart = func(g *TriviaGame) {
		gameStarted = true
	}

	game.Callbacks.OnQuestionAsked = func(g *TriviaGame, q *Question) {
		questionAsked = true
	}

	game.Callbacks.OnAnswer = func(g *TriviaGame, p *Player, a *Answer) {
		answerSubmitted = true
	}

	game.Callbacks.OnQuestionEnd = func(g *TriviaGame, q *Question, correct int) {
		questionEnded = true
	}

	game.Start()
	time.Sleep(10 * time.Millisecond) // Wait for goroutine

	if !gameStarted {
		t.Error("OnGameStart callback not called")
	}

	game.NextQuestion()
	time.Sleep(10 * time.Millisecond)

	if !questionAsked {
		t.Error("OnQuestionAsked callback not called")
	}

	game.SubmitAnswer("player1", 2)
	time.Sleep(10 * time.Millisecond)

	if !answerSubmitted {
		t.Error("OnAnswer callback not called")
	}

	game.RevealAnswer()
	time.Sleep(10 * time.Millisecond)

	if !questionEnded {
		t.Error("OnQuestionEnd callback not called")
	}
}

func TestTimeBonusPoints(t *testing.T) {
	game := NewTriviaGame("test", 10*time.Second)
	q := &Question{
		ID:         "q1",
		Text:       "Test?",
		Options:    []string{"A", "B"},
		Correct:    0,
		Category:   "Test",
		Difficulty: 1,
		Points:     100,
	}
	game.AddQuestion(q)
	game.AddPlayer("player1", "Alice")
	game.Start()
	game.NextQuestion()

	// Answer immediately (should get time bonus)
	game.SubmitAnswer("player1", 0)

	player := game.Players["player1"]
	if player.Score <= 100 {
		t.Errorf("Expected time bonus, got score %d", player.Score)
	}

	// Time bonus can be up to 50% more, so max 150
	if player.Score > 150 {
		t.Errorf("Time bonus too high, got %d", player.Score)
	}
}

func TestJSONRoundtrip(t *testing.T) {
	questions := createTestQuestions()

	data, err := json.Marshal(questions)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	game := NewTriviaGame("test", 30*time.Second)
	err = game.LoadQuestionsFromJSON(data)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if len(game.Questions) != 3 {
		t.Errorf("Expected 3 questions, got %d", len(game.Questions))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkAddQuestion(b *testing.B) {
	game := NewTriviaGame("test", 30*time.Second)
	q := createTestQuestions()[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game.AddQuestion(q)
	}
}

func BenchmarkSubmitAnswer(b *testing.B) {
	game := NewTriviaGame("test", 30*time.Second)
	game.AddQuestions(createTestQuestions())
	game.AddPlayer("player1", "Alice")
	game.Start()
	game.NextQuestion()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset for each iteration
		game.Players["player1"].Answers = game.Players["player1"].Answers[:0]
		game.SubmitAnswer("player1", 2)
	}
}

func BenchmarkGetScoreboard(b *testing.B) {
	game := NewTriviaGame("test", 30*time.Second)
	for i := 0; i < 100; i++ {
		game.AddPlayer(string(rune(i)), string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game.GetScoreboard()
	}
}
