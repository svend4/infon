package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/svend4/infon/pkg/brain"
)

// wordle.go is the THIRD worked Game on the session runner, and the first that is
// SOLITAIRE: one player against a hidden word, with per-letter feedback. It proves
// the Game/Participant frame is not limited to two-player adversarial play — the
// same runner, tournament and debate machinery drive a single-player, hidden-state,
// feedback game too. It speaks the existing tvcp-ai game:wordle protocol, so the
// reference brain (and any LLM) can play unchanged.

// WordleGuess is one guess and its per-letter marks.
type WordleGuess struct {
	Word  string   `json:"word"`
	Marks []string `json:"marks"` // per letter: "correct" | "present" | "absent"
}

// WordleState is the full game state. Target is secret and never sent to the
// player (Brief redacts it).
type WordleState struct {
	Target  string        `json:"target"`
	Guesses []WordleGuess `json:"guesses"`
	Max     int           `json:"max"`
}

// WordleMarks scores a guess against the target with standard Wordle rules:
// exact matches first, then present-elsewhere accounting for letter multiplicity,
// the rest absent. Exported so external bots can reason about feedback.
func WordleMarks(target, guess string) []string {
	target = strings.ToUpper(target)
	guess = strings.ToUpper(guess)
	tr, gr := []rune(target), []rune(guess)
	marks := make([]string, len(gr))
	rem := map[rune]int{}
	for i := range gr {
		if i < len(tr) && gr[i] == tr[i] {
			marks[i] = "correct"
		} else if i < len(tr) {
			rem[tr[i]]++
		}
	}
	for i := range gr {
		if marks[i] == "correct" {
			continue
		}
		if rem[gr[i]] > 0 {
			marks[i] = "present"
			rem[gr[i]]--
		} else {
			marks[i] = "absent"
		}
	}
	return marks
}

// Wordle implements Game[WordleState]. Target is the secret answer; MaxGuesses
// defaults to 6.
type Wordle struct {
	Target     string
	MaxGuesses int
}

func (Wordle) Name() string { return "wordle" }

func (w Wordle) Start() WordleState {
	mg := w.MaxGuesses
	if mg <= 0 {
		mg = 6
	}
	return WordleState{Target: strings.ToUpper(w.Target), Max: mg}
}

func wordleSolved(s WordleState) bool {
	return len(s.Guesses) > 0 &&
		strings.EqualFold(s.Guesses[len(s.Guesses)-1].Word, s.Target)
}

func (Wordle) Turn(s WordleState) int {
	if wordleSolved(s) || len(s.Guesses) >= s.Max {
		return -1
	}
	return 0
}

// Over reports the solitaire outcome: player 0 "wins" by solving, otherwise the
// game ends with no winner once guesses run out.
func (Wordle) Over(s WordleState) (bool, int) {
	if wordleSolved(s) {
		return true, 0
	}
	if len(s.Guesses) >= s.Max {
		return true, -1
	}
	return false, -1
}

func (Wordle) Brief(s WordleState, _ int) brain.Request {
	pub := struct {
		Length  int           `json:"length"`
		Guesses []WordleGuess `json:"guesses"`
		Max     int           `json:"max"`
	}{Length: len(s.Target), Guesses: s.Guesses, Max: s.Max}
	data, _ := json.Marshal(pub)
	return brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "wordle", State: data}
}

type wordWord struct {
	Word string `json:"word"`
}

func (Wordle) Parse(resp brain.Response) (Move, error) {
	if resp.Move == nil || resp.Move.Word == "" {
		return nil, errors.New("no word in response")
	}
	return json.Marshal(wordWord{Word: resp.Move.Word})
}

func (Wordle) Apply(s WordleState, _ int, m Move) (WordleState, error) {
	var v wordWord
	if json.Unmarshal(m, &v) != nil {
		return s, errors.New("bad move encoding")
	}
	word := strings.ToUpper(strings.TrimSpace(v.Word))
	if len(word) != len(s.Target) {
		return s, fmt.Errorf("guess must be %d letters, got %q", len(s.Target), word)
	}
	for _, r := range word {
		if r < 'A' || r > 'Z' {
			return s, fmt.Errorf("guess has a non-letter: %q", word)
		}
	}
	s.Guesses = append(append([]WordleGuess(nil), s.Guesses...),
		WordleGuess{Word: word, Marks: WordleMarks(s.Target, word)})
	return s, nil
}
