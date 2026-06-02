// Package session is the unifying frame the experimental games/bots/agents notes
// asked for: ONE turn-based game runner where every player — a human, a scripted
// bot, or a live tvcp-ai brain — is just a Participant, and any game is a Game.
// A GameSession drives them to a result with a transcript, over the same
// tvcp-ai/1 request the rest of the project speaks. Tic-tac-toe is included as a
// worked Game; wordle/uno/chess/etc. slot in the same way.
package session

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/svend4/infon/pkg/brain"
)

// Move is a game-specific move, JSON-encoded.
type Move = []byte

// Participant is any decision-maker: a brain, a bot, or a human bridge. It
// answers a tvcp-ai request — so the reference brain, an LLM over HTTP, and a Go
// function are all interchangeable here.
type Participant interface {
	Name() string
	Decide(req brain.Request) (brain.Response, error)
}

// Game is a turn-based game over a state type S.
type Game[S any] interface {
	Name() string
	Start() S
	Turn(s S) int                       // player to move (0-based), or -1 if none
	Over(s S) (done bool, winner int)    // winner: player index, or -1 for a draw
	Brief(s S, player int) brain.Request // the request shown to the player to move
	Parse(resp brain.Response) (Move, error)
	Apply(s S, player int, m Move) (S, error)
}

// TurnRec is one recorded move.
type TurnRec struct {
	Player int  `json:"player"`
	Move   Move `json:"move"`
}

// Result is the outcome of a session.
type Result[S any] struct {
	Winner     int // player index, or -1 for draw/unfinished
	Turns      int
	Final      S
	Transcript []TurnRec
}

// Play runs a Game between participants until it is over (or maxTurns is hit),
// returning the result and a transcript. An illegal or unparseable move ends the
// session with an error (the offending participant named).
func Play[S any](g Game[S], players []Participant, maxTurns int) (Result[S], error) {
	s := g.Start()
	var tr []TurnRec
	for t := 0; t < maxTurns; t++ {
		if done, win := g.Over(s); done {
			return Result[S]{Winner: win, Turns: len(tr), Final: s, Transcript: tr}, nil
		}
		p := g.Turn(s)
		if p < 0 || p >= len(players) {
			break
		}
		resp, err := players[p].Decide(g.Brief(s, p))
		if err != nil {
			return Result[S]{Final: s, Transcript: tr}, fmt.Errorf("session: %s: %w", players[p].Name(), err)
		}
		m, err := g.Parse(resp)
		if err != nil {
			return Result[S]{Final: s, Transcript: tr}, fmt.Errorf("session: %s: parse: %w", players[p].Name(), err)
		}
		ns, err := g.Apply(s, p, m)
		if err != nil {
			return Result[S]{Final: s, Transcript: tr}, fmt.Errorf("session: %s: illegal move: %w", players[p].Name(), err)
		}
		s = ns
		tr = append(tr, TurnRec{Player: p, Move: m})
	}
	done, win := g.Over(s)
	if !done {
		win = -1
	}
	return Result[S]{Winner: win, Turns: len(tr), Final: s, Transcript: tr}, nil
}

// BrainPlayer adapts any brain.Brain (reference, Local, HTTPBrain to an LLM) into
// a Participant.
type BrainPlayer struct {
	N string
	B brain.Brain
}

func (b BrainPlayer) Name() string                                   { return b.N }
func (b BrainPlayer) Decide(req brain.Request) (brain.Response, error) { return b.B.Decide(req) }

// FuncPlayer adapts a plain function into a Participant (a scripted bot).
type FuncPlayer struct {
	N string
	F func(brain.Request) brain.Response
}

func (f FuncPlayer) Name() string                                   { return f.N }
func (f FuncPlayer) Decide(req brain.Request) (brain.Response, error) { return f.F(req), nil }

// ---- a worked Game: tic-tac-toe -------------------------------------------

// Board is a tic-tac-toe position ("", "X", "O").
type Board = [3][3]string

type rc struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// TicTacToe implements Game[Board] over the tvcp-ai tictactoe protocol.
type TicTacToe struct{}

func (TicTacToe) Name() string { return "tictactoe" }
func (TicTacToe) Start() Board { return Board{} }

func mark(p int) string {
	if p == 0 {
		return "X"
	}
	return "O"
}

func ttCount(b Board) int {
	n := 0
	for i := range b {
		for j := range b[i] {
			if b[i][j] != "" {
				n++
			}
		}
	}
	return n
}

func ttWin(b Board) string {
	lines := [8][3][2]int{
		{{0, 0}, {0, 1}, {0, 2}}, {{1, 0}, {1, 1}, {1, 2}}, {{2, 0}, {2, 1}, {2, 2}},
		{{0, 0}, {1, 0}, {2, 0}}, {{0, 1}, {1, 1}, {2, 1}}, {{0, 2}, {1, 2}, {2, 2}},
		{{0, 0}, {1, 1}, {2, 2}}, {{0, 2}, {1, 1}, {2, 0}},
	}
	for _, l := range lines {
		a := b[l[0][0]][l[0][1]]
		if a != "" && a == b[l[1][0]][l[1][1]] && a == b[l[2][0]][l[2][1]] {
			return a
		}
	}
	return ""
}

func ttLegal(b Board) [][2]int {
	var out [][2]int
	for i := range b {
		for j := range b[i] {
			if b[i][j] == "" {
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}

func (TicTacToe) Turn(b Board) int {
	if ttWin(b) != "" || ttCount(b) >= 9 {
		return -1
	}
	return ttCount(b) % 2
}

func (TicTacToe) Over(b Board) (bool, int) {
	switch ttWin(b) {
	case "X":
		return true, 0
	case "O":
		return true, 1
	}
	if ttCount(b) >= 9 {
		return true, -1
	}
	return false, -1
}

func (TicTacToe) Brief(b Board, p int) brain.Request {
	st := brain.TicTacToeState{Board: b, Turn: mark(p), You: mark(p), Legal: ttLegal(b)}
	data, _ := json.Marshal(st)
	return brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "tictactoe", State: data}
}

func (TicTacToe) Parse(resp brain.Response) (Move, error) {
	if resp.Move == nil {
		return nil, errors.New("no move in response")
	}
	return json.Marshal(rc{Row: resp.Move.Row, Col: resp.Move.Col})
}

func (TicTacToe) Apply(b Board, p int, m Move) (Board, error) {
	var v rc
	if json.Unmarshal(m, &v) != nil {
		return b, errors.New("bad move encoding")
	}
	if v.Row < 0 || v.Row > 2 || v.Col < 0 || v.Col > 2 {
		return b, fmt.Errorf("out of bounds (%d,%d)", v.Row, v.Col)
	}
	if b[v.Row][v.Col] != "" {
		return b, fmt.Errorf("cell (%d,%d) occupied", v.Row, v.Col)
	}
	b[v.Row][v.Col] = mark(p)
	return b, nil
}
