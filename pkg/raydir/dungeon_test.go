package raydir

import "testing"

func TestLivingDungeonRunsAndCouples(t *testing.T) {
	q := NewQuest(16, 0.6)
	route, ok := q.Solve()
	if !ok {
		t.Fatal("quest must be solvable")
	}
	d := NewLivingDungeon(route, 10, 10, 20, 0.06, 1)
	if len(d.Populations()) != len(route) {
		t.Fatalf("one population per room expected")
	}
	for i := 0; i < 120; i++ {
		d.Step()
	}
	if d.Migrated() == 0 {
		t.Error("with coupling on, some creatures should have crossed edges")
	}
	if d.Tick() != 120 {
		t.Errorf("tick = %d, want 120", d.Tick())
	}
	total := 0
	for _, p := range d.Populations() {
		if p < 0 {
			t.Fatal("population went negative")
		}
		total += p
	}
	if total == 0 {
		t.Error("the whole dungeon went extinct — expected life to persist")
	}
}

func TestLivingDungeonRichnessVaries(t *testing.T) {
	// a lush room (sunny, dense, clear) must out-fertilise a barren one (foggy, sunless)
	lush := roomRichness(SceneVector{0.1, 0.5, 0.9, 0.9, 0.5, 0.5}.Hexagram())
	barren := roomRichness(SceneVector{0.9, 0.5, 0.1, 0.1, 0.5, 0.5}.Hexagram())
	if !(lush > barren) {
		t.Errorf("lush room (%.2f) should be more fertile than barren (%.2f)", lush, barren)
	}
}

func TestLivingDungeonSourceSink(t *testing.T) {
	// A barren room flanked by lush rooms should hold MORE life with coupling on than
	// in isolation: immigration sustains a sink the room could not feed itself.
	lush := SceneVector{0.0, 0.75, 0.75, 0.75, 0.75, 0.75}.Hexagram()    // fertile
	barren := SceneVector{0.75, 0.25, 0.25, 0.25, 0.25, 0.25}.Hexagram() // poor
	route := []Hexagram{lush, barren, lush}

	coupled := NewLivingDungeon(route, 10, 10, 25, 0.1, 7)
	isolated := NewLivingDungeon(route, 10, 10, 25, 0.0, 7) // migration off
	for i := 0; i < 200; i++ {
		coupled.Step()
		isolated.Step()
	}
	withMig := coupled.Populations()[1]
	without := isolated.Populations()[1]
	if withMig <= without {
		t.Errorf("immigration should sustain the barren middle room: coupled %d vs isolated %d", withMig, without)
	}
}

func TestLivingDungeonDeterministic(t *testing.T) {
	route := []Hexagram{HexagramFromNumber(0b011100), HexagramFromNumber(0b011110), HexagramFromNumber(0b001110)}
	run := func() (int, int) {
		d := NewLivingDungeon(route, 8, 8, 15, 0.08, 3)
		for i := 0; i < 80; i++ {
			d.Step()
		}
		tot := 0
		for _, p := range d.Populations() {
			tot += p
		}
		return tot, d.Migrated()
	}
	p1, m1 := run()
	p2, m2 := run()
	if p1 != p2 || m1 != m2 {
		t.Errorf("same seed must replay identically: pop %d/%d migrated %d/%d", p1, p2, m1, m2)
	}
}
