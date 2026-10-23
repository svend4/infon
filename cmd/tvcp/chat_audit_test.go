package main

import (
	"testing"

	"github.com/svend4/infon/internal/network"
	"github.com/svend4/infon/pkg/terminal"
)

// A remote peer controls the chat sender and text; printing them must not let
// it emit terminal control sequences (C0, DEL, C1).
func TestChatLineStripsControlCharacters(t *testing.T) {
	m := &network.TextMessage{
		Sender:  "ev\x1b]0;owned\x07e",
		Message: "hi\x1b[2J\u009b31m\x7f\r\nthere ✓ привет 🙂",
	}
	sender, msg := chatLine(m)
	if want := "ev]0;ownede"; sender != want {
		t.Errorf("sender = %q, want %q", sender, want)
	}
	if want := "hi[2J31mthere ✓ привет 🙂"; msg != want {
		t.Errorf("message = %q, want %q", msg, want)
	}
}

func TestChatLineDefaultsEmptySender(t *testing.T) {
	for _, s := range []string{"", "\x1b\x07\u0085"} {
		if sender, _ := chatLine(&network.TextMessage{Sender: s, Message: "x"}); sender != "Remote" {
			t.Errorf("chatLine sender for %q = %q, want Remote", s, sender)
		}
	}
}

// The share handshake payload is printed as the peer name.
func TestHandshakePayloadSanitized(t *testing.T) {
	got := terminal.SanitizeText(string([]byte("peer\x1b[2J\x00\x9b-ok")))
	if want := "peer[2J�-ok"; got != want {
		t.Errorf("SanitizeText = %q, want %q", got, want)
	}
}
