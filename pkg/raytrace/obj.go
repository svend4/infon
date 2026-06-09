// obj.go is a minimal Wavefront .obj reader. It understands `v x y z` vertices,
// `vn x y z` normals and `f` faces (any polygon, fan-triangulated). Face tokens
// may be `a`, `a/b`, `a/b/c` or `a//c`; the vertex index is always used and the
// normal index, when present, gives smooth (interpolated) shading. Both 1-based
// and negative/relative indices are accepted. It is enough to load classic test
// models (a cube, Suzanne, …) without a third-party parser.
package raytrace

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// LoadOBJ parses an .obj stream into a Mesh, giving every triangle mat.
func LoadOBJ(r io.Reader, mat Material) (*Mesh, error) {
	var verts, norms []Vec3
	var tris []Triangle

	// resolve a "v/vt/vn" token to a vertex index and an optional normal index.
	resolve := func(tok string) (vi, ni int, ok bool) {
		parts := strings.Split(tok, "/")
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, false
		}
		if v < 0 {
			v += len(verts)
		} else {
			v--
		}
		if v < 0 || v >= len(verts) {
			return 0, 0, false
		}
		ni = -1
		if len(parts) == 3 && parts[2] != "" {
			if n, e := strconv.Atoi(parts[2]); e == nil {
				if n < 0 {
					n += len(norms)
				} else {
					n--
				}
				if n >= 0 && n < len(norms) {
					ni = n
				}
			}
		}
		return v, ni, true
	}

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
			if len(f) >= 4 {
				x, _ := strconv.ParseFloat(f[1], 64)
				y, _ := strconv.ParseFloat(f[2], 64)
				z, _ := strconv.ParseFloat(f[3], 64)
				verts = append(verts, Vec3{X: x, Y: y, Z: z})
			}
		case "vn":
			if len(f) >= 4 {
				x, _ := strconv.ParseFloat(f[1], 64)
				y, _ := strconv.ParseFloat(f[2], 64)
				z, _ := strconv.ParseFloat(f[3], 64)
				norms = append(norms, Vec3{X: x, Y: y, Z: z}.Norm())
			}
		case "f":
			var vs, ns []int
			for _, tok := range f[1:] {
				if vi, ni, okv := resolve(tok); okv {
					vs = append(vs, vi)
					ns = append(ns, ni)
				}
			}
			for i := 1; i+1 < len(vs); i++ { // fan triangulation
				t := Triangle{A: verts[vs[0]], B: verts[vs[i]], C: verts[vs[i+1]], Mat: mat}
				if ns[0] >= 0 && ns[i] >= 0 && ns[i+1] >= 0 {
					t.Na, t.Nb, t.Nc = norms[ns[0]], norms[ns[i]], norms[ns[i+1]]
				}
				tris = append(tris, t)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return NewMesh(tris), nil
}
