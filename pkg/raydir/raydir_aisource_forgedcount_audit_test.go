package raydir

import (
	"encoding/binary"
	"runtime"
	"testing"

	"github.com/svend4/infon/pkg/raytrace"
)

// allocated reports the bytes allocated while f runs.
func allocated(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// header builds an 8-byte file header: magic + a forged 32-bit count, no entries.
func header(magic [4]byte, n uint32) []byte {
	b := append([]byte(nil), magic[:]...)
	return binary.BigEndian.AppendUint32(b, n)
}

// A world file's region count must not size an allocation the file can't fill: an
// 8-byte file claiming 2^20 regions used to allocate 144 MiB up front, and one
// claiming 2^32-1 made the runtime abort with "out of memory".
func TestDecodeWorldForgedCount(t *testing.T) {
	var err error
	if n := allocated(func() { _, err = DecodeWorld(header(worldMagic, 1<<20)) }); n > 1<<20 {
		t.Fatalf("DecodeWorld of an 8-byte file allocated %d bytes", n)
	}
	if err == nil {
		t.Error("forged region count accepted")
	}
	for _, n := range []uint32{1, 1 << 31, 0xFFFFFFFF} {
		if _, err := DecodeWorld(header(worldMagic, n)); err == nil {
			t.Errorf("count %d with no regions accepted", n)
		}
	}
	regions := []Region{{Index: 0, At: raytrace.Vec3{Z: 6}}, {Index: 1, At: raytrace.Vec3{Z: 18}}}
	if got, err := DecodeWorld(EncodeWorld(regions)); err != nil || len(got) != 2 || got[1].Index != 1 {
		t.Errorf("round trip = %v, %v", got, err)
	}
	if got, err := DecodeWorld(EncodeWorld(nil)); err != nil || len(got) != 0 {
		t.Errorf("empty world = %v, %v", got, err)
	}
}

// Same for a recording's event count (an event is at least 9 bytes).
func TestDecodeRecordingForgedCount(t *testing.T) {
	var err error
	if n := allocated(func() { _, err = DecodeRecording(header(recMagic, 1<<22)) }); n > 1<<20 {
		t.Fatalf("DecodeRecording of an 8-byte file allocated %d bytes", n)
	}
	if err == nil {
		t.Error("forged event count accepted")
	}
	for _, n := range []uint32{1, 1 << 31, 0xFFFFFFFF} {
		if _, err := DecodeRecording(header(recMagic, n)); err == nil {
			t.Errorf("count %d with no events accepted", n)
		}
	}
	ev := []RecEvent{{TMs: 1, Type: RecChat}, {TMs: 2, Type: RecPoses, Data: []byte{0, 0}}}
	if got, err := DecodeRecording(EncodeRecording(ev)); err != nil || len(got) != 2 || got[1].TMs != 2 || len(got[0].Data) != 0 {
		t.Errorf("round trip = %v, %v", got, err)
	}
}
