package registry

import (
	"encoding/json"
	"errors"

	"github.com/svend4/infon/pkg/sign"
)

// signed.go makes the brain directory FEDERATED and verifiable: a registry entry
// can be signed (pkg/sign) by whoever publishes it, and a SignedRegistry admits
// only entries whose signature checks out under a shared key. Combined with the
// capability lookup, this is the project's "DNS for brains" with trust.

// canon is the deterministic byte form of an entry that gets signed.
func canon(e Entry) []byte {
	b, _ := json.Marshal(e)
	return b
}

// SignedEntry is a directory entry plus its authentication stamp.
type SignedEntry struct {
	Entry Entry      `json:"entry"`
	Stamp sign.Stamp `json:"stamp"`
}

// SignEntry signs e under key.
func SignEntry(key []byte, e Entry) SignedEntry {
	return SignedEntry{Entry: e, Stamp: sign.Sign(sign.Version, key, canon(e))}
}

// VerifyEntry reports whether se's signature authenticates its entry under key.
func VerifyEntry(key []byte, se SignedEntry) bool {
	return sign.Verify(key, canon(se.Entry), se.Stamp, sign.Version)
}

// SignedRegistry is a registry that only admits signature-verified entries.
type SignedRegistry struct {
	reg *Registry
	key []byte
}

// NewSigned returns an empty signed registry trusting key.
func NewSigned(key []byte) *SignedRegistry {
	return &SignedRegistry{reg: New(), key: key}
}

// Admit verifies se and, if valid, registers its entry; otherwise it is rejected.
func (s *SignedRegistry) Admit(se SignedEntry) error {
	if !VerifyEntry(s.key, se) {
		return errors.New("registry: entry rejected (bad or missing signature)")
	}
	s.reg.Register(se.Entry)
	return nil
}

// Find delegates capability lookup to the underlying registry.
func (s *SignedRegistry) Find(kind, game string) []Entry { return s.reg.Find(kind, game) }

// List returns all admitted entries.
func (s *SignedRegistry) List() []Entry { return s.reg.List() }
