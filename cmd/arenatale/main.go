// Command arenatale plays an arena match, distils it into a few-byte chronicle
// (pkg/chronicle), and retells that byte-log as prose — the project's chain
// number -> image -> world -> story, end to end. It prints the chronicle and
// writes it to a text file.
//
//	go run ./cmd/arenatale
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/chronicle"
)

func board() *arena.Arena {
	const W, H = 9, 5
	a := &arena.Arena{W: W, H: H, Terrain: make([]uint8, W*H)}
	for y := 0; y < H; y++ {
		a.Terrain[y*W+4] = 3
	}
	for n, k := range []uint8{4, 3, 5} {
		y := uint8(1 + n)
		a.Units = append(a.Units,
			arena.Unit{Kind: k, X: 1, Y: y, HP: arena.Bestiary[k].HP, Faction: 0, Alive: true},
			arena.Unit{Kind: k, X: 7, Y: y, HP: arena.Bestiary[k].HP, Faction: 1, Alive: true},
		)
	}
	return a
}

func main() {
	out := flag.String("out", "_arena", "output directory")
	ticks := flag.Int("ticks", 60, "max ticks")
	flag.Parse()
	_ = os.MkdirAll(*out, 0o755)

	l := chronicle.Record(board(), arena.Planner{}, arena.RefCommander{}, *ticks)
	story := l.Narrate()
	fmt.Printf("the whole war in %d bytes (%d events) — then retold:\n\n", len(l.Bytes()), len(l.Events))
	fmt.Print(story)

	path := fmt.Sprintf("%s/chronicle.txt", *out)
	doc := fmt.Sprintf("Arena chronicle — %d bytes of war, %d events\n\n%s",
		len(l.Bytes()), len(l.Events), story)
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("\nwrote %s\n", path)
}
