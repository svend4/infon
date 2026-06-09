package raydir

import (
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/raytrace"
)

func TestAckRoundTrip(t *testing.T) {
	for _, in := range [][]int{{}, {0}, {0, 2, 5, 9}, {3, 1, 4, 1, 5}} {
		got, err := DecodeAck(EncodeAck(in))
		if err != nil {
			t.Fatalf("decode %v: %v", in, err)
		}
		if !reflect.DeepEqual(got, in) {
			t.Errorf("round trip %v -> %v", in, got)
		}
	}
	if _, err := DecodeAck([]byte{0}); err == nil {
		t.Error("short ack should error")
	}
}

func TestWorldKnownSorted(t *testing.T) {
	w := NewWorld()
	spec := brain.SceneSpec{Objects: []brain.ObjSpec{{X: 0, Y: 1, R: 1}}}
	for _, idx := range []int{2, 0, 5} {
		w.applyRegion(idx, raytrace.Vec3{}, spec)
	}
	if k := w.Known(); !reflect.DeepEqual(k, []int{0, 2, 5}) {
		t.Errorf("Known() = %v, want [0 2 5]", k)
	}
}

func TestMissingRegions(t *testing.T) {
	regions := []Region{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}}
	miss := MissingRegions([]int{0, 2}, regions)
	if len(miss) != 2 || miss[0].Index != 1 || miss[1].Index != 3 {
		t.Errorf("missing = %+v, want indices 1 and 3", miss)
	}
	if n := len(MissingRegions([]int{0, 1, 2, 3}, regions)); n != 0 {
		t.Errorf("nothing should be missing when all are had, got %d", n)
	}
	if n := len(MissingRegions(nil, regions)); n != 4 {
		t.Errorf("everything is missing when have is empty, got %d", n)
	}
}
