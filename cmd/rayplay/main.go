// Command rayplay replays a recorded shared-world session (see raymeet -record):
// it reconstructs the world, the walkers and the chat over time from a tiny file of
// timestamped events — meaning, not pixels — and plays it back in the terminal from
// a participant's point of view. Watch a walk you (or an AI director) took earlier.
//
//	go run ./cmd/rayplay walk.rrec            # replay at real time
//	go run ./cmd/rayplay -speed 4 walk.rrec   # 4x faster
//	go run ./cmd/rayplay -path walk.rrec      # path-traced playback
package main

import (
	"flag"
	"fmt"
	"image"
	"math"
	"os"
	"sort"
	"time"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/pkg/raydir"
	"github.com/svend4/infon/pkg/raytrace"
	"github.com/svend4/infon/pkg/terminal"
)

func main() {
	modeFlag := flag.String("mode", "auto", "terminal render mode")
	speed := flag.Float64("speed", 1.0, "playback speed multiplier")
	pathT := flag.Bool("path", false, "path-trace (prettier, slower)")
	cols := flag.Int("w", 80, "width in cells")
	rows := flag.Int("h", 36, "height in cells")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: rayplay [flags] <recording.rrec>")
		os.Exit(1)
	}
	ev, err := raydir.LoadRecording(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	player := raydir.NewPlayer(ev)
	pxW, pxH := *cols*2, *rows*4

	mode := *modeFlag
	if mode == "" || mode == "auto" {
		mode = terminal.DetectCapability().BestBlitMode()
	}
	rm, _ := babe.ParseRenderMode(mode)
	dr := terminal.NewDiffRenderer()
	pathOpt := raytrace.PathOptions{Samples: 3, MaxDepth: 4, Seed: 1, NEE: true, MIS: true, Sobol: true}

	dur := player.Duration()
	fmt.Printf("rayplay — replaying %d events over %.1fs at %.1fx (meaning, not pixels).\n", len(ev), float64(dur)/1000, *speed)
	start := time.Now()
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		tMs := uint32(time.Since(start).Seconds() * 1000 * *speed)
		player.Advance(tMs)

		poses := player.Poses()
		ids := make([]uint32, 0, len(poses))
		for id := range poses {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

		// follow the first walker; draw the others as avatars.
		cam := raytrace.Camera{Pos: raytrace.Vec3{X: 0, Y: 6, Z: -6}, Pitch: -0.5, FOV: math.Pi / 3}
		var follow uint32
		var extra []raytrace.Object
		if len(ids) > 0 {
			follow = ids[0]
			fp := poses[follow]
			cam = raydir.FlyCam{Pos: fp.Pos, Yaw: fp.Yaw, Pitch: fp.Pitch, FOV: math.Pi / 3}.Camera()
		}
		for _, id := range ids {
			if id != follow {
				extra = append(extra, raydir.AvatarSpheres(poses[id], raydir.AvatarColor(id))...)
			}
		}
		scene := player.World().SceneWith(extra)

		var im image.Image
		if *pathT {
			im = raytrace.PathRender(scene, cam, pxW, pxH, pathOpt)
		} else {
			im = raytrace.Render(scene, cam, pxW, pxH, raytrace.Options{Samples: 1})
		}
		fmt.Print(dr.Render(babe.ImageToFrameMode(im, *cols, *rows, rm)))
		fmt.Printf("\n[replay %.1fs/%.1fs | chunks:%d | walkers:%d]", float64(tMs)/1000, float64(dur)/1000, player.World().Chunks(), len(ids))
		for _, l := range player.Chat().Lines() {
			fmt.Printf("\n  💬 %s", l)
		}
		fmt.Print(" ")
		if tMs > dur+1000 { // a beat past the end, then stop
			fmt.Println("\n— end of recording —")
			return
		}
	}
}
