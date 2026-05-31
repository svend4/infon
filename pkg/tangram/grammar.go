package tangram

import (
	"fmt"
	"math/rand"
)

// Creature procedurally assembles a four-legged silhouette from the piece
// vocabulary by a tiny shape grammar: a rectangular body, a neck + head + snout
// + ear on the facing side, a swept-diagonal tail on the other, and evenly
// spaced legs. Everything is derived deterministically from seed, so the same
// seed always yields the same animal — a reproducible procedural bestiary. Each
// creature is well-formed by construction (one piece per cell).
func Creature(seed int64) Figure {
	rng := rand.New(rand.NewSource(seed))
	bodyW := 4 + rng.Intn(3) // 4..6
	bodyH := 2 + rng.Intn(2) // 2..3
	faceRight := rng.Intn(2) == 0
	tailUp := rng.Intn(2) == 0
	cols := []string{"amber", "teal", "coral", "mint", "gold", "skyblue", "purple", "green"}
	col := cols[rng.Intn(len(cols))]

	var ps []Piece
	add := func(g string, y, x int, c string) { ps = append(ps, Piece{Glyph: g, Y: y, X: x, Color: c}) }

	ox, oy := 2, 3 // room: ear/head above, snout/tail to the sides, legs below

	// Body: a solid rectangle.
	for y := 0; y < bodyH; y++ {
		for x := 0; x < bodyW; x++ {
			add("full", oy+y, ox+x, col)
		}
	}

	// Head on the facing side: neck + head stacked above, snout out, ear on top.
	hx := ox + bodyW - 1
	if !faceRight {
		hx = ox
	}
	add("full", oy-1, hx, col)
	add("full", oy-2, hx, col)
	if faceRight {
		add("tri-lr", oy-2, hx+1, col)
		add("tri-ul", oy-3, hx, col)
	} else {
		add("tri-ll", oy-2, hx-1, col)
		add("tri-ur", oy-3, hx, col)
	}

	// Tail: a diagonal sweeping off the rump, just OUTSIDE the body, so it never
	// collides with a leg or the body. Up/down varies with the seed.
	if faceRight {
		g := "tri-ll"
		if tailUp {
			g = "tri-ul"
		}
		add(g, oy, ox-1, col)
	} else {
		g := "tri-lr"
		if tailUp {
			g = "tri-ur"
		}
		add(g, oy, ox+bodyW, col)
	}

	// Legs: evenly spaced stubs below the body, with gaps between them.
	legY := oy + bodyH
	for x := 0; x < bodyW; x += 2 {
		add("full", legY, ox+x, col)
	}

	maxX, maxY := 0, 0
	for _, p := range ps {
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return Figure{
		Name:   fmt.Sprintf("creature-%d", seed),
		Bg:     "navy",
		H:      maxY + 2,
		W:      maxX + 2,
		Pieces: ps,
	}
}
