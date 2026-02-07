//go:build experimental

package board

import (
	"strings"
	"testing"
)

func TestNewTicTacToe(t *testing.T) {
	game := NewTicTacToe()

	if game.State != StateSetup {
		t.Errorf("Expected state Setup, got %v", game.State)
	}

	// Check empty board
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if game.Board[i][j] != MarkEmpty {
				t.Errorf("Board[%d][%d] should be empty", i, j)
			}
		}
	}
}

func TestSetPlayers(t *testing.T) {
	game := NewTicTacToe()

	p1 := &Player{ID: "p1", Name: "Alice"}
	p2 := &Player{ID: "p2", Name: "Bob"}

	err := game.SetPlayers(p1, p2)
	if err != nil {
		t.Fatalf("Failed to set players: %v", err)
	}

	if game.Players[0].Mark != MarkX {
		t.Errorf("Player 1 should have mark X, got %v", game.Players[0].Mark)
	}

	if game.Players[1].Mark != MarkO {
		t.Errorf("Player 2 should have mark O, got %v", game.Players[1].Mark)
	}
}

func TestSetupVsAI(t *testing.T) {
	game := NewTicTacToe()

	game.SetupVsAI("human1", "Alice", 2)

	if !game.AIEnabled {
		t.Error("AI should be enabled")
	}

	if game.Players[0].IsAI {
		t.Error("Player 0 should be human")
	}

	if !game.Players[1].IsAI {
		t.Error("Player 1 should be AI")
	}

	if game.AIDifficulty != 2 {
		t.Errorf("Expected difficulty 2, got %d", game.AIDifficulty)
	}
}

func TestStart(t *testing.T) {
	game := NewTicTacToe()

	// Try starting without players
	err := game.Start()
	if err == nil {
		t.Error("Expected error when starting without players")
	}

	// Set players and start
	p1 := &Player{ID: "p1", Name: "Alice"}
	p2 := &Player{ID: "p2", Name: "Bob"}
	game.SetPlayers(p1, p2)

	err = game.Start()
	if err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	if game.State != StatePlaying {
		t.Errorf("Expected state Playing, got %v", game.State)
	}

	if game.Turn != 0 {
		t.Errorf("Expected turn 0, got %d", game.Turn)
	}
}

func TestMakeMove(t *testing.T) {
	game := setupGame()

	err := game.MakeMove(0, 0)
	if err != nil {
		t.Fatalf("Failed to make move: %v", err)
	}

	if game.Board[0][0] != MarkX {
		t.Errorf("Expected X at [0][0], got %v", game.Board[0][0])
	}

	if game.Turn != 1 {
		t.Errorf("Expected turn 1, got %d", game.Turn)
	}

	if len(game.Moves) != 1 {
		t.Errorf("Expected 1 move, got %d", len(game.Moves))
	}
}

func TestMakeMoveInvalidPosition(t *testing.T) {
	game := setupGame()

	// Out of bounds
	err := game.MakeMove(-1, 0)
	if err == nil {
		t.Error("Expected error for out of bounds")
	}

	err = game.MakeMove(3, 0)
	if err == nil {
		t.Error("Expected error for out of bounds")
	}
}

func TestMakeMoveAlreadyTaken(t *testing.T) {
	game := setupGame()

	game.MakeMove(0, 0)

	err := game.MakeMove(0, 0)
	if err == nil {
		t.Error("Expected error for already taken position")
	}
}

func TestMakeMoveByPlayer(t *testing.T) {
	game := setupGame()

	// Player 1's turn
	err := game.MakeMoveByPlayer("p1", 0, 0)
	if err != nil {
		t.Fatalf("Failed to make move: %v", err)
	}

	// Try player 1 again (not their turn)
	err = game.MakeMoveByPlayer("p1", 0, 1)
	if err == nil {
		t.Error("Expected error when not player's turn")
	}

	// Player 2's turn
	err = game.MakeMoveByPlayer("p2", 0, 1)
	if err != nil {
		t.Fatalf("Failed to make move: %v", err)
	}
}

func TestWinRow(t *testing.T) {
	game := setupGame()

	// X wins top row
	game.MakeMove(0, 0) // X
	game.MakeMove(1, 0) // O
	game.MakeMove(0, 1) // X
	game.MakeMove(1, 1) // O
	game.MakeMove(0, 2) // X - wins!

	if game.State != StateXWins {
		t.Errorf("Expected state XWins, got %v", game.State)
	}

	if game.Winner == nil || game.Winner.Mark != MarkX {
		t.Error("Expected X to win")
	}
}

func TestWinColumn(t *testing.T) {
	game := setupGame()

	// O wins middle column
	game.MakeMove(0, 0) // X
	game.MakeMove(0, 1) // O
	game.MakeMove(1, 0) // X
	game.MakeMove(1, 1) // O
	game.MakeMove(2, 2) // X
	game.MakeMove(2, 1) // O - wins!

	if game.State != StateOWins {
		t.Errorf("Expected state OWins, got %v", game.State)
	}
}

func TestWinDiagonal(t *testing.T) {
	game := setupGame()

	// X wins main diagonal
	game.MakeMove(0, 0) // X
	game.MakeMove(0, 1) // O
	game.MakeMove(1, 1) // X
	game.MakeMove(0, 2) // O
	game.MakeMove(2, 2) // X - wins!

	if game.State != StateXWins {
		t.Errorf("Expected state XWins, got %v", game.State)
	}
}

func TestWinAntiDiagonal(t *testing.T) {
	game := setupGame()

	// O wins anti-diagonal
	game.MakeMove(0, 0) // X
	game.MakeMove(0, 2) // O
	game.MakeMove(0, 1) // X
	game.MakeMove(1, 1) // O
	game.MakeMove(1, 0) // X
	game.MakeMove(2, 0) // O - wins!

	if game.State != StateOWins {
		t.Errorf("Expected state OWins, got %v", game.State)
	}
}

func TestDraw(t *testing.T) {
	game := setupGame()

	// Fill board without winner
	game.MakeMove(0, 0) // X
	game.MakeMove(0, 1) // O
	game.MakeMove(0, 2) // X
	game.MakeMove(1, 0) // O
	game.MakeMove(1, 1) // X
	game.MakeMove(2, 2) // O
	game.MakeMove(1, 2) // X
	game.MakeMove(2, 0) // O
	game.MakeMove(2, 1) // X - draw

	if game.State != StateDraw {
		t.Errorf("Expected state Draw, got %v", game.State)
	}
}

func TestReset(t *testing.T) {
	game := setupGame()

	game.MakeMove(0, 0)
	game.MakeMove(1, 1)
	game.MakeMove(2, 2)

	game.Reset()

	// Check board is empty
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if game.Board[i][j] != MarkEmpty {
				t.Errorf("Board[%d][%d] should be empty after reset", i, j)
			}
		}
	}

	if game.State != StatePlaying {
		t.Errorf("Expected state Playing after reset, got %v", game.State)
	}

	if game.Turn != 0 {
		t.Errorf("Expected turn 0 after reset, got %d", game.Turn)
	}

	if len(game.Moves) != 0 {
		t.Errorf("Expected 0 moves after reset, got %d", len(game.Moves))
	}
}

func TestFormatBoard(t *testing.T) {
	game := setupGame()

	game.MakeMove(0, 0) // X
	game.MakeMove(1, 1) // O

	formatted := game.FormatBoard()

	if !strings.Contains(formatted, "X") {
		t.Error("Formatted board should contain X")
	}

	if !strings.Contains(formatted, "O") {
		t.Error("Formatted board should contain O")
	}
}

func TestGetCurrentPlayer(t *testing.T) {
	game := setupGame()

	current := game.GetCurrentPlayer()
	if current.ID != "p1" {
		t.Errorf("Expected player p1, got %s", current.ID)
	}

	game.MakeMove(0, 0)

	current = game.GetCurrentPlayer()
	if current.ID != "p2" {
		t.Errorf("Expected player p2 after move, got %s", current.ID)
	}
}

func TestGetGameInfo(t *testing.T) {
	game := setupGame()

	info := game.GetGameInfo()

	if !strings.Contains(info, "Current turn") {
		t.Error("Game info should contain current turn")
	}

	if !strings.Contains(info, "Moves played: 0") {
		t.Error("Game info should show 0 moves")
	}
}

func TestAIRandomMove(t *testing.T) {
	game := NewTicTacToe()
	game.SetupVsAI("human", "Alice", 0) // Random AI
	game.Start()

	// Human makes first move
	game.MakeMove(0, 0)

	// Now it's AI's turn
	row, col, err := game.GetAIMove()
	if err != nil {
		t.Fatalf("AI failed to get move: %v", err)
	}

	if row < 0 || row > 2 || col < 0 || col > 2 {
		t.Errorf("AI returned invalid position: [%d][%d]", row, col)
	}
}

func TestAIMinimaxMove(t *testing.T) {
	game := NewTicTacToe()
	game.SetupVsAI("human", "Alice", 2) // Hard AI
	game.Start()

	// Human (X) makes first move
	game.MakeMove(0, 0)

	// AI should make a move
	row, col, err := game.GetAIMove()
	if err != nil {
		t.Fatalf("AI failed to get move: %v", err)
	}

	// Make AI move
	err = game.MakeMove(row, col)
	if err != nil {
		t.Fatalf("Failed to make AI move: %v", err)
	}

	if game.Board[row][col] != MarkO {
		t.Error("AI move should place O")
	}
}

func TestAIBlocksWin(t *testing.T) {
	game := NewTicTacToe()
	game.SetupVsAI("human", "Alice", 2) // Hard AI
	game.Start()

	// Setup: X has two in a row, AI should block
	game.MakeMove(0, 0) // X
	game.MakeMove(1, 1) // AI - O (center)
	game.MakeMove(0, 1) // X (two in top row)

	// AI's turn - should block [0][2]
	row, col, err := game.GetAIMove()
	if err != nil {
		t.Fatalf("AI failed: %v", err)
	}

	// AI should block the winning move
	game.MakeMove(row, col)

	if game.Board[0][2] != MarkO {
		t.Error("AI should have blocked [0][2]")
	}
}

func TestAITakesWin(t *testing.T) {
	game := NewTicTacToe()
	game.SetupVsAI("human", "Alice", 2)
	game.Start()

	// Setup: AI has two in a row
	game.MakeMove(0, 0) // X
	game.MakeMove(1, 0) // O
	game.MakeMove(2, 2) // X
	game.MakeMove(1, 1) // O (two in middle column)
	game.MakeMove(0, 2) // X

	// AI should win at [1][2]
	row, col, err := game.GetAIMove()
	if err != nil {
		t.Fatalf("AI failed: %v", err)
	}

	game.MakeMove(row, col)

	// AI should take the win, but the column check above was wrong
	// Let me verify AI has two in a column or somewhere
	// Actually with the current board:
	// X . O
	// O O .
	// . . X
	// AI should complete middle row at [1][2]
	if game.State != StateOWins {
		t.Errorf("AI should have won, state is %v", game.State)
	}
}

func TestAIPerfectPlay(t *testing.T) {
	// AI playing perfectly should never lose
	for gameNum := 0; gameNum < 10; gameNum++ {
		game := NewTicTacToe()
		game.SetupVsAI("human", "Human", 2) // Hard AI
		game.Start()

		// Play game with semi-random human moves
		for game.State == StatePlaying {
			if game.GetCurrentPlayer().IsAI {
				row, col, _ := game.GetAIMove()
				game.MakeMove(row, col)
			} else {
				// Human makes random move
				moved := false
				for i := 0; i < 3 && !moved; i++ {
					for j := 0; j < 3 && !moved; j++ {
						if game.Board[i][j] == MarkEmpty {
							game.MakeMove(i, j)
							moved = true
						}
					}
				}
			}
		}

		// AI should never lose (only win or draw)
		if game.State == StateXWins {
			t.Error("Perfect AI should never lose")
		}
	}
}

func TestMinMaxAlphaBeta(t *testing.T) {
	game := setupGame()

	// Test that minimax with alpha-beta pruning works
	game.Board[0][0] = MarkX
	game.Board[0][1] = MarkO
	game.Board[1][1] = MarkX

	score := game.minimax(true, MarkO, MarkX, -1000, 1000)

	// Score should be bounded
	if score < -10 || score > 10 {
		t.Errorf("Minimax score out of bounds: %d", score)
	}
}

func TestCheckWinForMark(t *testing.T) {
	game := NewTicTacToe()

	game.Board[0][0] = MarkX
	game.Board[0][1] = MarkX
	game.Board[0][2] = MarkX

	game.mu.RLock()
	hasWon := game.checkWinForMarkLocked(MarkX)
	game.mu.RUnlock()

	if !hasWon {
		t.Error("X should have won")
	}

	game.mu.RLock()
	hasWonO := game.checkWinForMarkLocked(MarkO)
	game.mu.RUnlock()

	if hasWonO {
		t.Error("O should not have won")
	}
}

func TestStateToString(t *testing.T) {
	states := []GameState{StateSetup, StatePlaying, StateXWins, StateOWins, StateDraw}
	expected := []string{"Setup", "Playing", "X Wins", "O Wins", "Draw"}

	for i, state := range states {
		if state.String() != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], state.String())
		}
	}
}

// Helper function to setup a basic game
func setupGame() *TicTacToe {
	game := NewTicTacToe()
	p1 := &Player{ID: "p1", Name: "Alice"}
	p2 := &Player{ID: "p2", Name: "Bob"}
	game.SetPlayers(p1, p2)
	game.Start()
	return game
}

// Benchmark tests
func BenchmarkMakeMove(b *testing.B) {
	game := setupGame()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game.Reset()
		game.MakeMove(0, 0)
		game.MakeMove(1, 1)
		game.MakeMove(2, 2)
	}
}

func BenchmarkAIMinimaxMove(b *testing.B) {
	game := NewTicTacToe()
	game.SetupVsAI("human", "Alice", 2)
	game.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game.Reset()
		game.Start()
		game.GetAIMove()
	}
}

func BenchmarkFullGameWithAI(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		game := NewTicTacToe()
		game.SetupVsAI("human", "Human", 2)
		game.Start()

		for game.State == StatePlaying {
			if game.GetCurrentPlayer().IsAI {
				row, col, _ := game.GetAIMove()
				game.MakeMove(row, col)
			} else {
				// Random move
				for i := 0; i < 3; i++ {
					for j := 0; j < 3; j++ {
						if game.Board[i][j] == MarkEmpty {
							game.MakeMove(i, j)
							goto next
						}
					}
				}
			next:
			}
		}
	}
}
