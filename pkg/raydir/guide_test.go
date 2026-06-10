package raydir

import (
	"strings"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// The guide tours the landmarks nearest-first, marking each visited, then
// accompanies the player when the tour is done.
func TestGuideTour(t *testing.T) {
	marks := []Landmark{
		{Index: 0, At: raytrace.Vec3{Z: 10}, Name: "Forest"},
		{Index: 1, At: raytrace.Vec3{Z: 30}, Name: "Crystal Cave"},
	}
	g := NewGuide("Guide", raytrace.Vec3{})
	player := raytrace.Vec3{}

	first := g.Pos
	g.Step(0.5, player, marks)
	if g.Pos.Sub(marks[0].At).Len() >= first.Sub(marks[0].At).Len() {
		t.Error("the guide should move toward the nearest landmark")
	}
	if g.Yaw == 0 && g.Pos.Z == 0 {
		t.Error("the guide should face its motion")
	}
	for i := 0; i < 200; i++ { // walk the whole tour (curMark dips to -1 between stops)
		g.Step(0.5, player, marks)
	}
	if !g.visited[0] || !g.visited[1] {
		t.Errorf("the guide should visit both landmarks, visited=%v", g.visited)
	}
	if g.pickNext(marks) != -1 {
		t.Error("after the tour every landmark should be visited")
	}
	if g.Pos.Sub(player).Len() > 6 {
		t.Errorf("an accompanying guide should stay near the player, dist=%.1f", g.Pos.Sub(player).Len())
	}
}

// The guide comments on where it is leading you, throttled in time.
func TestGuideRemark(t *testing.T) {
	marks := []Landmark{{Index: 0, At: raytrace.Vec3{Z: 10}, Name: "Forest"}}
	g := NewGuide("Guide", raytrace.Vec3{})
	g.Step(0.1, raytrace.Vec3{}, marks) // pick a target

	r, ok := g.Remark(0, marks)
	if !ok || !strings.Contains(r, "Forest") {
		t.Errorf("the guide should announce the Forest, got %q ok=%v", r, ok)
	}
	if _, ok := g.Remark(1, marks); ok {
		t.Error("remarks should be throttled (nothing again so soon)")
	}
	if _, ok := g.Remark(7, marks); !ok {
		t.Error("after the interval the guide should speak again")
	}
}

func TestGuidePose(t *testing.T) {
	g := NewGuide("G", raytrace.Vec3{X: 1, Y: 2, Z: 3})
	if p := g.Pose(); p.Pos != (raytrace.Vec3{X: 1, Y: 2, Z: 3}) {
		t.Errorf("pose should reflect position, got %+v", p)
	}
}
