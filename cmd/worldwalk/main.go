// Command worldwalk is the project's FIRST capability married to its newest one:
// it renders the living pseudo-3D world (pkg/world) to an image, then shows that
// image in the terminal with the original block codec (internal/codec/babe ->
// terminal.Frame quadrant blocks, 24-bit colour) - redrawn in place, frame by
// frame. A walk through a living surreal world, in pure terminal video.
//
//	go run ./cmd/worldwalk                 # reference controller drives the world
//	go run ./cmd/worldwalk -fps 12 -cols 80 -rows 36
//	go run ./cmd/worldwalk -brain http://127.0.0.1:8092/v1/decide   # a live LLM directs it
package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/world"
)

func main() {
	fps := flag.Int("fps", 8, "frames per second")
	cols := flag.Int("cols", 64, "terminal width in cells")
	rows := flag.Int("rows", 30, "terminal height in cells")
	turns := flag.Int("turns", 48, "world frames to generate")
	loops := flag.Int("loops", 2, "how many times to replay the world")
	url := flag.String("brain", "", "tvcp-ai/1 URL to direct the world via game:world (default: reference)")
	flag.Parse()
	if *fps < 1 {
		*fps = 1
	}

	var ctrl world.Controller = world.RefController{}
	driver := "reference"
	if *url != "" {
		ctrl = &world.BrainController{B: brain.HTTPBrain{URL: *url, HTTP: &http.Client{Timeout: 60 * time.Second}}}
		driver = "live brain"
	}
	states, _ := world.LoopC(ctrl, world.Init(), *turns)
	if len(states) == 0 {
		fmt.Println("no world frames")
		return
	}

	imgW, imgH := *cols*2, *rows*2
	delay := time.Second / time.Duration(*fps)
	total := *loops * len(states)

	fmt.Print("\x1b[2J\x1b[?25l")       // clear, hide cursor
	defer fmt.Print("\x1b[?25h\x1b[0m") // restore cursor on exit

	for i := 0; i < total; i++ {
		st := states[i%len(states)]
		fr := babe.ImageToFrame(st.Render(imgW, imgH), *cols, *rows)
		fmt.Printf("\x1b[H\x1b[1mTVCP worldwalk\x1b[0m  image->blocks (babe)  driver=%s  frame %3d/%d        \r\n%s",
			driver, i+1, total, fr.Render())
		time.Sleep(delay)
	}
	fmt.Print("\r\ndone.\r\n")
}
