package stun

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestTURNParseXorAddress_IPv6Audit(t *testing.T) {
	tc := NewTURNClient("turn.example.com:3478", "user", "pass")
	txID := [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

	want := net.ParseIP("2001:db8::1234")
	port := 40000

	key := make([]byte, 16)
	binary.BigEndian.PutUint32(key[0:4], MagicCookie)
	copy(key[4:], txID[:])

	data := make([]byte, 20)
	data[1] = 0x02
	binary.BigEndian.PutUint16(data[2:4], uint16(port)^0x2112)
	for i := 0; i < 16; i++ {
		data[4+i] = want[i] ^ key[i]
	}

	got, err := tc.parseXorAddress(data, txID)
	if err != nil {
		t.Fatalf("parseXorAddress(IPv6) error: %v", err)
	}
	if !got.IP.Equal(want) || got.Port != port {
		t.Fatalf("parseXorAddress(IPv6) = %v, want [%v]:%d", got, want, port)
	}

	// IPv6 family with a truncated value must be rejected.
	if a, err := tc.parseXorAddress(data[:8], txID); err == nil {
		t.Fatalf("truncated IPv6 address accepted: %v", a)
	}
}

func TestTURNParseXorAddress_UnknownFamilyAudit(t *testing.T) {
	tc := NewTURNClient("turn.example.com:3478", "user", "pass")
	var txID [12]byte

	data := []byte{0x00, 0x7f, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}
	a, err := tc.parseXorAddress(data, txID)
	if err == nil {
		t.Fatalf("unknown address family accepted: %+v", a)
	}
}

func TestGenerateFoundation_EmptyIPAudit(t *testing.T) {
	ia := NewICEAgent(nil, nil)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("generateFoundation(nil IP) panicked: %v", r)
		}
	}()
	if f := ia.generateFoundation(CandidateTypeRelay, nil); f == "" {
		t.Fatal("generateFoundation(nil IP) returned empty foundation")
	}
}
