// Package sign adds message authentication and explicit versioning to the
// tvcp-ai/1 transport. A brain message is stamped with its protocol version and
// an HMAC-SHA256 over (version || payload) under a shared key, so a receiver can
// reject tampered or wrong-version messages BEFORE acting on them. It is the
// minimal "signing + versioning in the live transport" the audit asked for, and
// it is transport-agnostic — the same stamp travels over HTTP, a pipe or a packet.
package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Version is the current protocol version this package stamps by default.
const Version = "tvcp-ai/1"

// Stamp is the authentication envelope attached to a message.
type Stamp struct {
	Version string `json:"v"`
	Sig     string `json:"sig"` // hex HMAC-SHA256
}

func mac(version string, key, payload []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(version))
	h.Write([]byte{0}) // domain separator between version and payload
	h.Write(payload)
	return h.Sum(nil)
}

// Sign stamps payload for the given version under key.
func Sign(version string, key, payload []byte) Stamp {
	return Stamp{Version: version, Sig: hex.EncodeToString(mac(version, key, payload))}
}

// Verify reports whether the stamp authenticates payload under key AND matches
// wantVersion. It is constant-time in the signature comparison.
func Verify(key, payload []byte, s Stamp, wantVersion string) bool {
	if s.Version != wantVersion {
		return false
	}
	got, err := hex.DecodeString(s.Sig)
	if err != nil {
		return false
	}
	return hmac.Equal(got, mac(s.Version, key, payload))
}
