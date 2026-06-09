package raytrace

import (
	"strings"
	"testing"
)

func TestLoadMTL(t *testing.T) {
	src := `# materials
newmtl red
Kd 0.8 0.1 0.1
Ks 0.5 0.5 0.5
Ns 64
newmtl lamp
Ke 5 5 5
newmtl steel
Pm 1.0
Pr 0.2
newmtl glass
Ni 1.5
d 0.2
`
	m, err := LoadMTL(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if m["red"].Color != (Vec3{X: 0.8, Y: 0.1, Z: 0.1}) || m["red"].Shine != 64 || m["red"].Spec <= 0 {
		t.Errorf("red wrong: %+v", m["red"])
	}
	if m["lamp"].Emit != (Vec3{X: 5, Y: 5, Z: 5}) {
		t.Errorf("lamp emit wrong: %+v", m["lamp"].Emit)
	}
	if m["steel"].Metal != 1.0 || m["steel"].Rough != 0.2 {
		t.Errorf("steel pbr wrong: %+v", m["steel"])
	}
	if m["glass"].Glass != 1.5 {
		t.Errorf("glass should be promoted to IOR 1.5, got %v", m["glass"].Glass)
	}
}

func TestLoadOBJMTLUsesNamedMaterials(t *testing.T) {
	mtls := map[string]Material{"shiny": {Color: Vec3{X: 0.1, Y: 0.9, Z: 0.2}, Metal: 1}}
	obj := "v 0 0 0\nv 1 0 0\nv 0 1 0\nusemtl shiny\nf 1 2 3\n"
	mesh, err := LoadOBJMTL(strings.NewReader(obj), mtls, Material{})
	if err != nil {
		t.Fatal(err)
	}
	if len(mesh.Tris) != 1 || mesh.Tris[0].Mat.Metal != 1 || mesh.Tris[0].Mat.Color != (Vec3{X: 0.1, Y: 0.9, Z: 0.2}) {
		t.Errorf("usemtl not applied: %+v", mesh.Tris[0].Mat)
	}
}
