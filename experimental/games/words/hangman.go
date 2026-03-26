//go:build experimental

package words

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// HangmanGame represents a hangman game
type HangmanGame struct {
	mu            sync.RWMutex
	ID            string
	Word          string
	Category      string
	Hint          string
	GuessedLetters map[rune]bool
	WrongGuesses  []rune
	MaxWrong      int
	Players       map[string]*HangmanPlayer
	CurrentTurn   string // PlayerID
	State         HangmanState
	StartTime     time.Time
	EndTime       time.Time
	Multiplayer   bool
}

// HangmanState represents game state
type HangmanState int

const (
	HangmanWaiting HangmanState = iota
	HangmanPlaying
	HangmanWon
	HangmanLost
)

func (s HangmanState) String() string {
	switch s {
	case HangmanWaiting:
		return "Waiting"
	case HangmanPlaying:
		return "Playing"
	case HangmanWon:
		return "Won"
	case HangmanLost:
		return "Lost"
	default:
		return "Unknown"
	}
}

// HangmanPlayer represents a player
type HangmanPlayer struct {
	ID             string
	Name           string
	CorrectGuesses int
	WrongGuesses   int
	Score          int
	LastGuessTime  time.Time
}

// NewHangmanGame creates a new hangman game
func NewHangmanGame(id, word, category, hint string) *HangmanGame {
	word = strings.ToUpper(word)

	return &HangmanGame{
		ID:             id,
		Word:           word,
		Category:       category,
		Hint:           hint,
		GuessedLetters: make(map[rune]bool),
		WrongGuesses:   make([]rune, 0, 7),
		MaxWrong:       6, // Classic hangman: head, body, 2 arms, 2 legs
		Players:        make(map[string]*HangmanPlayer),
		State:          HangmanWaiting,
		Multiplayer:    false,
	}
}

// NewMultiplayerHangman creates a competitive hangman game
func NewMultiplayerHangman(id, word, category, hint string) *HangmanGame {
	game := NewHangmanGame(id, word, category, hint)
	game.Multiplayer = true
	return game
}

// AddPlayer adds a player to the game
func (g *HangmanGame) AddPlayer(id, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.Players[id]; exists {
		return errors.New("player already exists")
	}

	player := &HangmanPlayer{
		ID:   id,
		Name: name,
	}

	g.Players[id] = player

	if g.CurrentTurn == "" {
		g.CurrentTurn = id
	}

	return nil
}

// Start starts the game
func (g *HangmanGame) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != HangmanWaiting {
		return errors.New("game already started")
	}

	if len(g.Players) == 0 {
		return errors.New("no players")
	}

	g.State = HangmanPlaying
	g.StartTime = time.Now()

	return nil
}

// GuessLetter processes a letter guess
func (g *HangmanGame) GuessLetter(playerID string, letter rune) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != HangmanPlaying {
		return errors.New("game not in playing state")
	}

	player, exists := g.Players[playerID]
	if !exists {
		return errors.New("player not found")
	}

	if g.Multiplayer && g.CurrentTurn != playerID {
		return errors.New("not your turn")
	}

	letter = toUpperRune(letter)

	// Check if already guessed
	if g.GuessedLetters[letter] {
		return errors.New("letter already guessed")
	}

	g.GuessedLetters[letter] = true
	player.LastGuessTime = time.Now()

	// Check if letter is in word
	found := false
	for _, ch := range g.Word {
		if ch == letter {
			found = true
			break
		}
	}

	if found {
		player.CorrectGuesses++
		player.Score += 10

		// Check if won
		if g.isWordCompleteLocked() {
			g.State = HangmanWon
			g.EndTime = time.Now()
			player.Score += 50 // Bonus for winning
		}
	} else {
		g.WrongGuesses = append(g.WrongGuesses, letter)
		player.WrongGuesses++

		// Check if lost
		if len(g.WrongGuesses) >= g.MaxWrong {
			g.State = HangmanLost
			g.EndTime = time.Now()
		}
	}

	// Switch turn if multiplayer
	if g.Multiplayer && g.State == HangmanPlaying {
		g.switchTurnLocked()
	}

	return nil
}

// GuessWord processes a full word guess
func (g *HangmanGame) GuessWord(playerID, word string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != HangmanPlaying {
		return errors.New("game not in playing state")
	}

	player, exists := g.Players[playerID]
	if !exists {
		return errors.New("player not found")
	}

	if g.Multiplayer && g.CurrentTurn != playerID {
		return errors.New("not your turn")
	}

	word = strings.ToUpper(word)
	player.LastGuessTime = time.Now()

	if word == g.Word {
		// Correct word guess
		g.State = HangmanWon
		g.EndTime = time.Now()
		player.Score += 100 // Big bonus for word guess
		return nil
	} else {
		// Wrong word guess - penalty
		g.WrongGuesses = append(g.WrongGuesses, []rune("*")[0]) // Use * as marker for word guess
		player.WrongGuesses++

		if len(g.WrongGuesses) >= g.MaxWrong {
			g.State = HangmanLost
			g.EndTime = time.Now()
		}

		// Switch turn if multiplayer
		if g.Multiplayer && g.State == HangmanPlaying {
			g.switchTurnLocked()
		}

		return errors.New("wrong word")
	}
}

// isWordCompleteLocked checks if all letters have been guessed (must hold lock)
func (g *HangmanGame) isWordCompleteLocked() bool {
	for _, ch := range g.Word {
		if ch == ' ' || ch == '-' {
			continue // Skip spaces and hyphens
		}
		if !g.GuessedLetters[ch] {
			return false
		}
	}
	return true
}

// switchTurnLocked switches to next player (must hold lock)
func (g *HangmanGame) switchTurnLocked() {
	if !g.Multiplayer || len(g.Players) <= 1 {
		return
	}

	// Find next player
	playerIDs := make([]string, 0, len(g.Players))
	for id := range g.Players {
		playerIDs = append(playerIDs, id)
	}

	currentIdx := -1
	for i, id := range playerIDs {
		if id == g.CurrentTurn {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + 1) % len(playerIDs)
	g.CurrentTurn = playerIDs[nextIdx]
}

// GetRevealedWord returns the word with unrevealed letters as underscores
func (g *HangmanGame) GetRevealedWord() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sb strings.Builder

	for _, ch := range g.Word {
		if ch == ' ' {
			sb.WriteString("  ")
		} else if ch == '-' {
			sb.WriteString(" - ")
		} else if g.GuessedLetters[ch] {
			sb.WriteString(fmt.Sprintf(" %c ", ch))
		} else {
			sb.WriteString(" _ ")
		}
	}

	return sb.String()
}

// GetScoreboard returns player rankings
func (g *HangmanGame) GetScoreboard() []*HangmanPlayer {
	g.mu.RLock()
	defer g.mu.RUnlock()

	players := make([]*HangmanPlayer, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, p)
	}

	// Sort by score (descending)
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			if players[j].Score > players[i].Score {
				players[i], players[j] = players[j], players[i]
			}
		}
	}

	return players
}

// FormatGame formats the game state
func (g *HangmanGame) FormatGame() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("╔═══════════════════════════╗\n")
	sb.WriteString("║       HANGMAN GAME        ║\n")
	sb.WriteString("╚═══════════════════════════╝\n\n")

	// Draw hangman figure
	sb.WriteString(g.drawHangmanLocked())
	sb.WriteString("\n")

	// Category and hint
	if g.Category != "" {
		sb.WriteString(fmt.Sprintf("Category: %s\n", g.Category))
	}
	if g.Hint != "" {
		sb.WriteString(fmt.Sprintf("Hint: %s\n", g.Hint))
	}
	sb.WriteString("\n")

	// Revealed word
	sb.WriteString("Word: " + g.GetRevealedWord() + "\n\n")

	// Wrong guesses
	if len(g.WrongGuesses) > 0 {
		sb.WriteString("Wrong guesses: ")
		for _, ch := range g.WrongGuesses {
			if ch == '*' {
				sb.WriteString("[WORD] ")
			} else {
				sb.WriteString(fmt.Sprintf("%c ", ch))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Remaining wrong guesses: %d/%d\n\n", g.MaxWrong-len(g.WrongGuesses), g.MaxWrong))

	// Game state
	switch g.State {
	case HangmanWon:
		sb.WriteString(fmt.Sprintf("🎉 Victory! The word was: %s\n", g.Word))
	case HangmanLost:
		sb.WriteString(fmt.Sprintf("💀 Game Over! The word was: %s\n", g.Word))
	case HangmanPlaying:
		if g.Multiplayer {
			currentPlayer := g.Players[g.CurrentTurn]
			sb.WriteString(fmt.Sprintf("Current turn: %s\n", currentPlayer.Name))
		}
	}

	return sb.String()
}

// drawHangmanLocked draws the hangman figure based on wrong guesses (must hold lock)
func (g *HangmanGame) drawHangmanLocked() string {
	stage := len(g.WrongGuesses)

	stages := []string{
		// 0 wrong
		`
   ┌───┐
   │
   │
   │
   │
   │
  ━┷━
`,
		// 1 wrong (head)
		`
   ┌───┐
   │   ○
   │
   │
   │
   │
  ━┷━
`,
		// 2 wrong (body)
		`
   ┌───┐
   │   ○
   │   │
   │
   │
   │
  ━┷━
`,
		// 3 wrong (left arm)
		`
   ┌───┐
   │   ○
   │  ╱│
   │
   │
   │
  ━┷━
`,
		// 4 wrong (right arm)
		`
   ┌───┐
   │   ○
   │  ╱│╲
   │
   │
   │
  ━┷━
`,
		// 5 wrong (left leg)
		`
   ┌───┐
   │   ○
   │  ╱│╲
   │   │
   │  ╱
   │
  ━┷━
`,
		// 6 wrong (right leg - game over)
		`
   ┌───┐
   │   ○
   │  ╱│╲
   │   │
   │  ╱ ╲
   │
  ━┷━
`,
	}

	if stage > 6 {
		stage = 6
	}

	return stages[stage]
}

// GetStats returns game statistics
func (g *HangmanGame) GetStats() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["word_length"] = len(g.Word)
	stats["max_wrong"] = g.MaxWrong
	stats["wrong_count"] = len(g.WrongGuesses)
	stats["total_players"] = len(g.Players)
	stats["state"] = g.State.String()
	stats["multiplayer"] = g.Multiplayer

	lettersRevealed := 0
	for _, ch := range g.Word {
		if g.GuessedLetters[ch] {
			lettersRevealed++
		}
	}
	stats["letters_revealed"] = lettersRevealed

	if !g.StartTime.IsZero() {
		if g.EndTime.IsZero() {
			stats["duration"] = time.Since(g.StartTime).String()
		} else {
			stats["duration"] = g.EndTime.Sub(g.StartTime).String()
		}
	}

	return stats
}

// Helper function
func toUpperRune(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}
