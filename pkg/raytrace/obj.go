// obj.go is a minimal Wavefront .obj reader. It understands `v x y z` vertices
// and `f` faces (any polygon, fan-triangulated). Face tokens may be `a`, `a/b`,
// `a/b/c` or `a//c`; only the vertex index is used (normals are computed per
// triangle at render time). Both 1-based and negative/relative indices are
// accepted. It is enough to load classic test models (a cube, Suzanne, …)
// without a third-party parser.
package raytrace

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// LoadOBJ parses an .obj stream into a Mesh, giving every triangle mat.
func LoadOBJ(r io.Reader, mat Material) (*Mesh, error) {
	var verts []Vec3
	var tris []Triangle

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		f := strings.Fields(line)
		switch f[0] {
		case "v":
			if len(f) < 4 {
				continue
			}
			x, _ := strconv.ParseFloat(f[1], 64)
			y, _ := strconv.ParseFloat(f[2], 64)
			z, _ := strconv.ParseFloat(f[3], 64)
			verts = append(verts, Vec3{X: x, Y: y, Z: z})
		case "f":
			idx := make([]int, 0, len(f)-1)
			for _, tok := range f[1:] {
				vs := tok
				if i := strings.IndexByte(tok, '/'); i >= 0 {
					vs = tok[:i]
				}
				n, err := strconv.Atoi(vs)
				if err != nil {
					continue
				}
				if n < 0 {
					n += len(verts) // relative: -1 == last vertex
				} else {
					n-- // 1-based -> 0-based
				}
				if n >= 0 && n < len(verts) {
					idx = append(idx, n)
				}
			}
			for i := 1; i+1 < len(idx); i++ { // fan triangulation
				tris = append(tris, Triangle{A: verts[idx[0]], B: verts[idx[i]], C: verts[idx[i+1]], Mat: mat})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return NewMesh(tris), nil
}
