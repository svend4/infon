package scene

import (
	"testing"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func renderNoPanic(t *testing.T, data string) (f *terminal.Frame) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Render(%s) panicked: %v", data, r)
		}
	}()
	f, err := Render([]byte(data))
	if err != nil {
		t.Fatalf("Render(%s): %v", data, err)
	}
	return f
}

// "from"/"to" are optional in JSON; a gradient missing them must not crash.
func TestGradientMissingEndpointsDoNotPanic(t *testing.T) {
	cases := []struct {
		name string
		data string
		bg   bool // check Bg (vgradient) or Fg (hgradient)
		want color.RGB
	}{
		{"vgradient none", `{"width":3,"height":3,"ops":[{"op":"vgradient","w":3,"h":3}]}`, true, color.NewRGB(0, 0, 0)},
		{"vgradient from only", `{"width":3,"height":3,"ops":[{"op":"vgradient","w":3,"h":3,"from":[10,20,30]}]}`, true, color.NewRGB(10, 20, 30)},
		{"vgradient to only", `{"width":3,"height":3,"ops":[{"op":"vgradient","w":3,"h":3,"to":[40,50,60]}]}`, true, color.NewRGB(40, 50, 60)},
		{"hgradient none", `{"width":3,"height":3,"ops":[{"op":"hgradient","w":3,"h":3}]}`, false, color.NewRGB(0, 0, 0)},
		{"hgradient from only", `{"width":3,"height":3,"ops":[{"op":"hgradient","w":3,"h":3,"from":[70,80,90]}]}`, false, color.NewRGB(70, 80, 90)},
		{"hgradient to only", `{"width":3,"height":3,"ops":[{"op":"hgradient","w":3,"h":3,"to":[200,100,50]}]}`, false, color.NewRGB(200, 100, 50)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := renderNoPanic(t, tc.data)
			for y := 0; y < f.Height; y++ {
				for x := 0; x < f.Width; x++ {
					got := f.Blocks[y][x].Fg
					if tc.bg {
						got = f.Blocks[y][x].Bg
					}
					if got != tc.want {
						t.Fatalf("cell (%d,%d) colour = %+v, want %+v (missing endpoint takes the other one, black if both missing)", x, y, got, tc.want)
					}
				}
			}
		})
	}
}
