package raydir

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/brain"
)

// These scenes were produced by ai/adapters/yijing_brain.scene_from_vector. Each
// vector sits where a fused multiply-add (which the Go compiler emits for x*y+z on
// arm64, ppc64le and s390x) lands an ulp away from Python's twice-rounded result
// and flips a 3-decimal tie: ring x (1.056 vs 1.055), the ground colour, the form
// colours, their emission, and the lower sky. Run on arm64 to exercise it:
//
//	GOARCH=arm64 go test -exec qemu-aarch64-static -run SceneSpecUnfused ./pkg/raydir
func TestSceneSpecUnfusedMatchesPython(t *testing.T) {
	goldens := []struct {
		v    SceneVector
		want string
	}{
		{SceneVector{0.5, 0.5, 0.5, 0.5, 0.074, 0.5}, `{"objects":[{"kind":"plane","color":[0.4,0.38,0.31]},{"kind":"sphere","x":2.111,"y":0.459,"z":6.0,"r":0.459,"color":[0.56,0.425,0.475],"rough":0.45},{"kind":"sphere","x":1.055,"y":0.459,"z":7.649,"r":0.459,"color":[0.76,0.625,0.675],"rough":0.45},{"kind":"sphere","x":-1.055,"y":0.459,"z":7.649,"r":0.459,"color":[0.56,0.425,0.475],"rough":0.45},{"kind":"sphere","x":-2.111,"y":0.459,"z":6.0,"r":0.459,"color":[0.76,0.625,0.675],"rough":0.45},{"kind":"sphere","x":-1.055,"y":0.459,"z":4.351,"r":0.459,"color":[0.56,0.425,0.475],"rough":0.45},{"kind":"sphere","x":1.055,"y":0.459,"z":4.351,"r":0.459,"color":[0.76,0.625,0.675],"rough":0.45},{"x":0,"y":7.0,"z":-1,"r":0.7,"emit":[10.5,10.5,9.975]}],"light":[6,8.0,-4],"skyTop":[0.42,0.407,0.468],"skyBottom":[0.572,0.558,0.619],"name":"Q6 vector (hex 101111)"}`},
		{SceneVector{0.5, 0.075, 0.5, 0.5, 0.5, 0.5}, `{"objects":[{"kind":"plane","color":[0.315,0.346,0.302]},{"kind":"sphere","x":2.75,"y":0.8,"z":6.0,"r":0.8,"color":[0.339,0.489,0.751],"rough":0.45},{"kind":"sphere","x":1.375,"y":0.8,"z":8.165,"r":0.8,"color":[0.539,0.689,0.951],"rough":0.45},{"kind":"sphere","x":-1.375,"y":0.8,"z":8.165,"r":0.8,"color":[0.339,0.489,0.751],"rough":0.45},{"kind":"sphere","x":-2.75,"y":0.8,"z":6.0,"r":0.8,"color":[0.539,0.689,0.951],"rough":0.45},{"kind":"sphere","x":-1.375,"y":0.8,"z":3.835,"r":0.8,"color":[0.339,0.489,0.751],"rough":0.45},{"kind":"sphere","x":1.375,"y":0.8,"z":3.835,"r":0.8,"color":[0.539,0.689,0.951],"rough":0.45},{"x":0,"y":7.0,"z":-1,"r":0.7,"emit":[10.5,10.5,9.975]}],"light":[6,8.0,-4],"skyTop":[0.427,0.523,0.698],"skyBottom":[0.578,0.674,0.849],"name":"Q6 vector (hex 111101)"}`},
		{SceneVector{0.5, 0.35, 0.5, 0.5, 0.5, 0.5}, `{"objects":[{"kind":"plane","color":[0.37,0.368,0.307]},{"kind":"sphere","x":2.75,"y":0.8,"z":6.0,"r":0.8,"color":[0.482,0.448,0.573],"rough":0.45},{"kind":"sphere","x":1.375,"y":0.8,"z":8.165,"r":0.8,"color":[0.682,0.647,0.773],"rough":0.45},{"kind":"sphere","x":-1.375,"y":0.8,"z":8.165,"r":0.8,"color":[0.482,0.448,0.573],"rough":0.45},{"kind":"sphere","x":-2.75,"y":0.8,"z":6.0,"r":0.8,"color":[0.682,0.647,0.773],"rough":0.45},{"kind":"sphere","x":-1.375,"y":0.8,"z":3.835,"r":0.8,"color":[0.482,0.448,0.573],"rough":0.45},{"kind":"sphere","x":1.375,"y":0.8,"z":3.835,"r":0.8,"color":[0.682,0.647,0.773],"rough":0.45},{"x":0,"y":7.0,"z":-1,"r":0.7,"emit":[10.5,10.5,9.975]}],"light":[6,8.0,-4],"skyTop":[0.423,0.448,0.549],"skyBottom":[0.574,0.599,0.7],"name":"Q6 vector (hex 111101)"}`},
		{SceneVector{0.57, 0.57, 0.57, 0.57, 0.57, 0.57}, `{"objects":[{"kind":"plane","color":[0.414,0.386,0.311]},{"kind":"sphere","x":2.855,"y":0.856,"z":6.0,"r":0.856,"color":[0.596,0.414,0.43],"rough":0.415,"emit":[0.501,0.348,0.361]},{"kind":"sphere","x":1.78,"y":0.856,"z":8.031,"r":0.856,"color":[0.796,0.614,0.63],"rough":0.415},{"kind":"sphere","x":-0.635,"y":0.856,"z":8.533,"r":0.856,"color":[0.596,0.414,0.43],"rough":0.415,"emit":[0.501,0.348,0.361]},{"kind":"sphere","x":-2.572,"y":0.856,"z":7.127,"r":0.856,"color":[0.796,0.614,0.63],"rough":0.415},{"kind":"sphere","x":-2.572,"y":0.856,"z":4.873,"r":0.856,"color":[0.596,0.414,0.43],"rough":0.415,"emit":[0.501,0.348,0.361]},{"kind":"sphere","x":-0.635,"y":0.856,"z":3.467,"r":0.856,"color":[0.796,0.614,0.63],"rough":0.415},{"kind":"sphere","x":1.78,"y":0.856,"z":3.969,"r":0.856,"color":[0.596,0.414,0.43],"rough":0.415,"emit":[0.501,0.348,0.361]},{"x":0,"y":7.28,"z":-1,"r":0.7,"emit":[11.13,11.13,10.574]}],"light":[6,8.56,-4],"skyTop":[0.448,0.409,0.446],"skyBottom":[0.592,0.554,0.59],"name":"Q6 vector (hex 111111)"}`},
		// fog*0.65 feeds lerp3's 1-t: fused into one multiply-subtract it lands an ulp
		// off and flips skyTop's rounding on arm64.
		{SceneVector{21.0 / 255, 85.0 / 255, 0.5, 0, 0.5, 0.5}, `{"objects":[{"kind":"plane","color":[0.367,0.367,0.307]},{"kind":"sphere","x":2.75,"y":0.8,"z":6.0,"r":0.8,"color":[0.473,0.45,0.583],"rough":0.45},{"kind":"sphere","x":1.375,"y":0.8,"z":8.165,"r":0.8,"color":[0.673,0.65,0.783],"rough":0.45},{"kind":"sphere","x":-1.375,"y":0.8,"z":8.165,"r":0.8,"color":[0.473,0.45,0.583],"rough":0.45},{"kind":"sphere","x":-2.75,"y":0.8,"z":6.0,"r":0.8,"color":[0.673,0.65,0.783],"rough":0.45},{"kind":"sphere","x":-1.375,"y":0.8,"z":3.835,"r":0.8,"color":[0.473,0.45,0.583],"rough":0.45},{"kind":"sphere","x":1.375,"y":0.8,"z":3.835,"r":0.8,"color":[0.673,0.65,0.783],"rough":0.45},{"x":0,"y":5,"z":-1,"r":0.7,"emit":[6,6,5.7]}],"light":[6,4,-4],"skyTop":[0.233,0.328,0.47],"skyBottom":[0.425,0.52,0.662],"name":"Q6 vector (hex 110100)"}`},
		{VectorFromPrompt("prompt 487"), `{"objects":[{"kind":"plane","color":[0.324,0.349,0.302]},{"kind":"sphere","x":2.418,"y":0.623,"z":6.0,"r":0.623,"color":[0.361,0.482,0.724],"rough":0.402,"emit":[0.416,0.556,0.835]},{"kind":"sphere","x":-1.209,"y":0.623,"z":7.896,"r":0.623,"color":[0.561,0.682,0.924],"rough":0.402},{"kind":"sphere","x":-1.209,"y":0.623,"z":4.104,"r":0.623,"color":[0.361,0.482,0.724],"rough":0.402,"emit":[0.416,0.556,0.835]},{"x":0,"y":6.6,"z":-1,"r":0.7,"emit":[9.6,9.6,9.12]}],"light":[6,7.2,-4],"skyTop":[0.405,0.49,0.649],"skyBottom":[0.555,0.641,0.8],"name":"Q6 vector (hex 100001)"}`},
	}
	for _, g := range goldens {
		var want brain.SceneSpec
		if err := json.Unmarshal([]byte(g.want), &want); err != nil {
			t.Fatalf("golden for %v: %v", g.v, err)
		}
		if got := g.v.SceneSpec(); !reflect.DeepEqual(got, want) {
			gj, _ := json.Marshal(got)
			wj, _ := json.Marshal(want)
			t.Errorf("SceneSpec(%v) differs from the Python director:\n got  %s\n want %s", g.v, gj, wj)
		}
	}
}
