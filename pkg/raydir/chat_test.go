package raydir

import "testing"

func TestChatLogRing(t *testing.T) {
	c := NewChatLog(3)
	c.Add("a", "one")
	c.Add("b", "two")
	c.Add("a", "three")
	c.Add("b", "four") // pushes "one" out
	lines := c.Lines()
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "b: two" || lines[2] != "b: four" {
		t.Errorf("ring order wrong: %v", lines)
	}
	// Lines returns a copy (mutating it must not corrupt the log).
	lines[0] = "x"
	if c.Lines()[0] != "b: two" {
		t.Error("Lines should return a copy")
	}
}
