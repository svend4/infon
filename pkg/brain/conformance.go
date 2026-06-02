package brain

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/scene"
	"github.com/svend4/infon/pkg/sketch"
	"github.com/svend4/infon/pkg/tangram"
)

// ConformanceCase is one labeled request from the standard battery.
type ConformanceCase struct {
	Name string
	Req  Request
}

// ConformanceResult is the outcome of one case.
type ConformanceResult struct {
	Name   string
	Pass   bool
	Detail string
}

// ConformanceBattery returns the standard set of requests every tvcp-ai brain
// must answer correctly. Implementers run it against their endpoint (see
// cmd/conform) to claim tvcp-ai conformance.
func ConformanceBattery() []ConformanceCase {
	ttState, _ := json.Marshal(TicTacToeState{
		Board: [3][3]string{{"X", "", ""}, {"", "O", ""}, {"", "", ""}},
		You:   "X", Turn: "X",
		Legal: [][2]int{{0, 1}, {0, 2}, {1, 0}, {1, 2}, {2, 0}, {2, 1}, {2, 2}},
	})
	wState, _ := json.Marshal(map[string]any{"length": 5, "guesses": []any{}})
	uState, _ := json.Marshal(map[string]any{"hand": []string{"Red 5", "Blue 2"}, "color": "Red", "playable": []int{0}})
	tgState, _ := json.Marshal(PuzzleStateFromFigureTree())
	wlState, _ := json.Marshal(map[string]int{"fold_pct": 40, "relief_pct": 30, "camera_deg": 0, "orbit_phase": 0})
	return []ConformanceCase{
		{"move/tictactoe", Request{Kind: "move", Game: "tictactoe", State: ttState}},
		{"move/wordle", Request{Kind: "move", Game: "wordle", State: wState}},
		{"move/uno", Request{Kind: "move", Game: "uno", State: uState}},
		{"move/tangram", Request{Kind: "move", Game: "tangram", State: tgState}},
		{"move/world", Request{Kind: "move", Game: "world", State: wlState}},
		{"draw", Request{Kind: "draw", Prompt: "a calm harbor at dawn", Canvas: &Canvas{Width: 48, Height: 20}}},
		{"sketch", Request{Kind: "sketch", Prompt: "sunset over mountains"}},
		{"image/grid", Request{Kind: "image", Format: "grid", Prompt: "a calm harbor", Canvas: &Canvas{Width: 48, Height: 20}}},
		{"image/sigils", Request{Kind: "image", Format: "sigils", Prompt: "a harbor"}},
		{"image/glyphs", Request{Kind: "image", Format: "glyphs", Prompt: "a harbor"}},
		{"image/marks", Request{Kind: "image", Format: "marks", Prompt: "a mountain"}},
		{"react", Request{Kind: "react", Prompt: "great job!"}},
	}
}

// PuzzleStateFromFigureTree builds the tangram conformance puzzle (a tree).
func PuzzleStateFromFigureTree() tangram.PuzzleState {
	return tangram.PuzzleFromFigure(tangram.Tree())
}

var wordRe = regexp.MustCompile(`^[A-Za-z]{5}$`)

// CheckResponse validates a response against the request per the tvcp-ai
// contract. Returns ("", true) on pass or (reason, false) on failure.
func CheckResponse(req Request, resp Response) (string, bool) {
	if resp.Error != "" {
		return "brain returned error: " + resp.Error, false
	}
	switch req.Kind {
	case "move":
		if req.Game == "world" {
			var d struct{ Fold, Rise, Spin, Camera int }
			if len(resp.World) == 0 || json.Unmarshal(resp.World, &d) != nil {
				return "world: missing or invalid directives", false
			}
			for _, v := range []int{d.Fold, d.Rise, d.Spin, d.Camera} {
				if v < -1 || v > 1 {
					return "world: directive out of range", false
				}
			}
			return "", true
		}
		if req.Game == "tangram" {
			var ps tangram.PuzzleState
			if json.Unmarshal(req.State, &ps) != nil {
				return "tangram: bad state", false
			}
			var sol tangram.Figure
			if len(resp.Tangram) == 0 || json.Unmarshal(resp.Tangram, &sol) != nil {
				return "tangram: missing or invalid solution", false
			}
			sc := tangram.ScorePuzzle(ps, sol)
			if !sc.WellFormed {
				return "tangram: solution has overlaps", false
			}
			if sc.IoU < 0.5 {
				return fmt.Sprintf("tangram: IoU too low: %.2f", sc.IoU), false
			}
			return "", true
		}
		if resp.Move == nil {
			return "no move in response", false
		}
		switch req.Game {
		case "wordle":
			if !wordRe.MatchString(resp.Move.Word) {
				return "word must be 5 letters, got " + fmt.Sprintf("%q", resp.Move.Word), false
			}
		case "uno":
			if resp.Move.CardIndex == nil && !resp.Move.Draw {
				return "uno move is neither a card_index nor draw", false
			}
		default: // tic-tac-toe
			if resp.Move.Row < 0 || resp.Move.Row > 2 || resp.Move.Col < 0 || resp.Move.Col > 2 {
				return fmt.Sprintf("tictactoe move out of range: %d,%d", resp.Move.Row, resp.Move.Col), false
			}
		}
	case "draw":
		var sc scene.Scene
		if json.Unmarshal(resp.Scene, &sc) != nil || len(sc.Ops) == 0 {
			return "draw must carry a scene with ops", false
		}
	case "sketch":
		var sk sketch.Sketch
		if json.Unmarshal(resp.Sketch, &sk) != nil || len(sk.Shapes) == 0 {
			return "sketch must carry shapes", false
		}
	case "image":
		var sp pseudo.Spec
		if json.Unmarshal(resp.Image, &sp) != nil {
			return "image spec failed to parse", false
		}
		if f, err := sp.Frame(); err != nil || f == nil || f.Width == 0 {
			return "image spec failed to render", false
		}
	case "react":
		if len(resp.Cards) == 0 {
			return "react must carry at least one card", false
		}
	default:
		return "unknown kind: " + req.Kind, false
	}
	return "", true
}

// RunConformance runs the battery through a decide function (Reference, or an
// HTTPBrain's Decide) and returns per-case results plus an overall pass.
func RunConformance(decide func(Request) (Response, error)) ([]ConformanceResult, bool) {
	all := true
	out := make([]ConformanceResult, 0, len(ConformanceBattery()))
	for _, c := range ConformanceBattery() {
		resp, err := decide(c.Req)
		if err != nil {
			out = append(out, ConformanceResult{c.Name, false, "transport: " + err.Error()})
			all = false
			continue
		}
		detail, ok := CheckResponse(c.Req, resp)
		out = append(out, ConformanceResult{c.Name, ok, detail})
		if !ok {
			all = false
		}
	}
	return out, all
}
