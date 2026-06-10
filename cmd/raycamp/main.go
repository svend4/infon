// Command raycamp runs a robot logistics yard — Part 1's "logistics in a camp".
// The yard floor is a square forest (the navigable grid of blocks and roads); a
// small swarm of robots carries cargo from a hub to depots, with routes planned by
// the BubbleGraph router. Each robot has a role placed on the Q6 hypercube, so the
// swarm's relationships draw as a HexBridges graph and a Q6 navigator — the same
// 6-bit coordination info150's portal uses. It prints a text logistics report and
// renders three views: the 3-D yard with a highlighted route, the swarm graph, and
// the Q6 map.
//
//	go run ./cmd/raycamp                 # writes camp.png, camp.swarm.png, camp.q6.png
//	go run ./cmd/raycamp -gx 8 -gz 8 -seed 4 -spp 64
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"

	"github.com/svend4/infon/pkg/fleet"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
)

func main() {
	var (
		out  = flag.String("out", "camp", "output basename")
		gx   = flag.Int("gx", 8, "yard width in cells")
		gz   = flag.Int("gz", 8, "yard depth in cells")
		seed = flag.Int64("seed", 0, "layout seed (0 = search for a well-connected yard)")
		w    = flag.Int("w", 760, "render width")
		h    = flag.Int("h", 480, "render height")
		spp  = flag.Int("spp", 64, "samples per pixel")
	)
	flag.Parse()

	// Build a yard and four robots with jobs from a hub to reachable depots. If no
	// seed is given, search a few until the yard connects a hub to >= 4 depots.
	camp, hub, dests := buildYard(*gx, *gz, *seed)
	if len(dests) < 1 {
		fmt.Fprintln(os.Stderr, "no connected depots in this yard; try another -seed")
		os.Exit(1)
	}

	hex := func(s string) raydir.Hexagram { v, _ := raydir.ParseHexagram(s); return v }
	roles := []struct {
		name string
		role raydir.Hexagram
	}{
		{"haul-1", hex("110000")},
		{"haul-2", hex("110000")},    // same role as haul-1 (Hamming 0)
		{"charge-1", hex("110001")},  // one line from the haulers
		{"inspect-1", hex("001100")}, // a distant role
	}
	// Pick depots spread across the yard (evenly through the sorted reachable set),
	// so the swarm does not bunch up next to the hub.
	pick := make([][2]int, len(roles))
	for i := range roles {
		pick[i] = dests[(i+1)*len(dests)/(len(roles)+1)]
	}

	// Each robot starts at its own depot and carries cargo to the hub.
	units := make([]fleet.CampUnit, 0, len(roles))
	for i, r := range roles {
		units = append(units, fleet.CampUnit{Name: r.name, Role: r.role, From: pick[i], To: hub})
	}

	// Logistics report.
	fmt.Printf("logistics yard %dx%d — hub %v, %d depots reachable\n", *gx, *gz, hub, len(dests))
	for _, u := range units {
		route := camp.PlanRoute(u.From, u.To)
		if route == nil {
			fmt.Printf("  %-10s depot %v -> hub %v   NO ROUTE\n", u.Name, u.From, u.To)
			continue
		}
		fmt.Printf("  %-10s depot %v -> hub %v   %d cells\n", u.Name, u.From, u.To, len(route))
	}

	// Render the 3-D yard with haul-1's route highlighted.
	route := camp.PlanRoute(units[0].From, units[0].To)
	scene := camp.CampScene(units, route)
	span := math.Max(float64(*gx), float64(*gz)) * 8 // forest pitch is 8
	cam := raytrace.Camera{
		Pos:   raytrace.Vec3{X: span * 0.5, Y: span * 0.55, Z: -span * 0.4},
		Pitch: -0.5, FOV: math.Pi / 3,
	}
	img := raytrace.PathRender(scene, cam, *w, *h, raytrace.PathOptions{
		Samples: *spp, MaxDepth: 5, Seed: 9, NEE: true, MIS: true, Sobol: true,
	})
	writePNG(*out+".png", img)

	// The swarm graph (HexBridges) and the Q6 navigator (the haulers' role).
	swarm := fleet.SwarmGraph(units, 1)
	writePNG(*out+".swarm.png", swarm.BubbleMap(swarm.Home(), nil, 640, 420))
	writePNG(*out+".q6.png", raydir.Q6Map(roles[0].role, 480, 480))

	fmt.Printf("wrote %s.png, %s.swarm.png, %s.q6.png\n", *out, *out, *out)
}

// buildYard returns a camp, a hub cell, and the depots reachable from it. With
// seed 0 it searches seeds 1..40 for a yard whose hub reaches at least four
// depots, so the demo always has work to do.
func buildYard(gx, gz int, seed int64) (*fleet.Camp, [2]int, [][2]int) {
	try := func(s int64) (*fleet.Camp, [2]int, [][2]int) {
		camp := fleet.NewCamp(gx, gz, s)
		open := camp.OpenCells()
		if len(open) == 0 {
			return camp, [2]int{}, nil
		}
		hub := open[0]
		var dests [][2]int
		for _, c := range open[1:] {
			if camp.PlanRoute(hub, c) != nil {
				dests = append(dests, c)
			}
		}
		return camp, hub, dests
	}
	if seed != 0 {
		c, hub, d := try(seed)
		return c, hub, d
	}
	var best *fleet.Camp
	var bestHub [2]int
	var bestDests [][2]int
	for s := int64(1); s <= 40; s++ {
		c, hub, d := try(s)
		if len(d) >= 4 {
			return c, hub, d
		}
		if len(d) > len(bestDests) {
			best, bestHub, bestDests = c, hub, d
		}
	}
	return best, bestHub, bestDests
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
