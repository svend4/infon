package tangram7

import "sort"

// Figures returns named classic tangram silhouettes. Each value is a valid
// assembly of all seven pieces (verified by Verify: 7 pieces, area 16, ~zero
// overlap) and by the figures test (each piece congruent to its base shape).
// "square" is the canonical solution; "cat" and "swan" are authored animals.
func Figures() map[string][]Piece {
	return map[string][]Piece{
		"square": Square(),
		"cat":    cat(),
		"swan":   swan(),
	}
}

// FigureNames returns the figure names in a stable order.
func FigureNames() []string {
	out := make([]string, 0, 3)
	for k := range Figures() {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// cat is a sitting cat facing left: blue+teal large triangles form the seated
// body, the red square is the head, green+pink small triangles are the ears,
// the gold medium is a front paw (edge-joined to the chest) and the orange
// parallelogram is the upright tail.
func cat() []Piece {
	return []Piece{
		{"large1", "blue", Poly{{2, 8}, {2, 4}, {4, 6}}},
		{"large2", "teal", Poly{{2, 8}, {6, 8}, {4, 6}}},
		{"square", "red", Poly{{1, 3}, {2, 2}, {3, 3}, {2, 4}}},
		{"small1", "green", Poly{{1, 3}, {2, 2}, {1, 1}}},
		{"small2", "pink", Poly{{3, 3}, {2, 2}, {3, 1}}},
		{"medium", "gold", Poly{{0, 8}, {2, 8}, {2, 6}}},
		{"parallelogram", "orange", Poly{{6, 8}, {7, 7}, {7, 5}, {6, 6}}},
	}
}

// swan is a standing bird: blue+teal large triangles form the diamond body,
// the gold medium and red square sit on the back, the orange parallelogram is
// the leg, the green small the foot and the pink small the tail.
func swan() []Piece {
	return []Piece{
		{"large1", "blue", Poly{{1, 3}, {3, 1}, {3, 5}}},
		{"large2", "teal", Poly{{5, 3}, {3, 1}, {3, 5}}},
		{"medium", "gold", Poly{{1, 1}, {3, 1}, {1, 3}}},
		{"square", "red", Poly{{3, 1}, {4, 0}, {5, 1}, {4, 2}}},
		{"parallelogram", "orange", Poly{{3, 5}, {2, 6}, {2, 8}, {3, 7}}},
		{"small1", "green", Poly{{2, 8}, {0, 8}, {1, 9}}},
		{"small2", "pink", Poly{{5, 3}, {5, 5}, {6, 4}}},
	}
}
