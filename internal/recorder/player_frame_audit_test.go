package recorder

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

func auditFrameHeader(width, height uint16) []byte {
	d := make([]byte, 4)
	binary.BigEndian.PutUint16(d[0:2], width)
	binary.BigEndian.PutUint16(d[2:4], height)
	return d
}

func TestDeserializeFrameTruncatedAudit(t *testing.T) {
	// 4 bytes of data claiming a 2000x2000 frame (40 MB of blocks).
	if f := deserializeFrame(auditFrameHeader(2000, 2000)); f != nil {
		t.Fatalf("deserializeFrame returned a %dx%d frame from 4 bytes, want nil", f.Width, f.Height)
	}
}

func TestDeserializeFrameZeroWidthAudit(t *testing.T) {
	// 0x65535: no blocks, but NewFrame would still allocate 65535 rows.
	f := deserializeFrame(auditFrameHeader(0, 65535))
	if f != nil && len(f.Blocks) != 0 {
		t.Fatalf("deserializeFrame allocated %d rows for a zero-area frame", len(f.Blocks))
	}
}

func TestDeserializeFrameRoundTripAudit(t *testing.T) {
	want := terminal.NewFrame(4, 3)
	want.SetBlock(3, 2, 'Q', color.RGB{R: 9, G: 8, B: 7}, color.RGB{R: 6, G: 5, B: 4})
	got := deserializeFrame(serializeFrame(want))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}

func TestLoadTruncatedFrameAudit(t *testing.T) {
	data := auditFrameHeader(2000, 2000)
	path := writeAuditFile(t, auditHeader(1, 0), auditEntry(0, uint32(len(data))), data)
	if err := NewPlayer().Load(path); err == nil {
		t.Fatal("Load accepted a frame whose data is shorter than its dimensions require")
	}
}
