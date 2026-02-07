//go:build experimental

package trivia

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// TriviaGame represents a trivia quiz game that can be played during calls
type TriviaGame struct {
	mu            sync.RWMutex
	ID            string
	Questions     []*Question
	Players       map[string]*Player
	CurrentQ      int
	Scores        map[string]int
	TimeLimit     time.Duration
	State         GameState
	StartTime     time.Time
	EndTime       time.Time
	QuestionStart time.Time
	Callbacks     *Callbacks
}

// GameState represents the current state of the game
type GameState int

const (
	StateWaiting GameState = iota // Waiting for players
	StatePlaying                  // Game in progress
	StateQuestion                 // Question being answered
	StateRevealing                // Revealing answer
	StateFinished                 // Game finished
)

func (s GameState) String() string {
	switch s {
	case StateWaiting:
		return "Waiting"
	case StatePlaying:
		return "Playing"
	case StateQuestion:
		return "Question"
	case StateRevealing:
		return "Revealing"
	case StateFinished:
		return "Finished"
	default:
		return "Unknown"
	}
}

// Question represents a trivia question
type Question struct {
	ID         string   `json:"id"`
	Text       string   `json:"text"`
	Options    []string `json:"options"`
	Correct    int      `json:"correct"`
	Category   string   `json:"category"`
	Difficulty int      `json:"difficulty"` // 1-5
	Points     int      `json:"points"`
}

// Player represents a player in the game
type Player struct {
	ID          string
	Name        string
	Score       int
	Answers     []Answer
	JoinedAt    time.Time
	LastAnswerAt time.Time
}

// Answer represents a player's answer to a question
type Answer struct {
	QuestionID string
	Selected   int
	Correct    bool
	TimeSpent  time.Duration
	Points     int
}

// Callbacks for game events
type Callbacks struct {
	OnGameStart      func(game *TriviaGame)
	OnQuestionAsked  func(game *TriviaGame, q *Question)
	OnAnswer         func(game *TriviaGame, player *Player, answer *Answer)
	OnQuestionEnd    func(game *TriviaGame, q *Question, correctAnswer int)
	OnGameEnd        func(game *TriviaGame, winner *Player)
	OnScoreUpdate    func(game *TriviaGame)
}

// NewTriviaGame creates a new trivia game
func NewTriviaGame(id string, timeLimit time.Duration) *TriviaGame {
	return &TriviaGame{
		ID:        id,
		Questions: make([]*Question, 0),
		Players:   make(map[string]*Player),
		Scores:    make(map[string]int),
		TimeLimit: timeLimit,
		State:     StateWaiting,
		Callbacks: &Callbacks{},
	}
}

// AddQuestion adds a question to the game
func (g *TriviaGame) AddQuestion(q *Question) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if q.Points == 0 {
		// Default points based on difficulty
		q.Points = q.Difficulty * 100
	}

	g.Questions = append(g.Questions, q)
}

// AddQuestions adds multiple questions
func (g *TriviaGame) AddQuestions(questions []*Question) {
	for _, q := range questions {
		g.AddQuestion(q)
	}
}

// LoadQuestionsFromJSON loads questions from JSON data
func (g *TriviaGame) LoadQuestionsFromJSON(data []byte) error {
	var questions []*Question
	if err := json.Unmarshal(data, &questions); err != nil {
		return fmt.Errorf("failed to parse questions: %w", err)
	}

	g.AddQuestions(questions)
	return nil
}

// ShuffleQuestions randomizes question order
func (g *TriviaGame) ShuffleQuestions() {
	g.mu.Lock()
	defer g.mu.Unlock()

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(g.Questions), func(i, j int) {
		g.Questions[i], g.Questions[j] = g.Questions[j], g.Questions[i]
	})
}

// AddPlayer adds a player to the game
func (g *TriviaGame) AddPlayer(id, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != StateWaiting {
		return errors.New("cannot join: game already started")
	}

	if _, exists := g.Players[id]; exists {
		return errors.New("player already joined")
	}

	player := &Player{
		ID:       id,
		Name:     name,
		Score:    0,
		Answers:  make([]Answer, 0),
		JoinedAt: time.Now(),
	}

	g.Players[id] = player
	g.Scores[id] = 0

	return nil
}

// RemovePlayer removes a player from the game
func (g *TriviaGame) RemovePlayer(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.Players, id)
	delete(g.Scores, id)
}

// Start starts the game
func (g *TriviaGame) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != StateWaiting {
		return errors.New("game already started")
	}

	if len(g.Players) == 0 {
		return errors.New("no players")
	}

	if len(g.Questions) == 0 {
		return errors.New("no questions")
	}

	g.State = StatePlaying
	g.StartTime = time.Now()
	g.CurrentQ = -1

	if g.Callbacks.OnGameStart != nil {
		go g.Callbacks.OnGameStart(g)
	}

	return nil
}

// NextQuestion moves to the next question
func (g *TriviaGame) NextQuestion() (*Question, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != StatePlaying {
		return nil, errors.New("game not in playing state")
	}

	g.CurrentQ++

	if g.CurrentQ >= len(g.Questions) {
		// Game finished
		g.State = StateFinished
		g.EndTime = time.Now()

		winner := g.getWinnerLocked()
		if g.Callbacks.OnGameEnd != nil {
			go g.Callbacks.OnGameEnd(g, winner)
		}

		return nil, errors.New("no more questions")
	}

	g.State = StateQuestion
	g.QuestionStart = time.Now()

	question := g.Questions[g.CurrentQ]

	if g.Callbacks.OnQuestionAsked != nil {
		go g.Callbacks.OnQuestionAsked(g, question)
	}

	// Auto-reveal after time limit
	if g.TimeLimit > 0 {
		go g.autoRevealAfterTimeout()
	}

	return question, nil
}

// autoRevealAfterTimeout automatically reveals the answer after timeout
func (g *TriviaGame) autoRevealAfterTimeout() {
	time.Sleep(g.TimeLimit)

	g.mu.Lock()
	if g.State == StateQuestion {
		g.mu.Unlock()
		g.RevealAnswer()
	} else {
		g.mu.Unlock()
	}
}

// SubmitAnswer submits a player's answer
func (g *TriviaGame) SubmitAnswer(playerID string, selected int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != StateQuestion {
		return errors.New("no active question")
	}

	player, exists := g.Players[playerID]
	if !exists {
		return errors.New("player not found")
	}

	// Check if already answered this question
	for _, ans := range player.Answers {
		if ans.QuestionID == g.Questions[g.CurrentQ].ID {
			return errors.New("already answered this question")
		}
	}

	question := g.Questions[g.CurrentQ]

	if selected < 0 || selected >= len(question.Options) {
		return errors.New("invalid answer option")
	}

	timeSpent := time.Since(g.QuestionStart)
	correct := selected == question.Correct

	// Calculate points (bonus for quick answers)
	points := 0
	if correct {
		points = question.Points

		// Time bonus (up to 50% more points)
		if g.TimeLimit > 0 {
			timeBonus := 1.0 - (float64(timeSpent) / float64(g.TimeLimit))
			if timeBonus > 0 {
				points += int(float64(question.Points) * 0.5 * timeBonus)
			}
		}

		player.Score += points
		g.Scores[playerID] = player.Score
	}

	answer := Answer{
		QuestionID: question.ID,
		Selected:   selected,
		Correct:    correct,
		TimeSpent:  timeSpent,
		Points:     points,
	}

	player.Answers = append(player.Answers, answer)
	player.LastAnswerAt = time.Now()

	if g.Callbacks.OnAnswer != nil {
		go g.Callbacks.OnAnswer(g, player, &answer)
	}

	if g.Callbacks.OnScoreUpdate != nil {
		go g.Callbacks.OnScoreUpdate(g)
	}

	return nil
}

// RevealAnswer reveals the correct answer and moves to revealing state
func (g *TriviaGame) RevealAnswer() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != StateQuestion {
		return errors.New("no active question")
	}

	g.State = StateRevealing
	question := g.Questions[g.CurrentQ]

	if g.Callbacks.OnQuestionEnd != nil {
		go g.Callbacks.OnQuestionEnd(g, question, question.Correct)
	}

	return nil
}

// ContinueToNext continues to the next question after revealing
func (g *TriviaGame) ContinueToNext() error {
	g.mu.Lock()
	if g.State != StateRevealing {
		g.mu.Unlock()
		return errors.New("not in revealing state")
	}

	g.State = StatePlaying
	g.mu.Unlock()

	_, err := g.NextQuestion()
	return err
}

// GetCurrentQuestion returns the current question
func (g *TriviaGame) GetCurrentQuestion() (*Question, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.CurrentQ < 0 || g.CurrentQ >= len(g.Questions) {
		return nil, errors.New("no current question")
	}

	return g.Questions[g.CurrentQ], nil
}

// GetScoreboard returns the current scoreboard
func (g *TriviaGame) GetScoreboard() []*Player {
	g.mu.RLock()
	defer g.mu.RUnlock()

	players := make([]*Player, 0, len(g.Players))
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

// GetWinner returns the winning player
func (g *TriviaGame) GetWinner() *Player {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.getWinnerLocked()
}

func (g *TriviaGame) getWinnerLocked() *Player {
	var winner *Player
	maxScore := -1

	for _, player := range g.Players {
		if player.Score > maxScore {
			maxScore = player.Score
			winner = player
		}
	}

	return winner
}

// GetStats returns game statistics
func (g *TriviaGame) GetStats() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_questions"] = len(g.Questions)
	stats["current_question"] = g.CurrentQ + 1
	stats["total_players"] = len(g.Players)
	stats["state"] = g.State.String()

	if !g.StartTime.IsZero() {
		if g.EndTime.IsZero() {
			stats["duration"] = time.Since(g.StartTime).String()
		} else {
			stats["duration"] = g.EndTime.Sub(g.StartTime).String()
		}
	}

	return stats
}

// FormatQuestion formats a question for display
func (q *Question) FormatQuestion() string {
	result := fmt.Sprintf("❓ %s\n\n", q.Text)
	result += fmt.Sprintf("Category: %s | Difficulty: %d/5 | Points: %d\n\n", q.Category, q.Difficulty, q.Points)

	for i, option := range q.Options {
		result += fmt.Sprintf("%d) %s\n", i+1, option)
	}

	return result
}

// FormatScoreboard formats the scoreboard for display
func (g *TriviaGame) FormatScoreboard() string {
	scoreboard := g.GetScoreboard()

	result := "🏆 SCOREBOARD 🏆\n"
	result += "━━━━━━━━━━━━━━━━━━━━━━\n"

	for i, player := range scoreboard {
		medal := ""
		switch i {
		case 0:
			medal = "🥇"
		case 1:
			medal = "🥈"
		case 2:
			medal = "🥉"
		default:
			medal = fmt.Sprintf("%2d.", i+1)
		}

		result += fmt.Sprintf("%s %-20s %6d pts\n", medal, player.Name, player.Score)
	}

	result += "━━━━━━━━━━━━━━━━━━━━━━\n"

	return result
}
