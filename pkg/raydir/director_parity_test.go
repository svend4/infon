package raydir

import (
	"testing"
)

// The expected values below were produced by ai/adapters/yijing_brain.py
// (scene_from_vector / Python's round) and pin the Go side to the Python director:
// correctly rounded 3-decimal output with ties to even, a ties-to-even form count,
// and cos/sin pre-rounded to 12 decimals. A 41k-vector sweep (hash-derived prompts,
// 1/1000 grids and random vectors) matched field for field with these rules.

func TestRound3MatchesPythonRound(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{2.0105, 2.01}, // binary 2.0105 is just below the tie: Python 2.01; the old math.Round(x*1000) gave 2.011
		{0.4105, 0.41},
		{1.0005, 1.0},
		{0.0005, 0.001}, // just above the tie
		{2.675, 2.675},
		{0.1235, 0.123},
		{0.1245, 0.124},
		{5.785, 5.785},
		{0.9995, 1.0},
		{12.3455, 12.345},
		{0.6055, 0.606},
		{-1.0495, -1.05},
	}
	for _, c := range cases {
		if got := round3(c.in); got != c.want {
			t.Errorf("round3(%v) = %v, want %v (Python round(x, 3))", c.in, got, c.want)
		}
	}
}

func TestSceneSpecMatchesPythonDirector(t *testing.T) {
	type golden struct {
		v                      SceneVector
		nobj                   int
		xs, zs                 []float64
		rough                  float64
		light, top, bot, plane [3]float64
		name                   string
	}
	goldens := []golden{
		{ // cos(pi/3): Go gives exactly 0.5, libm 0.5000000000000001 -> x=1.3375 tie
			v: SceneVector{0.3, 0.6, 0.45, 0.45, 0.45, 0.45}, nobj: 8,
			xs:    []float64{0, 2.675, 1.337, -1.337, -2.675, -1.337, 1.337, 0},
			zs:    []float64{0, 6.0, 8.104, 8.104, 6.0, 3.896, 3.896, -1},
			rough: 0.475, light: [3]float64{6, 7.6, -4},
			top: [3]float64{0.399, 0.361, 0.404}, bot: [3]float64{0.57, 0.532, 0.575},
			plane: [3]float64{0.42, 0.388, 0.312}, name: "Q6 vector (hex 000010)",
		},
		{ // density 0.0625*8 = 0.5 exactly: ties to even -> 2 forms (math.Round gave 3)
			v: SceneVector{0.4999, 0.5001, 0.0625, 0.1875, 0.3125, 0.4375}, nobj: 4,
			xs: []float64{0, 2.469, -2.469, 0}, zs: []float64{0, 6.0, 6.0, -1},
			rough: 0.481, light: [3]float64{6, 5.5, -4},
			top: [3]float64{0.299, 0.324, 0.391}, bot: [3]float64{0.45, 0.475, 0.542},
			plane: [3]float64{0.4, 0.38, 0.31}, name: "Q6 vector (hex 000010)",
		},
		{ // density 0.8125*8 = 6.5 exactly: ties to even -> 6 forms
			v: SceneVector{0.5625, 0.6875, 0.8125, 0.9375, 0.0625, 0.5}, nobj: 10,
			xs:    []float64{0, 2.094, 1.481, 0.0, -1.481, -2.094, -1.481, -0.0, 1.481, 0},
			zs:    []float64{0, 6.0, 7.335, 7.888, 7.335, 6.0, 4.665, 4.112, 4.665, -1},
			rough: 0.45, light: [3]float64{6, 11.5, -4},
			top: [3]float64{0.616, 0.488, 0.475}, bot: [3]float64{0.761, 0.633, 0.62},
			plane: [3]float64{0.438, 0.395, 0.314}, name: "Q6 vector (hex 101111)",
		},
		{ // skyTop[0] = 0.6055 sits on a tie
			v: SceneVector{1, 1, 1, 1, 1, 1}, nobj: 12,
			xs:    []float64{0, 3.5, 2.832, 1.082, -1.082, -2.832, -3.5, -2.832, -1.082, 1.082, 2.832, 0},
			zs:    []float64{0, 6.0, 7.881, 9.043, 9.043, 7.881, 6.0, 4.119, 2.957, 2.957, 4.119, -1},
			rough: 0.2, light: [3]float64{6, 12, -4},
			top: [3]float64{0.605, 0.466, 0.399}, bot: [3]float64{0.701, 0.568, 0.502},
			plane: [3]float64{0.5, 0.42, 0.32}, name: "Q6 vector (hex 111111)",
		},
	}
	for _, g := range goldens {
		spec := g.v.SceneSpec()
		if len(spec.Objects) != g.nobj {
			t.Errorf("%v: %d objects, want %d", g.v, len(spec.Objects), g.nobj)
			continue
		}
		for i, o := range spec.Objects {
			if o.X != g.xs[i] || o.Z != g.zs[i] {
				t.Errorf("%v: object %d at (%v, %v), want (%v, %v)", g.v, i, o.X, o.Z, g.xs[i], g.zs[i])
			}
		}
		if spec.Objects[1].Rough != g.rough {
			t.Errorf("%v: rough %v, want %v", g.v, spec.Objects[1].Rough, g.rough)
		}
		if spec.Light != g.light || spec.SkyTop != g.top || spec.SkyBot != g.bot || spec.Objects[0].Color != g.plane {
			t.Errorf("%v: light/sky/plane %v %v %v %v, want %v %v %v %v", g.v, spec.Light, spec.SkyTop, spec.SkyBot, spec.Objects[0].Color, g.light, g.top, g.bot, g.plane)
		}
		if spec.Name != g.name {
			t.Errorf("%v: name %q, want %q", g.v, spec.Name, g.name)
		}
	}
}
