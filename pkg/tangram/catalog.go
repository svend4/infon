package tangram

// The catalog proves the headline property: ONE small piece vocabulary, many
// figures. Every figure below is 8 rows × 9 columns, so any two can be morphed into each
// (see semframe). Each is authored as a grid of two-character cells — a shape
// char and a colour char — which keeps the silhouettes legible in source.
//
// Shape chars:  . empty   F █   q ◤   w ◥   a ◣   s ◢   T ▀   B ▄   L ▌   R ▐
//               / ▞   \ ▚
// Colour chars: a amber o orange g gold w white n navy t teal b blue k black
//               s slate r brown d darkgreen y yellow c coral l skyblue m mint
//               p purple e green u sand . none

var shapeOf = map[byte]string{
	'F': "full", 'q': "tri-ul", 'w': "tri-ur", 'a': "tri-ll", 's': "tri-lr",
	'T': "top", 'B': "bottom", 'L': "left", 'R': "right", '/': "diag-/", '\\': "diag-\\",
}

var colourOf = map[byte]string{
	'a': "amber", 'o': "orange", 'g': "gold", 'w': "white", 'n': "navy", 't': "teal",
	'b': "blue", 'k': "black", 's': "slate", 'r': "brown", 'd': "darkgreen", 'y': "yellow",
	'c': "coral", 'l': "skyblue", 'm': "mint", 'p': "purple", 'e': "green", 'u': "sand", '.': "",
}

// parseFig turns a grid of two-char cells into a Figure (single piece per cell).
func parseFig(name, bg string, rows []string) Figure {
	f := Figure{Name: name, Bg: bg, H: len(rows)}
	for _, r := range rows {
		if len(r)/2 > f.W {
			f.W = len(r) / 2
		}
	}
	for y, r := range rows {
		for x := 0; x*2+1 < len(r); x++ {
			sc, cc := r[x*2], r[x*2+1]
			g := shapeOf[sc]
			if g == "" { // '.' or space -> empty cell
				continue
			}
			f.Pieces = append(f.Pieces, Piece{Glyph: g, Y: y, X: x, Color: colourOf[cc]})
		}
	}
	return f
}

// pair places two complementary pieces in one cell (a two-colour diagonal).
func (f *Figure) pair(y, x int, g0, c0, g1, c1 string) {
	f.Pieces = append(f.Pieces,
		Piece{Glyph: g0, Y: y, X: x, Color: c0},
		Piece{Glyph: g1, Y: y, X: x, Color: c1},
	)
}

// Cat — an amber face on navy, rounded by corner triangles, white eyes, and a
// gold nose built from a complementary pair (gold over amber in one cell).
func Cat() Figure {
	f := parseFig("cat", "navy", []string{
		"..qa..........wa..",
		"..FaFaFaFaFaFa....",
		"..FaFwFaFaFwFa....",
		"..FaFa....FaFa....",
		"..FaFaFaFaFaFa....",
		"..waFaFaFaFaqa....",
		"..................",
		"..................",
	})
	f.pair(3, 3, "tri-lr", "gold", "tri-ul", "amber")
	f.pair(3, 4, "tri-ll", "gold", "tri-ur", "amber")
	return f
}

// Swan — a white body with a diagonal neck rising to the right, on blue water.
func Swan() Figure {
	return parseFig("swan", "navy", []string{
		"..........qw......",
		"........qwFw......",
		"......qwFw........",
		"....qwFwFw........",
		"..FwFwFwFwsw......",
		"qwFwFwFwFwFwww....",
		"FbFbFbFbFbFbFbFb..",
		"..................",
	})
}

// Boat — a triangular white sail above a brown hull that narrows to the keel.
func Boat() Figure {
	return parseFig("boat", "navy", []string{
		"........qw........",
		"......qwFw........",
		"....qwFwFw........",
		"..qwFwFwFw........",
		"........Lr........",
		"..srFrFrFrFrar....",
		"FbFbFbFbFbFbFbFb..",
		"..................",
	})
}

// Tree — a stacked-triangle pine in dark green over a brown trunk.
func Tree() Figure {
	return parseFig("tree", "navy", []string{
		"......sdad........",
		"......FdFd........",
		"....sdFdFdad......",
		"....FdFdFdFd......",
		"..sdFdFdFdFdad....",
		"......FrFr........",
		"......FrFr........",
		"..................",
	})
}

// House — a coral roof with slanted eaves over sand walls and a brown door.
func House() Figure {
	return parseFig("house", "navy", []string{
		"..................",
		"..srFcFcFcFcar....",
		"..FuFuFuFuFuFu....",
		"..FuFuFrFrFuFu....",
		"..FuFuFrFrFuFu....",
		"..................",
		"..................",
		"..................",
	})
}

// Catalog returns every named figure.
func Catalog() map[string]Figure {
	return map[string]Figure{
		"cat": Cat(), "swan": Swan(), "boat": Boat(), "tree": Tree(), "house": House(),
		"dog": Dog(), "fox": Fox(), "rabbit": Rabbit(), "fish": Fish(), "bird": Bird(),
		"duck": Duck(), "owl": Owl(), "butterfly": Butterfly(), "crab": Crab(),
		"turtle": Turtle(), "snake": Snake(),
	}
}

// Named returns a catalog figure by name and whether it exists.
func Named(name string) (Figure, bool) {
	f, ok := Catalog()[name]
	return f, ok
}

// --- the address book: eleven more animals, each a trigram-clean silhouette ---

// Dog - a brown dog seen from the side, ears up over a blocky body and legs.
func Dog() Figure {
	return parseFig("dog", "navy", []string{
		"..ar....wr........",
		"..FrFr..FrFr......",
		"..FrFrFrFrqr......",
		"..FrFrFrFr........",
		"..FrFrFrFrFr......",
		"..FrFrFrFrFr......",
		"..Fr..FrFr..Fr....",
		"..................",
	})
}

// Fox - an orange fox with a white chest, pointed ears and a swept tail.
func Fox() Figure {
	return parseFig("fox", "navy", []string{
		"..ao....wo........",
		"..FoFo..FoFo......",
		"..FoFoFoFoqo......",
		"..FoFwFwFo....wo..",
		"..FoFwFwFoFo..Foao",
		"..FoFoFoFoFo..Fo..",
		"..Fo..FoFo..Fo....",
		"..................",
	})
}

// Rabbit - two tall ears over a slate body with a single white eye.
func Rabbit() Figure {
	return parseFig("rabbit", "navy", []string{
		"..Fs..Fs..........",
		"..Fs..Fs..........",
		"..Fs..Fs..........",
		"..qsFsFsws........",
		"..FsFwFsFs........",
		"..FsFsFsFs........",
		"..Fs....Fs........",
		"..................",
	})
}

// Fish - a coral fish facing left, white eye, tail fanning to the right.
func Fish() Figure {
	return parseFig("fish", "navy", []string{
		"..................",
		"....qcFcFcwc......",
		"..qcFcFwFcFcwc....",
		"..FcFcFcFcFcFcwc..",
		"..acFcFcFcFcscsc..",
		"....acFcFcsc......",
		"..................",
		"..................",
	})
}

// Bird - a small blue songbird with an orange beak on brown legs.
func Bird() Figure {
	return parseFig("bird", "navy", []string{
		"......qbwb........",
		"....qbFbFbwb......",
		"....FbFwFbFb..ao..",
		"....FbFbFbFb......",
		"....qbFbFbwb......",
		"......FbFb........",
		"......ar..ar......",
		"..................",
	})
}

// Duck - a yellow duck facing left with an orange beak.
func Duck() Figure {
	return parseFig("duck", "navy", []string{
		"..................",
		"....qyFyFywy......",
		"..ao..FyFwFy......",
		"..FyFyFyFyFy......",
		"..FyFyFyFyFyFy....",
		"..qyFyFyFyFywy....",
		"..................",
		"..................",
	})
}

// Owl - a brown owl with amber eyes, an orange beak and tufted ears.
func Owl() Figure {
	return parseFig("owl", "navy", []string{
		"..qr......wr......",
		"..FrFrFrFrFr......",
		"..FrFaFrFaFr......",
		"..FrFaFrFaFr......",
		"..FrFrsoFrFr......",
		"..FrFrFrFrFr......",
		"..wrFrFrFrqr......",
		"..................",
	})
}

// Butterfly - four purple wings around an orange body, mirror-symmetric.
func Butterfly() Figure {
	return parseFig("butterfly", "navy", []string{
		"..qpwp..qpwp......",
		"..FpFpFoFpFp......",
		"..FpFpFoFpFp......",
		"..apsp..apsp......",
		"..qpwp..qpwp......",
		"..FpFpFoFpFp......",
		"..apsp..apsp......",
		"..................",
	})
}

// Crab - a coral crab: two claws raised over a wide body on splayed legs.
func Crab() Figure {
	return parseFig("crab", "navy", []string{
		"..................",
		"..qc......wc......",
		"..FcFc..FcFc......",
		"....qcFcFcwc......",
		"..FcFcFcFcFcFc....",
		"..ac..ac..ac..sc..",
		"..................",
		"..................",
	})
}

// Turtle - a dark-green domed shell over sand head and feet.
func Turtle() Figure {
	return parseFig("turtle", "navy", []string{
		"..................",
		"......qdFdwd......",
		"....qdFdFdFdwd....",
		"..qdFdFdFdFdFdwd..",
		"..FdFdFdFdFdFdFd..",
		"..au..FuFu..su....",
		"..................",
		"..................",
	})
}

// Snake - a green snake coiled in an S, head upper-left, tail lower-left.
func Snake() Figure {
	return parseFig("snake", "navy", []string{
		"..qeFewe..........",
		"..FeFeFe..........",
		"....FeFeFe........",
		"......FeFeFe......",
		"......FeFeFe......",
		"....FeFeFe........",
		"..FeFeFe..........",
		"..ae..............",
	})
}
