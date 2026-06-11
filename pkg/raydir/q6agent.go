package raydir

// q6agent.go is an agent that PLAYS the Q6 roguelike (joining the maze of Block D with
// the learning spirit of Block C). Unlike the omniscient solver, the agent navigates
// with fog of war: it cannot see a collapsed room until it tries to step into one and
// bumps. It heads for the goal along the shortest route it currently believes exists
// (assuming unseen rooms are open — optimism), re-planning whenever a bump reveals a
// wall, and it REMEMBERS every wall it has found. Replayed on the same maze its route
// shortens toward the omniscient optimum as its map fills in: it learns the dungeon by
// playing it.

// QuestAgent explores a Quest by bumping (it learns a room is a wall only by trying to
// enter it) with persistent memory across episodes.
type QuestAgent struct {
	q         *Quest
	knownWall [64]bool // walls the agent has bumped into (persists across episodes)
	knownOpen [64]bool // rooms it has stood in
	Bumps     int      // total bumps over the agent's life (a cost it learns to avoid)
}

// NewQuestAgent makes an agent that knows nothing of the maze's walls yet.
func NewQuestAgent(q *Quest) *QuestAgent { return &QuestAgent{q: q} }

// planBelief is a breadth-first shortest route from cur to goal over the agent's
// belief: known walls are blocked, everything else (open or unseen) is assumed
// passable. Returns room indices inclusive, or nil if the belief offers no route.
func (a *QuestAgent) planBelief(cur, goal int) []int {
	if a.knownWall[cur] || a.knownWall[goal] {
		return nil
	}
	prev := [64]int{}
	for i := range prev {
		prev[i] = -2
	}
	prev[cur] = -1
	queue := []int{cur}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == goal {
			break
		}
		h := HexagramFromNumber(c)
		for line := 0; line < 6; line++ {
			nb := h.Flip(line).Number()
			if a.knownWall[nb] || prev[nb] != -2 {
				continue
			}
			prev[nb] = c
			queue = append(queue, nb)
		}
	}
	if prev[goal] == -2 {
		return nil
	}
	var path []int
	for c := goal; c != -1; c = prev[c] {
		path = append(path, c)
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// Run plays one episode from Start to Goal, re-planning whenever it bumps a wall. It
// returns the route walked (inclusive) and whether it reached the goal within maxSteps
// moves. Knowledge gathered persists on the agent, so a later Run on the same maze
// tends to be shorter.
func (a *QuestAgent) Run(maxSteps int) ([]Hexagram, bool) {
	cur := a.q.Start.Number()
	goal := a.q.Goal.Number()
	route := []Hexagram{HexagramFromNumber(cur)}
	steps := 0
	for steps < maxSteps {
		if cur == goal {
			return route, true
		}
		a.knownOpen[cur] = true
		path := a.planBelief(cur, goal)
		if len(path) < 2 {
			return route, false // belief says boxed in (cannot happen on a solvable maze)
		}
		next := path[1]
		if a.q.walls[next] { // bump: the believed-open room is a wall — learn it, replan
			a.knownWall[next] = true
			a.Bumps++
			continue
		}
		cur = next
		route = append(route, HexagramFromNumber(cur))
		steps++
	}
	return route, cur == goal
}

// Explored is how many of the 64 rooms the agent has confirmed (open or wall).
func (a *QuestAgent) Explored() int {
	n := 0
	for i := 0; i < 64; i++ {
		if a.knownWall[i] || a.knownOpen[i] {
			n++
		}
	}
	return n
}

// Learn replays the maze for episodes, returning the number of steps walked each
// episode. The series falls toward the optimum as the agent's map fills in.
func (a *QuestAgent) Learn(episodes, maxSteps int) []int {
	steps := make([]int, 0, episodes)
	for e := 0; e < episodes; e++ {
		route, ok := a.Run(maxSteps)
		if !ok {
			steps = append(steps, maxSteps)
			continue
		}
		steps = append(steps, len(route)-1)
	}
	return steps
}
