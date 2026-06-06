// Command tangram7demo renders the AUTHENTIC seven-piece geometric tangram: the
// canonical square (verified to tile) and an exploded tray of all seven pieces.
//
//	go run ./cmd/tangram7demo [outdir]
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/svend4/infon/pkg/tangram7"
)

func main() {
	out := "./_tangram7"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	_ = os.MkdirAll(out, 0o755)

	sq := tangram7.Square()
	tray := tangram7.Tray()
	diamond := tangram7.Place(sq, 1, false, 0, 0) // rotate the square 45 degrees

	save(filepath.Join(out, "square.png"), tangram7.Render(sq, 460, 460))
	save(filepath.Join(out, "tray.png"), tangram7.Render(tray, 760, 360))
	save(filepath.Join(out, "diamond.png"), tangram7.Render(diamond, 460, 460))

	r := tangram7.Verify(sq, 260)
	fmt.Println("=== authentic 7-piece geometric tangram ===")
	fmt.Printf("canonical square: %d pieces, total area %.2f, overlap %.4f%% -> tiles exactly\n",
		r.Pieces, r.Area, r.Overlap*100)
	fmt.Printf("renders in %s: square.png, tray.png (all 7 pieces), diamond.png\n", out)
}

func save(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
