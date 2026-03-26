//go:build experimental

package words

import (
	"strings"
	"testing"
)

// ============ WORDLE TESTS ============

func TestNewWordleGame(t *testing.T) {
	game := NewWordleGame("test", "HELLO")

	if game.Word != "HELLO" {
		t.Errorf("Expected word HELLO, got %s", game.Word)
	}

	if game.WordLength != 5 {
		t.Errorf("Expected length 5, got %d", game.WordLength)
	}

	if game.MaxAttempts != 6 {
		t.Errorf("Expected 6 attempts, got %d", game.MaxAttempts)
	}

	if game.State != WordleWaiting {
		t.Errorf("Expected state Waiting, got %v", game.State)
	}
}

func TestNewDailyWordleGame(t *testing.T) {
	game := NewDailyWordleGame("daily", 42)

	if !game.DailyMode {
		t.Error("Expected daily mode to be true")
	}

	if len(game.Word) == 0 {
		t.Error("Word should not be empty")
	}
}

func TestWordleAddPlayer(t *testing.T) {
	game := NewWordleGame("test", "HELLO")

	err := game.AddPlayer("p1", "Alice")
	if err != nil {
		t.Fatalf("Failed to add player: %v", err)
	}

	if len(game.Players) != 1 {
		t.Errorf("Expected 1 player, got %d", len(game.Players))
	}

	// Try adding same player again
	err = game.AddPlayer("p1", "Alice")
	if err == nil {
		t.Error("Expected error when adding duplicate player")
	}
}

func TestWordleStart(t *testing.T) {
	game := NewWordleGame("test", "HELLO")

	// Try starting without players
	err := game.Start()
	if err == nil {
		t.Error("Expected error when starting without players")
	}

	game.AddPlayer("p1", "Alice")

	err = game.Start()
	if err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	if game.State != WordlePlaying {
		t.Errorf("Expected state Playing, got %v", game.State)
	}
}

func TestWordleMakeGuess(t *testing.T) {
	game := NewWordleGame("test", "PIANO")
	game.AddPlayer("p1", "Alice")
	game.Start()

	attempt, err := game.MakeGuess("p1", "HOUSE")
	if err != nil {
		t.Fatalf("Failed to make guess: %v", err)
	}

	if attempt.Word != "HOUSE" {
		t.Errorf("Expected HOUSE, got %s", attempt.Word)
	}

	if len(attempt.Result) != 5 {
		t.Errorf("Expected 5 results, got %d", len(attempt.Result))
	}

	player := game.Players["p1"]
	if player.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", player.Attempts)
	}
}

func TestWordleCorrectGuess(t *testing.T) {
	game := NewWordleGame("test", "PIANO")
	game.AddPlayer("p1", "Alice")
	game.Start()

	attempt, err := game.MakeGuess("p1", "PIANO")
	if err != nil {
		t.Fatalf("Failed to make guess: %v", err)
	}

	// All letters should be correct
	for i, status := range attempt.Result {
		if status != LetterCorrect {
			t.Errorf("Letter %d should be correct, got %v", i, status)
		}
	}

	player := game.Players["p1"]
	if !player.Won {
		t.Error("Player should have won")
	}
}

func TestWordleCheckGuess(t *testing.T) {
	game := NewWordleGame("test", "PIANO")

	// Test exact match
	result := game.checkGuess("PIANO")
	for i, status := range result {
		if status != LetterCorrect {
			t.Errorf("Position %d should be correct", i)
		}
	}

	// Test wrong position
	result = game.checkGuess("PAINO")
	// P - correct position
	// A - correct position
	// I - wrong position (should be at index 1)
	// N - wrong position (should be at index 3)
	// O - correct position

	if result[0] != LetterCorrect {
		t.Errorf("P should be correct")
	}

	// Test no matches
	result = game.checkGuess("HELLO")
	// Should not have any of PIANO's letters in correct positions
}

func TestWordleInvalidGuess(t *testing.T) {
	game := NewWordleGame("test", "PIANO")
	game.AddPlayer("p1", "Alice")
	game.Start()

	// Wrong length
	_, err := game.MakeGuess("p1", "HI")
	if err == nil {
		t.Error("Expected error for wrong length")
	}

	// Invalid characters
	_, err = game.MakeGuess("p1", "12345")
	if err == nil {
		t.Error("Expected error for invalid characters")
	}
}

func TestWordleMaxAttempts(t *testing.T) {
	game := NewWordleGame("test", "PIANO")
	game.AddPlayer("p1", "Alice")
	game.Start()

	// Make 6 wrong guesses
	words := []string{"HOUSE", "BEACH", "GRAPE", "TIGER", "SMILE", "CLOUD"}
	for i, word := range words {
		_, err := game.MakeGuess("p1", word)
		if err != nil {
			t.Fatalf("Guess %d failed: %v", i, err)
		}
	}

	player := game.Players["p1"]
	if player.Attempts != 6 {
		t.Errorf("Expected 6 attempts, got %d", player.Attempts)
	}

	// Try one more - should fail
	_, err := game.MakeGuess("p1", "ROBOT")
	if err == nil {
		t.Error("Expected error after max attempts")
	}

	if game.State != WordleLost {
		t.Errorf("Expected state Lost, got %v", game.State)
	}
}

func TestWordleFormatBoard(t *testing.T) {
	game := NewWordleGame("test", "PIANO")
	game.AddPlayer("p1", "Alice")
	game.Start()

	game.MakeGuess("p1", "HOUSE")

	board := game.FormatBoard("p1")

	if !strings.Contains(board, "WORDLE") {
		t.Error("Board should contain WORDLE title")
	}

	if !strings.Contains(board, "HOUSE") {
		t.Error("Board should show the guess")
	}
}

func TestWordleGetScoreboard(t *testing.T) {
	game := NewWordleGame("test", "PIANO")
	game.AddPlayer("p1", "Alice")
	game.AddPlayer("p2", "Bob")
	game.Start()

	game.MakeGuess("p1", "PIANO") // Alice wins immediately
	game.MakeGuess("p2", "HOUSE") // Bob makes one wrong guess

	scoreboard := game.GetScoreboard()

	if len(scoreboard) != 2 {
		t.Errorf("Expected 2 players in scoreboard, got %d", len(scoreboard))
	}

	// Winner should be first
	if !scoreboard[0].Won {
		t.Error("First player should be the winner")
	}
}

// ============ HANGMAN TESTS ============

func TestNewHangmanGame(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "Greetings", "Common greeting")

	if game.Word != "HELLO" {
		t.Errorf("Expected word HELLO, got %s", game.Word)
	}

	if game.Category != "Greetings" {
		t.Errorf("Expected category Greetings, got %s", game.Category)
	}

	if game.MaxWrong != 6 {
		t.Errorf("Expected 6 max wrong, got %d", game.MaxWrong)
	}

	if game.State != HangmanWaiting {
		t.Errorf("Expected state Waiting, got %v", game.State)
	}
}

func TestNewMultiplayerHangman(t *testing.T) {
	game := NewMultiplayerHangman("test", "HELLO", "Words", "Greeting")

	if !game.Multiplayer {
		t.Error("Expected multiplayer to be true")
	}
}

func TestHangmanAddPlayer(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")

	err := game.AddPlayer("p1", "Alice")
	if err != nil {
		t.Fatalf("Failed to add player: %v", err)
	}

	if len(game.Players) != 1 {
		t.Errorf("Expected 1 player, got %d", len(game.Players))
	}

	if game.CurrentTurn != "p1" {
		t.Error("First player should have first turn")
	}

	// Try adding duplicate
	err = game.AddPlayer("p1", "Alice")
	if err == nil {
		t.Error("Expected error for duplicate player")
	}
}

func TestHangmanStart(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")

	// Try starting without players
	err := game.Start()
	if err == nil {
		t.Error("Expected error when starting without players")
	}

	game.AddPlayer("p1", "Alice")

	err = game.Start()
	if err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	if game.State != HangmanPlaying {
		t.Errorf("Expected state Playing, got %v", game.State)
	}
}

func TestHangmanGuessLetter(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")
	game.AddPlayer("p1", "Alice")
	game.Start()

	// Guess correct letter
	err := game.GuessLetter("p1", 'H')
	if err != nil {
		t.Fatalf("Failed to guess letter: %v", err)
	}

	if !game.GuessedLetters['H'] {
		t.Error("H should be marked as guessed")
	}

	player := game.Players["p1"]
	if player.CorrectGuesses != 1 {
		t.Errorf("Expected 1 correct guess, got %d", player.CorrectGuesses)
	}

	if player.Score != 10 {
		t.Errorf("Expected score 10, got %d", player.Score)
	}
}

func TestHangmanGuessWrongLetter(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")
	game.AddPlayer("p1", "Alice")
	game.Start()

	// Guess wrong letter
	err := game.GuessLetter("p1", 'Z')
	if err != nil {
		t.Fatalf("Failed to guess letter: %v", err)
	}

	if len(game.WrongGuesses) != 1 {
		t.Errorf("Expected 1 wrong guess, got %d", len(game.WrongGuesses))
	}

	if game.WrongGuesses[0] != 'Z' {
		t.Errorf("Expected Z in wrong guesses, got %c", game.WrongGuesses[0])
	}
}

func TestHangmanDuplicateGuess(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")
	game.AddPlayer("p1", "Alice")
	game.Start()

	game.GuessLetter("p1", 'H')

	// Try guessing H again
	err := game.GuessLetter("p1", 'H')
	if err == nil {
		t.Error("Expected error for duplicate guess")
	}
}

func TestHangmanWinByLetters(t *testing.T) {
	game := NewHangmanGame("test", "HI", "", "")
	game.AddPlayer("p1", "Alice")
	game.Start()

	game.GuessLetter("p1", 'H')

	if game.State != HangmanPlaying {
		t.Error("Game should still be playing after first letter")
	}

	game.GuessLetter("p1", 'I')

	if game.State != HangmanWon {
		t.Errorf("Expected state Won, got %v", game.State)
	}

	player := game.Players["p1"]
	if player.Score < 60 { // 2*10 + 50 bonus = 70
		t.Errorf("Expected bonus points, got score %d", player.Score)
	}
}

func TestHangmanLoseByLetters(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")
	game.AddPlayer("p1", "Alice")
	game.Start()

	// Make 6 wrong guesses
	wrongLetters := []rune{'Z', 'X', 'Q', 'W', 'K', 'J'}
	for _, letter := range wrongLetters {
		game.GuessLetter("p1", letter)
	}

	if game.State != HangmanLost {
		t.Errorf("Expected state Lost, got %v", game.State)
	}
}

func TestHangmanGuessWord(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")
	game.AddPlayer("p1", "Alice")
	game.Start()

	err := game.GuessWord("p1", "HELLO")
	if err != nil {
		t.Fatalf("Failed to guess word: %v", err)
	}

	if game.State != HangmanWon {
		t.Errorf("Expected state Won, got %v", game.State)
	}

	player := game.Players["p1"]
	if player.Score != 100 {
		t.Errorf("Expected score 100 for word guess, got %d", player.Score)
	}
}

func TestHangmanGuessWrongWord(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")
	game.AddPlayer("p1", "Alice")
	game.Start()

	err := game.GuessWord("p1", "WORLD")
	if err == nil {
		t.Error("Expected error for wrong word")
	}

	if len(game.WrongGuesses) != 1 {
		t.Errorf("Expected 1 wrong guess, got %d", len(game.WrongGuesses))
	}
}

func TestHangmanGetRevealedWord(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")
	game.AddPlayer("p1", "Alice")
	game.Start()

	revealed := game.GetRevealedWord()

	if !strings.Contains(revealed, "_") {
		t.Error("Revealed word should contain underscores")
	}

	game.GuessLetter("p1", 'H')
	revealed = game.GetRevealedWord()

	if !strings.Contains(revealed, "H") {
		t.Error("Revealed word should contain H after guessing")
	}

	if !strings.Contains(revealed, "_") {
		t.Error("Revealed word should still have underscores")
	}
}

func TestHangmanMultiplayerTurns(t *testing.T) {
	game := NewMultiplayerHangman("test", "HELLO", "", "")
	game.AddPlayer("p1", "Alice")
	game.AddPlayer("p2", "Bob")
	game.Start()

	if game.CurrentTurn != "p1" {
		t.Error("Alice should have first turn")
	}

	// Alice guesses
	game.GuessLetter("p1", 'H')

	if game.CurrentTurn != "p2" {
		t.Error("Turn should switch to Bob")
	}

	// Bob tries to guess, and it's his turn
	err := game.GuessLetter("p2", 'E')
	if err != nil {
		t.Fatalf("Bob should be able to guess on his turn: %v", err)
	}

	// Turn should switch back to Alice
	if game.CurrentTurn != "p1" {
		t.Error("Turn should switch back to Alice")
	}

	// Bob tries to guess but it's not his turn anymore
	err = game.GuessLetter("p2", 'L')
	if err == nil {
		t.Error("Expected error when not player's turn")
	}
}

func TestHangmanFormatGame(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "Greetings", "Common word")
	game.AddPlayer("p1", "Alice")
	game.Start()

	game.GuessLetter("p1", 'H')
	game.GuessLetter("p1", 'Z')

	formatted := game.FormatGame()

	if !strings.Contains(formatted, "HANGMAN") {
		t.Error("Formatted game should contain HANGMAN title")
	}

	if !strings.Contains(formatted, "Greetings") {
		t.Error("Formatted game should show category")
	}

	if !strings.Contains(formatted, "Common word") {
		t.Error("Formatted game should show hint")
	}

	if !strings.Contains(formatted, "H") {
		t.Error("Formatted game should show revealed letter H")
	}

	if !strings.Contains(formatted, "_") {
		t.Error("Formatted game should show unrevealed letters")
	}

	if !strings.Contains(formatted, "Z") {
		t.Error("Formatted game should show wrong guesses")
	}
}

func TestHangmanDrawHangman(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")

	// Test each stage
	for i := 0; i <= 6; i++ {
		game.WrongGuesses = make([]rune, i)
		for j := 0; j < i; j++ {
			game.WrongGuesses[j] = rune('A' + j)
		}

		game.mu.RLock()
		drawing := game.drawHangmanLocked()
		game.mu.RUnlock()

		if len(drawing) == 0 {
			t.Errorf("Drawing should not be empty at stage %d", i)
		}

		// Check that gallows is always there
		if !strings.Contains(drawing, "┌───┐") {
			t.Error("Drawing should contain gallows")
		}
	}
}

func TestHangmanGetScoreboard(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "", "")
	game.AddPlayer("p1", "Alice")
	game.AddPlayer("p2", "Bob")
	game.Start()

	// Alice guesses correctly
	game.GuessLetter("p1", 'H')
	game.GuessLetter("p2", 'Z') // Bob guesses wrong

	scoreboard := game.GetScoreboard()

	if len(scoreboard) != 2 {
		t.Errorf("Expected 2 players in scoreboard, got %d", len(scoreboard))
	}

	// Alice should be first (higher score)
	if scoreboard[0].ID != "p1" {
		t.Error("Alice should be first in scoreboard")
	}
}

func TestWordleGetStats(t *testing.T) {
	game := NewWordleGame("test", "HELLO")
	game.AddPlayer("p1", "Alice")
	game.Start()

	stats := game.GetStats()

	if stats["word_length"] != 5 {
		t.Errorf("Expected word length 5, got %v", stats["word_length"])
	}

	if stats["total_players"] != 1 {
		t.Errorf("Expected 1 player, got %v", stats["total_players"])
	}

	if stats["state"] != "Playing" {
		t.Errorf("Expected state Playing, got %v", stats["state"])
	}
}

func TestHangmanGetStats(t *testing.T) {
	game := NewHangmanGame("test", "HELLO", "Words", "Greeting")
	game.AddPlayer("p1", "Alice")
	game.Start()

	stats := game.GetStats()

	if stats["word_length"] != 5 {
		t.Errorf("Expected word length 5, got %v", stats["word_length"])
	}

	if stats["total_players"] != 1 {
		t.Errorf("Expected 1 player, got %v", stats["total_players"])
	}

	if stats["multiplayer"] != false {
		t.Errorf("Expected multiplayer false, got %v", stats["multiplayer"])
	}
}

func TestLetterStatusString(t *testing.T) {
	if LetterWrong.String() != "⬜" {
		t.Errorf("Wrong emoji incorrect")
	}

	if LetterMisplaced.String() != "🟨" {
		t.Errorf("Misplaced emoji incorrect")
	}

	if LetterCorrect.String() != "🟩" {
		t.Errorf("Correct emoji incorrect")
	}
}

func TestWordleStateString(t *testing.T) {
	states := []WordleState{WordleWaiting, WordlePlaying, WordleWon, WordleLost}
	expected := []string{"Waiting", "Playing", "Won", "Lost"}

	for i, state := range states {
		if state.String() != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], state.String())
		}
	}
}

func TestHangmanStateString(t *testing.T) {
	states := []HangmanState{HangmanWaiting, HangmanPlaying, HangmanWon, HangmanLost}
	expected := []string{"Waiting", "Playing", "Won", "Lost"}

	for i, state := range states {
		if state.String() != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], state.String())
		}
	}
}

// Benchmarks
func BenchmarkWordleCheckGuess(b *testing.B) {
	game := NewWordleGame("bench", "PIANO")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game.checkGuess("HOUSE")
	}
}

func BenchmarkHangmanGuessLetter(b *testing.B) {
	game := NewHangmanGame("bench", "HELLO", "", "")
	game.AddPlayer("p1", "Test")
	game.Start()

	letters := []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'I', 'J', 'K'}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game.mu.Lock()
		game.GuessedLetters = make(map[rune]bool)
		game.WrongGuesses = game.WrongGuesses[:0]
		game.State = HangmanPlaying
		game.mu.Unlock()

		game.GuessLetter("p1", letters[i%len(letters)])
	}
}

func BenchmarkWordleFullGame(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game := NewWordleGame("bench", "PIANO")
		game.AddPlayer("p1", "Test")
		game.Start()

		guesses := []string{"HOUSE", "BEACH", "GRAPE", "PIANO"}
		for _, guess := range guesses {
			game.MakeGuess("p1", guess)
			if game.State != WordlePlaying {
				break
			}
		}
	}
}

func BenchmarkHangmanFullGame(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game := NewHangmanGame("bench", "HELLO", "", "")
		game.AddPlayer("p1", "Test")
		game.Start()

		letters := []rune{'H', 'E', 'L', 'O'}
		for _, letter := range letters {
			game.GuessLetter("p1", letter)
			if game.State != HangmanPlaying {
				break
			}
		}
	}
}
