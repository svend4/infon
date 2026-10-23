package raydir

import (
	"strings"
	"testing"
	"unicode"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// hasControl reports a control character other than the newlines a caller adds.
func hasControl(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return r != '\n' && unicode.IsControl(r) }) >= 0
}

// A region name arrives from a peer (DecodeRegion) or a model and is printed to the
// terminal in the minimap legend every frame; escape sequences in it must not reach
// the terminal.
func TestRegionNameDropsControlCharacters(t *testing.T) {
	w := NewWorld()
	obj := []brain.ObjSpec{{Kind: "sphere", Y: 1, Z: 6, R: 1}}
	evil := "Grove\x1b]0;owned\x07\x1b[2J\u009b31m\xff"
	w.AddRegion(Region{Index: 0, At: raytrace.Vec3{Z: 6}, Spec: brain.SceneSpec{Objects: obj, Name: evil}})
	w.AddRegion(Region{Index: 1, At: raytrace.Vec3{Z: 18}, Spec: brain.SceneSpec{Objects: obj, Name: "\x1b\x07\r"}})

	marks := w.Landmarks()
	if len(marks) != 2 {
		t.Fatalf("%d landmarks, want 2", len(marks))
	}
	if hasControl(marks[0].Name) || !strings.HasPrefix(marks[0].Name, "Grove") {
		t.Errorf("landmark name %q keeps control characters", marks[0].Name)
	}
	if marks[1].Name != "Region 1" {
		t.Errorf("a name of only control characters = %q, want the default", marks[1].Name)
	}
	if m := Minimap(marks, nil, raytrace.Vec3{}, 20, 6); hasControl(m) {
		t.Errorf("minimap writes control characters to the terminal: %q", m)
	}
	if l, ok := w.FindLandmark("grove"); !ok || l.Index != 0 {
		t.Errorf("FindLandmark(grove) = %v, %v", l, ok)
	}

	// The chat lines printed under the same frame carry peer text too.
	c := NewChatLog(4)
	c.Add("mallory\x1b[2J", "hi\x1b]52;c;cGF5bG9hZA==\x07 there\u009b")
	c.Add("alice", "plain text, ünïcödé")
	lines := c.Lines()
	for _, l := range lines {
		if hasControl(l) {
			t.Errorf("chat line %q keeps control characters", l)
		}
	}
	if lines[1] != "alice: plain text, ünïcödé" {
		t.Errorf("printable chat changed: %q", lines[1])
	}
}
