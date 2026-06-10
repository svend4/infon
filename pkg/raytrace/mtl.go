// mtl.go reads Wavefront .mtl material libraries into raytrace Materials, so
// textured/credentialled models load with their authored look. Mapping:
// Kd -> Color, Ke -> Emit, Ks(+Ns) -> specular strength/exponent, Pr -> Rough,
// Pm -> Metal, and a transparent material (d<1 or Tr>0) with Ni -> Glass. Use the
// returned map with LoadOBJMTL.
package raytrace

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

func mtlVec(f []string) Vec3 {
	if len(f) < 4 {
		return Vec3{}
	}
	x, _ := strconv.ParseFloat(f[1], 64)
	y, _ := strconv.ParseFloat(f[2], 64)
	z, _ := strconv.ParseFloat(f[3], 64)
	return Vec3{X: x, Y: y, Z: z}
}

func mtlNum(f []string) float64 {
	if len(f) < 2 {
		return 0
	}
	v, _ := strconv.ParseFloat(f[1], 64)
	return v
}

// LoadMTL parses a .mtl stream into a map of name -> Material.
func LoadMTL(r io.Reader) (map[string]Material, error) {
	mats := map[string]Material{}
	ni := map[string]float64{}  // Ni (index of refraction) per material
	transp := map[string]bool{} // transparent (d<1 or Tr>0)
	cur := ""

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		f := strings.Fields(line)
		key := strings.ToLower(f[0])
		if key == "newmtl" {
			if len(f) >= 2 {
				cur = f[1]
				mats[cur] = Material{}
			}
			continue
		}
		if cur == "" {
			continue
		}
		m := mats[cur]
		switch key {
		case "kd":
			m.Color = mtlVec(f)
		case "ke":
			m.Emit = mtlVec(f)
		case "ks":
			ks := mtlVec(f)
			m.Spec = clamp01((ks.X + ks.Y + ks.Z) / 3)
		case "ns":
			m.Shine = mtlNum(f)
		case "pr": // PBR roughness extension
			m.Rough = clamp01(mtlNum(f))
		case "pm": // PBR metalness extension
			m.Metal = clamp01(mtlNum(f))
		case "ni":
			ni[cur] = mtlNum(f)
		case "d":
			if mtlNum(f) < 1 {
				transp[cur] = true
			}
		case "tr":
			if mtlNum(f) > 0 {
				transp[cur] = true
			}
		}
		mats[cur] = m
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	// Promote transparent materials to glass using their IOR (default 1.5).
	for name := range mats {
		if transp[name] {
			m := mats[name]
			if m.Glass = ni[name]; m.Glass <= 1 {
				m.Glass = 1.5
			}
			mats[name] = m
		}
	}
	return mats, nil
}
