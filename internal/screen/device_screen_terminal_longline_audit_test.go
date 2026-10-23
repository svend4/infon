package screen

import (
	"strings"
	"testing"
)

// A line longer than bufio.Scanner's 64 KiB token limit must not stop the
// reader: it would stop draining the pipe and the shared command would block on
// a full pipe, freezing the session. The long line is shown truncated and the
// lines after it still arrive.
func TestReadOutputSurvivesOverlongLine(t *testing.T) {
	ss := NewScreenShare("unused", 8, 3)
	for i := range ss.buffer {
		ss.buffer[i] = []rune(strings.Repeat(" ", 8))
	}

	in := strings.NewReader(strings.Repeat("x", 200*1024) + "\nafter\nlast\n")
	ss.readOutput(in)

	if in.Len() != 0 {
		t.Errorf("reader stopped with %d bytes unread", in.Len())
	}
	got := ss.GetBuffer()
	want := []string{"xxxxxxxx", "after   ", "last    "}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q, want %q (buffer %q)", i, got[i], want[i], got)
		}
	}
}
