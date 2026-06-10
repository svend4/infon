package raydir

import "testing"

func TestHexCAStatesInRange(t *testing.T) {
	c := NewHexCA(16, 16, 5)
	for _, s := range c.Grid() {
		if s > 63 {
			t.Fatalf("state out of range: %d", s)
		}
	}
	for step := 0; step < 20; step++ {
		c.Step()
		for _, s := range c.Grid() {
			if s > 63 {
				t.Fatalf("state out of range after step %d: %d", step, s)
			}
		}
	}
}

func TestHexCADeterministic(t *testing.T) {
	a := NewHexCA(20, 12, 9)
	b := NewHexCA(20, 12, 9)
	for i := 0; i < 15; i++ {
		a.Step()
		b.Step()
	}
	for i := range a.Grid() {
		if a.Grid()[i] != b.Grid()[i] {
			t.Fatalf("same seed should evolve identically (diff at %d)", i)
		}
	}
}

func TestHexCAEvolves(t *testing.T) {
	c := NewHexCA(24, 24, 1)
	before := append([]uint8(nil), c.Grid()...)
	c.Step()
	changed := 0
	for i := range before {
		if before[i] != c.Grid()[i] {
			changed++
		}
	}
	if changed == 0 {
		t.Error("a random grid should change on the first step")
	}
}

func TestHexCARenderDims(t *testing.T) {
	c := NewHexCA(10, 8, 2)
	img := c.Render(6)
	if img.Bounds().Dx() != 60 || img.Bounds().Dy() != 48 {
		t.Errorf("unexpected render size %v", img.Bounds())
	}
}
