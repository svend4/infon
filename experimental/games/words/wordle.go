//go:build experimental

package words

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// WordleGame represents a Wordle game
type WordleGame struct {
	mu           sync.RWMutex
	ID           string
	Word         string
	WordLength   int
	MaxAttempts  int
	Attempts     []WordleAttempt
	Players      map[string]*WordlePlayer
	State        WordleState
	StartTime    time.Time
	EndTime      time.Time
	DailyMode    bool
	Dictionary   []string
}

// WordleState represents game state
type WordleState int

const (
	WordleWaiting WordleState = iota
	WordlePlaying
	WordleWon
	WordleLost
)

func (s WordleState) String() string {
	switch s {
	case WordleWaiting:
		return "Waiting"
	case WordlePlaying:
		return "Playing"
	case WordleWon:
		return "Won"
	case WordleLost:
		return "Lost"
	default:
		return "Unknown"
	}
}

// WordleAttempt represents a guess attempt
type WordleAttempt struct {
	PlayerID string
	Word     string
	Result   []LetterStatus
	Time     time.Time
}

// LetterStatus represents the status of a letter
type LetterStatus int

const (
	LetterWrong    LetterStatus = iota // Gray - not in word
	LetterMisplaced                     // Yellow - in word, wrong position
	LetterCorrect                       // Green - correct position
)

func (l LetterStatus) String() string {
	switch l {
	case LetterWrong:
		return "⬜"
	case LetterMisplaced:
		return "🟨"
	case LetterCorrect:
		return "🟩"
	default:
		return "❓"
	}
}

// WordlePlayer represents a player
type WordlePlayer struct {
	ID           string
	Name         string
	Attempts     int
	Won          bool
	WinTime      time.Time
	GamesPlayed  int
	GamesWon     int
	CurrentStreak int
	MaxStreak    int
}

// NewWordleGame creates a new Wordle game
func NewWordleGame(id string, word string) *WordleGame {
	word = strings.ToUpper(word)

	return &WordleGame{
		ID:          id,
		Word:        word,
		WordLength:  len(word),
		MaxAttempts: 6,
		Attempts:    make([]WordleAttempt, 0, 6),
		Players:     make(map[string]*WordlePlayer),
		State:       WordleWaiting,
		DailyMode:   false,
		Dictionary:  getDefaultDictionary(),
	}
}

// NewDailyWordleGame creates a daily challenge game
func NewDailyWordleGame(id string, dayNumber int) *WordleGame {
	// Use day number to pick a word deterministically
	dict := getDefaultDictionary()
	word := dict[dayNumber%len(dict)]

	game := NewWordleGame(id, word)
	game.DailyMode = true

	return game
}

// AddPlayer adds a player to the game
func (g *WordleGame) AddPlayer(id, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.Players[id]; exists {
		return errors.New("player already exists")
	}

	player := &WordlePlayer{
		ID:   id,
		Name: name,
	}

	g.Players[id] = player

	return nil
}

// Start starts the game
func (g *WordleGame) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != WordleWaiting {
		return errors.New("game already started")
	}

	if len(g.Players) == 0 {
		return errors.New("no players")
	}

	g.State = WordlePlaying
	g.StartTime = time.Now()

	return nil
}

// MakeGuess processes a guess
func (g *WordleGame) MakeGuess(playerID, guess string) (*WordleAttempt, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != WordlePlaying {
		return nil, errors.New("game not in playing state")
	}

	player, exists := g.Players[playerID]
	if !exists {
		return nil, errors.New("player not found")
	}

	if player.Won {
		return nil, errors.New("player already won")
	}

	if player.Attempts >= g.MaxAttempts {
		return nil, errors.New("max attempts reached")
	}

	guess = strings.ToUpper(guess)

	// Validate guess
	if len(guess) != g.WordLength {
		return nil, fmt.Errorf("guess must be %d letters", g.WordLength)
	}

	if !g.isValidWord(guess) {
		return nil, errors.New("not a valid word")
	}

	// Check guess
	result := g.checkGuess(guess)

	attempt := WordleAttempt{
		PlayerID: playerID,
		Word:     guess,
		Result:   result,
		Time:     time.Now(),
	}

	g.Attempts = append(g.Attempts, attempt)
	player.Attempts++

	// Check if won
	if guess == g.Word {
		player.Won = true
		player.WinTime = time.Now()
		player.GamesWon++
		player.CurrentStreak++
		if player.CurrentStreak > player.MaxStreak {
			player.MaxStreak = player.CurrentStreak
		}

		// Check if all players finished or won
		allFinished := true
		for _, p := range g.Players {
			if !p.Won && p.Attempts < g.MaxAttempts {
				allFinished = false
				break
			}
		}

		if allFinished {
			g.State = WordleWon
			g.EndTime = time.Now()
		}

		return &attempt, nil
	}

	// Check if lost
	if player.Attempts >= g.MaxAttempts {
		player.GamesPlayed++
		player.CurrentStreak = 0

		// Check if all players finished
		allFinished := true
		for _, p := range g.Players {
			if !p.Won && p.Attempts < g.MaxAttempts {
				allFinished = false
				break
			}
		}

		if allFinished {
			g.State = WordleLost
			g.EndTime = time.Now()
		}
	}

	return &attempt, nil
}

// checkGuess checks a guess against the target word
func (g *WordleGame) checkGuess(guess string) []LetterStatus {
	result := make([]LetterStatus, g.WordLength)
	wordLetters := []rune(g.Word)
	guessLetters := []rune(guess)

	// Count remaining letters in word (for yellow logic)
	letterCounts := make(map[rune]int)
	for _, ch := range wordLetters {
		letterCounts[ch]++
	}

	// First pass: mark correct positions (green)
	for i := 0; i < g.WordLength; i++ {
		if guessLetters[i] == wordLetters[i] {
			result[i] = LetterCorrect
			letterCounts[guessLetters[i]]--
		}
	}

	// Second pass: mark misplaced letters (yellow)
	for i := 0; i < g.WordLength; i++ {
		if result[i] == LetterCorrect {
			continue
		}

		if letterCounts[guessLetters[i]] > 0 {
			result[i] = LetterMisplaced
			letterCounts[guessLetters[i]]--
		} else {
			result[i] = LetterWrong
		}
	}

	return result
}

// isValidWord checks if a word is in the dictionary
func (g *WordleGame) isValidWord(word string) bool {
	// For now, accept any alphabetic word of correct length
	for i := 0; i < len(word); i++ {
		if word[i] < 'A' || word[i] > 'Z' {
			return false
		}
	}
	return true
}

// GetPlayerAttempts returns all attempts for a player
func (g *WordleGame) GetPlayerAttempts(playerID string) []WordleAttempt {
	g.mu.RLock()
	defer g.mu.RUnlock()

	attempts := make([]WordleAttempt, 0)
	for _, attempt := range g.Attempts {
		if attempt.PlayerID == playerID {
			attempts = append(attempts, attempt)
		}
	}

	return attempts
}

// GetScoreboard returns player statistics
func (g *WordleGame) GetScoreboard() []*WordlePlayer {
	g.mu.RLock()
	defer g.mu.RUnlock()

	players := make([]*WordlePlayer, 0, len(g.Players))
	for _, p := range g.Players {
		players = append(players, p)
	}

	// Sort by won (yes first), then by attempts (fewer is better)
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			if (!players[i].Won && players[j].Won) ||
				(players[i].Won == players[j].Won && players[i].Attempts > players[j].Attempts) {
				players[i], players[j] = players[j], players[i]
			}
		}
	}

	return players
}

// FormatAttempt formats a single attempt
func (a *WordleAttempt) FormatAttempt() string {
	var sb strings.Builder

	// Show the word with colored blocks
	for i := range a.Word {
		sb.WriteString(fmt.Sprintf("%s", a.Result[i].String()))
	}
	sb.WriteString(" ")
	sb.WriteString(a.Word)

	return sb.String()
}

// FormatBoard formats the entire board for a player
func (g *WordleGame) FormatBoard(playerID string) string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("   WORDLE GAME\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━\n\n")

	attempts := make([]WordleAttempt, 0)
	for _, attempt := range g.Attempts {
		if attempt.PlayerID == playerID {
			attempts = append(attempts, attempt)
		}
	}

	// Show attempts
	for i, attempt := range attempts {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, attempt.FormatAttempt()))
	}

	// Show remaining empty slots
	player := g.Players[playerID]
	for i := player.Attempts; i < g.MaxAttempts; i++ {
		sb.WriteString(fmt.Sprintf("%d. ", i+1))
		for j := 0; j < g.WordLength; j++ {
			sb.WriteString("⬛")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	if player.Won {
		sb.WriteString("🎉 You won!\n")
	} else if player.Attempts >= g.MaxAttempts {
		sb.WriteString(fmt.Sprintf("😢 Game over! The word was: %s\n", g.Word))
	} else {
		sb.WriteString(fmt.Sprintf("Attempts remaining: %d/%d\n", g.MaxAttempts-player.Attempts, g.MaxAttempts))
	}

	return sb.String()
}

// GetStats returns game statistics
func (g *WordleGame) GetStats() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["word_length"] = g.WordLength
	stats["max_attempts"] = g.MaxAttempts
	stats["total_players"] = len(g.Players)
	stats["state"] = g.State.String()

	winnersCount := 0
	for _, p := range g.Players {
		if p.Won {
			winnersCount++
		}
	}
	stats["winners"] = winnersCount

	if !g.StartTime.IsZero() {
		if g.EndTime.IsZero() {
			stats["duration"] = time.Since(g.StartTime).String()
		} else {
			stats["duration"] = g.EndTime.Sub(g.StartTime).String()
		}
	}

	return stats
}

// getDefaultDictionary returns a small word list
func getDefaultDictionary() []string {
	return []string{
		"PIANO", "GUITAR", "VIOLIN", "DRUMS", "FLUTE",
		"HOUSE", "BEACH", "MOUNTAIN", "FOREST", "RIVER",
		"APPLE", "BANANA", "ORANGE", "GRAPE", "LEMON",
		"TIGER", "ELEPHANT", "GIRAFFE", "ZEBRA", "LION",
		"HAPPY", "SMILE", "LAUGH", "DANCE", "SING",
		"ROBOT", "ALIEN", "SPACE", "PLANET", "STAR",
		"BOOK", "STORY", "MAGIC", "DREAM", "WISH",
		"CLOUD", "RAIN", "STORM", "WIND", "SNOW",
		"LIGHT", "DARK", "BRIGHT", "GLOW", "SHINE",
		"OCEAN", "WAVE", "CORAL", "SHELL", "PEARL",
	}
}
