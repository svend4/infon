package raydir

import (
	"image/color"
	"strings"
	"testing"
)

func TestYangCount(t *testing.T) {
	cases := map[int]int{0: 0, 0b111111: 6, 0b101010: 3, 0b1: 1}
	for n, want := range cases {
		if got := yangCount(n); got != want {
			t.Errorf("yangCount(%b) = %d, want %d", n, got, want)
		}
	}
}

func TestBubbleDOT(t *testing.T) {
	g, ids := sampleGraph()
	dot := g.DOT()
	if !strings.HasPrefix(dot, "graph bubbles {") || !strings.Contains(dot, "}") {
		t.Error("DOT should be a graphviz graph")
	}
	for _, id := range ids {
		if !strings.Contains(dot, "n"+itoa(id)+" [") {
			t.Errorf("DOT should declare node n%d", id)
		}
	}
	if strings.Count(dot, " -- ") != 4 { // sampleGraph has 4 transits
		t.Errorf("DOT should list 4 edges, got %d", strings.Count(dot, " -- "))
	}
}

func TestBubbleSVG(t *testing.T) {
	g, ids := sampleGraph()
	g.Layout(200)
	svg := g.SVG(ids[2], g.Route(ids[0], ids[3]), 400, 300)
	if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(svg, "</svg>") {
		t.Error("SVG should be a well-formed svg document")
	}
	if strings.Count(svg, "<circle") != g.Len() {
		t.Errorf("SVG should draw one circle per bubble, got %d", strings.Count(svg, "<circle"))
	}
	if !strings.Contains(svg, "#5adc8c") {
		t.Error("SVG should colour the route edges green")
	}
}

func TestQ6MapRenders(t *testing.T) {
	img := Q6Map(HexagramFromNumber(0b101010), 320, 320)
	if img.Bounds().Dx() != 320 {
		t.Fatalf("unexpected size %v", img.Bounds())
	}
	// the all-yang cell is gold-ish, the all-yin cell is dark blue: they differ
	if !hasColor(img, color.RGBA{R: 250, G: 250, B: 255}, 8) {
		t.Error("the current hexagram should be outlined in white")
	}
	if !hasColor(img, color.RGBA{R: 255, G: 216, B: 102}, 24) {
		t.Error("a high-yang cell should be gold-ish")
	}
}

// itoa avoids importing strconv just for the test.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
