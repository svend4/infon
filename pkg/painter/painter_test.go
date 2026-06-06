package painter

import (
	"encoding/json"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/pseudo"
)

func gridHas(g *pseudo.Grid, tok string) bool {
	for _, row := range g.Rows {
		for _, c := range row {
			if c == tok {
				return true
			}
		}
	}
	return false
}

func TestPaintDeterministic(t *testing.T) {
	a := Paint("a calm harbor at sunset", 64, 28)
	b := Paint("a calm harbor at sunset", 64, 28)
	if len(a.Grid.Rows) != len(b.Grid.Rows) {
		t.Fatal("row count differs")
	}
	for y := range a.Grid.Rows {
		for x := range a.Grid.Rows[y] {
			if a.Grid.Rows[y][x] != b.Grid.Rows[y][x] {
				t.Fatalf("cell %d,%d differs: %q vs %q", y, x, a.Grid.Rows[y][x], b.Grid.Rows[y][x])
			}
		}
	}
}

func TestPaintReflectsPrompt(t *testing.T) {
	night := Paint("a forest under a starry night", 48, 24)
	if night.Format != pseudo.FormatGrid || night.Grid == nil {
		t.Fatal("expected a grid spec")
	}
	if !gridHas(night.Grid, "navy") || !gridHas(night.Grid, "white") {
		t.Fatal("night scene should have a navy sky and white stars")
	}
	sunsetSea := Paint("a harbor at sunset", 48, 24)
	if !gridHas(sunsetSea.Grid, "orange") && !gridHas(sunsetSea.Grid, "dusk") {
		t.Fatal("sunset sky should be orange/dusk")
	}
	if !gridHas(sunsetSea.Grid, "teal") && !gridHas(sunsetSea.Grid, "skyblue") {
		t.Fatal("a harbor should have water")
	}
}

func TestBrainPaintsAndDelegates(t *testing.T) {
	var b Brain
	// image kind -> painted locally
	resp, err := b.Decide(brain.Request{Kind: "image", Prompt: "snowy mountains at dawn"})
	if err != nil || resp.Kind != "image" || len(resp.Image) == 0 {
		t.Fatalf("image decide failed: %+v err=%v", resp, err)
	}
	var spec pseudo.Spec
	if json.Unmarshal(resp.Image, &spec) != nil || spec.Grid == nil {
		t.Fatal("painted image is not a grid spec")
	}
	// non-image kind -> delegated to the reference policy
	st, _ := json.Marshal(brain.TicTacToeState{Turn: "X", You: "X", Legal: [][2]int{{1, 1}}})
	mv, err := b.Decide(brain.Request{Kind: "move", Game: "tictactoe", State: st})
	if err != nil || mv.Move == nil {
		t.Fatalf("delegate move failed: %+v err=%v", mv, err)
	}
}
