// Command arenaplay is LIVE, animated terminal graphics: it plays an arena battle
// out frame by frame, redrawing in place with ANSI escapes (cursor-home, no
// scroll) so you watch the tangram-block armies move and fight in real time.
// This is the project's "terminal video" thesis as motion, not a still.
//
//	go run ./cmd/arenaplay              # ~12 fps until one side wins
//	go run ./cmd/arenaplay -fps 20 -seed 3
//
// Works in any VT/ANSI terminal (Windows Terminal, modern PowerShell, xterm).
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/tangram"
)

const (
	reset = "\x1b[0m"
	blue  = "\x1b[38;5;33m"
	red   = "\x1b[38;5;196m"
	dim   = "\x1b[38;5;240m"
	bold  = "\x1b[1m"
)

func build(seed int64) *arena.Arena {
	a := &arena.Arena{W: 20, H: 12, Terrain: make([]uint8, 20*12)}
	cat := tangram.Catalog()
	r := rand.New(rand.NewSource(seed))
	roster := []string{"cat", "fox", "owl", "crab", "turtle", "swan"}
	add := func(name string, f, x, y uint8) {
		if fig, ok := cat[name]; ok {
			if u, _, ok2 := arena.Summon(fig, f, x, y); ok2 {
				a.Units = append(a.Units, u)
			}
		}
	}
	for i := 0; i < 5; i++ {
		add(roster[r.Intn(len(roster))], 0, 2, uint8(1+i*2))
		add(roster[r.Intn(len(roster))], 1, 17, uint8(1+i*2))
	}
	return a
}

// frame renders one snapshot, addressed from the cursor home position so each
// frame overwrites the previous one (no scrolling).
func frame(a *arena.Arena, tick int) string {
	at := map[[2]int]uint8{} // x,y -> faction+1
	for _, u := range a.Units {
		if u.Alive && int(u.X) < a.W && int(u.Y) < a.H {
			at[[2]int{int(u.X), int(u.Y)}] = u.Faction + 1
		}
	}
	var b strings.Builder
	b.WriteString("\x1b[H") // cursor home
	b.WriteString(fmt.Sprintf("%sTVCP arena%s  tick %s%3d%s   %s██ blue %d%s  vs  %s██ red %d%s        \r\n",
		bold, reset, bold, tick, reset, blue, a.AliveCount(0), reset, red, a.AliveCount(1), reset))
	b.WriteString(dim + "+" + strings.Repeat("--", a.W) + "+" + reset + "\r\n")
	for y := 0; y < a.H; y++ {
		b.WriteString(dim + "|" + reset)
		for x := 0; x < a.W; x++ {
			switch at[[2]int{x, y}] {
			case 1:
				b.WriteString(blue + "██" + reset)
			case 2:
				b.WriteString(red + "██" + reset)
			default:
				b.WriteString(dim + " ·" + reset)
			}
		}
		b.WriteString(dim + "|" + reset + "\r\n")
	}
	b.WriteString(dim + "+" + strings.Repeat("--", a.W) + "+" + reset + "\r\n")
	return b.String()
}

func main() {
	fps := flag.Int("fps", 12, "frames per second")
	seed := flag.Int64("seed", 7, "battlefield seed")
	maxTurns := flag.Int("turns", 300, "safety cap on ticks")
	flag.Parse()
	if *fps < 1 {
		*fps = 1
	}

	a := build(*seed)
	fmt.Print("\x1b[2J\x1b[?25l")        // clear screen, hide cursor
	defer fmt.Print("\x1b[?25h" + reset) // restore cursor on exit

	delay := time.Second / time.Duration(*fps)
	tick := 0
	for ; tick < *maxTurns; tick++ {
		fmt.Print(frame(a, tick))
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			break
		}
		a.Step(arena.RefCommander{}, arena.RefCommander{})
		time.Sleep(delay)
	}

	winner := "a draw"
	switch {
	case a.AliveCount(0) > a.AliveCount(1):
		winner = blue + "blue holds the field" + reset
	case a.AliveCount(1) > a.AliveCount(0):
		winner = red + "red holds the field" + reset
	}
	fmt.Printf("\r\n%sresult:%s %s after %d ticks.\r\n", bold, reset, winner, tick)
}
