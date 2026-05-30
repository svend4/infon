// Command aiimg renders a raster image into the terminal via the repo's own
// internal/codec/babe.ImageToFrame. This is the Level-1b "pixel path":
//
//	(image model / any PNG-JPG) -> babe.ImageToFrame -> Frame -> terminal
//
// Place at cmd/aiimg/main.go inside the svend4/infon module and run:
//
//	go run ./cmd/aiimg assets/aurora.png 64 40
package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"

	"github.com/svend4/infon/internal/codec/babe"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: aiimg <image> [cols] [rows]")
		os.Exit(1)
	}
	cols, rows := 64, 40
	if len(os.Args) >= 4 {
		if v, err := strconv.Atoi(os.Args[2]); err == nil {
			cols = v
		}
		if v, err := strconv.Atoi(os.Args[3]); err == nil {
			rows = v
		}
	}
	fh, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "aiimg:", err)
		os.Exit(1)
	}
	defer fh.Close()
	img, _, err := image.Decode(fh)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aiimg: decode:", err)
		os.Exit(1)
	}
	babe.ImageToFrame(img, cols, rows).RenderToTerminal()
}
