// Package blazon compiles a (simplified) heraldic blazon — the medieval language
// for describing a coat of arms — into a pkg/pseudo spec that renders in the
// terminal. Heraldry was an ultra-low-bandwidth visual-identity system: a knight
// recognizable at a distance from a few symbols on a colored field. That is
// exactly tvcp's problem, a thousand years early. So a blazon is, quite
// literally, a pseudo-image: "Azure, a sun Or" -> a blue field with a gold sun.
package blazon

import (
	"math"
	"strings"

	"github.com/svend4/infon/pkg/pseudo"
)

// tincture maps heraldic colors (and a few plain names) to pseudo palette names.
var tincture = map[string]string{
	"or": "gold", "argent": "white", "gules": "red", "azure": "blue",
	"vert": "green", "sable": "black", "purpure": "purple", "tenne": "orange",
	"sanguine": "coral", "murrey": "purple",
	"gold": "gold", "silver": "white", "white": "white", "red": "red",
	"blue": "blue", "green": "green", "black": "black", "purple": "purple",
	"navy": "navy", "teal": "teal",
}

// charge maps heraldic charges to pseudo sigil names.
var charge = map[string]string{
	"sun": "sun", "moon": "moon", "crescent": "moon",
	"star": "star", "mullet": "star", "estoile": "star", "etoile": "star",
	"heart": "heart", "rose": "flower", "flower": "flower",
	"fleur": "fleur", "lily": "fleur",
	"anchor": "anchor", "key": "key", "sword": "sword", "crown": "crown",
	"tower": "tower", "castle": "tower", "lion": "lion", "ship": "boat",
	"boat": "boat", "tree": "tree", "mountain": "mountain", "fish": "fish",
	"shield": "shield", "flag": "flag", "wave": "wave", "bishop": "bishop",
	"knight": "knight",
}

// ordinary keywords are geometric charges painted onto the field.
var ordinary = map[string]bool{
	"chief": true, "base": true, "fess": true, "pale": true, "bend": true,
	"cross": true, "saltire": true, "bordure": true, "chevron": true,
}

var number = map[string]int{
	"a": 1, "an": 1, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6,
}

var position = map[string][2]float64{ // normalized centers on the shield
	"chief": {0.5, 0.20}, "base": {0.5, 0.80}, "dexter": {0.24, 0.5},
	"sinister": {0.76, 0.5}, "fess": {0.5, 0.5}, "center": {0.5, 0.5},
	"centre": {0.5, 0.5}, "honour": {0.5, 0.34}, "nombril": {0.5, 0.66},
}

func chargeOf(w string) (string, bool) {
	if c, ok := charge[w]; ok {
		return c, true
	}
	if len(w) > 1 && strings.HasSuffix(w, "s") {
		if c, ok := charge[w[:len(w)-1]]; ok {
			return c, true
		}
	}
	return "", false
}

func tokenize(s string) []string {
	var out []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func contrastOf(field string) string {
	switch field {
	case "gold", "white", "yellow", "coral":
		return "red"
	default:
		return "gold"
	}
}

const fsize = 16 // field grid resolution

func newField(color string) *pseudo.Grid {
	rows := make([][]string, fsize)
	for y := 0; y < fsize; y++ {
		row := make([]string, fsize)
		for x := 0; x < fsize; x++ {
			row[x] = color
		}
		rows[y] = row
	}
	return &pseudo.Grid{Rows: rows}
}

func paintOrdinary(g *pseudo.Grid, name, color string) {
	set := func(pred func(fx, fy float64) bool) {
		for y := 0; y < fsize; y++ {
			for x := 0; x < fsize; x++ {
				fx := float64(x) / float64(fsize-1)
				fy := float64(y) / float64(fsize-1)
				if pred(fx, fy) {
					g.Rows[y][x] = color
				}
			}
		}
	}
	switch name {
	case "chief":
		set(func(fx, fy float64) bool { return fy < 0.22 })
	case "base":
		set(func(fx, fy float64) bool { return fy > 0.78 })
	case "fess":
		set(func(fx, fy float64) bool { return fy > 0.40 && fy < 0.60 })
	case "pale":
		set(func(fx, fy float64) bool { return fx > 0.40 && fx < 0.60 })
	case "bend":
		set(func(fx, fy float64) bool { return math.Abs(fx-fy) < 0.13 })
	case "cross":
		set(func(fx, fy float64) bool { return (fx > 0.40 && fx < 0.60) || (fy > 0.40 && fy < 0.60) })
	case "saltire":
		set(func(fx, fy float64) bool { return math.Abs(fx-fy) < 0.13 || math.Abs(fx-(1-fy)) < 0.13 })
	case "bordure":
		set(func(fx, fy float64) bool { return fx < 0.08 || fx > 0.92 || fy < 0.08 || fy > 0.92 })
	case "chevron":
		set(func(fx, fy float64) bool { return math.Abs(math.Abs(fx-0.5)-(1-fy)*0.5) < 0.08 && fy > 0.25 })
	}
}

func placeCharges(out *[]pseudo.Sigil, name, color string, pos [2]float64, n int) {
	if n <= 1 {
		*out = append(*out, pseudo.Sigil{Name: name, X: pos[0], Y: pos[1], Color: color})
		return
	}
	// spread n charges horizontally around the position
	for i := 0; i < n; i++ {
		t := (float64(i)+0.5)/float64(n)*0.5 + 0.25 // 0.25..0.75
		*out = append(*out, pseudo.Sigil{Name: name, X: t, Y: pos[1], Color: color})
	}
}

func lookTincture(words []string, from int) string {
	for j := from; j < len(words); j++ {
		if t, ok := tincture[words[j]]; ok {
			return t
		}
		if ordinary[words[j]] {
			return ""
		}
		if _, ok := chargeOf(words[j]); ok {
			return ""
		}
	}
	return ""
}

func lookPosition(words []string, from int) [2]float64 {
	for j := from; j+1 < len(words); j++ {
		if words[j] == "in" {
			if p, ok := position[words[j+1]]; ok {
				return p
			}
		}
		if ordinary[words[j]] {
			break
		}
		if _, ok := chargeOf(words[j]); ok {
			break
		}
	}
	return position["center"]
}

// Parse compiles a blazon string into a renderable pseudo.Spec (mixed format:
// a tinctured field with ordinaries, plus charges as sigils).
func Parse(s string) pseudo.Spec {
	words := tokenize(s)
	field := "azure"
	for _, w := range words {
		if t, ok := tincture[w]; ok {
			field = t
			break
		}
	}
	fieldColor := tincture[field]
	if fieldColor == "" {
		fieldColor = field
	}
	g := newField(fieldColor)
	contrast := contrastOf(fieldColor)

	var sigils []pseudo.Sigil
	curN := 1
	for i := 0; i < len(words); i++ {
		w := words[i]
		if nm, ok := number[w]; ok {
			curN = nm
			continue
		}
		if ordinary[w] {
			oc := lookTincture(words, i+1)
			if oc == "" {
				oc = contrast
			}
			paintOrdinary(g, w, oc)
			curN = 1
			continue
		}
		if ch, ok := chargeOf(w); ok {
			cc := lookTincture(words, i+1)
			if cc == "" {
				cc = contrast
			}
			n := curN
			if n < 1 {
				n = 1
			}
			if n > 6 {
				n = 6
			}
			placeCharges(&sigils, ch, cc, lookPosition(words, i+1), n)
			curN = 1
			continue
		}
	}

	return pseudo.Spec{
		Format: pseudo.FormatMixed,
		Cols:   48,
		Rows:   24,
		Title:  strings.TrimSpace(s),
		Mixed:  &pseudo.Mixed{Grid: g, Sigils: sigils},
	}
}
