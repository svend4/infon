// Command raygates runs the sim-to-real conformance gates — Part 1's variant E.
// The same demo logistics scenario is assessed twice: once as clean "sim"
// readings and once perturbed by a sim-to-real gap (bias + noise). The two runs'
// conclusions are scored against triage4-style gates (level agreement, dominant-
// cause agreement, severity MAE, shared-cause bridge agreement). It prints the
// gate table and draws a dashboard, so you can see how far the gap can grow before
// the engine, validated in simulation, would diverge on real telemetry.
//
//	go run ./cmd/raygates            # writes gates.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/svend4/infon/pkg/fleet"
	"github.com/svend4/infon/pkg/microfont"
)

type profile struct {
	name string
	gap  fleet.GapModel
}

func main() {
	out := flag.String("out", "gates", "output basename (writes <out>.png)")
	flag.Parse()

	sim := fleet.DemoScenario()
	profiles := []profile{
		{"tight   noise .05", fleet.GapModel{Noise: 0.05, Seed: 1}},
		{"loose   noise .15", fleet.GapModel{Noise: 0.15, Seed: 1}},
		{"biased  bias .12", fleet.GapModel{Bias: 0.12, Seed: 1}},
		{"broken  noise .45", fleet.GapModel{Noise: 0.45, Seed: 1}},
	}
	reps := make([]fleet.GateReport, len(profiles))

	fmt.Println("sim-to-real gates (sim: demo logistics scenario)")
	fmt.Printf("  %-18s %-7s %-7s %-8s %-7s %s\n", "profile", "level", "cause", "sevMAE", "bridge", "verdict")
	for i, p := range profiles {
		rep := fleet.Compare(sim, fleet.ApplyGap(sim, p.gap))
		reps[i] = rep
		m := metricByName(rep)
		verdict := "PASS"
		if !rep.Pass {
			verdict = "FAIL"
		}
		fmt.Printf("  %-18s %-7.2f %-7.2f %-8.3f %-7.2f %s\n", p.name,
			m["level-agreement"], m["cause-agreement"], m["severity-MAE"], m["bridge-agreement"], verdict)
	}

	img := drawDashboard(profiles, reps)
	f, err := os.Create(*out + ".png")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s.png\n", *out)
}

func metricByName(rep fleet.GateReport) map[string]float64 {
	m := map[string]float64{}
	for _, g := range rep.Gates {
		m[g.Name] = g.Metric
	}
	return m
}

func drawDashboard(profiles []profile, reps []fleet.GateReport) image.Image {
	const (
		w     = 760
		rowH  = 40
		top   = 64
		nameX = 16
	)
	h := top + len(profiles)*rowH + 14
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	dim := color.RGBA{R: 150, G: 150, B: 170, A: 255}
	white := color.RGBA{R: 225, G: 225, B: 235, A: 255}
	ink := color.RGBA{R: 18, G: 18, B: 24, A: 255}
	pass := color.RGBA{R: 60, G: 160, B: 90, A: 255}
	fail := color.RGBA{R: 185, G: 65, B: 65, A: 255}
	fillRect(img, 0, 0, w, h, ink)
	microfont.Draw(img, 16, 12, 2, "SIM -> REAL CONFORMANCE GATES", white)
	microfont.Draw(img, 16, 40, 1, "same conclusions on noisy telemetry? (info150 triage gates)", dim)

	gates := []string{"level-agreement", "cause-agreement", "severity-MAE", "bridge-agreement"}
	short := []string{"level", "cause", "sevMAE", "bridge"}
	colX := []int{210, 320, 430, 545}
	for i, s := range short {
		microfont.Draw(img, colX[i], top-12, 1, s, dim)
	}
	microfont.Draw(img, 662, top-12, 1, "verdict", dim)

	for r, p := range profiles {
		y := top + r*rowH
		microfont.Draw(img, nameX, y+rowH/2-4, 1, p.name, white)
		gmap := map[string]fleet.Gate{}
		for _, g := range reps[r].Gates {
			gmap[g.Name] = g
		}
		for i, gn := range gates {
			g := gmap[gn]
			cell := fail
			if g.Pass {
				cell = pass
			}
			fillRect(img, colX[i]-6, y+6, 96, rowH-14, cell)
			microfont.Draw(img, colX[i], y+rowH/2-4, 1, fmt.Sprintf("%.2f", g.Metric), ink)
		}
		vcol, vtxt := fail, "FAIL"
		if reps[r].Pass {
			vcol, vtxt = pass, "PASS"
		}
		fillRect(img, 656, y+6, 78, rowH-14, vcol)
		microfont.Draw(img, 672, y+rowH/2-4, 1, vtxt, ink)
	}
	return img
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			img.SetRGBA(i, j, c)
		}
	}
}
