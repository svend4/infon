package session

import (
	"encoding/json"
	"testing"

	"github.com/svend4/infon/pkg/brain"
)

func cfMv(col int) Move {
	m, _ := json.Marshal(CFMove{Col: col})
	return m
}

func TestCFVerticalWin(t *testing.T) {
	g := ConnectFour{}
	b := g.Start()
	seq := []struct{ p, col int }{{0, 3}, {1, 0}, {0, 3}, {1, 0}, {0, 3}, {1, 0}, {0, 3}}
	for _, s := range seq {
		nb, err := g.Apply(b, s.p, cfMv(s.col))
		if err != nil {
			t.Fatal(err)
		}
		b = nb
	}
	if done, w := g.Over(b); !done || w != 0 {
		t.Fatalf("four X in a column should win: done=%v winner=%d", done, w)
	}
}

func TestCFFullColumnRejected(t *testing.T) {
	g := ConnectFour{}
	b := g.Start()
	for i := 0; i < 6; i++ {
		nb, err := g.Apply(b, i%2, cfMv(0))
		if err != nil {
			t.Fatal(err)
		}
		b = nb
	}
	if _, err := g.Apply(b, 0, cfMv(0)); err == nil {
		t.Fatal("a move into a full column was accepted")
	}
}

func TestCFGameCompletes(t *testing.T) {
	bot := func(name string, order []int) FuncPlayer {
		return FuncPlayer{N: name, F: func(req brain.Request) brain.Response {
			var st CFState
			_ = json.Unmarshal(req.State, &st)
			for _, c := range order {
				for _, l := range st.Legal {
					if l == c {
						return brain.Response{Kind: "move", Move: &brain.Move{Col: c}}
					}
				}
			}
			if len(st.Legal) > 0 {
				return brain.Response{Kind: "move", Move: &brain.Move{Col: st.Legal[0]}}
			}
			return brain.Response{Kind: "move", Move: &brain.Move{}}
		}}
	}
	players := []Participant{
		bot("center", []int{3, 2, 4, 1, 5, 0, 6}),
		bot("left", []int{0, 1, 2, 3, 4, 5, 6}),
	}
	r, err := Play[CFBoard](ConnectFour{}, players, 60)
	if err != nil {
		t.Fatal(err)
	}
	if done, _ := (ConnectFour{}).Over(r.Final); !done {
		t.Fatal("connect four game did not finish")
	}
}
