package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

func TestEcosystemLives(t *testing.T) {
	// A seeded world with food should run its boom/bust on its own: population
	// stays bounded by the cap and never goes negative, and it does not sit frozen.
	clim := NewClimate(7)
	e := NewEcosystem(12, 12, 30, clim, 1)
	start := e.Population()
	moved := false
	for i := 0; i < 200; i++ {
		e.Step()
		p := e.Population()
		if p < 0 || p > e.maxPop {
			t.Fatalf("population out of bounds at tick %d: %d (cap %d)", i, p, e.maxPop)
		}
		if p != start {
			moved = true
		}
	}
	if !moved {
		t.Error("population never changed over 200 ticks — the world is not alive")
	}
	if e.Tick() != 200 {
		t.Errorf("tick = %d, want 200", e.Tick())
	}
}

func TestEcosystemStarvation(t *testing.T) {
	// Overcrowd a small grid with the larder emptied: demand for energy (pop × the
	// 0.06 cost of living) vastly outstrips what the slow regrowth can supply, so the
	// population must crash as creatures starve down toward the carrying capacity.
	e := NewEcosystem(4, 4, 60, nil, 2)
	for i := range e.food {
		e.food[i] = 0
	}
	start := e.Population()
	for i := 0; i < 60; i++ {
		e.Step()
	}
	if p := e.Population(); p >= start {
		t.Errorf("an overcrowded, food-starved world should crash; start %d, end %d", start, p)
	}

	// And a single creature with almost no energy and nothing to eat dies promptly:
	// the slow regrowth cannot cover even one cost of living.
	e2 := NewEcosystem(6, 6, 1, nil, 3)
	for i := range e2.food {
		e2.food[i] = 0
	}
	e2.creatures[0].energy = 0.05
	for i := 0; i < 5; i++ {
		e2.Step()
	}
	if e2.Population() != 0 {
		t.Error("a starving creature with nothing to eat should die")
	}
}

func TestEcosystemFoodRegrows(t *testing.T) {
	// No grazers, emptied grid: the climate must grow food back.
	e := NewEcosystem(8, 8, 0, nil, 3)
	for i := range e.food {
		e.food[i] = 0
	}
	if f := e.TotalFood(); f != 0 {
		t.Fatalf("emptied grid should hold no food, got %g", f)
	}
	e.Step()
	if f := e.TotalFood(); f <= 0 {
		t.Errorf("food should regrow with no grazers, got %g", f)
	}
}

func TestEcosystemDeterministic(t *testing.T) {
	clim := NewClimate(5)
	run := func() (int, float64) {
		e := NewEcosystem(10, 10, 20, clim, 42)
		for i := 0; i < 120; i++ {
			e.Step()
		}
		return e.Population(), e.TotalFood()
	}
	p1, f1 := run()
	p2, f2 := run()
	if p1 != p2 || f1 != f2 {
		t.Errorf("same seed must replay identically: pop %d/%d food %g/%g", p1, p2, f1, f2)
	}
}

func TestEcosystemObjects(t *testing.T) {
	e := NewEcosystem(8, 8, 12, NewClimate(1), 9)
	e.Step()
	objs := e.Objects(raytrace.Vec3{})
	if len(objs) == 0 {
		t.Fatal("a populated world should render some objects")
	}
	// every creature contributes a sphere, so at least Population() objects exist
	if len(objs) < e.Population() {
		t.Errorf("got %d objects for %d creatures (plus food)", len(objs), e.Population())
	}
}
