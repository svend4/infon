package tangram

import (
	"fmt"
	"math/rand"
)

// Creature procedurally assembles a silhouette from the piece vocabulary by a
// small shape grammar. The seed picks one of four BODY PLANS — quadruped, bird,
// fish, biped — plus colour and facing, so a batch of seeds yields a varied
// bestiary (a gallery wall of distinct shapes), each well-formed by construction.
func Creature(seed int64) Figure {
	rng := rand.New(rand.NewSource(seed))
	cols := []string{"amber", "teal", "coral", "mint", "gold", "skyblue", "purple", "green", "orange"}
	col := cols[rng.Intn(len(cols))]
	plan := rng.Intn(4)
	faceRight := rng.Intn(2) == 0

	var ps []Piece
	switch plan {
	case 0:
		ps = quadruped(rng, col)
	case 1:
		ps = bird(col)
	case 2:
		ps = fish(col)
	default:
		ps = biped(col)
	}
	if !faceRight && plan != 3 { // biped is symmetric
		ps = mirrorPieces(ps)
	}
	return creatureFrom(fmt.Sprintf("creature-%d", seed), ps)
}

func creatureFrom(name string, ps []Piece) Figure {
	maxX, maxY := 0, 0
	for _, p := range ps {
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return Figure{Name: name, Bg: "navy", H: maxY + 2, W: maxX + 2, Pieces: ps}
}

// mirrorPieces flips a facing-right plan to face left (glyphs mirrored too).
func mirrorPieces(ps []Piece) []Piece {
	maxX := 0
	for _, p := range ps {
		if p.X > maxX {
			maxX = p.X
		}
	}
	out := make([]Piece, len(ps))
	for i, p := range ps {
		g := mirrorXName[p.Glyph]
		if g == "" && p.Glyph != "" {
			g = p.Glyph
		}
		out[i] = Piece{Glyph: g, Y: p.Y, X: maxX - p.X, Color: p.Color}
	}
	return out
}

func quadruped(rng *rand.Rand, col string) []Piece {
	bodyW := 4 + rng.Intn(3)
	bodyH := 2 + rng.Intn(2)
	tailUp := rng.Intn(2) == 0
	ox, oy := 2, 3
	var ps []Piece
	add := func(g string, y, x int) { ps = append(ps, Piece{Glyph: g, Y: y, X: x, Color: col}) }
	for y := 0; y < bodyH; y++ {
		for x := 0; x < bodyW; x++ {
			add("full", oy+y, ox+x)
		}
	}
	hx := ox + bodyW - 1
	add("full", oy-1, hx)
	add("full", oy-2, hx)
	add("tri-lr", oy-2, hx+1) // snout
	add("tri-ul", oy-3, hx)   // ear
	if tailUp {
		add("tri-ul", oy, ox-1)
	} else {
		add("tri-ll", oy, ox-1)
	}
	for x := 0; x < bodyW; x += 2 {
		add("full", oy+bodyH, ox+x)
	}
	return ps
}

func bird(col string) []Piece {
	ox, oy := 2, 3
	var ps []Piece
	add := func(g string, y, x int) { ps = append(ps, Piece{Glyph: g, Y: y, X: x, Color: col}) }
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			add("full", oy+y, ox+x)
		}
	}
	add("full", oy-1, ox+2)   // head
	add("tri-lr", oy-1, ox+3) // beak
	add("tri-ur", oy-1, ox+1) // wing
	add("tri-ur", oy, ox-1)   // tail fan
	add("tri-lr", oy+1, ox-1)
	add("full", oy+2, ox+1) // leg
	return ps
}

func fish(col string) []Piece {
	ox, oy := 3, 3
	var ps []Piece
	add := func(g string, y, x int) { ps = append(ps, Piece{Glyph: g, Y: y, X: x, Color: col}) }
	add("tri-ul", oy, ox)
	add("full", oy, ox+1)
	add("full", oy, ox+2)
	add("tri-ur", oy, ox+3)
	add("tri-ll", oy+1, ox)
	add("full", oy+1, ox+1)
	add("full", oy+1, ox+2)
	add("tri-lr", oy+1, ox+3)
	add("tri-ur", oy, ox-1) // tail fan
	add("tri-lr", oy+1, ox-1)
	add("tri-ll", oy-1, ox+1) // top fin
	return ps
}

func biped(col string) []Piece {
	ox, oy := 3, 2
	var ps []Piece
	add := func(g string, y, x int) { ps = append(ps, Piece{Glyph: g, Y: y, X: x, Color: col}) }
	add("full", oy, ox+1)   // head
	add("full", oy+1, ox)   // arm
	add("full", oy+1, ox+1) // torso top
	add("full", oy+1, ox+2) // arm
	add("full", oy+2, ox+1) // torso
	add("full", oy+3, ox)   // leg L
	add("full", oy+4, ox)
	add("full", oy+3, ox+2) // leg R
	add("full", oy+4, ox+2)
	return ps
}
