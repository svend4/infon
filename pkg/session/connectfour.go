package session

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/svend4/infon/pkg/brain"
)

// connectfour.go is the SECOND worked Game on the session runner — proof the
// frame is game-agnostic. Connect Four is 7 columns x 6 rows; a move is a column
// (the disc falls to the lowest empty cell). State, move and helpers are exported
// so any external Participant (a bot or an LLM speaking game:connect4) can play.

const cfRows, cfCols = 6, 7

// CFBoard is a Connect Four position ("", "X", "O"); row 0 is the bottom.
type CFBoard = [cfRows][cfCols]string

// CFState is the brief shown to the player to move.
type CFState struct {
	Board CFBoard `json:"board"`
	You   string  `json:"you"`
	Legal []int   `json:"legal"` // columns that are not full
}

// CFMove is a column choice.
type CFMove struct {
	Col int `json:"col"`
}

// CFPlace drops mark into col, returning the new board and whether it fit.
func CFPlace(b CFBoard, col int, m string) (CFBoard, bool) {
	if col < 0 || col >= cfCols {
		return b, false
	}
	for r := 0; r < cfRows; r++ {
		if b[r][col] == "" {
			b[r][col] = m
			return b, true
		}
	}
	return b, false
}

// CFWinner returns "X" or "O" if four are connected, else "".
func CFWinner(b CFBoard) string {
	dirs := [4][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for r := 0; r < cfRows; r++ {
		for c := 0; c < cfCols; c++ {
			p := b[r][c]
			if p == "" {
				continue
			}
			for _, d := range dirs {
				ok := true
				for k := 1; k < 4; k++ {
					rr, cc := r+d[0]*k, c+d[1]*k
					if rr < 0 || rr >= cfRows || cc < 0 || cc >= cfCols || b[rr][cc] != p {
						ok = false
						break
					}
				}
				if ok {
					return p
				}
			}
		}
	}
	return ""
}

func cfCount(b CFBoard) int {
	n := 0
	for r := 0; r < cfRows; r++ {
		for c := 0; c < cfCols; c++ {
			if b[r][c] != "" {
				n++
			}
		}
	}
	return n
}

func cfLegal(b CFBoard) []int {
	var l []int
	for c := 0; c < cfCols; c++ {
		if b[cfRows-1][c] == "" {
			l = append(l, c)
		}
	}
	return l
}

// ConnectFour implements Game[CFBoard].
type ConnectFour struct{}

func (ConnectFour) Name() string  { return "connect4" }
func (ConnectFour) Start() CFBoard { return CFBoard{} }

func (ConnectFour) Turn(b CFBoard) int {
	if CFWinner(b) != "" || cfCount(b) >= cfRows*cfCols {
		return -1
	}
	return cfCount(b) % 2
}

func (ConnectFour) Over(b CFBoard) (bool, int) {
	switch CFWinner(b) {
	case "X":
		return true, 0
	case "O":
		return true, 1
	}
	if cfCount(b) >= cfRows*cfCols {
		return true, -1
	}
	return false, -1
}

func (ConnectFour) Brief(b CFBoard, p int) brain.Request {
	st := CFState{Board: b, You: mark(p), Legal: cfLegal(b)}
	data, _ := json.Marshal(st)
	return brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "connect4", State: data}
}

func (ConnectFour) Parse(resp brain.Response) (Move, error) {
	if resp.Move == nil {
		return nil, errors.New("no move in response")
	}
	return json.Marshal(CFMove{Col: resp.Move.Col})
}

func (ConnectFour) Apply(b CFBoard, p int, m Move) (CFBoard, error) {
	var v CFMove
	if json.Unmarshal(m, &v) != nil {
		return b, errors.New("bad move encoding")
	}
	if v.Col < 0 || v.Col >= cfCols {
		return b, fmt.Errorf("column %d out of range", v.Col)
	}
	nb, ok := CFPlace(b, v.Col, mark(p))
	if !ok {
		return b, fmt.Errorf("column %d is full", v.Col)
	}
	return nb, nil
}
