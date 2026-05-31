package brain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- tic-tac-toe reference policy (minimax) ---

func ttRequest(t *testing.T, board [3][3]string, you string) Response {
	t.Helper()
	raw, err := json.Marshal(TicTacToeState{Board: board, Turn: you, You: you})
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	return Reference(Request{Kind: "move", Game: "tictactoe", State: raw})
}

func TestReferenceTakesWinningMove(t *testing.T) {
	// X has two in the top row; the only winning move is (0,2).
	board := [3][3]string{
		{"X", "X", ""},
		{"O", "O", ""},
		{"", "", ""},
	}
	resp := ttRequest(t, board, "X")
	if resp.Move == nil {
		t.Fatal("no move returned")
	}
	if resp.Move.Row != 0 || resp.Move.Col != 2 {
		t.Errorf("winning move = (%d,%d), want (0,2)", resp.Move.Row, resp.Move.Col)
	}
}

func TestReferenceBlocksOpponentWin(t *testing.T) {
	// O threatens to complete the top row at (0,2); X must block there.
	board := [3][3]string{
		{"O", "O", ""},
		{"X", "", ""},
		{"", "", ""},
	}
	resp := ttRequest(t, board, "X")
	if resp.Move == nil {
		t.Fatal("no move returned")
	}
	if resp.Move.Row != 0 || resp.Move.Col != 2 {
		t.Errorf("block move = (%d,%d), want (0,2)", resp.Move.Row, resp.Move.Col)
	}
}

// TestReferenceNeverLoses plays the reference brain (O) against an exhaustive
// opponent (X) from the empty board. A correct minimax must never lose.
func TestReferenceNeverLoses(t *testing.T) {
	var board [3][3]string
	// X moves first; explore every X line, brain answers as O.
	if lost := playOut(t, board, "X"); lost {
		t.Fatal("reference brain (O) lost a game it should at worst draw")
	}
}

// playOut recurses: when it's X's turn it tries every legal X move (adversary);
// when it's O's turn the reference brain answers. Returns true if O ever loses.
func playOut(t *testing.T, board [3][3]string, turn string) bool {
	t.Helper()
	if w := ttWinner(board); w != "" {
		return w == "X" // O losing is the only failure
	}
	if ttFull(board) {
		return false
	}
	if turn == "O" {
		resp := Reference(Request{Kind: "move", Game: "tictactoe",
			State: mustJSON(t, TicTacToeState{Board: board, Turn: "O", You: "O"})})
		if resp.Move == nil || board[resp.Move.Row][resp.Move.Col] != "" {
			t.Fatalf("brain returned illegal move %+v for board %v", resp.Move, board)
		}
		board[resp.Move.Row][resp.Move.Col] = "O"
		return playOut(t, board, "X")
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if board[i][j] == "" {
				board[i][j] = "X"
				if playOut(t, board, "O") {
					return true
				}
				board[i][j] = ""
			}
		}
	}
	return false
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// --- Wordle reference policy ---

func TestWordleConsistencyFilter(t *testing.T) {
	// wConsistent is the core feedback filter. Spot-check each mark kind.
	cases := []struct {
		word, guess string
		marks       []string
		want        bool
	}{
		// All-correct feedback: a word is always consistent with guessing itself.
		{"WATER", "WATER", []string{"correct", "correct", "correct", "correct", "correct"}, true},
		// GHOST vs GUESS: G correct@0, U/E absent, S correct@3, trailing S present.
		{"GHOST", "GUESS", []string{"correct", "absent", "absent", "correct", "present"}, true},
		// BRAIN vs CRANE: C absent, R correct@1, A correct@2, N present (N is in
		// BRAIN at index 4, not 3), E absent.
		{"BRAIN", "CRANE", []string{"absent", "correct", "correct", "present", "absent"}, true},
	}
	for _, c := range cases {
		if got := wConsistent(c.word, c.guess, c.marks); got != c.want {
			t.Errorf("wConsistent(%q,%q,%v) = %v, want %v", c.word, c.guess, c.marks, got, c.want)
		}
	}
}

func TestWordleRejectsInconsistentCandidate(t *testing.T) {
	// If 'Z' is marked absent, no candidate containing Z is consistent.
	if wConsistent("ZEBRA", "ZZZZZ", []string{"absent", "absent", "absent", "absent", "absent"}) {
		t.Error("candidate containing absent-marked letter should be inconsistent")
	}
	// A 'correct' mark pins the position.
	if wConsistent("WATER", "WXXXX", []string{"correct", "absent", "absent", "absent", "absent"}) == false {
		t.Error("WATER should be consistent with W correct at index 0")
	}
	if wConsistent("OCEAN", "WXXXX", []string{"correct", "absent", "absent", "absent", "absent"}) {
		t.Error("OCEAN should be inconsistent with W correct at index 0")
	}
}

func TestWordleGuessIsConsistentWithFeedback(t *testing.T) {
	// Give feedback that only GHOST satisfies among the built-in list, then check
	// the brain's guess is consistent with all prior feedback (not necessarily
	// the exact word, but never a contradiction).
	prior := []wMark{
		{Word: "CRANE", Marks: []string{"absent", "absent", "absent", "absent", "absent"}},
	}
	st := wState{Length: 5, Guesses: prior}
	resp := Reference(Request{Kind: "move", Game: "wordle", State: mustJSON(t, st)})
	if resp.Move == nil || resp.Move.Word == "" {
		t.Fatal("no word guessed")
	}
	for _, g := range prior {
		if !wConsistent(resp.Move.Word, g.Word, g.Marks) {
			t.Errorf("guess %q contradicts prior feedback %+v", resp.Move.Word, g)
		}
	}
}

// --- UNO reference policy ---

func TestUnoDrawsWhenNoPlayable(t *testing.T) {
	st := uState{Hand: []string{"Red 5", "Blue 2"}, Color: "Green", Playable: nil}
	resp := Reference(Request{Kind: "move", Game: "uno", State: mustJSON(t, st)})
	if resp.Move == nil || !resp.Move.Draw {
		t.Errorf("expected draw move, got %+v", resp.Move)
	}
}

func TestUnoPlaysFirstPlayableAndPicksDominantColor(t *testing.T) {
	// Hand is mostly Blue, so a wild's chosen color should be Blue.
	st := uState{
		Hand:     []string{"Blue 1", "Blue 7", "Blue 9", "Red 3", "Wild"},
		Color:    "Blue",
		Playable: []int{4, 0},
	}
	resp := Reference(Request{Kind: "move", Game: "uno", State: mustJSON(t, st)})
	if resp.Move == nil || resp.Move.CardIndex == nil {
		t.Fatalf("expected a card play, got %+v", resp.Move)
	}
	if *resp.Move.CardIndex != 4 {
		t.Errorf("card index = %d, want 4 (first playable)", *resp.Move.CardIndex)
	}
	if resp.Move.Color != "Blue" {
		t.Errorf("chosen color = %q, want Blue (dominant in hand)", resp.Move.Color)
	}
}

// --- dispatch / draw / sketch / react / errors ---

func TestReferenceUnknownKind(t *testing.T) {
	resp := Reference(Request{Kind: "teleport"})
	if resp.Error == "" {
		t.Error("expected an error for an unknown kind")
	}
}

func TestReferenceDrawAndSketchProduceContent(t *testing.T) {
	draw := Reference(Request{Kind: "draw", Prompt: "a calm horizon", Canvas: &Canvas{Width: 40, Height: 16}})
	if draw.Kind != "draw" || len(draw.Scene) == 0 {
		t.Errorf("draw response missing scene: %+v", draw)
	}
	sketch := Reference(Request{Kind: "sketch", Prompt: "a calm horizon"})
	if sketch.Kind != "sketch" || len(sketch.Sketch) == 0 {
		t.Errorf("sketch response missing sketch: %+v", sketch)
	}
	react := Reference(Request{Kind: "react"})
	if react.Kind != "react" || len(react.Cards) == 0 {
		t.Errorf("react response missing cards: %+v", react)
	}
}

// --- HTTP round-trip through the real handler + HTTPBrain client ---

func TestHTTPBrainRoundTrip(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()

	hb := HTTPBrain{URL: srv.URL, HTTP: srv.Client()}
	board := [3][3]string{{"X", "X", ""}, {"O", "", ""}, {"O", "", ""}}
	resp, err := hb.Decide(Request{Kind: "move", Game: "tictactoe",
		State: mustJSON(t, TicTacToeState{Board: board, Turn: "X", You: "X"})})
	if err != nil {
		t.Fatalf("Decide over HTTP: %v", err)
	}
	if resp.Protocol != Protocol {
		t.Errorf("protocol = %q, want %q", resp.Protocol, Protocol)
	}
	if resp.Move == nil || resp.Move.Row != 0 || resp.Move.Col != 2 {
		t.Errorf("winning move over HTTP = %+v, want (0,2)", resp.Move)
	}
}

func TestHandlerRejectsBadJSON(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for bad request", resp.StatusCode)
	}
}

func TestLocalBrainMatchesReference(t *testing.T) {
	req := Request{Kind: "react"}
	got, err := Local{}.Decide(req)
	if err != nil {
		t.Fatalf("Local.Decide: %v", err)
	}
	want := Reference(req)
	if got.Kind != want.Kind || len(got.Cards) != len(want.Cards) {
		t.Errorf("Local diverged from Reference: %+v vs %+v", got, want)
	}
}
