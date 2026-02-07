//go:build experimental

package trivia_test

import (
	"fmt"
	"time"

	"github.com/svend4/infon/experimental/games/trivia"
)

// Example demonstrates basic trivia game usage
func Example() {
	// Create a new game with 30 second time limit per question
	game := trivia.NewTriviaGame("quiz-2025", 30*time.Second)

	// Add questions
	questions := []*trivia.Question{
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
	}

	game.AddQuestions(questions)

	// Add players
	game.AddPlayer("player1", "Alice")
	game.AddPlayer("player2", "Bob")

	// Set up callbacks
	game.Callbacks.OnQuestionAsked = func(g *trivia.TriviaGame, q *trivia.Question) {
		fmt.Println(q.FormatQuestion())
	}

	game.Callbacks.OnScoreUpdate = func(g *trivia.TriviaGame) {
		fmt.Println("Scores updated!")
	}

	// Start the game
	game.Start()

	// Ask first question
	game.NextQuestion()

	// Players submit answers
	game.SubmitAnswer("player1", 2) // Alice answers Paris (correct)
	game.SubmitAnswer("player2", 1) // Bob answers Berlin (wrong)

	// Reveal answer
	game.RevealAnswer()

	// Show scoreboard
	fmt.Println(game.FormatScoreboard())

	// Continue to next question
	game.ContinueToNext()
}

// Example_fullGame demonstrates a complete game flow
func Example_fullGame() {
	game := trivia.NewTriviaGame("full-game", 10*time.Second)

	// Load questions from JSON (in real usage)
	jsonData := `[
		{
			"id": "geography1",
			"text": "Which river is the longest in the world?",
			"options": ["Nile", "Amazon", "Yangtze", "Mississippi"],
			"correct": 0,
			"category": "Geography",
			"difficulty": 2,
			"points": 200
		},
		{
			"id": "history1",
			"text": "In which year did World War II end?",
			"options": ["1943", "1944", "1945", "1946"],
			"correct": 2,
			"category": "History",
			"difficulty": 2,
			"points": 200
		}
	]`

	game.LoadQuestionsFromJSON([]byte(jsonData))

	// Add players
	game.AddPlayer("alice", "Alice")
	game.AddPlayer("bob", "Bob")
	game.AddPlayer("charlie", "Charlie")

	// Set up event handlers
	questionCount := 0

	game.Callbacks.OnGameStart = func(g *trivia.TriviaGame) {
		fmt.Println("🎮 Game Started!")
		fmt.Printf("Players: %d, Questions: %d\n\n", len(g.Players), len(g.Questions))
	}

	game.Callbacks.OnQuestionAsked = func(g *trivia.TriviaGame, q *trivia.Question) {
		questionCount++
		fmt.Printf("\n📝 Question %d/%d\n", questionCount, len(g.Questions))
		fmt.Println(q.FormatQuestion())
	}

	game.Callbacks.OnAnswer = func(g *trivia.TriviaGame, p *trivia.Player, a *trivia.Answer) {
		if a.Correct {
			fmt.Printf("✅ %s answered correctly! (+%d points)\n", p.Name, a.Points)
		} else {
			fmt.Printf("❌ %s answered incorrectly\n", p.Name)
		}
	}

	game.Callbacks.OnQuestionEnd = func(g *trivia.TriviaGame, q *trivia.Question, correct int) {
		fmt.Printf("\n💡 Correct answer: %s\n", q.Options[correct])
	}

	game.Callbacks.OnGameEnd = func(g *trivia.TriviaGame, winner *trivia.Player) {
		fmt.Println("\n🎉 Game Finished!")
		if winner != nil {
			fmt.Printf("👑 Winner: %s with %d points!\n\n", winner.Name, winner.Score)
		}
		fmt.Println(g.FormatScoreboard())
	}

	// Start game
	game.Start()

	// Play through all questions
	for {
		_, err := game.NextQuestion()
		if err != nil {
			break
		}

		// Simulate players answering (in real usage, this would be async)
		time.Sleep(100 * time.Millisecond)
		game.SubmitAnswer("alice", 0)

		time.Sleep(50 * time.Millisecond)
		game.SubmitAnswer("bob", 1)

		time.Sleep(75 * time.Millisecond)
		game.SubmitAnswer("charlie", 2)

		// Reveal and continue
		game.RevealAnswer()
		time.Sleep(200 * time.Millisecond)
		game.ContinueToNext()
	}

	// Get final statistics
	stats := game.GetStats()
	fmt.Printf("\n📊 Game Statistics:\n")
	for key, value := range stats {
		fmt.Printf("  %s: %v\n", key, value)
	}
}

// Example_chatIntegration shows how to integrate with chat
func Example_chatIntegration() {
	game := trivia.NewTriviaGame("chat-quiz", 30*time.Second)

	// Add sample questions
	game.AddQuestion(&trivia.Question{
		ID:         "q1",
		Text:       "What programming language is TVCP written in?",
		Options:    []string{"Python", "JavaScript", "Go", "Rust"},
		Correct:    2,
		Category:   "Programming",
		Difficulty: 1,
		Points:     100,
	})

	game.AddPlayer("user123", "Alice")
	game.AddPlayer("user456", "Bob")

	// Simulate chat integration
	game.Callbacks.OnQuestionAsked = func(g *trivia.TriviaGame, q *trivia.Question) {
		// Send to chat
		message := fmt.Sprintf("[TRIVIA] %s", q.FormatQuestion())
		sendToChat(message)
	}

	game.Callbacks.OnAnswer = func(g *trivia.TriviaGame, p *trivia.Player, a *trivia.Answer) {
		emoji := "❌"
		if a.Correct {
			emoji = "✅"
		}
		message := fmt.Sprintf("%s %s submitted answer", emoji, p.Name)
		sendToChat(message)
	}

	game.Start()
	game.NextQuestion()

	// Users answer via chat commands
	// /answer 3
	handleChatCommand := func(userID string, command string, arg string) {
		switch command {
		case "/answer":
			var option int
			fmt.Sscanf(arg, "%d", &option)
			if err := game.SubmitAnswer(userID, option-1); err != nil {
				sendToChat(fmt.Sprintf("Error: %v", err))
			}
		case "/scores":
			sendToChat(game.FormatScoreboard())
		}
	}

	// Simulate chat commands
	handleChatCommand("user123", "/answer", "3")
	handleChatCommand("user456", "/answer", "1")

	game.RevealAnswer()

	handleChatCommand("user123", "/scores", "")
}

// Mock function for example
func sendToChat(message string) {
	fmt.Println("[CHAT]", message)
}
