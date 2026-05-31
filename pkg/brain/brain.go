// Package brain defines an OPEN, transport-agnostic format ("tvcp-ai/1") for
// connecting ANY neural network as a partner for games and visual messages.
//
// A brain is anything that can answer a Request with a Response. The wire form
// is JSON over HTTP POST, so the backend can be local (Ollama), cloud (OpenAI/
// Anthropic), a script, or this process itself. Swap brains by changing a URL.
package brain

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/scene"
	"github.com/svend4/infon/pkg/sketch"
)

// Protocol is the format version string.
const Protocol = "tvcp-ai/1"

// Request is what we ask a brain to decide.
type Request struct {
	Protocol string          `json:"protocol"`
	Kind     string          `json:"kind"`            // "move" | "draw" | "sketch" | "image" | "react"
	Game     string          `json:"game,omitempty"`  // e.g. "tictactoe"
	State    json.RawMessage `json:"state,omitempty"` // game-specific state
	Prompt   string          `json:"prompt,omitempty"`
	Canvas   *Canvas         `json:"canvas,omitempty"`
	Format   string          `json:"format,omitempty"` // image kind: grid|pixels|glyphs|sigils|vector|sketch
}

// Canvas is the target size for a draw request (in terminal cells).
type Canvas struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Move is a board move.
type Move struct {
	Row       int    `json:"row"`
	Col       int    `json:"col"`
	Word      string `json:"word,omitempty"`       // word games (Wordle)
	CardIndex *int   `json:"card_index,omitempty"` // card games (UNO): index in hand
	Draw      bool   `json:"draw,omitempty"`       // UNO: draw instead of play
	Color     string `json:"color,omitempty"`      // UNO: chosen color for a wild
}

// Response is the brain's answer.
type Response struct {
	Protocol  string          `json:"protocol"`
	Kind      string          `json:"kind"`
	Move      *Move           `json:"move,omitempty"`
	Scene     json.RawMessage `json:"scene,omitempty"`  // a draw-DSL document
	Sketch    json.RawMessage `json:"sketch,omitempty"` // a high-level sketch
	Image     json.RawMessage `json:"image,omitempty"`  // a pseudo-image spec (pkg/pseudo)
	Cards     []string        `json:"cards,omitempty"`  // glyph "cards" for react
	Reasoning string          `json:"reasoning,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// TicTacToeState is the standard state payload for game "tictactoe".
type TicTacToeState struct {
	Board [3][3]string `json:"board"` // "X", "O", or ""
	Turn  string       `json:"turn"`  // whose turn
	You   string       `json:"you"`   // the mark the brain plays
	Legal [][2]int     `json:"legal"`
}

// Brain is any decision backend.
type Brain interface {
	Decide(Request) (Response, error)
}

// HTTPBrain talks to any endpoint that speaks tvcp-ai/1.
type HTTPBrain struct {
	URL  string
	HTTP *http.Client
}

// Decide POSTs the request as JSON and decodes the JSON response.
func (h HTTPBrain) Decide(req Request) (Response, error) {
	if req.Protocol == "" {
		req.Protocol = Protocol
	}
	body, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}
	cl := h.HTTP
	if cl == nil {
		cl = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := cl.Post(h.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	var out Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Response{}, err
	}
	return out, nil
}

// ----- Reference brain (a self-contained backend implementing tvcp-ai/1) -----

func other(m string) string {
	if m == "X" {
		return "O"
	}
	return "X"
}

func ttWinner(b [3][3]string) string {
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

func ttFull(b [3][3]string) bool {
	for i := range b {
		for j := range b[i] {
			if b[i][j] == "" {
				return false
			}
		}
	}
	return true
}

func ttMinimax(b [3][3]string, me, cur string) int {
	if w := ttWinner(b); w != "" {
		if w == me {
			return 1
		}
		return -1
	}
	if ttFull(b) {
		return 0
	}
	best := -2
	if cur != me {
		best = 2
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if b[i][j] == "" {
				b[i][j] = cur
				s := ttMinimax(b, me, other(cur))
				b[i][j] = ""
				if cur == me {
					if s > best {
						best = s
					}
				} else if s < best {
					best = s
				}
			}
		}
	}
	return best
}

func bestMove(b [3][3]string, me string) (int, int) {
	br, bc, best := -1, -1, -2
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if b[i][j] == "" {
				b[i][j] = me
				s := ttMinimax(b, me, other(me))
				b[i][j] = ""
				if s > best {
					best, br, bc = s, i, j
				}
			}
		}
	}
	return br, bc
}

func refMove(req Request) Response {
	var st TicTacToeState
	_ = json.Unmarshal(req.State, &st)
	me := st.You
	if me == "" {
		me = st.Turn
	}
	if me == "" {
		me = "X"
	}
	r, c := bestMove(st.Board, me)
	return Response{Protocol: Protocol, Kind: "move", Move: &Move{Row: r, Col: c}, Reasoning: "reference minimax"}
}

func refDraw(req Request) Response {
	w, h := 64, 24
	if req.Canvas != nil {
		if req.Canvas.Width > 0 {
			w = req.Canvas.Width
		}
		if req.Canvas.Height > 0 {
			h = req.Canvas.Height
		}
	}
	hs := 0
	for _, ch := range req.Prompt {
		hs += int(ch)
	}
	c1 := [3]int{(hs*7)%180 + 40, (hs*13)%150 + 40, (hs*17)%140 + 70}
	c2 := [3]int{(hs*5)%90 + 10, (hs*11)%90 + 20, (hs*19)%150 + 60}
	sun := [3]int{255, 228, 150}
	white := [3]int{245, 245, 250}
	tag := [3]int{200, 210, 230}
	txt := req.Prompt
	if len(txt) > w-4 {
		txt = txt[:w-4]
	}
	sc := scene.Scene{Width: w, Height: h, Ops: []scene.Op{
		{Op: "vgradient", X: 0, Y: 0, W: w, H: h, Glyph: " ", From: &c1, To: &c2},
		{Op: "disc", CX: w * 3 / 4, CY: h / 3, R: 3, Glyph: "█", Fg: &sun},
		{Op: "text", X: 2, Y: 1, S: txt, Fg: &white},
		{Op: "text", X: 2, Y: h - 2, S: "tvcp-ai/1 | brain draw", Fg: &tag},
	}}
	data, _ := json.Marshal(sc)
	return Response{Protocol: Protocol, Kind: "draw", Scene: data, Reasoning: "reference scene"}
}

func refReact(req Request) Response {
	return Response{Protocol: Protocol, Kind: "react", Cards: []string{"★", "♥", "✓"}, Reasoning: "reference"}
}

func refSketch(req Request) Response {
	hs := 0
	for _, ch := range req.Prompt {
		hs += int(ch)
	}
	skies := []string{"orange", "purple", "navy", "teal", "dusk"}
	grounds := []string{"navy", "darkgreen", "brown", "black"}
	txt := req.Prompt
	if len(txt) > 40 {
		txt = txt[:40]
	}
	sk := sketch.Sketch{
		Sky:    skies[hs%len(skies)],
		Ground: grounds[hs%len(grounds)],
		Shapes: []sketch.Shape{
			{Kind: "sun", X: 48, Y: 6, W: 3, Color: "gold"},
			{Kind: "mountain", X: 16, H: 8, Color: "darkgreen"},
			{Kind: "mountain", X: 40, H: 6, Color: "darkgreen"},
			{Kind: "star", X: 8, Y: 3, Color: "white"},
			{Kind: "star", X: 30, Y: 2, Color: "white"},
			{Kind: "text", X: 2, Y: 1, S: txt, Color: "white"},
		},
	}
	data, _ := json.Marshal(sk)
	return Response{Protocol: Protocol, Kind: "sketch", Sketch: data, Reasoning: "reference sketch"}
}

// refImage answers a kind:"image" request with a pkg/pseudo spec. It proves the
// whole point of pseudo-images: even a backend with NO image model (this one)
// can "paint" by emitting a few tokens of structured JSON. The Format field of
// the request selects the encoding; the default is the pseudo-diffusion grid.
func refImage(req Request) Response {
	cols, rows := 64, 28
	if req.Canvas != nil {
		if req.Canvas.Width > 0 {
			cols = req.Canvas.Width
		}
		if req.Canvas.Height > 0 {
			rows = req.Canvas.Height
		}
	}
	hs := 0
	for _, ch := range req.Prompt {
		hs += int(ch)
	}
	spec := pseudo.Spec{Cols: cols, Rows: rows, Title: req.Prompt}
	switch strings.ToLower(strings.TrimSpace(req.Format)) {
	case "sketch":
		spec.Format = pseudo.FormatSketch
		spec.Sketch = refSketch(req).Sketch
	case "vector":
		spec.Format = pseudo.FormatVector
		spec.Vector = refDraw(req).Scene
	case "pixels":
		spec.Format = pseudo.FormatPixels
		spec.Pixels = refGridSeed(hs)
	case "sigils":
		spec.Format = pseudo.FormatSigils
		spec.Sigils = refSigilScene(hs)
	case "glyphs":
		spec.Format = pseudo.FormatGlyphs
		spec.Glyphs = refGlyphArt()
	default: // "grid" or unspecified -> pseudo-diffusion
		spec.Format = pseudo.FormatGrid
		spec.Grid = refGridSeed(hs)
	}
	data, _ := json.Marshal(spec)
	return Response{Protocol: Protocol, Kind: "image", Image: data, Reasoning: "reference pseudo-image (" + string(spec.Format) + ")"}
}

// refGridSeed builds a tiny 12x8 landscape seed (sky + sun glow + water) that the
// pseudo-diffusion renderer upscales and blurs into a painterly image.
func refGridSeed(hs int) *pseudo.Grid {
	skies := [][2]string{{"navy", "purple"}, {"dusk", "navy"}, {"navy", "blue"}, {"purple", "navy"}}
	waters := []string{"teal", "blue", "skyblue"}
	s := skies[hs%len(skies)]
	w := waters[hs%len(waters)]
	sun := 3 + hs%6
	rows := make([][]string, 8)
	for y := 0; y < 8; y++ {
		row := make([]string, 12)
		for x := 0; x < 12; x++ {
			switch {
			case y >= 5:
				row[x] = w
				if (x+y)%3 == 0 {
					row[x] = "skyblue"
				}
			case y == 4:
				row[x] = "slate"
			default:
				d := x - sun
				if d < 0 {
					d = -d
				}
				switch {
				case d+y <= 2:
					row[x] = "gold"
				case d+y <= 4:
					row[x] = "amber"
				case x < 6:
					row[x] = s[0]
				default:
					row[x] = s[1]
				}
			}
		}
		rows[y] = row
	}
	return &pseudo.Grid{Rows: rows}
}

func refSigilScene(hs int) *pseudo.SigilScene {
	skies := []string{"navy", "dusk", "purple"}
	return &pseudo.SigilScene{
		Sky:    skies[hs%len(skies)],
		Ground: "teal",
		Items: []pseudo.Sigil{
			{Name: "sun", X: 0.72, Y: 0.24, Color: "gold"},
			{Name: "cloud", X: 0.3, Y: 0.18, Color: "gray"},
			{Name: "star", X: 0.12, Y: 0.12, Color: "white"},
			{Name: "mountain", X: 0.24, Y: 0.58, Color: "slate"},
			{Name: "mountain", X: 0.36, Y: 0.62, Color: "dusk"},
			{Name: "anchor", X: 0.62, Y: 0.82, Color: "white"},
		},
	}
}

func refGlyphArt() *pseudo.GlyphArt {
	return &pseudo.GlyphArt{
		Bg: "navy", Fg: "white",
		Palette: map[string]string{"*": "gold", "^": "slate", "~": "skyblue", "=": "teal"},
		Rows: []string{
			"          *           ",
			"     ^^        ^^^     ",
			"   ^^^^^^^^^^^^^^^^^^  ",
			" ==~~==~~==~~==~~==~~= ",
			"~~==~~==~~==~~==~~==~~~",
		},
	}
}

var wordleWords = []string{
	"WATER", "CRANE", "SLATE", "ROBOT", "PIXEL", "LIGHT", "SOUND", "BRAIN", "CLOUD", "STONE",
	"RIVER", "OCEAN", "PLANT", "HOUSE", "MUSIC", "DREAM", "FLAME", "GHOST", "KNIFE", "LEMON",
	"MONEY", "NIGHT", "PAPER", "QUEEN", "SUGAR", "TIGER", "VOICE", "WHALE", "ZEBRA", "APPLE",
	"BEACH", "CHAIR", "DANCE", "EAGLE", "FROST", "GRAPE", "HONEY", "STORM",
}

type wMark struct {
	Word  string   `json:"word"`
	Marks []string `json:"marks"`
}
type wState struct {
	Length  int     `json:"length"`
	Guesses []wMark `json:"guesses"`
}

func wConsistent(word, guess string, marks []string) bool {
	word = strings.ToUpper(word)
	guess = strings.ToUpper(guess)
	if len(guess) != len(word) || len(marks) != len(word) {
		return true
	}
	for i := 0; i < len(word); i++ {
		g := guess[i]
		switch marks[i] {
		case "correct":
			if word[i] != g {
				return false
			}
		case "present":
			if word[i] == g || !strings.ContainsRune(word, rune(g)) {
				return false
			}
		case "absent":
			if strings.ContainsRune(word, rune(g)) {
				return false
			}
		}
	}
	return true
}

func refWordle(req Request) Response {
	var st wState
	_ = json.Unmarshal(req.State, &st)
	used := map[string]bool{}
	for _, gx := range st.Guesses {
		used[strings.ToUpper(gx.Word)] = true
	}
	for _, w := range wordleWords {
		if used[w] {
			continue
		}
		ok := true
		for _, gx := range st.Guesses {
			if !wConsistent(w, gx.Word, gx.Marks) {
				ok = false
				break
			}
		}
		if ok {
			return Response{Protocol: Protocol, Kind: "move", Move: &Move{Word: w}, Reasoning: "reference wordle"}
		}
	}
	return Response{Protocol: Protocol, Kind: "move", Move: &Move{Word: wordleWords[0]}, Reasoning: "reference wordle (fallback)"}
}

type uState struct {
	Hand     []string `json:"hand"`
	Color    string   `json:"color"`
	Playable []int    `json:"playable"`
}

func refUno(req Request) Response {
	var st uState
	_ = json.Unmarshal(req.State, &st)
	if len(st.Playable) == 0 {
		return Response{Protocol: Protocol, Kind: "move", Move: &Move{Draw: true}, Reasoning: "reference uno: draw"}
	}
	idx := st.Playable[0]
	counts := map[string]int{}
	for _, c := range st.Hand {
		for _, name := range []string{"Red", "Blue", "Green", "Yellow"} {
			if strings.Contains(c, name) {
				counts[name]++
			}
		}
	}
	best, bn := "Red", -1
	for _, name := range []string{"Red", "Blue", "Green", "Yellow"} {
		if counts[name] > bn {
			bn, best = counts[name], name
		}
	}
	return Response{Protocol: Protocol, Kind: "move", Move: &Move{CardIndex: &idx, Color: best}, Reasoning: "reference uno"}
}

// Reference is a self-contained brain implementing tvcp-ai/1. Any other backend
// (Ollama, OpenAI, Anthropic, this model) just needs to produce the same shapes.
func Reference(req Request) Response {
	switch req.Kind {
	case "move":
		if req.Game == "wordle" {
			return refWordle(req)
		}
		if req.Game == "uno" {
			return refUno(req)
		}
		return refMove(req)
	case "draw":
		return refDraw(req)
	case "sketch":
		return refSketch(req)
	case "image":
		return refImage(req)
	case "react":
		return refReact(req)
	default:
		return Response{Protocol: Protocol, Error: "unknown kind: " + req.Kind}
	}
}

// Handler serves the reference brain over HTTP (POST tvcp-ai/1 JSON).
func Handler() http.HandlerFunc {
	return func(wr http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			wr.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(wr).Encode(Response{Protocol: Protocol, Error: "bad request: " + err.Error()})
			return
		}
		wr.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(wr).Encode(Reference(req))
	}
}

// Local is an in-process Brain using the reference policy (no network).
type Local struct{}

// Decide implements Brain.
func (Local) Decide(req Request) (Response, error) { return Reference(req), nil }
