package session_test

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/session"
)

func wordPlayer(name string, pick func() string) session.FuncPlayer {
	return session.FuncPlayer{N: name, F: func(brain.Request) brain.Response {
		return brain.Response{Kind: "move", Move: &brain.Move{Word: pick()}}
	}}
}

func TestWordleMarksDuplicates(t *testing.T) {
	// target ALLOY, guess LOLLY: two L's exercise present-vs-absent multiplicity.
	got := session.WordleMarks("ALLOY", "LOLLY")
	want := []string{"present", "present", "correct", "absent", "correct"}
	if len(got) != len(want) {
		t.Fatalf("len %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pos %d: got %s, want %s (%v)", i, got[i], want[i], got)
		}
	}
}

func TestWordleOracleSolvesInOne(t *testing.T) {
	target := "ROBOT"
	r, err := session.Play[session.WordleState](
		session.Wordle{Target: target, MaxGuesses: 6},
		[]session.Participant{wordPlayer("oracle", func() string { return target })},
		10)
	if err != nil {
		t.Fatal(err)
	}
	if r.Winner != 0 || r.Turns != 1 {
		t.Fatalf("oracle should solve in 1 turn: winner=%d turns=%d", r.Winner, r.Turns)
	}
}

func TestWordleFailsWhenExhausted(t *testing.T) {
	r, err := session.Play[session.WordleState](
		session.Wordle{Target: "ROBOT", MaxGuesses: 3},
		[]session.Participant{wordPlayer("wrong", func() string { return "ZZZZZ" })},
		10)
	if err != nil {
		t.Fatal(err)
	}
	if r.Winner != -1 || r.Turns != 3 {
		t.Fatalf("should exhaust to no winner in 3 turns: winner=%d turns=%d", r.Winner, r.Turns)
	}
}

func TestWordleRejectsBadGuesses(t *testing.T) {
	w := session.Wordle{Target: "ROBOT"}
	s := w.Start()
	if _, err := w.Apply(s, 0, []byte(`{"word":"TOOLONGWORD"}`)); err == nil {
		t.Error("wrong-length guess should be rejected")
	}
	if _, err := w.Apply(s, 0, []byte(`{"word":"RO8OT"}`)); err == nil {
		t.Error("non-letter guess should be rejected")
	}
	if _, err := w.Apply(s, 0, []byte(`{"word":"crane"}`)); err != nil {
		t.Errorf("valid lower-case guess should be accepted: %v", err)
	}
}

// The reference brain, speaking game:wordle, actually solves a word from its own
// vocabulary on the session runner (no oracle) within a generous budget.
func TestWordleReferenceBrainSolves(t *testing.T) {
	r, err := session.Play[session.WordleState](
		session.Wordle{Target: "CRANE", MaxGuesses: 20},
		[]session.Participant{session.BrainPlayer{N: "reference", B: brain.Local{}}},
		40)
	if err != nil {
		t.Fatal(err)
	}
	if r.Winner != 0 {
		t.Fatalf("reference brain failed to solve CRANE: winner=%d turns=%d", r.Winner, r.Turns)
	}
}
