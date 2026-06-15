// Package bench is a self-contained technical test-bench for the tvcp-ai layer:
// it exercises a battery of real in-process capabilities (the protocol, the
// games, the symbolic substrate, security, the directory, and the agent-world)
// and returns a structured Report. The same Report renders to the terminal
// (cmd/testbench), serializes to JSON (for tools, dashboards, or another model),
// and feeds the Cowork/browser panels — one engine, many faces.
package bench

import (
	"bytes"
	"fmt"
	"time"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/glyphqr"
	"github.com/svend4/infon/pkg/registry"
	"github.com/svend4/infon/pkg/republic"
	"github.com/svend4/infon/pkg/session"
	"github.com/svend4/infon/pkg/sign"
	"github.com/svend4/infon/pkg/tangram"
)

// Check is one capability probe and its outcome.
type Check struct {
	Group  string `json:"group"`
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Metric string `json:"metric"`           // the headline number
	Detail string `json:"detail,omitempty"` // a secondary note
	Bytes  int    `json:"bytes,omitempty"`  // payload/budget where meaningful
	Millis int64  `json:"ms"`
}

// Report is the whole bench run.
type Report struct {
	Protocol string  `json:"protocol"`
	When     string  `json:"when"`
	Passed   int     `json:"passed"`
	Total    int     `json:"total"`
	Checks   []Check `json:"checks"`
}

// probe returns (pass, metric, detail, bytes).
type probe func() (bool, string, string, int)

func timed(group, name string, fn probe) Check {
	t0 := time.Now()
	pass, metric, detail, by := fn()
	return Check{Group: group, Name: name, Pass: pass, Metric: metric, Detail: detail, Bytes: by, Millis: time.Since(t0).Milliseconds()}
}

// Run executes the full battery and returns the Report.
func Run() Report {
	checks := []Check{
		timed("protocol", "conformance", checkConformance),
		timed("games", "hanabi", checkHanabi),
		timed("games", "uno", checkUno),
		timed("substrate", "tangram-hard", checkTangram),
		timed("substrate", "glyphqr-loss", checkGlyphQR),
		timed("security", "sign", checkSign),
		timed("directory", "registry", checkRegistry),
		timed("agent-world", "arena", checkArena),
		timed("agent-world", "republic", checkRepublic),
	}
	r := Report{Protocol: brain.Protocol, When: time.Now().UTC().Format(time.RFC3339), Checks: checks}
	for _, c := range checks {
		r.Total++
		if c.Pass {
			r.Passed++
		}
	}
	return r
}

func refDecide(req brain.Request) (brain.Response, error) { return brain.Reference(req), nil }

func checkConformance() (bool, string, string, int) {
	res, ok := brain.RunConformance(refDecide)
	n := 0
	for _, x := range res {
		if x.Pass {
			n++
		}
	}
	return ok, fmt.Sprintf("%d/%d", n, len(res)), "tvcp-ai/1 battery", 0
}

func twoRef() []session.Participant {
	return []session.Participant{
		session.BrainPlayer{N: "ref-A", B: brain.Local{}},
		session.BrainPlayer{N: "ref-B", B: brain.Local{}},
	}
}

func checkHanabi() (bool, string, string, int) {
	r, err := session.Play[session.HanabiState](session.Hanabi{Seed: 7}, twoRef(), 400)
	if err != nil {
		return false, "error", err.Error(), 0
	}
	score := session.HanabiScore(r.Final)
	total := 0
	for _, t := range r.Transcript {
		total += len(t.Move)
	}
	bpt := 0
	if r.Turns > 0 {
		bpt = total / r.Turns
	}
	pass := score > 0 && r.Final.Fuses == 3
	return pass, fmt.Sprintf("%d/25 · %d fuses", score, r.Final.Fuses), fmt.Sprintf("%d B/turn cooperative", bpt), bpt
}

func checkUno() (bool, string, string, int) {
	r, err := session.Play[session.UnoState](session.Uno{Seed: 7}, twoRef(), 3000)
	if err != nil {
		return false, "error", err.Error(), 0
	}
	return r.Winner >= 0, fmt.Sprintf("winner P%d", r.Winner), fmt.Sprintf("%d turns, hidden info", r.Turns), 0
}

func checkTangram() (bool, string, string, int) {
	cat := tangram.Catalog()
	var sumE, sumH float64
	n := 0
	for _, f := range cat {
		ps := tangram.PuzzleFromFigure(f)
		hp := tangram.HardFromFigure(f)
		sumE += tangram.ScorePuzzle(ps, tangram.Solve(ps)).IoU
		sumH += tangram.ScorePuzzle(ps, tangram.SolveHard(hp)).IoU
		n++
	}
	if n == 0 {
		return false, "no figures", "", 0
	}
	me, mh := sumE/float64(n), sumH/float64(n)
	return mh > 0.9 && me > 0.99, fmt.Sprintf("IoU hard %.3f", mh), fmt.Sprintf("easy %.3f over %d figures", me, n), 0
}

func checkGlyphQR() (bool, string, string, int) {
	data := []byte("tvcp-ai: transmit meaning, not pixels 0123456789")
	ecc := 10
	w := glyphqr.RSEncodeBytes(data, ecc)
	cw := append([]byte(nil), w...)
	nerr := 4 // well within Reed-Solomon's ecc/2 correction budget, spread by interleaving
	for i := 0; i < nerr && len(cw) > 0; i++ {
		cw[(i*11+5)%len(cw)] ^= 0xFF
	}
	got, err := glyphqr.RSDecodeBytes(cw, ecc)
	pass := err == nil && bytes.Equal(got, data)
	return pass, fmt.Sprintf("recovered %d errors", nerr), fmt.Sprintf("ecc=%d, %d-byte word", ecc, len(w)), len(w)
}

func checkSign() (bool, string, string, int) {
	key := []byte("shared-secret-key")
	payload := []byte(`{"protocol":"tvcp-ai/1","kind":"move","game":"hanabi"}`)
	s := sign.Sign(brain.Protocol, key, payload)
	good := sign.Verify(key, payload, s, brain.Protocol)
	tampered := append([]byte(nil), payload...)
	tampered[10] ^= 0x20
	badMsg := sign.Verify(key, tampered, s, brain.Protocol)
	badVer := sign.Verify(key, payload, s, "tvcp-ai/2")
	pass := good && !badMsg && !badVer
	return pass, "verify ok · tamper rejected", "HMAC-SHA256 + version", 0
}

func checkRegistry() (bool, string, string, int) {
	r := registry.New()
	r.Register(registry.Entry{Name: "haiku", URL: "http://h/v1/decide", Kinds: []string{"move", "image"}, Games: []string{"hanabi", "tangram"}})
	r.Register(registry.Entry{Name: "ollama", URL: "http://o/v1/decide", Kinds: []string{"move"}, Games: []string{"uno"}})
	hits := r.Find("move", "hanabi")
	pass := len(hits) == 1 && hits[0].Name == "haiku"
	return pass, fmt.Sprintf("find move/hanabi → %d", len(hits)), "capability lookup", 0
}

func checkArena() (bool, string, string, int) {
	cat := tangram.Catalog()
	a := &arena.Arena{W: 12, H: 8, Terrain: make([]uint8, 12*8)}
	add := func(name string, f uint8, x, y uint8) {
		if fig, ok := cat[name]; ok {
			if u, _, ok2 := arena.Summon(fig, f, x, y); ok2 {
				a.Units = append(a.Units, u)
			}
		}
	}
	add("cat", 0, 2, 2)
	add("fox", 0, 2, 5)
	add("owl", 1, 9, 2)
	add("crab", 1, 9, 5)
	if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
		return false, "no armies", "summon failed", 0
	}
	hp0 := a.TotalHP()
	ticks := 0
	for ; ticks < 80 && a.AliveCount(0) > 0 && a.AliveCount(1) > 0; ticks++ {
		a.Step(arena.RefCommander{}, arena.RefCommander{})
	}
	snap := len(a.Snapshot())
	return a.TotalHP() < hp0, fmt.Sprintf("%d ticks · %d vs %d", ticks, a.AliveCount(0), a.AliveCount(1)), fmt.Sprintf("snapshot %d B", snap), snap
}

// checkRepublic exercises the self-organizing world: brains register, an ACL
// Contract-Net round awards the two faction seats to distinct lowest bidders,
// and the arena is then fought out by those awarded commanders.
func checkRepublic() (bool, string, string, int) {
	cands := []republic.Candidate{
		{Name: "scout", Brain: brain.Local{}, Cost: func(int) int { return 3 }},
		{Name: "planner", Brain: brain.Local{}, Cost: func(int) int { return 4 }},
		{Name: "warden", Brain: brain.Local{}, Cost: func(int) int { return 5 }},
	}
	r := republic.Convene(cands, 7)
	ok := len(r.Seats) == 2 && r.Seats[0].Awarded && r.Seats[1].Awarded && r.Seats[0].Brain != r.Seats[1].Brain && r.Ticks > 0
	win := "draw"
	switch r.Winner {
	case 0:
		win = "faction 0"
	case 1:
		win = "faction 1"
	}
	return ok, "2 seats negotiated", fmt.Sprintf("%s · %d ticks", win, r.Ticks), 0
}
