package raydir

import (
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// A pose or presence set's 2-byte count must not pre-size a map the payload can't
// fill: a 2-byte packet claiming 65535 entries used to allocate ~7 MB.
func TestDecodePoseSetForgedCount(t *testing.T) {
	forged := []byte{0xFF, 0xFF}
	var err error
	if n := allocated(func() { _, err = DecodePoseSet(forged) }); n > 64<<10 {
		t.Errorf("DecodePoseSet of a 2-byte payload allocated %d bytes", n)
	}
	if err == nil {
		t.Error("forged pose count accepted")
	}
	if n := allocated(func() { _, err = DecodePresenceSet(forged) }); n > 64<<10 {
		t.Errorf("DecodePresenceSet of a 2-byte payload allocated %d bytes", n)
	}
	if err == nil {
		t.Error("forged presence count accepted")
	}

	poses := PoseSet{7: {Pos: raytrace.Vec3{X: 1, Z: 2}, Yaw: 0.5}, 9: {Pitch: -0.1}}
	if got, err := DecodePoseSet(poses.Encode()); err != nil || !reflect.DeepEqual(got, poses) {
		t.Errorf("pose set round trip = %v, %v", got, err)
	}
	pres := PresenceSet{3: {ID: 3, Pose: Pose{Yaw: 1}}}
	if got, err := DecodePresenceSet(pres.Encode()); err != nil || len(got) != 1 || got[3].Pose.Yaw != 1 {
		t.Errorf("presence set round trip = %v, %v", got, err)
	}
	if got, err := DecodePoseSet(PoseSet{}.Encode()); err != nil || len(got) != 0 {
		t.Errorf("empty pose set = %v, %v", got, err)
	}
}
