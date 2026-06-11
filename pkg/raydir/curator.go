package raydir

// curator.go is an agent that searches the Q6 hypercube for the world a viewer loves
// MOST (joining Block C's engagement, Block D's maze and Block H's exploration). Its
// goal is not a fixed coordinate but a feeling: maximise the viewer's engagement. It
// evaluates the open worlds adjacent to where it stands (the viewer's reaction to each
// neighbouring world) and expands toward the most-loved — a best-first climb over the
// engagement landscape that detours around collapsed worlds and settles on the best
// reachable one.

// Curator searches a walled hypercube (a Quest) guided by a viewer's taste.
type Curator struct {
	q  *Quest
	vm ViewerModel
}

// NewCurator makes a curator for a maze and a viewer.
func NewCurator(q *Quest, vm ViewerModel) *Curator { return &Curator{q: q, vm: vm} }

// engagementOf is how much the viewer loves a hexagram's world.
func (c *Curator) engagementOf(h Hexagram) float64 {
	return c.vm.Engagement(VectorFromHexagram(h))
}

// Curate explores from start, expanding up to budget open worlds best-first (always
// the most-engaging frontier world next), never crossing a wall. It returns the
// most-loved world found, its engagement, and the worlds it visited in order.
func (c *Curator) Curate(start Hexagram, budget int) (Hexagram, float64, []Hexagram) {
	if c.q.IsWall(start) {
		return start, 0, nil
	}
	seen := map[int]bool{start.Number(): true}
	frontier := []Hexagram{start}
	feng := []float64{c.engagementOf(start)}
	best, bestE := start, c.engagementOf(start)
	var visited []Hexagram
	for len(frontier) > 0 && len(visited) < budget {
		bi := 0 // pick the most-engaging frontier world
		for i := 1; i < len(frontier); i++ {
			if feng[i] > feng[bi] {
				bi = i
			}
		}
		cur := frontier[bi]
		frontier = append(frontier[:bi], frontier[bi+1:]...)
		feng = append(feng[:bi], feng[bi+1:]...)
		visited = append(visited, cur)
		if e := c.engagementOf(cur); e > bestE {
			best, bestE = cur, e
		}
		for _, nb := range cur.Neighbors() {
			if c.q.IsWall(nb) || seen[nb.Number()] {
				continue
			}
			seen[nb.Number()] = true
			frontier = append(frontier, nb)
			feng = append(feng, c.engagementOf(nb))
		}
	}
	return best, bestE, visited
}

// Engagement is the viewer's love for a hexagram's world (0..1) — exposed so a caller
// can paint the engagement landscape.
func (c *Curator) Engagement(h Hexagram) float64 { return c.engagementOf(h) }
