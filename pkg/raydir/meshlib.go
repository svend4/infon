package raydir

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/svend4/infon/pkg/raytrace"
)

// meshlib.go is the named-mesh library the director draws from. A scene object of
// kind "mesh" names a model here; it is placed as an Instance (one shared mesh,
// transformed), so a forest of 100 trees costs one mesh + 100 cheap transforms.
// Built-in procedural meshes are registered out of the box; LoadMeshDir adds
// arbitrary .obj models from disk, so a deployment can extend the vocabulary with
// no code changes — and still ship only a name over the wire.

var (
	meshMu  sync.RWMutex
	meshReg = map[string]*raytrace.Mesh{}
)

// RegisterMesh adds (or replaces) a named instanceable mesh.
func RegisterMesh(name string, m *raytrace.Mesh) {
	if m == nil {
		return
	}
	meshMu.Lock()
	meshReg[strings.ToLower(name)] = m
	meshMu.Unlock()
}

// meshFor returns the named mesh, or nil if unknown.
func meshFor(name string) *raytrace.Mesh {
	meshMu.RLock()
	defer meshMu.RUnlock()
	return meshReg[strings.ToLower(name)]
}

// meshFromObjects builds a Mesh from the triangles among objs.
func meshFromObjects(objs []raytrace.Object) *raytrace.Mesh {
	tris := make([]raytrace.Triangle, 0, len(objs))
	for _, o := range objs {
		if t, ok := o.(raytrace.Triangle); ok {
			tris = append(tris, t)
		}
	}
	return raytrace.NewMesh(tris)
}

// LoadMeshDir registers every .obj file in dir under its base name, so the
// director can place custom models by name without any code change.
func LoadMeshDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".obj") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		m, err := raytrace.LoadOBJ(f, raytrace.Material{Color: raytrace.Vec3{X: 0.72, Y: 0.72, Z: 0.74}})
		_ = f.Close()
		if err != nil {
			continue
		}
		RegisterMesh(strings.TrimSuffix(strings.ToLower(e.Name()), ".obj"), m)
		n++
	}
	return n, nil
}

func init() {
	o := raytrace.Vec3{}
	RegisterMesh("tree", meshFromObjects(treeObjects(o, 1, raytrace.Material{})))
	RegisterMesh("house", meshFromObjects(houseObjects(o, 1, raytrace.Material{})))
	RegisterMesh("crystal", meshFromObjects(crystalTris(o, 1)))
	RegisterMesh("rock", meshFromObjects(rockTris(o, 1)))
}
