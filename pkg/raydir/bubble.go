package raydir

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/microfont"
	"github.com/svend4/infon/pkg/raytrace"
)

// bubble.go models the dream world the way the "dream hackers" map it: not a flat
// map but a "molecular structure of bubbles". Each bubble is a discrete place (a
// scene), bubbles connect through transits (instantaneous links — our portals),
// and one bubble is home, the anchor of the whole structure. The graph can be
// routed (shortest chain of transits between two places) and drawn as the bubble
// diagram the hackers sketch. Clean-room; pure data, so it's testable.

// Bubble is one place in the dream structure.
type Bubble struct {
	ID   int
	Name string
	At   raytrace.Vec3   // layout position (X,Z used on the diagram)
	Spec brain.SceneSpec // the bubble's scene (optional)
}

// BubbleGraph is the molecular structure of bubbles connected by transits.
type BubbleGraph struct {
	bubbles map[int]*Bubble
	links   map[int]map[int]bool // undirected adjacency
	order   []int                // insertion order (home first)
	nextID  int
}

// NewBubbleGraph returns an empty structure.
func NewBubbleGraph() *BubbleGraph {
	return &BubbleGraph{bubbles: map[int]*Bubble{}, links: map[int]map[int]bool{}}
}

// Add inserts a bubble, assigning the next ID. The first bubble added is home.
func (g *BubbleGraph) Add(name string, at raytrace.Vec3, spec brain.SceneSpec) int {
	id := g.nextID
	g.nextID++
	g.bubbles[id] = &Bubble{ID: id, Name: name, At: at, Spec: spec}
	g.links[id] = map[int]bool{}
	g.order = append(g.order, id)
	return id
}

// Link adds an undirected transit between two bubbles.
func (g *BubbleGraph) Link(a, b int) {
	if a == b || g.bubbles[a] == nil || g.bubbles[b] == nil {
		return
	}
	g.links[a][b] = true
	g.links[b][a] = true
}

// Home is the anchor bubble (the first added), or -1 if empty.
func (g *BubbleGraph) Home() int {
	if len(g.order) == 0 {
		return -1
	}
	return g.order[0]
}

// Get returns a bubble by ID.
func (g *BubbleGraph) Get(id int) (Bubble, bool) {
	b, ok := g.bubbles[id]
	if !ok {
		return Bubble{}, false
	}
	return *b, true
}

// Neighbors returns the bubbles reachable by one transit, sorted.
func (g *BubbleGraph) Neighbors(id int) []int {
	out := make([]int, 0, len(g.links[id]))
	for n := range g.links[id] {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// Bubbles returns every bubble in insertion order.
func (g *BubbleGraph) Bubbles() []Bubble {
	out := make([]Bubble, 0, len(g.order))
	for _, id := range g.order {
		out = append(out, *g.bubbles[id])
	}
	return out
}

// Len is the number of bubbles.
func (g *BubbleGraph) Len() int { return len(g.order) }

// Route is the shortest chain of transits from `from` to `to` (inclusive of both),
// or nil if unreachable. Breadth-first, so it minimises the number of transits.
func (g *BubbleGraph) Route(from, to int) []int {
	if g.bubbles[from] == nil || g.bubbles[to] == nil {
		return nil
	}
	if from == to {
		return []int{from}
	}
	prev := map[int]int{from: from}
	queue := []int{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			break
		}
		for _, n := range g.Neighbors(cur) {
			if _, seen := prev[n]; !seen {
				prev[n] = cur
				queue = append(queue, n)
			}
		}
	}
	if _, ok := prev[to]; !ok {
		return nil
	}
	var path []int
	for n := to; ; n = prev[n] {
		path = append([]int{n}, path...)
		if n == from {
			break
		}
	}
	return path
}

// drawLine rasterises a line into an RGBA image (DDA).
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx, dy := x1-x0, y1-y0
	steps := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))
	if steps == 0 {
		img.SetRGBA(x0, y0, c)
		return
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		img.SetRGBA(x0+int(float64(dx)*t+0.5), y0+int(float64(dy)*t+0.5), c)
	}
}

// fillCircle draws a filled disc.
func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.SetRGBA(cx+x, cy+y, c)
			}
		}
	}
}

// BubbleMap draws the bubble structure: transits as lines, bubbles as discs (home
// gold, the current bubble cyan, dead-ends dim), numbered and named; if route is
// non-empty its transits are highlighted. It is the hackers' bubble diagram.
func (g *BubbleGraph) BubbleMap(current int, route []int, w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 16, G: 16, B: 22, A: 255}}, image.Point{}, draw.Src)
	if len(g.order) == 0 {
		return img
	}
	minX, maxX := math.Inf(1), math.Inf(-1)
	minZ, maxZ := math.Inf(1), math.Inf(-1)
	for _, b := range g.bubbles {
		minX, maxX = math.Min(minX, b.At.X), math.Max(maxX, b.At.X)
		minZ, maxZ = math.Min(minZ, b.At.Z), math.Max(maxZ, b.At.Z)
	}
	pad := 28.0
	sx := func(x float64) int {
		if maxX == minX {
			return w / 2
		}
		return int(pad + (x-minX)/(maxX-minX)*(float64(w)-2*pad))
	}
	sy := func(z float64) int {
		if maxZ == minZ {
			return h / 2
		}
		return int(pad + (z-minZ)/(maxZ-minZ)*(float64(h)-2*pad))
	}
	routeEdge := map[[2]int]bool{}
	for i := 0; i+1 < len(route); i++ {
		a, b := route[i], route[i+1]
		routeEdge[[2]int{a, b}] = true
		routeEdge[[2]int{b, a}] = true
	}
	// edges
	for _, a := range g.order {
		for _, b := range g.Neighbors(a) {
			if a >= b {
				continue // draw each undirected edge once
			}
			col := color.RGBA{R: 70, G: 70, B: 90, A: 255}
			if routeEdge[[2]int{a, b}] {
				col = color.RGBA{R: 90, G: 220, B: 140, A: 255}
			}
			drawLine(img, sx(g.bubbles[a].At.X), sy(g.bubbles[a].At.Z), sx(g.bubbles[b].At.X), sy(g.bubbles[b].At.Z), col)
		}
	}
	// nodes
	for _, id := range g.order {
		b := g.bubbles[id]
		cx, cy := sx(b.At.X), sy(b.At.Z)
		col := color.RGBA{R: 130, G: 150, B: 200, A: 255}
		if len(g.links[id]) <= 1 {
			col = color.RGBA{R: 80, G: 80, B: 95, A: 255} // dead-end bubble, dim
		}
		if id == g.Home() {
			col = color.RGBA{R: 240, G: 200, B: 80, A: 255} // home, gold
		}
		if id == current {
			col = color.RGBA{R: 90, G: 220, B: 240, A: 255} // here, cyan
		}
		fillCircle(img, cx, cy, 7, col)
		microfont.Draw(img, cx+9, cy-3, 1, b.Name, color.RGBA{R: 220, G: 220, B: 230, A: 255})
	}
	return img
}
