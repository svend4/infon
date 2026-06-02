// Package discovery implements the DNS-TXT peer discovery sketched in
// docs/dns-peer-discovery.md: instead of sharing a cryptographic Yggdrasil
// address out of band, a user publishes a TXT record at _tvcp.<user>.<domain>
// and is reached by a human handle like alice@example.com. This package owns the
// record format, its parser, and a resolver seam (real DNS, or a map for tests).
package discovery

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// Record is a published peer endpoint.
type Record struct {
	Ygg     string // Yggdrasil IPv6 address + port, e.g. "200:abc:def::1:5000"
	Version string // record format version (default "1")
	PubKey  string // optional Yggdrasil public key for verification
}

// Name is the TXT owner name for a handle's parts.
func Name(user, domain string) string { return "_tvcp." + user + "." + domain }

// ParseHandle splits "user@domain".
func ParseHandle(handle string) (user, domain string, err error) {
	handle = strings.TrimSpace(handle)
	i := strings.IndexByte(handle, '@')
	if i <= 0 || i >= len(handle)-1 || strings.Contains(handle[i+1:], "@") {
		return "", "", fmt.Errorf("discovery: bad handle %q (want user@domain)", handle)
	}
	return handle[:i], handle[i+1:], nil
}

// Format renders a record as the TXT string body.
func Format(r Record) string {
	v := r.Version
	if v == "" {
		v = "1"
	}
	s := "ygg=" + r.Ygg + " v=" + v
	if r.PubKey != "" {
		s += " pubkey=" + r.PubKey
	}
	return s
}

// Parse reads a TXT body ("ygg=... v=1 pubkey=..."). Unknown keys are ignored for
// forward compatibility; a record with no ygg= is rejected.
func Parse(txt string) (Record, error) {
	var r Record
	for _, tok := range strings.Fields(txt) {
		k, v, ok := strings.Cut(tok, "=")
		if !ok {
			continue
		}
		switch strings.ToLower(k) {
		case "ygg":
			r.Ygg = v
		case "v":
			r.Version = v
		case "pubkey":
			r.PubKey = v
		}
	}
	if r.Ygg == "" {
		return Record{}, errors.New("discovery: TXT record has no ygg= field")
	}
	if r.Version == "" {
		r.Version = "1"
	}
	return r, nil
}

// Resolver looks up TXT records for a name. net.LookupTXT satisfies the intent;
// tests use a MapResolver.
type Resolver interface {
	LookupTXT(name string) ([]string, error)
}

// NetResolver resolves against the real DNS.
type NetResolver struct{}

func (NetResolver) LookupTXT(name string) ([]string, error) { return net.LookupTXT(name) }

// MapResolver is an in-memory resolver for tests/offline use.
type MapResolver map[string][]string

func (m MapResolver) LookupTXT(name string) ([]string, error) {
	if v, ok := m[name]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("discovery: no TXT record for %s", name)
}

// Lookup resolves a handle (user@domain) to its peer Record via the resolver,
// returning the first TXT entry that parses as a valid _tvcp record.
func Lookup(r Resolver, handle string) (Record, error) {
	user, domain, err := ParseHandle(handle)
	if err != nil {
		return Record{}, err
	}
	txts, err := r.LookupTXT(Name(user, domain))
	if err != nil {
		return Record{}, err
	}
	for _, txt := range txts {
		if rec, err := Parse(txt); err == nil {
			return rec, nil
		}
	}
	return Record{}, fmt.Errorf("discovery: no valid _tvcp record for %s", handle)
}
