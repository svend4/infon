package raydir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// A named mesh becomes a single Instance placed at the spec's position, and a ray
// toward it connects.
func TestMeshInstance(t *testing.T) {
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{
		{Kind: "mesh", Name: "crystal", X: 0, Y: 1, Z: 5, R: 2},
	}})
	if len(s.Objects) != 1 {
		t.Fatalf("a mesh should be one Instance, got %d objects", len(s.Objects))
	}
	if _, ok := s.Objects[0].(*raytrace.Instance); !ok {
		t.Fatalf("mesh object should be an *Instance, got %T", s.Objects[0])
	}
	ray := raytrace.Ray{Origin: raytrace.Vec3{X: 0, Y: 1, Z: 0}, Dir: raytrace.Vec3{X: 0, Y: 0, Z: 1}}
	if _, ok := s.Objects[0].Intersect(ray, 1e-9, 1e9); !ok {
		t.Error("ray should hit the instanced crystal")
	}
}

// An unknown model name is dropped (sanitisation), so a bad reference can't break
// the scene.
func TestMeshUnknownDropped(t *testing.T) {
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "mesh", Name: "nope-not-real", Y: 1}}})
	if len(s.Objects) != 0 {
		t.Errorf("unknown mesh name should be dropped, got %d objects", len(s.Objects))
	}
	if hasRenderable(brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "mesh", Name: "nope", Y: 1}}}) {
		t.Error("an unknown mesh should not count as renderable")
	}
}

// .obj files in a directory register as named, instanceable models.
func TestLoadMeshDir(t *testing.T) {
	dir := t.TempDir()
	obj := "v 0 0 0\nv 1 0 0\nv 0 1 0\nf 1 2 3\n"
	if err := os.WriteFile(filepath.Join(dir, "widget.obj"), []byte(obj), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := LoadMeshDir(dir)
	if err != nil || n != 1 {
		t.Fatalf("LoadMeshDir = %d,%v, want 1,nil", n, err)
	}
	if meshFor("widget") == nil {
		t.Fatal("widget.obj should be registered as 'widget'")
	}
	s := BuildScene(brain.SceneSpec{Objects: []brain.ObjSpec{{Kind: "mesh", Name: "widget", Y: 1, R: 1}}})
	if _, ok := s.Objects[0].(*raytrace.Instance); !ok {
		t.Errorf("a loaded .obj should place an Instance, got %T", s.Objects[0])
	}
}

func TestRefSceneMeshKeyword(t *testing.T) {
	meshNamed := func(prompt, name string) bool {
		_, spec, _ := AuthorScene(brain.Local{}, prompt)
		for _, o := range spec.Objects {
			if o.Kind == "mesh" && o.Name == name {
				return true
			}
		}
		return false
	}
	if !meshNamed("a glittering crystal cave", "crystal") {
		t.Error("'crystal' should author a crystal mesh")
	}
	if !meshNamed("a field of boulders", "rock") {
		t.Error("'boulder' should author a rock mesh")
	}
}
