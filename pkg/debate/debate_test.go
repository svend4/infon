package debate

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/brain"
)

func TestProWinsWhenJudgeFavorsIt(t *testing.T) {
	a := FuncDebater{N: "pro", F: func(_ string, _ int, _ string) string { return "a strong, evidenced point" }}
	b := FuncDebater{N: "con", F: func(_ string, _ int, _ string) string { return "weak" }}
	j := FuncJudge{N: "judge", F: func(_, x, y string) int {
		sx, sy := strings.Contains(x, "strong"), strings.Contains(y, "strong")
		switch {
		case sx && !sy:
			return 0
		case sy && !sx:
			return 1
		default:
			return -1
		}
	}}
	res := Run("topic", 3, a, b, j)
	if res.Winner != 0 || res.ScoreA != 3 {
		t.Fatalf("pro should sweep: %+v", res)
	}
	if len(res.Rounds) != 3 {
		t.Fatalf("rounds %d, want 3", len(res.Rounds))
	}
}

func TestTieWhenJudgeTies(t *testing.T) {
	d := FuncDebater{N: "x", F: func(_ string, _ int, _ string) string { return "same" }}
	j := FuncJudge{N: "j", F: func(_, _, _ string) int { return -1 }}
	res := Run("t", 4, d, FuncDebater{N: "y", F: d.F}, j)
	if res.Winner != -1 || res.Ties != 4 {
		t.Fatalf("expected a 0-0 tie over 4 rounds: %+v", res)
	}
}

func TestBrainDebaterProducesArgument(t *testing.T) {
	d := BrainDebater{N: "ref", B: brain.Local{}}
	s := d.Argue("is the terminal the future of media?", 0, "pixels are universal")
	if s == "" || s == "(no argument)" {
		t.Fatalf("brain debater produced no argument: %q", s)
	}
}
