package stun

import (
	"net"
	"testing"
)

func TestFormatParseCandidateRoundTripAudit(t *testing.T) {
	cases := []*ICECandidate{
		{Type: CandidateTypeHost, Address: &net.UDPAddr{IP: net.IPv4(192, 168, 1, 10).To4(), Port: 5000}, Priority: 2130706431, Foundation: "host-192", Component: 1},
		{Type: CandidateTypeSrflx, Address: &net.UDPAddr{IP: net.IPv4(203, 0, 113, 7).To4(), Port: 61000}, Priority: 1694498815, Foundation: "srflx-203", Component: 1},
		{Type: CandidateTypeRelay, Address: &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 3478}, Priority: 16777215, Foundation: "relay-32", Component: 2},
	}

	for _, want := range cases {
		sdp := FormatCandidate(want)
		got, err := ParseCandidate(sdp)
		if err != nil {
			t.Fatalf("ParseCandidate(%q) error: %v", sdp, err)
		}
		if got.Type != want.Type || got.Priority != want.Priority ||
			got.Foundation != want.Foundation || got.Component != want.Component ||
			!got.Address.IP.Equal(want.Address.IP) || got.Address.Port != want.Address.Port {
			t.Fatalf("round trip mismatch for %q: got %+v (addr %v), want %+v (addr %v)",
				sdp, got, got.Address, want, want.Address)
		}
	}
}

func TestParseCandidate_MalformedAudit(t *testing.T) {
	for _, sdp := range []string{
		"",
		"candidate:",
		"candidate:f 1 udp 100 10.0.0.1 5000 typ bogus",
		"candidate:f x udp 100 10.0.0.1 5000 typ host",
	} {
		if c, err := ParseCandidate(sdp); err == nil {
			t.Errorf("ParseCandidate(%q) = %+v, want error", sdp, c)
		}
	}
}
