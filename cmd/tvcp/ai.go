package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/svend4/infon/internal/aisource"
	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/pkg/color"
)

func runAI() {
	fs := flag.NewFlagSet("ai", flag.ContinueOnError)
	brainURL := fs.String("brain", "", "paint frames with a tvcp-ai/1 brain")
	promptFlag := fs.String("prompt", "a calm landscape", "scene prompt (with -brain)")
	_ = fs.Parse(os.Args[2:])

	cols, rows := 80, 24
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	fmt.Print(color.ClearScreen)

	if *brainURL != "" {
		phases := []string{"at dawn", "in the morning", "at noon", "at sunset", "at night"}
		i := 0
		for {
			select {
			case <-sig:
				fmt.Print(color.Reset, color.ClearScreen)
				return
			default:
				aisource.BrainFrame(*brainURL, *promptFlag+" "+phases[i%len(phases)], cols, rows).RenderToTerminal()
				i++
			}
		}
	}

	cam := aisource.NewAICamera(cols*2, rows*2, 15)
	_ = cam.Open()
	defer cam.Close()
	tick := time.NewTicker(time.Duration(1000.0/cam.GetFPS()) * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-sig:
			fmt.Print(color.Reset, color.ClearScreen)
			return
		case <-tick.C:
			img, _ := cam.Read()
			babe.ImageToFrame(img, cols, rows).RenderToTerminal()
		}
	}
}
