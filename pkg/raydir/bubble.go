package raydir

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"
	"strings"

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

// DOT exports the structure as a Graphviz `graph` (home filled gold, dead-ends
// grey) — the way svend4/meta's hexvis exports subgraphs, for `dot -Tsvg` etc.
func (g *BubbleGraph) DOT() string {
	var b strings.Builder
	b.WriteString("graph bubbles {\n  node [style=filled,fontname=mono];\n")
	for _, id := range g.order {
		fill := "lightsteelblue"
		if len(g.links[id]) <= 1 {
			fill = "gray70"
		}
		if id == g.Home() {
			fill = "gold"
		}
		fmt.Fprintf(&b, "  n%d [label=%q,fillcolor=%s];\n", id, g.bubbles[id].Name, fill)
	}
	for _, a := range g.order {
		for _, c := range g.Neighbors(a) {
			if a < c {
				fmt.Fprintf(&b, "  n%d -- n%d;\n", a, c)
			}
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// SVG exports the structure as a standalone scalable diagram (transits as lines,
// route edges green, bubbles as discs with labels) using the laid-out positions —
// a vector companion to BubbleMap (cf. hexvis SVG export).
func (g *BubbleGraph) SVG(current int, route []int, w, h int) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, w, h, w, h)
	b.WriteString(`<rect width="100%" height="100%" fill="#101016"/>`)
	if len(g.order) == 0 {
		b.WriteString("</svg>")
		return b.String()
	}
	minX, maxX, minZ, maxZ := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, bb := range g.bubbles {
		minX, maxX = math.Min(minX, bb.At.X), math.Max(maxX, bb.At.X)
		minZ, maxZ = math.Min(minZ, bb.At.Z), math.Max(maxZ, bb.At.Z)
	}
	pad := 30.0
	sx := func(x float64) float64 {
		if maxX == minX {
			return float64(w) / 2
		}
		return pad + (x-minX)/(maxX-minX)*(float64(w)-2*pad)
	}
	sy := func(z float64) float64 {
		if maxZ == minZ {
			return float64(h) / 2
		}
		return pad + (z-minZ)/(maxZ-minZ)*(float64(h)-2*pad)
	}
	routeEdge := map[[2]int]bool{}
	for i := 0; i+1 < len(route); i++ {
		routeEdge[[2]int{route[i], route[i+1]}] = true
		routeEdge[[2]int{route[i+1], route[i]}] = true
	}
	for _, a := range g.order {
		for _, c := range g.Neighbors(a) {
			if a >= c {
				continue
			}
			col := "#46465a"
			if routeEdge[[2]int{a, c}] {
				col = "#5adc8c"
			}
			fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1.5"/>`,
				sx(g.bubbles[a].At.X), sy(g.bubbles[a].At.Z), sx(g.bubbles[c].At.X), sy(g.bubbles[c].At.Z), col)
		}
	}
	for _, id := range g.order {
		bb := g.bubbles[id]
		fill := "#8296c8"
		if len(g.links[id]) <= 1 {
			fill = "#50505f"
		}
		if id == g.Home() {
			fill = "#f0c850"
		}
		if id == current {
			fill = "#5adcf0"
		}
		x, y := sx(bb.At.X), sy(bb.At.Z)
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="7" fill="%s"/>`, x, y, fill)
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="#dcdce6" font-family="monospace" font-size="11">%s</text>`,
			x+10, y+4, bb.Name)
	}
	b.WriteString("</svg>")
	return b.String()
}

// Layout arranges the bubbles on the X/Z plane by a force-directed simulation
// (Coulomb repulsion between every pair, Hooke springs along transits, mild
// centring, with the home bubble pinned at the origin), the way the InfoAquarium
// graph (svend4/in4n) lays itself out. It starts from a deterministic circle, so
// the result is reproducible, and overwrites each bubble's At.X/At.Z (Y is kept).
func (g *BubbleGraph) Layout(iters int) {
	n := len(g.order)
	if n == 0 {
		return
	}
	for i, id := range g.order { // deterministic initial ring
		a := float64(i) / float64(n) * 2 * math.Pi
		g.bubbles[id].At.X = math.Cos(a) * 10
		g.bubbles[id].At.Z = math.Sin(a) * 10
	}
	const (
		rep  = 220.0 // Coulomb repulsion
		spr  = 0.09  // Hooke spring stiffness
		rest = 8.0   // spring rest length
		damp = 0.85  // velocity damping
		dt   = 0.1
	)
	home := g.Home()
	vel := make(map[int][2]float64, n)
	for it := 0; it < iters; it++ {
		force := make(map[int][2]float64, n)
		for i := 0; i < n; i++ { // pairwise repulsion
			for j := i + 1; j < n; j++ {
				a, b := g.bubbles[g.order[i]], g.bubbles[g.order[j]]
				dx, dz := a.At.X-b.At.X, a.At.Z-b.At.Z
				d2 := dx*dx + dz*dz + 0.01
				d := math.Sqrt(d2)
				fx, fz := rep/d2*dx/d, rep/d2*dz/d
				f := force[a.ID]
				f[0], f[1] = f[0]+fx, f[1]+fz
				force[a.ID] = f
				f = force[b.ID]
				f[0], f[1] = f[0]-fx, f[1]-fz
				force[b.ID] = f
			}
		}
		for _, a := range g.order { // springs along transits (each edge once)
			for b := range g.links[a] {
				if a >= b {
					continue
				}
				A, B := g.bubbles[a], g.bubbles[b]
				dx, dz := B.At.X-A.At.X, B.At.Z-A.At.Z
				d := math.Hypot(dx, dz) + 1e-6
				fx, fz := spr*(d-rest)*dx/d, spr*(d-rest)*dz/d
				f := force[a]
				f[0], f[1] = f[0]+fx, f[1]+fz
				force[a] = f
				f = force[b]
				f[0], f[1] = f[0]-fx, f[1]-fz
				force[b] = f
			}
		}
		for _, id := range g.order { // mild centring + integrate
			b := g.bubbles[id]
			fx, fz := force[id][0]-b.At.X*0.01, force[id][1]-b.At.Z*0.01
			v := vel[id]
			v[0], v[1] = (v[0]+fx)*damp, (v[1]+fz)*damp
			vel[id] = v
			b.At.X += v[0] * dt
			b.At.Z += v[1] * dt
		}
		if home >= 0 { // pin home at the origin — the anchor of the structure
			g.bubbles[home].At.X, g.bubbles[home].At.Z = 0, 0
			vel[home] = [2]float64{}
		}
	}
}

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

// fillRect fills an axis-aligned rectangle.
func fillRect(img *image.RGBA, x0, y0, w, h int, c color.RGBA) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			img.SetRGBA(x, y, c)
		}
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
