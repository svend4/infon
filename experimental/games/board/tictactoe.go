//go:build experimental

package board

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// TicTacToe represents a tic-tac-toe game
type TicTacToe struct {
	mu       sync.RWMutex
	Board    [3][3]Mark
	Players  [2]*Player
	Turn     int           // 0 or 1
	State    GameState
	Winner   *Player
	Moves    []Move
	AIEnabled bool
	AIDifficulty int // 0=Random, 1=Easy, 2=Hard (Minimax)
}

// Mark represents a cell mark
type Mark string

const (
	MarkEmpty Mark = " "
	MarkX     Mark = "X"
	MarkO     Mark = "O"
)

// Player represents a player
type Player struct {
	ID   string
	Name string
	Mark Mark
	IsAI bool
}

// Move represents a move
type Move struct {
	Row    int
	Col    int
	Mark   Mark
	Player *Player
}

// GameState represents game state
type GameState int

const (
	StateSetup GameState = iota
	StatePlaying
	StateXWins
	StateOWins
	StateDraw
)

func (s GameState) String() string {
	switch s {
	case StateSetup:
		return "Setup"
	case StatePlaying:
		return "Playing"
	case StateXWins:
		return "X Wins"
	case StateOWins:
		return "O Wins"
	case StateDraw:
		return "Draw"
	default:
		return "Unknown"
	}
}

// NewTicTacToe creates a new tic-tac-toe game
func NewTicTacToe() *TicTacToe {
	game := &TicTacToe{
		State:        StateSetup,
		Moves:        make([]Move, 0, 9),
		AIEnabled:    false,
		AIDifficulty: 2, // Hard by default
	}

	// Initialize empty board
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			game.Board[i][j] = MarkEmpty
		}
	}

	return game
}

// SetPlayers sets the two players
func (g *TicTacToe) SetPlayers(player1, player2 *Player) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != StateSetup {
		return errors.New("game already started")
	}

	player1.Mark = MarkX
	player2.Mark = MarkO

	g.Players[0] = player1
	g.Players[1] = player2

	return nil
}

// SetupVsAI sets up game against AI
func (g *TicTacToe) SetupVsAI(humanID, humanName string, difficulty int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	human := &Player{
		ID:   humanID,
		Name: humanName,
		Mark: MarkX,
		IsAI: false,
	}

	ai := &Player{
		ID:   "ai",
		Name: "AI",
		Mark: MarkO,
		IsAI: true,
	}

	g.Players[0] = human
	g.Players[1] = ai
	g.AIEnabled = true
	g.AIDifficulty = difficulty
}

// Start starts the game
func (g *TicTacToe) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Players[0] == nil || g.Players[1] == nil {
		return errors.New("players not set")
	}

	g.State = StatePlaying
	g.Turn = 0

	return nil
}

// MakeMove makes a move at the given position
func (g *TicTacToe) MakeMove(row, col int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != StatePlaying {
		return errors.New("game not in playing state")
	}

	if row < 0 || row > 2 || col < 0 || col > 2 {
		return errors.New("invalid position")
	}

	if g.Board[row][col] != MarkEmpty {
		return errors.New("position already taken")
	}

	currentPlayer := g.Players[g.Turn]
	g.Board[row][col] = currentPlayer.Mark

	move := Move{
		Row:    row,
		Col:    col,
		Mark:   currentPlayer.Mark,
		Player: currentPlayer,
	}
	g.Moves = append(g.Moves, move)

	// Check for win or draw
	if g.checkWinLocked() {
		g.Winner = currentPlayer
		if currentPlayer.Mark == MarkX {
			g.State = StateXWins
		} else {
			g.State = StateOWins
		}
		return nil
	}

	if g.checkDrawLocked() {
		g.State = StateDraw
		return nil
	}

	// Switch turn
	g.Turn = 1 - g.Turn

	return nil
}

// MakeMoveByPlayer makes a move for a specific player
func (g *TicTacToe) MakeMoveByPlayer(playerID string, row, col int) error {
	g.mu.RLock()
	currentPlayer := g.Players[g.Turn]
	g.mu.RUnlock()

	if currentPlayer.ID != playerID {
		return errors.New("not your turn")
	}

	return g.MakeMove(row, col)
}

// GetAIMove gets the best move for AI
func (g *TicTacToe) GetAIMove() (row, col int, err error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.AIEnabled || !g.Players[g.Turn].IsAI {
		return 0, 0, errors.New("not AI's turn")
	}

	switch g.AIDifficulty {
	case 0: // Random
		return g.getRandomMove()
	case 1: // Easy (blocks obvious wins, but makes mistakes)
		// 70% chance of good move, 30% random
		if randInt(0, 100) < 70 {
			return g.getMinimaxMove()
		}
		return g.getRandomMove()
	case 2: // Hard (perfect minimax)
		return g.getMinimaxMove()
	default:
		return g.getMinimaxMove()
	}
}

// getRandomMove returns a random valid move
func (g *TicTacToe) getRandomMove() (int, int, error) {
	empty := make([][2]int, 0, 9)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if g.Board[i][j] == MarkEmpty {
				empty = append(empty, [2]int{i, j})
			}
		}
	}

	if len(empty) == 0 {
		return 0, 0, errors.New("no valid moves")
	}

	choice := empty[randInt(0, len(empty))]
	return choice[0], choice[1], nil
}

// getMinimaxMove returns the best move using minimax algorithm
func (g *TicTacToe) getMinimaxMove() (int, int, error) {
	bestScore := -1000
	bestRow, bestCol := -1, -1

	aiMark := g.Players[g.Turn].Mark
	humanMark := MarkX
	if aiMark == MarkX {
		humanMark = MarkO
	}

	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if g.Board[i][j] == MarkEmpty {
				// Try move
				g.Board[i][j] = aiMark
				score := g.minimax(false, aiMark, humanMark, -1000, 1000)
				g.Board[i][j] = MarkEmpty

				if score > bestScore {
					bestScore = score
					bestRow = i
					bestCol = j
				}
			}
		}
	}

	if bestRow == -1 {
		return 0, 0, errors.New("no valid moves")
	}

	return bestRow, bestCol, nil
}

// minimax algorithm with alpha-beta pruning
func (g *TicTacToe) minimax(isMax bool, aiMark, humanMark Mark, alpha, beta int) int {
	// Check terminal states
	if g.checkWinForMarkLocked(aiMark) {
		return 10
	}
	if g.checkWinForMarkLocked(humanMark) {
		return -10
	}
	if g.checkDrawLocked() {
		return 0
	}

	if isMax {
		bestScore := -1000
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if g.Board[i][j] == MarkEmpty {
					g.Board[i][j] = aiMark
					score := g.minimax(false, aiMark, humanMark, alpha, beta)
					g.Board[i][j] = MarkEmpty
					bestScore = max(bestScore, score)
					alpha = max(alpha, score)
					if beta <= alpha {
						break
					}
				}
			}
		}
		return bestScore
	} else {
		bestScore := 1000
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if g.Board[i][j] == MarkEmpty {
					g.Board[i][j] = humanMark
					score := g.minimax(true, aiMark, humanMark, alpha, beta)
					g.Board[i][j] = MarkEmpty
					bestScore = min(bestScore, score)
					beta = min(beta, score)
					if beta <= alpha {
						break
					}
				}
			}
		}
		return bestScore
	}
}

// checkWinLocked checks if current board has a winner (must hold lock)
func (g *TicTacToe) checkWinLocked() bool {
	// Check rows
	for i := 0; i < 3; i++ {
		if g.Board[i][0] != MarkEmpty &&
			g.Board[i][0] == g.Board[i][1] &&
			g.Board[i][1] == g.Board[i][2] {
			return true
		}
	}

	// Check columns
	for j := 0; j < 3; j++ {
		if g.Board[0][j] != MarkEmpty &&
			g.Board[0][j] == g.Board[1][j] &&
			g.Board[1][j] == g.Board[2][j] {
			return true
		}
	}

	// Check diagonals
	if g.Board[0][0] != MarkEmpty &&
		g.Board[0][0] == g.Board[1][1] &&
		g.Board[1][1] == g.Board[2][2] {
		return true
	}

	if g.Board[0][2] != MarkEmpty &&
		g.Board[0][2] == g.Board[1][1] &&
		g.Board[1][1] == g.Board[2][0] {
		return true
	}

	return false
}

// checkWinForMarkLocked checks if a specific mark has won
func (g *TicTacToe) checkWinForMarkLocked(mark Mark) bool {
	// Check rows
	for i := 0; i < 3; i++ {
		if g.Board[i][0] == mark && g.Board[i][1] == mark && g.Board[i][2] == mark {
			return true
		}
	}

	// Check columns
	for j := 0; j < 3; j++ {
		if g.Board[0][j] == mark && g.Board[1][j] == mark && g.Board[2][j] == mark {
			return true
		}
	}

	// Check diagonals
	if g.Board[0][0] == mark && g.Board[1][1] == mark && g.Board[2][2] == mark {
		return true
	}

	if g.Board[0][2] == mark && g.Board[1][1] == mark && g.Board[2][0] == mark {
		return true
	}

	return false
}

// checkDrawLocked checks if the game is a draw (must hold lock)
func (g *TicTacToe) checkDrawLocked() bool {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if g.Board[i][j] == MarkEmpty {
				return false
			}
		}
	}
	return true
}

// Reset resets the game
func (g *TicTacToe) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			g.Board[i][j] = MarkEmpty
		}
	}

	g.State = StatePlaying
	g.Turn = 0
	g.Winner = nil
	g.Moves = g.Moves[:0]
}

// FormatBoard returns a string representation of the board
func (g *TicTacToe) FormatBoard() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  0   1   2\n")

	for i := 0; i < 3; i++ {
		sb.WriteString(fmt.Sprintf("%d ", i))
		for j := 0; j < 3; j++ {
			sb.WriteString(string(g.Board[i][j]))
			if j < 2 {
				sb.WriteString(" │ ")
			}
		}
		sb.WriteString("\n")
		if i < 2 {
			sb.WriteString(" ───┼───┼───\n")
		}
	}

	return sb.String()
}

// GetCurrentPlayer returns the current player
func (g *TicTacToe) GetCurrentPlayer() *Player {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.Players[g.Turn]
}

// GetGameInfo returns game information
func (g *TicTacToe) GetGameInfo() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString(g.FormatBoard())
	sb.WriteString("\n")

	if g.State == StatePlaying {
		currentPlayer := g.Players[g.Turn]
		sb.WriteString(fmt.Sprintf("Current turn: %s (%s)\n", currentPlayer.Name, currentPlayer.Mark))
	} else {
		sb.WriteString(fmt.Sprintf("Game State: %s\n", g.State.String()))
		if g.Winner != nil {
			sb.WriteString(fmt.Sprintf("Winner: %s\n", g.Winner.Name))
		}
	}

	sb.WriteString(fmt.Sprintf("Moves played: %d\n", len(g.Moves)))

	return sb.String()
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Simple random number generator (for AI)
var randSeed = 12345

func randInt(min, max int) int {
	// Simple LCG (Linear Congruential Generator)
	randSeed = (randSeed*1103515245 + 12345) & 0x7fffffff
	return min + (randSeed % (max - min))
}
