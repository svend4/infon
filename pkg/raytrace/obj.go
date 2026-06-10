// obj.go is a minimal Wavefront .obj reader. It understands `v`, `vn`, `vt`, `f`
// (any polygon, fan-triangulated) and `usemtl` (when a material library is
// supplied via LoadOBJMTL). Face tokens may be `a`, `a/b`, `a/b/c` or `a//c`;
// 1-based and negative/relative indices are accepted. Enough to load classic test
// models without a third-party parser.
package raytrace

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// LoadOBJ parses an .obj stream into a Mesh, giving every triangle mat.
func LoadOBJ(r io.Reader, mat Material) (*Mesh, error) { return loadOBJWith(r, nil, mat) }

// LoadOBJMTL parses an .obj stream, switching material on `usemtl name` using the
// supplied material library (see LoadMTL); faces before any usemtl use fallback.
func LoadOBJMTL(r io.Reader, mtls map[string]Material, fallback Material) (*Mesh, error) {
	return loadOBJWith(r, mtls, fallback)
}

func loadOBJWith(r io.Reader, mtls map[string]Material, fallback Material) (*Mesh, error) {
	var verts, norms []Vec3
	var uvs [][2]float64
	var tris []Triangle
	curMat := fallback

	resolve := func(tok string) (vi, ti, ni int, ok bool) {
		parts := strings.Split(tok, "/")
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, 0, false
		}
		if v < 0 {
			v += len(verts)
		} else {
			v--
		}
		if v < 0 || v >= len(verts) {
			return 0, 0, 0, false
		}
		ti, ni = -1, -1
		if len(parts) >= 2 && parts[1] != "" {
			if t, e := strconv.Atoi(parts[1]); e == nil {
				if t < 0 {
					t += len(uvs)
				} else {
					t--
				}
				if t >= 0 && t < len(uvs) {
					ti = t
				}
			}
		}
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
		return v, ti, ni, true
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
		case "vt":
			if len(f) >= 3 {
				u, _ := strconv.ParseFloat(f[1], 64)
				v, _ := strconv.ParseFloat(f[2], 64)
				uvs = append(uvs, [2]float64{u, v})
			}
		case "usemtl":
			if mtls != nil && len(f) >= 2 {
				if m, ok := mtls[f[1]]; ok {
					curMat = m
				}
			}
		case "f":
			var vs, ts, ns []int
			for _, tok := range f[1:] {
				if vi, ti, ni, okv := resolve(tok); okv {
					vs = append(vs, vi)
					ts = append(ts, ti)
					ns = append(ns, ni)
				}
			}
			for i := 1; i+1 < len(vs); i++ { // fan triangulation
				t := Triangle{A: verts[vs[0]], B: verts[vs[i]], C: verts[vs[i+1]], Mat: curMat}
				if ns[0] >= 0 && ns[i] >= 0 && ns[i+1] >= 0 {
					t.Na, t.Nb, t.Nc = norms[ns[0]], norms[ns[i]], norms[ns[i+1]]
				}
				if ts[0] >= 0 && ts[i] >= 0 && ts[i+1] >= 0 {
					t.UVa, t.UVb, t.UVc = uvs[ts[0]], uvs[ts[i]], uvs[ts[i+1]]
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
