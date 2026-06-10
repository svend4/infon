// Command rayfleet watches a fleet of machines the way info150's triage4-drive
// watches a driver: it ingests per-unit signal readings, fuses them into a
// severity, tracks each unit against its own baseline, and reacts — printing
// short operator cues (text) and rendering the fleet in 3-D with an alarm glow
// (graphics), plus the relationship graph of shared faults. The reaction scene is
// authored locally, or by any tvcp-ai/1 brain when BRAIN_URL is set
// (see ai/adapters/equipment_brain.py).
//
//	go run ./cmd/rayfleet                       # built-in demo fleet
//	go run ./cmd/rayfleet readings.json         # assess readings from a JSON file
//	go run ./cmd/rayfleet -sixel                # also print the view in the terminal
//	BRAIN_URL=http://127.0.0.1:8096/v1/decide go run ./cmd/rayfleet
//
// A readings file is a JSON array of {"unit","t","signals":[{"name","value","weight"}]}
// in time order; a unit may appear several times to give the engine history.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/fleet"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
	"github.com/svend4/infon/pkg/terminal"
)

func main() {
	var (
		out   = flag.String("out", "fleet", "output basename (writes <out>.png and <out>.graph.png)")
		w     = flag.Int("w", 720, "render width")
		h     = flag.Int("h", 480, "render height")
		spp   = flag.Int("spp", 64, "samples per pixel")
		sixel = flag.Bool("sixel", false, "also print the fleet view as sixel to stdout")
		vfx   = flag.Bool("vfx", true, "apply bloom + dream post (the alarm glow)")
	)
	flag.Parse()

	readings := fleet.DemoScenario()
	if flag.NArg() > 0 {
		r, err := loadReadings(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, "read:", err)
			os.Exit(1)
		}
		readings = r
	}

	// Run the engine over every reading; keep the latest assessment per unit (its
	// temporal memory accumulates across the earlier ones).
	eng := fleet.NewMonitoringEngine()
	latest := map[string]fleet.Assessment{}
	var order []string
	for _, rd := range readings {
		a := eng.Assess(rd)
		if _, seen := latest[a.Unit]; !seen {
			order = append(order, a.Unit)
		}
		latest[a.Unit] = a
	}
	as := make([]fleet.Assessment, 0, len(latest))
	for _, u := range order {
		as = append(as, latest[u])
	}
	sort.Slice(as, func(i, j int) bool {
		if as[i].Severity != as[j].Severity {
			return as[i].Severity > as[j].Severity
		}
		return as[i].Unit < as[j].Unit
	})

	printReport(as)

	// Author the reaction scene: a brain if BRAIN_URL is set, else locally.
	spec := fleet.SceneFromAssessments(as)
	if url := os.Getenv("BRAIN_URL"); url != "" {
		if s, err := brainScene(url, as); err != nil {
			fmt.Fprintln(os.Stderr, "brain:", err, "(falling back to local scene)")
		} else {
			spec = s
			fmt.Println("scene authored by brain:", url)
		}
	}

	scene := raydir.BuildScene(spec)
	cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 2.4, Z: -4.5}, Pitch: -0.12, FOV: math.Pi / 3}
	img := raytrace.PathRender(scene, cam, *w, *h, raytrace.PathOptions{
		Samples: *spp, MaxDepth: 6, Seed: 7, NEE: true, MIS: true, Sobol: true,
	})
	if *vfx {
		img = raytrace.PostProcess(img, 1.0, 0.8, 0.6) // bloom the alarm lights
		img = raytrace.Dream(img, raytrace.DreamOptions{Chroma: 0.004, Grain: 0.02, Vignette: 0.35, Seed: 1})
	}
	writePNG(*out+".png", img)

	g := fleet.FleetGraph(as)
	writePNG(*out+".graph.png", g.BubbleMap(g.Home(), nil, 640, 420))

	fmt.Printf("wrote %s.png and %s.graph.png\n", *out, *out)
	if *sixel {
		fmt.Println(terminal.EncodeSixel(img, 200))
	}
}

func printReport(as []fleet.Assessment) {
	crit, warn := 0, 0
	for _, a := range as {
		switch a.Level {
		case fleet.LevelCritical:
			crit++
		case fleet.LevelWarn:
			warn++
		}
	}
	fmt.Printf("fleet monitor — %d units, %d critical, %d warning\n", len(as), crit, warn)
	for _, a := range as {
		fmt.Printf("  %-9s sev %.2f  %-34s [%s]\n", a.Level, a.Severity, a.Cue, a.Explain)
	}
	if br := fleet.Bridges(as); len(br) > 0 {
		fmt.Println("bridges (shared cause):")
		causes := make([]string, 0, len(br))
		for c := range br {
			causes = append(causes, c)
		}
		sort.Strings(causes)
		for _, c := range causes {
			fmt.Printf("  %s: %s\n", c, strings.Join(br[c], " — "))
		}
	}
}

func brainScene(url string, as []fleet.Assessment) (brain.SceneSpec, error) {
	hb := brain.HTTPBrain{URL: url}
	resp, err := hb.Decide(fleet.ReactRequest(as))
	if err != nil {
		return brain.SceneSpec{}, err
	}
	if len(resp.Ray) == 0 {
		return brain.SceneSpec{}, fmt.Errorf("brain returned no ray scene")
	}
	var spec brain.SceneSpec
	if err := json.Unmarshal(resp.Ray, &spec); err != nil {
		return brain.SceneSpec{}, err
	}
	return spec, nil
}

func loadReadings(path string) ([]fleet.Reading, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rs []fleet.Reading
	if err := json.Unmarshal(b, &rs); err != nil {
		return nil, err
	}
	return rs, nil
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
