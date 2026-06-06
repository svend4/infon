package arena

// planner.go adds a NON-reactive commander. RefCommander decides each unit
// greedily from the current tick; Planner instead PLANS the next exchange by
// simulating it: for each candidate set of moves it clones the world, runs one
// real Step with the enemy modelled as the reference bot, and scores the
// resulting trade (damage dealt and kills won, minus its own losses, plus a
// little for holding high ground). It hill-climbs the joint move unit by unit.
// Because it looks a full tick ahead through the true combat rules, it stops
// over-extending and out-trades the reactive baseline head to head.

// Planner is a rollout (one-ply lookahead) Commander. Deterministic.
type Planner struct{}

func (a *Arena) clone() *Arena {
	c := *a
	c.Units = append([]Unit(nil), a.Units...)
	return &c
}

func (a *Arena) factionHP(faction uint8) int {
	t := 0
	for _, u := range a.Units {
		if u.Alive && u.Faction == faction {
			t += int(u.HP)
		}
	}
	return t
}

func (a *Arena) factionAlive(faction uint8) int { return a.AliveCount(faction) }

func (a *Arena) onHighGround(faction uint8) int {
	n := 0
	for _, u := range a.Units {
		if u.Alive && u.Faction == faction && a.height(int(u.X), int(u.Y)) > 0 {
			n++
		}
	}
	return n
}

// fixed replays a precomputed action list for one faction (used inside rollouts).
type fixed struct {
	faction uint8
	acts    []Action
}

func (f fixed) Plan(_ *Arena, faction uint8) []Action {
	if faction == f.faction {
		return f.acts
	}
	return nil
}

// rolloutScore plays the move out: it applies my candidate actions for one tick
// (the enemy answering as the reference bot), then lets BOTH sides play the
// reference policy to the end, and scores the final material. Because Step
// resolves combat from positions held at the START of a tick, a move only pays
// off on later ticks — so the planner must simulate forward, not one ply.
func (a *Arena) rolloutScore(faction uint8, acts []Action) float64 {
	enemy := uint8(1 - faction)
	sim := a.clone()
	me := fixed{faction: faction, acts: acts}
	if faction == 0 {
		sim.Step(me, RefCommander{})
	} else {
		sim.Step(RefCommander{}, me)
	}
	for t := 0; t < 40; t++ {
		if sim.AliveCount(0) == 0 || sim.AliveCount(1) == 0 {
			break
		}
		sim.Step(RefCommander{}, RefCommander{})
	}
	return 3.0*float64(sim.factionAlive(faction)-sim.factionAlive(enemy)) +
		0.2*float64(sim.factionHP(faction)-sim.factionHP(enemy)) +
		0.1*float64(sim.onHighGround(faction))
}

// Plan implements Commander by coordinate-ascent over my units: each unit picks
// the move that best improves the simulated next-tick trade, holding the others
// fixed; two passes let early choices be revised against later ones.
func (Planner) Plan(a *Arena, faction uint8) []Action {
	var mine []int
	for i := range a.Units {
		if a.Units[i].Alive && a.Units[i].Faction == faction {
			mine = append(mine, i)
		}
	}
	if len(mine) == 0 {
		return nil
	}
	dirs := [5][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	chosen := make(map[int]Action, len(mine))
	for _, i := range mine {
		chosen[i] = Action{Unit: i, DX: 0, DY: 0, Attack: -1}
	}
	actsList := func() []Action {
		out := make([]Action, 0, len(mine))
		for _, i := range mine {
			out = append(out, chosen[i])
		}
		return out
	}
	for pass := 0; pass < 2; pass++ {
		for _, i := range mine {
			u := a.Units[i]
			bestScore := -1e18
			best := Action{Unit: i, DX: 0, DY: 0, Attack: -1}
			for _, d := range dirs {
				nx, ny := int(u.X)+d[0], int(u.Y)+d[1]
				if nx < 0 || ny < 0 || nx >= a.W || ny >= a.H {
					continue
				}
				chosen[i] = Action{Unit: i, DX: d[0], DY: d[1], Attack: -1}
				if s := a.rolloutScore(faction, actsList()); s > bestScore {
					bestScore, best = s, chosen[i]
				}
			}
			chosen[i] = best
		}
	}
	return actsList()
}
