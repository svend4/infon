package raydir

import (
	"encoding/json"
	"math"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

// WorldDirector evolves the *scene* (its spheres) one tick at a time. Combined
// with a (camera) Director and the broadcast codec, a brain can author a moving
// 3-D world that streams to spectators in a few bytes per tick.
type WorldDirector interface {
	NextWorld(spheres []raytrace.Sphere) []raytrace.Sphere
}

func worldCenter(sph []raytrace.Sphere) raytrace.Vec3 {
	if len(sph) == 0 {
		return raytrace.Vec3{}
	}
	var c raytrace.Vec3
	for _, s := range sph {
		c = c.Add(s.Center)
	}
	return c.Scale(1 / float64(len(sph)))
}

// applyWorld maps {spin,rise,fold} onto the scene: spin orbits the spheres about
// the vertical axis through their centroid, rise bobs their height, fold pulses
// their radius (all clamped so they stay above the floor and sane).
func applyWorld(sph []raytrace.Sphere, d directives) []raytrace.Sphere {
	out := make([]raytrace.Sphere, len(sph))
	copy(out, sph)
	c := worldCenter(out)
	th := float64(clampd(d.Spin)) * 0.12
	cs, sn := math.Cos(th), math.Sin(th)
	for i := range out {
		dx := out[i].Center.X - c.X
		dz := out[i].Center.Z - c.Z
		out[i].Center.X = c.X + dx*cs - dz*sn
		out[i].Center.Z = c.Z + dx*sn + dz*cs
		out[i].Radius = clampf(out[i].Radius+float64(clampd(d.Fold))*0.03, 0.3, 2.0)
		out[i].Center.Y = clampf(out[i].Center.Y+float64(clampd(d.Rise))*0.05, out[i].Radius, 5)
	}
	return out
}

// RefWorldDirector is a scripted scenographer: orbit the spheres and slowly
// breathe their size/height.
type RefWorldDirector struct{ t int }

// NextWorld implements WorldDirector.
func (w *RefWorldDirector) NextWorld(sph []raytrace.Sphere) []raytrace.Sphere {
	w.t++
	d := directives{Spin: 1}
	if (w.t/20)%2 == 0 {
		d.Rise, d.Fold = 1, 1
	} else {
		d.Rise, d.Fold = -1, -1
	}
	return applyWorld(sph, d)
}

// BrainWorldDirector lets any tvcp-ai brain author the world's motion via
// game:"world": it sends a Brief of the scene (sphere count, mean size/height,
// reusing the fields the reference world brain reads) and applies the returned
// {fold,rise,spin} directives. brain.Local works out of the box; a live model
// flies it too.
type BrainWorldDirector struct {
	B    brain.Brain
	last directives
}

// NextWorld implements WorldDirector.
func (w *BrainWorldDirector) NextWorld(sph []raytrace.Sphere) []raytrace.Sphere {
	var avgR, avgH float64
	for _, s := range sph {
		avgR += s.Radius
		avgH += s.Center.Y
	}
	if len(sph) > 0 {
		avgR /= float64(len(sph))
		avgH /= float64(len(sph))
	}
	b := brief{
		FoldPct:   int(avgR / 2.0 * 100),
		ReliefPct: int(avgH / 5.0 * 100),
		Menu:      "reply {fold,rise,spin,camera}, each -1|0|1",
	}
	raw, _ := json.Marshal(b)
	resp, err := w.B.Decide(brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "world", State: raw})
	d := w.last
	if err == nil && len(resp.World) > 0 {
		var nd directives
		if json.Unmarshal(resp.World, &nd) == nil {
			d = directives{Fold: clampd(nd.Fold), Rise: clampd(nd.Rise), Spin: clampd(nd.Spin), Camera: clampd(nd.Camera)}
			w.last = d
		}
	}
	return applyWorld(sph, d)
}

// RaySphereDirector authors the scene per sphere via the NATIVE game:"ray"
// protocol (not the {fold,rise,spin} reuse): it sends each sphere's position and
// applies the brain's {id,dx,dy,dz} moves. brain.Local (the reference
// scenographer) gathers the spheres toward their centroid; a live model can place
// them however it likes.
type RaySphereDirector struct{ B brain.Brain }

type raySphereState struct {
	ID int     `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	Z  float64 `json:"z"`
}

type rayMove struct {
	ID int `json:"id"`
	DX int `json:"dx"`
	DY int `json:"dy"`
	DZ int `json:"dz"`
}

// NextWorld implements WorldDirector via game:"ray".
func (c RaySphereDirector) NextWorld(sph []raytrace.Sphere) []raytrace.Sphere {
	brief := struct {
		Spheres []raySphereState `json:"spheres"`
	}{}
	for i, s := range sph {
		brief.Spheres = append(brief.Spheres, raySphereState{ID: i, X: s.Center.X, Y: s.Center.Y, Z: s.Center.Z})
	}
	raw, _ := json.Marshal(brief)
	resp, err := c.B.Decide(brain.Request{Protocol: brain.Protocol, Kind: "move", Game: "ray", State: raw})
	out := make([]raytrace.Sphere, len(sph))
	copy(out, sph)
	if err != nil || len(resp.Ray) == 0 {
		return out
	}
	var moves []rayMove
	if json.Unmarshal(resp.Ray, &moves) != nil {
		return out
	}
	const step = 0.15
	for _, m := range moves {
		if m.ID < 0 || m.ID >= len(out) {
			continue
		}
		out[m.ID].Center = out[m.ID].Center.Add(raytrace.Vec3{
			X: float64(clampd(m.DX)) * step,
			Y: float64(clampd(m.DY)) * step,
			Z: float64(clampd(m.DZ)) * step,
		})
		if out[m.ID].Center.Y < out[m.ID].Radius {
			out[m.ID].Center.Y = out[m.ID].Radius
		}
	}
	return out
}

// Choreograph runs a camera Director and a WorldDirector together for n ticks and
// returns a raytrace.SceneAnim (camera + light + spheres per frame), ready to
// render, stream over RS (EncodeAnim) or broadcast to spectators (EncodeBroadcast).
func Choreograph(cam Director, world WorldDirector, init CamState, spheres []raytrace.Sphere, light raytrace.Vec3, n int) raytrace.SceneAnim {
	cs := init
	cur := append([]raytrace.Sphere(nil), spheres...)
	anim := raytrace.SceneAnim{N: len(spheres)}
	for i := 0; i < n; i++ {
		cs = cam.Next(cs)
		cur = world.NextWorld(cur)
		fr := make([]raytrace.Sphere, len(cur))
		copy(fr, cur)
		anim.Frames = append(anim.Frames, raytrace.SceneWire{Cam: cs.Camera(), Light: light, Spheres: fr})
	}
	return anim
}
