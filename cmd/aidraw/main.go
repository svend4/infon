// Command aidraw renders to the terminal via pkg/scene. Scene from file/stdin,
// or from any tvcp-ai/1 brain (-brain). -sketch requests the simple high-level
// sketch format (best for small models); otherwise the full draw-DSL is used.
// The requested canvas size is always enforced.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/scene"
	"github.com/svend4/infon/pkg/sketch"
)

func main() {
	brainURL := flag.String("brain", "", "tvcp-ai/1 brain endpoint")
	prompt := flag.String("prompt", "", "draw prompt (with -brain)")
	canvas := flag.String("canvas", "64x28", "WxH canvas (with -brain)")
	sketchMode := flag.Bool("sketch", false, "request the simple sketch format (best for small models)")
	flag.Parse()

	if *brainURL != "" {
		w, h := 64, 28
		if p := strings.SplitN(*canvas, "x", 2); len(p) == 2 {
			if v, e := strconv.Atoi(p[0]); e == nil {
				w = v
			}
			if v, e := strconv.Atoi(p[1]); e == nil {
				h = v
			}
		}
		hb := brain.HTTPBrain{URL: *brainURL, HTTP: &http.Client{Timeout: 120 * time.Second}}
		if *sketchMode {
			resp, e := hb.Decide(brain.Request{Kind: "sketch", Prompt: *prompt, Canvas: &brain.Canvas{Width: w, Height: h}})
			if e != nil {
				fmt.Fprintln(os.Stderr, "aidraw brain:", e)
				os.Exit(1)
			}
			var sk sketch.Sketch
			if resp.Error != "" || json.Unmarshal(resp.Sketch, &sk) != nil {
				fmt.Fprintln(os.Stderr, "aidraw: brain returned no/invalid sketch:", resp.Error)
				os.Exit(1)
			}
			sc := sketch.ToScene(sk, w, h)
			scene.RenderScene(&sc).RenderToTerminal()
			return
		}
		resp, e := hb.Decide(brain.Request{Kind: "draw", Prompt: *prompt, Canvas: &brain.Canvas{Width: w, Height: h}})
		if e != nil {
			fmt.Fprintln(os.Stderr, "aidraw brain:", e)
			os.Exit(1)
		}
		var sc scene.Scene
		if resp.Error != "" || json.Unmarshal(resp.Scene, &sc) != nil {
			fmt.Fprintln(os.Stderr, "aidraw: brain returned no/invalid scene:", resp.Error)
			os.Exit(1)
		}
		sc.Width, sc.Height = w, h
		scene.RenderScene(&sc).RenderToTerminal()
		return
	}

	var data []byte
	var err error
	if args := flag.Args(); len(args) > 0 {
		data, err = os.ReadFile(args[0])
	} else {
		data, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "aidraw:", err)
		os.Exit(1)
	}
	f, e := scene.Render(data)
	if e != nil {
		fmt.Fprintln(os.Stderr, "aidraw render:", e)
		os.Exit(1)
	}
	f.RenderToTerminal()
}
