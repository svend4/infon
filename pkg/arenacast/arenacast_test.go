package arenacast

import (
	"testing"

	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/pseudo"
	"github.com/svend4/infon/pkg/semvideo"
)

func mkArena() *arena.Arena {
	return &arena.Arena{
		W: 4, H: 3, Terrain: make([]uint8, 12),
		Units: []arena.Unit{
			{Kind: 0, X: 0, Y: 0, HP: 5, Faction: 0, Alive: true},
			{Kind: 0, X: 3, Y: 2, HP: 5, Faction: 1, Alive: true},
		},
	}
}

func TestSpecShapeAndTokens(t *testing.T) {
	s := Spec(mkArena())
	if s.Format != pseudo.FormatMarks || s.Marks == nil {
		t.Fatal("expected a marks spec")
	}
	if len(s.Marks.Rows) != 3 || len(s.Marks.Rows[0]) != 4 {
		t.Fatalf("board shape %dx%d, want 3x4", len(s.Marks.Rows), len(s.Marks.Rows[0]))
	}
	if s.Marks.Rows[0][0] != FactionGlyph(0) || s.Marks.Colors[0][0] != FactionColor(0) {
		t.Fatalf("blue unit cell = %q/%q", s.Marks.Rows[0][0], s.Marks.Colors[0][0])
	}
	if s.Marks.Rows[2][3] != FactionGlyph(1) {
		t.Fatalf("red unit cell = %q", s.Marks.Rows[2][3])
	}
	if s.Marks.Rows[0][1] != "" {
		t.Fatalf("empty cell should be blank, got %q", s.Marks.Rows[0][1])
	}
}

// The match must round-trip through the semantic-video codec exactly: arena ->
// marks specs -> semvideo keyframe+delta -> decode -> identical board.
func TestSemvideoRoundTrip(t *testing.T) {
	a := mkArena()
	s0 := Spec(a)
	a.Units[0].X = 1 // the blue unit steps right
	s1 := Spec(a)

	pkts, err := semvideo.Encode([]pseudo.Spec{s0, s1}, 8)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := semvideo.Decode(pkts, 8)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("decoded %d frames, want 2", len(got))
	}
	for y := range s1.Marks.Rows {
		for x := range s1.Marks.Rows[y] {
			if got[1].Marks.Rows[y][x] != s1.Marks.Rows[y][x] {
				t.Fatalf("cell %d,%d: got %q want %q", y, x,
					got[1].Marks.Rows[y][x], s1.Marks.Rows[y][x])
			}
		}
	}
	if iou := semvideo.MarksIoU(s1, got[1]); iou != 1.0 {
		t.Fatalf("reconstructed IoU = %.3f, want 1.0", iou)
	}
}
