package raydir

import "testing"

func TestQuestAgentReachesGoal(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		q := NewQuest(seed, 0.5)
		a := NewQuestAgent(q)
		route, ok := a.Run(512)
		if !ok {
			t.Fatalf("seed %d: agent failed to reach the goal", seed)
		}
		if route[0] != q.Start || route[len(route)-1] != q.Goal {
			t.Fatalf("seed %d: route must run Start..Goal", seed)
		}
		for i := 1; i < len(route); i++ {
			if route[i-1].Hamming(route[i]) != 1 {
				t.Fatalf("seed %d: step %d is not a single line flip", seed, i)
			}
			if q.IsWall(route[i]) {
				t.Fatalf("seed %d: agent walked into a wall", seed)
			}
		}
	}
}

func TestQuestAgentLearns(t *testing.T) {
	// Replaying the same maze, the agent's route must shorten toward the omniscient
	// optimum as it maps the walls — and ultimately match it.
	improved := 0
	for seed := int64(0); seed < 30; seed++ {
		q := NewQuest(seed, 0.6)
		opt, ok := q.Solve()
		if !ok {
			continue
		}
		optimal := len(opt) - 1
		a := NewQuestAgent(q)
		steps := a.Learn(80, 512)
		first, last := steps[0], steps[len(steps)-1]
		if last > first {
			t.Fatalf("seed %d: agent got worse with experience (%d -> %d)", seed, first, last)
		}
		if last != optimal {
			t.Fatalf("seed %d: after learning, route %d != optimum %d", seed, last, optimal)
		}
		if first > optimal {
			improved++ // this maze actually demanded learning
		}
	}
	if improved == 0 {
		t.Error("no maze required the agent to learn — strengthen the test mazes")
	}
}

func TestQuestAgentExploresAndIsDeterministic(t *testing.T) {
	route := func() []Hexagram {
		q := NewQuest(11, 0.5)
		a := NewQuestAgent(q)
		r, _ := a.Run(512)
		return r
	}
	r1, r2 := route(), route()
	if len(r1) != len(r2) {
		t.Fatalf("same maze must replay identically: %d vs %d steps", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i] != r2[i] {
			t.Fatalf("route diverged at step %d", i)
		}
	}
	// having walked to the goal, the agent has confirmed a fair share of the cube
	q := NewQuest(11, 0.5)
	a := NewQuestAgent(q)
	a.Run(512)
	if a.Explored() < 8 {
		t.Errorf("agent should have sensed many rooms, only %d", a.Explored())
	}
}
