// Command aicam streams an AI video source. Default: procedural plasma camera
// through device.Camera -> babe -> terminal. With -brain URL: a model paints
// each frame as a sketch (slow, painterly video). Emits @@@FRAME@@@-separated.
package main

import (
	"flag"
	"fmt"

	"github.com/svend4/infon/internal/aisource"
	"github.com/svend4/infon/internal/codec/babe"
	tcolor "github.com/svend4/infon/pkg/color"
)

func main() {
	brainURL := flag.String("brain", "", "paint frames with a tvcp-ai/1 brain (sketch)")
	promptFlag := flag.String("prompt", "a calm landscape", "scene prompt (with -brain)")
	frames := flag.Int("frames", 16, "frames to emit")
	flag.Parse()

	hud := tcolor.NewRGB(12, 12, 18)
	white := tcolor.NewRGB(240, 240, 245)
	red := tcolor.NewRGB(235, 70, 70)

	if *brainURL != "" {
		cols, rows := 64, 20
		phases := []string{"at dawn", "in the morning", "at noon", "at sunset", "at night"}
		for i := 0; i < *frames; i++ {
			f := aisource.BrainFrame(*brainURL, *promptFlag+" "+phases[i%len(phases)], cols, rows)
			f.DrawText(1, 0, "AI CAM brain ", white, hud)
			f.SetBlock(13, 0, '●', red, hud)
			f.DrawText(15, 0, "REC", red, hud)
			f.DrawText(1, rows-1, fmt.Sprintf("brain video  frame %02d", i), white, hud)
			fmt.Print(f.Render())
			fmt.Print("\n@@@FRAME@@@\n")
		}
		return
	}

	cols, rows := 64, 36
	cam := aisource.NewAICamera(cols*2, rows*2, 16)
	_ = cam.Open()
	defer cam.Close()
	for i := 0; i < *frames; i++ {
		img, _ := cam.Read()
		frame := babe.ImageToFrame(img, cols, rows)
		frame.DrawText(1, 0, "AI CAM ", white, hud)
		frame.SetBlock(8, 0, '●', red, hud)
		frame.DrawText(10, 0, "REC", red, hud)
		frame.DrawText(1, rows-1, fmt.Sprintf("device.Camera -> babe -> terminal   frame %02d", i), white, hud)
		fmt.Print(frame.Render())
		fmt.Print("\n@@@FRAME@@@\n")
	}
}
