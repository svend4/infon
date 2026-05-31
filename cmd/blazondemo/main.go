// Command blazondemo renders example coats of arms (heraldic blazons) to PNGs,
// proving that a medieval visual-identity language compiles straight into the
// pseudo-image pipeline.
//
//	go run ./cmd/blazondemo [outdir]
package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/svend4/infon/pkg/blazon"
)

var arms = []struct{ name, blazon string }{
	{"01_sun", "Azure, a sun Or"},
	{"02_lions", "Gules, three lions Or"},
	{"03_fleurs", "Azure, three fleurs-de-lis Or"},
	{"04_cross", "Argent, a cross Gules"},
	{"05_saltire", "Azure, a saltire Argent"},
	{"06_chief", "Gules, on a chief Argent three mullets Sable"},
	{"07_tower", "Sable, a tower Or, a bordure Argent"},
	{"08_anchor", "Vert, an anchor Argent in fess, a chief Or"},
}

func main() {
	out := "./_blazon"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		panic(err)
	}
	for _, a := range arms {
		sp := blazon.Parse(a.blazon)
		if _, err := sp.Frame(); err != nil {
			panic(err)
		}
		img, err := sp.Image(384, 192)
		if err != nil {
			panic(err)
		}
		f, err := os.Create(filepath.Join(out, a.name+".png"))
		if err != nil {
			panic(err)
		}
		if err := png.Encode(f, img); err != nil {
			panic(err)
		}
		_ = f.Close()
		fmt.Printf("ok  %-11s  %s\n", a.name, strings.TrimSpace(a.blazon))
	}
	fmt.Printf("done: %s\n", out)
}
