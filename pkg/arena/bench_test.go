package arena

import "testing"

func BenchmarkStep(b *testing.B) {
	mk := func() *Arena {
		a := &Arena{W: 8, H: 6, Terrain: make([]uint8, 48)}
		a.Units = []Unit{
			{Kind: 0, X: 0, Y: 0, HP: 6, Faction: 0, Alive: true},
			{Kind: 0, X: 7, Y: 5, HP: 6, Faction: 1, Alive: true},
		}
		return a
	}
	a := mk()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			a = mk()
		}
		a.Step(RefCommander{}, RefCommander{})
	}
}

func BenchmarkPlannerStep(b *testing.B) {
	mk := func() *Arena {
		a := &Arena{W: 8, H: 6, Terrain: make([]uint8, 48)}
		a.Units = []Unit{
			{Kind: 4, X: 0, Y: 0, HP: 5, Faction: 0, Alive: true},
			{Kind: 3, X: 7, Y: 5, HP: 5, Faction: 1, Alive: true},
		}
		return a
	}
	a := mk()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if a.AliveCount(0) == 0 || a.AliveCount(1) == 0 {
			a = mk()
		}
		a.Step(Planner{}, RefCommander{})
	}
}
