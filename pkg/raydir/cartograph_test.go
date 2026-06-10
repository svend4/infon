package raydir

import (
	"image/color"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

func TestCartographReveal(t *testing.T) {
	c := NewCartograph(4)
	if c.Count() != 0 {
		t.Fatal("a fresh cartograph should be empty")
	}
	c.Reveal(raytrace.Vec3{}, 10)
	if !c.Seen(raytrace.Vec3{}) {
		t.Error("the spot you stand on should be revealed")
	}
	if !c.Seen(raytrace.Vec3{X: 6}) {
		t.Error("a point within the reveal radius should be revealed")
	}
	if c.Seen(raytrace.Vec3{X: 40, Z: 40}) {
		t.Error("a far point should remain a white spot")
	}
	if c.Count() == 0 {
		t.Error("revealing should record explored cells")
	}
}

func TestSpecialPlace(t *testing.T) {
	if _, ok := isSpecialPlace("The Reserve"); !ok {
		t.Error("a reserve should be a special place")
	}
	if _, ok := isSpecialPlace("Old Prison"); !ok {
		t.Error("a prison should be a special place")
	}
	if _, ok := isSpecialPlace("Forest"); ok {
		t.Error("an ordinary place should not be special")
	}
}

func TestCartographRender(t *testing.T) {
	c := NewCartograph(4)
	for z := 0.0; z < 40; z += 4 { // a walked corridor
		c.Reveal(raytrace.Vec3{Z: z}, 8)
	}
	marks := []Landmark{
		{Name: "Forest", At: raytrace.Vec3{X: 5, Z: 10}},
		{Name: "The Reserve", At: raytrace.Vec3{X: -6, Z: 20}},
		{Name: "Old Prison", At: raytrace.Vec3{X: 6, Z: 30}},
	}
	img := c.Render(marks, raytrace.Vec3{Z: 36}, 320, 320)
	if img.Bounds().Dx() != 320 || img.Bounds().Dy() != 320 {
		t.Fatalf("unexpected map size %v", img.Bounds())
	}
	if !hasColor(img, color.RGBA{R: 42, G: 66, B: 46}, 12) {
		t.Error("explored ground should be drawn")
	}
	if !hasColor(img, color.RGBA{R: 206, G: 202, B: 188}, 12) {
		t.Error("unexplored white spots should be drawn")
	}
	if !hasColor(img, color.RGBA{R: 240, G: 200, B: 80}, 14) {
		t.Error("the reserve should be marked in gold")
	}
	if !hasColor(img, color.RGBA{R: 220, G: 80, B: 80}, 14) {
		t.Error("the prison should be marked in red")
	}
}

func TestWorldRevealWiring(t *testing.T) {
	w := NewWorld()
	if w.Cartograph() != nil {
		t.Error("a fresh world has no map yet")
	}
	w.Reveal(raytrace.Vec3{Z: 5}, 10)
	if w.Cartograph() == nil || w.Cartograph().Count() == 0 {
		t.Error("revealing should create and populate the world's map")
	}
}
