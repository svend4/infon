package raydir

import (
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// The mandala and tessellation surface textures are registered and applyable to a
// scene object via the tex field.
func TestSurfacePatternTextures(t *testing.T) {
	for _, name := range []string{"mandala", "tiles", "tessellation"} {
		if TextureFor(name) == nil {
			t.Errorf("surface pattern %q should be registered as a texture", name)
		}
	}
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "box", Y: 0.05, Z: 5, S: [3]float64{4, 0.05, 4}, Tex: "mandala"},
	}})
	ray := raytrace.Ray{Origin: raytrace.Vec3{X: 0, Y: 3, Z: 5}, Dir: raytrace.Vec3{X: 0, Y: -1, Z: 0}}
	h, ok := nearestHit(s, ray)
	if !ok || h.Mat.Tex == nil {
		t.Fatalf("a mandala-textured surface should carry Material.Tex (ok=%v)", ok)
	}
}
