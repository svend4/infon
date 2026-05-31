// Command pseudodemo renders the same scene ("a calm harbor at dawn") in every
// pseudo-image format, proving that an ordinary text model — one that can only
// emit JSON, no diffusion head — can paint pictures for the terminal. Each
// format is saved as a PNG; the grid seed also produces a denoising GIF.
//
//	go run ./cmd/pseudodemo [outdir]
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"

	"github.com/svend4/infon/pkg/pseudo"
)

type example struct {
	name string
	doc  string
}

// All six formats describe the SAME subject so the styles can be compared.
var examples = []example{
	{"1_grid", `{"format":"grid","cols":80,"rows":36,"grid":{"rows":[
["navy","navy","navy","navy","navy","purple","purple","navy","navy","navy","navy","navy"],
["navy","navy","navy","purple","amber","gold","gold","amber","purple","navy","navy","navy"],
["navy","navy","purple","amber","gold","white","gold","gold","amber","purple","navy","navy"],
["navy","purple","amber","gold","gold","gold","gold","amber","amber","purple","navy","navy"],
["slate","slate","amber","amber","gold","amber","amber","teal","teal","slate","slate","slate"],
["teal","teal","teal","skyblue","teal","teal","skyblue","teal","teal","teal","skyblue","teal"],
["teal","skyblue","teal","teal","blue","teal","teal","skyblue","teal","blue","teal","teal"],
["blue","teal","blue","teal","teal","blue","teal","teal","blue","teal","teal","blue"]
]}}`},
	{"2_pixels", `{"format":"pixels","cols":80,"rows":36,"pixels":{"rows":[
["navy","navy","navy","navy","navy","purple","purple","navy","navy","navy","navy","navy"],
["navy","navy","navy","purple","amber","gold","gold","amber","purple","navy","navy","navy"],
["navy","navy","purple","amber","gold","white","gold","gold","amber","purple","navy","navy"],
["navy","purple","amber","gold","gold","gold","gold","amber","amber","purple","navy","navy"],
["slate","slate","amber","amber","gold","amber","amber","teal","teal","slate","slate","slate"],
["teal","teal","teal","skyblue","teal","teal","skyblue","teal","teal","teal","skyblue","teal"],
["teal","skyblue","teal","teal","blue","teal","teal","skyblue","teal","blue","teal","teal"],
["blue","teal","blue","teal","teal","blue","teal","teal","blue","teal","teal","blue"]
]}}`},
	{"3_glyphs", `{"format":"glyphs","cols":80,"rows":24,"glyphs":{"bg":"navy","fg":"white",
"palette":{"*":"gold","o":"amber","^":"slate","~":"skyblue","=":"teal","v":"white"},
"rows":[
"                                        ",
"                  *                     ",
"                 ooo                    ",
"                  *                     ",
"        ^^                    ^^^       ",
"      ^^^^^^      v        ^^^^^^^^^     ",
"    ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^   ",
"  ====~~~~====~~~~====~~~~====~~~~====~~ ",
"~~====~~~~====~~~~====~~~~====~~~~====~~~",
"====~~~~====~~~~====~~~~====~~~~====~~~~="
]}}`},
	{"4_sigils", `{"format":"sigils","cols":80,"rows":28,"sigils":{"sky":"navy","ground":"teal",
"items":[
{"name":"star","x":0.12,"y":0.12,"color":"white"},
{"name":"cloud","x":0.5,"y":0.12,"color":"gray"},
{"name":"cloud","x":0.25,"y":0.2,"color":"slate"},
{"name":"sun","x":0.7,"y":0.25,"color":"gold"},
{"name":"mountain","x":0.2,"y":0.55,"color":"slate"},
{"name":"mountain","x":0.32,"y":0.6,"color":"dusk"},
{"name":"anchor","x":0.6,"y":0.82,"color":"white"},
{"name":"wave","x":0.82,"y":0.86,"color":"skyblue"}
]}}`},
	{"5_vector", `{"format":"vector","cols":80,"rows":28,"vector":{"width":80,"height":28,
"bg":[28,38,88],
"ops":[
{"op":"vgradient","x":0,"y":0,"w":80,"h":16,"glyph":" ","from":[28,38,88],"to":[150,110,180]},
{"op":"disc","cx":56,"cy":8,"r":4,"glyph":"█","fg":[245,200,90]},
{"op":"rect","x":0,"y":16,"w":80,"h":12,"glyph":" ","fg":[70,160,170],"bg":[70,160,170]},
{"op":"hgradient","x":0,"y":16,"w":80,"h":12,"glyph":"▀","from":[60,150,170],"to":[120,200,210]}
]}}`},
	{"6_sketch", `{"format":"sketch","cols":80,"rows":24,"sketch":{"sky":"navy","ground":"teal",
"shapes":[
{"kind":"sun","x":56,"y":6,"w":3,"color":"gold"},
{"kind":"cloud","x":10,"y":4,"w":10,"color":"gray"},
{"kind":"mountain","x":24,"h":8,"color":"slate"},
{"kind":"mountain","x":40,"h":6,"color":"dusk"},
{"kind":"star","x":8,"y":3,"color":"white"},
{"kind":"band","y":21,"h":2,"color":"skyblue"}
]}}`},
}

func main() {
	out := "./_pseudo_go"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fail(err)
	}

	const pxW, pxH = 480, 216
	for _, ex := range examples {
		var s pseudo.Spec
		if err := json.Unmarshal([]byte(ex.doc), &s); err != nil {
			fail(fmt.Errorf("%s: parse: %w", ex.name, err))
		}
		if _, err := s.Frame(); err != nil { // terminal path must work too
			fail(fmt.Errorf("%s: frame: %w", ex.name, err))
		}
		img, err := s.Image(pxW, pxH)
		if err != nil {
			fail(fmt.Errorf("%s: image: %w", ex.name, err))
		}
		savePNG(filepath.Join(out, ex.name+".png"), img)
		fmt.Printf("ok  %-9s -> %s.png\n", ex.name, ex.name)
	}

	// Denoising animation from the grid seed (the "pseudo-diffusion" headline).
	var gs pseudo.Spec
	if err := json.Unmarshal([]byte(examples[0].doc), &gs); err != nil {
		fail(err)
	}
	frames := gs.Grid.DiffuseFrames(pxW, pxH, 14)
	saveGIF(filepath.Join(out, "pseudo_diffusion.gif"), frames)
	fmt.Printf("ok  gif       -> pseudo_diffusion.gif (%d frames)\n", len(frames))
	fmt.Printf("done: %s\n", out)
}

func savePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fail(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		fail(err)
	}
}

func saveGIF(path string, frames []image.Image) {
	g := &gif.GIF{}
	for _, fr := range frames {
		g.Image = append(g.Image, toPaletted(fr))
		g.Delay = append(g.Delay, 10)
	}
	f, err := os.Create(path)
	if err != nil {
		fail(err)
	}
	defer func() { _ = f.Close() }()
	if err := gif.EncodeAll(f, g); err != nil {
		fail(err)
	}
}

func toPaletted(img image.Image) *image.Paletted {
	p := image.NewPaletted(img.Bounds(), palette.Plan9)
	draw.Draw(p, p.Bounds(), img, img.Bounds().Min, draw.Src)
	return p
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "pseudodemo:", err)
	os.Exit(1)
}
