// noise.go adds procedural noise — the building block for clouds, marble, wood,
// terrain and weathering. Perlin3 is classic improved gradient noise (smooth,
// zero at the integer lattice); FBM sums octaves of it for natural detail;
// Worley3 is cellular (distance-to-nearest-feature) noise for cells, scales and
// cracks. NoiseTex and MarbleTex expose them as surface textures, and FBM doubles
// as a Medium density for real (non-uniform) clouds. Clean-room implementation.
package raytrace

import "math"

// perlinPerm is the doubled Ken Perlin permutation table.
var perlinPerm [512]int

func init() {
	base := [256]int{
		151, 160, 137, 91, 90, 15, 131, 13, 201, 95, 96, 53, 194, 233, 7, 225, 140, 36, 103, 30, 69, 142, 8, 99, 37,
		240, 21, 10, 23, 190, 6, 148, 247, 120, 234, 75, 0, 26, 197, 62, 94, 252, 219, 203, 117, 35, 11, 32, 57, 177,
		33, 88, 237, 149, 56, 87, 174, 20, 125, 136, 171, 168, 68, 175, 74, 165, 71, 134, 139, 48, 27, 166, 77, 146,
		158, 231, 83, 111, 229, 122, 60, 211, 133, 230, 220, 105, 92, 41, 55, 46, 245, 40, 244, 102, 143, 54, 65, 25,
		63, 161, 1, 216, 80, 73, 209, 76, 132, 187, 208, 89, 18, 169, 200, 196, 135, 130, 116, 188, 159, 86, 164, 100,
		109, 198, 173, 186, 3, 64, 52, 217, 226, 250, 124, 123, 5, 202, 38, 147, 118, 126, 255, 82, 85, 212, 207, 206,
		59, 227, 47, 16, 58, 17, 182, 189, 28, 42, 223, 183, 170, 213, 119, 248, 152, 2, 44, 154, 163, 70, 221, 153,
		101, 155, 167, 43, 172, 9, 129, 22, 39, 253, 19, 98, 108, 110, 79, 113, 224, 232, 178, 185, 112, 104, 218, 246,
		97, 228, 251, 34, 242, 193, 238, 210, 144, 12, 191, 179, 162, 241, 81, 51, 145, 235, 249, 14, 239, 107, 49, 192,
		214, 31, 181, 199, 106, 157, 184, 84, 204, 176, 115, 121, 50, 45, 127, 4, 150, 254, 138, 236, 205, 93, 222, 114,
		67, 29, 24, 72, 243, 141, 128, 195, 78, 66, 215, 61, 156, 180,
	}
	for i := 0; i < 256; i++ {
		perlinPerm[i] = base[i]
		perlinPerm[256+i] = base[i]
	}
}

func perlinFade(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }

func perlinLerp(t, a, b float64) float64 { return a + t*(b-a) }

func perlinGrad(hash int, x, y, z float64) float64 {
	h := hash & 15
	u := x
	if h >= 8 {
		u = y
	}
	v := z
	if h < 4 {
		v = y
	} else if h == 12 || h == 14 {
		v = x
	}
	if h&1 != 0 {
		u = -u
	}
	if h&2 != 0 {
		v = -v
	}
	return u + v
}

// Perlin3 is improved Perlin gradient noise, roughly in [-1,1] and exactly 0 at
// integer lattice points.
func Perlin3(p Vec3) float64 {
	fx, fy, fz := math.Floor(p.X), math.Floor(p.Y), math.Floor(p.Z)
	X, Y, Z := int(fx)&255, int(fy)&255, int(fz)&255
	x, y, z := p.X-fx, p.Y-fy, p.Z-fz
	u, v, w := perlinFade(x), perlinFade(y), perlinFade(z)
	pm := &perlinPerm
	a := pm[X] + Y
	aa, ab := pm[a]+Z, pm[a+1]+Z
	b := pm[X+1] + Y
	ba, bb := pm[b]+Z, pm[b+1]+Z
	return perlinLerp(w,
		perlinLerp(v,
			perlinLerp(u, perlinGrad(pm[aa], x, y, z), perlinGrad(pm[ba], x-1, y, z)),
			perlinLerp(u, perlinGrad(pm[ab], x, y-1, z), perlinGrad(pm[bb], x-1, y-1, z))),
		perlinLerp(v,
			perlinLerp(u, perlinGrad(pm[aa+1], x, y, z-1), perlinGrad(pm[ba+1], x-1, y, z-1)),
			perlinLerp(u, perlinGrad(pm[ab+1], x, y-1, z-1), perlinGrad(pm[bb+1], x-1, y-1, z-1))))
}

// FBM sums octaves of Perlin noise (fractal Brownian motion), normalised to about
// [-1,1]. lacunarity scales frequency per octave, gain scales amplitude.
func FBM(p Vec3, octaves int, lacunarity, gain float64) float64 {
	if octaves < 1 {
		octaves = 4
	}
	if lacunarity <= 0 {
		lacunarity = 2
	}
	if gain <= 0 {
		gain = 0.5
	}
	amp, freq, sum, norm := 1.0, 1.0, 0.0, 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * Perlin3(p.Scale(freq))
		norm += amp
		amp *= gain
		freq *= lacunarity
	}
	if norm == 0 {
		return 0
	}
	return sum / norm
}

func noiseHash3(i, j, k int) Vec3 {
	h := uint32(i)*73856093 ^ uint32(j)*19349663 ^ uint32(k)*83492791
	f := func(s uint32) float64 {
		s ^= s >> 15
		s *= 0x2c1b3c6d
		s ^= s >> 12
		s *= 0x297a2d39
		s ^= s >> 15
		return float64(s) / 4294967296.0
	}
	return Vec3{X: f(h), Y: f(h*0x9e3779b9 + 1), Z: f(h*0x85ebca6b + 2)}
}

// Worley3 is cellular (Worley) noise: the distance to the nearest randomly-placed
// feature point (F1), giving cell/scale/crack patterns. Roughly in [0, ~1.4].
func Worley3(p Vec3) float64 {
	bx, by, bz := int(math.Floor(p.X)), int(math.Floor(p.Y)), int(math.Floor(p.Z))
	minD := math.MaxFloat64
	for di := -1; di <= 1; di++ {
		for dj := -1; dj <= 1; dj++ {
			for dk := -1; dk <= 1; dk++ {
				ci, cj, ck := bx+di, by+dj, bz+dk
				fp := Vec3{X: float64(ci), Y: float64(cj), Z: float64(ck)}.Add(noiseHash3(ci, cj, ck))
				if d := p.Sub(fp).LenSq(); d < minD {
					minD = d
				}
			}
		}
	}
	return math.Sqrt(minD)
}

// NoiseTex blends colours A->B by FBM noise at the world point (Scale sets the
// feature size). Good for clouds, dirt and general variation.
type NoiseTex struct {
	A, B    Vec3
	Scale   float64
	Octaves int
}

// At implements Texture.
func (n NoiseTex) At(_, _ float64, p Vec3) Vec3 {
	s := n.Scale
	if s <= 0 {
		s = 1
	}
	t := 0.5 * (FBM(p.Scale(1/s), n.Octaves, 2, 0.5) + 1)
	return n.A.Add(n.B.Sub(n.A).Scale(t))
}

// MarbleTex is a marble vein pattern: a sine wave along X warped by turbulence,
// blending A (stone) and B (vein).
type MarbleTex struct {
	A, B        Vec3
	Scale, Turb float64
}

// At implements Texture.
func (m MarbleTex) At(_, _ float64, p Vec3) Vec3 {
	s := m.Scale
	if s <= 0 {
		s = 1
	}
	turb := m.Turb
	if turb <= 0 {
		turb = 1
	}
	val := math.Sin((p.X/s + turb*FBM(p.Scale(1/s), 5, 2, 0.5)) * math.Pi)
	t := 0.5 * (val + 1)
	return m.A.Add(m.B.Sub(m.A).Scale(t))
}
