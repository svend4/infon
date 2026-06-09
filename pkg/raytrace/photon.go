// photon.go adds caustics by photon mapping. A pre-pass shoots photons from the
// emissive spheres; a photon that passes through one or more specular surfaces
// (glass / mirror / metal) and then lands on a diffuse surface is stored as a
// "caustic photon". The photons are bucketed into a uniform grid; at render time
// a diffuse hit gathers the nearby photons (radius query) and density-estimates
// the focused light - the sharp bright patterns a forward path tracer finds only
// very slowly. It is a clean-room implementation.
//
// This is a consistent (bias -> 0 as photons -> inf, radius -> 0) estimator added
// on top of path tracing; pair it with modest light sizes for the cleanest look.
package raytrace

import "math"

type photon struct {
	pos, power Vec3
}

// PhotonMap is a grid-hashed set of caustic photons.
type PhotonMap struct {
	photons []photon
	grid    map[[3]int][]int32
	cell    float64
	radius  float64
}

func (pm *PhotonMap) cellOf(p Vec3) [3]int {
	return [3]int{
		int(math.Floor(p.X / pm.cell)),
		int(math.Floor(p.Y / pm.cell)),
		int(math.Floor(p.Z / pm.cell)),
	}
}

func (pm *PhotonMap) add(pos, power Vec3) {
	idx := int32(len(pm.photons))
	pm.photons = append(pm.photons, photon{pos, power})
	c := pm.cellOf(pos)
	pm.grid[c] = append(pm.grid[c], idx)
}

// Count is the number of stored caustic photons.
func (pm *PhotonMap) Count() int { return len(pm.photons) }

// BuildCaustics traces n photons from the scene's emissive spheres through
// specular surfaces and stores those that reach a diffuse surface. radius is the
// gather radius used both for bucketing and density estimation.
func BuildCaustics(s *Scene, n int, radius float64, seed uint64) *PhotonMap {
	pm := &PhotonMap{grid: map[[3]int][]int32{}, cell: radius, radius: radius}
	s.gatherEmitters()
	if len(s.emit) == 0 || radius <= 0 || n <= 0 {
		return pm
	}
	rg := newRNG(seed)
	ne := len(s.emit)
	for i := 0; i < n; i++ {
		e := s.emit[i%ne]
		nrm := rg.unit()
		sp := e.Center.Add(nrm.Scale(e.Radius))
		// power per photon (radiance -> flux over the sphere, split over n).
		power := e.Mat.Emit.Scale(4 * math.Pi * math.Pi * e.Radius * e.Radius / float64(n))
		r := Ray{Origin: sp.Add(nrm.Scale(shadowEps)), Dir: cosineSample(nrm, rg)}
		specular := false
		for d := 0; d < 8; d++ {
			h, ok := s.closest(r, shadowEps, tFar)
			if !ok {
				break
			}
			switch {
			case h.Mat.Glass > 0:
				r = s.scatterGlass(r, h, rg)
				specular = true
			case h.Mat.Reflect > 0 || h.Mat.Metal > 0:
				if a := h.albedo(); a.LenSq() > 0 {
					power = power.Mul(a)
				}
				r = Ray{Origin: h.P.Add(h.N.Scale(shadowEps)), Dir: r.Dir.Reflect(h.N).Norm()}
				specular = true
			default:
				if specular {
					pm.add(h.P, power) // a caustic photon (L S+ D)
				}
				d = 8 // stop at the first diffuse surface
			}
		}
	}
	return pm
}

// Estimate returns the caustic radiance at a diffuse point p with normal n and
// the given albedo, by gathering photons within the map's radius.
func (pm *PhotonMap) Estimate(p, n, albedo Vec3) Vec3 {
	if len(pm.photons) == 0 {
		return Vec3{}
	}
	r2 := pm.radius * pm.radius
	c := pm.cellOf(p)
	var flux Vec3
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for dz := -1; dz <= 1; dz++ {
				for _, idx := range pm.grid[[3]int{c[0] + dx, c[1] + dy, c[2] + dz}] {
					ph := pm.photons[idx]
					if ph.pos.Sub(p).LenSq() <= r2 {
						flux = flux.Add(ph.power)
					}
				}
			}
		}
	}
	// density estimation over the disc area pi*r^2; diffuse BRDF = albedo/pi.
	return albedo.Scale(1 / math.Pi).Mul(flux).Scale(1 / (math.Pi * r2))
}
