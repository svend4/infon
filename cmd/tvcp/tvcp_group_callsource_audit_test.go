package main

import (
	"net"
	"strings"
	"testing"

	"github.com/svend4/infon/internal/network"
)

// The call's receive loop must accept packets only from the dialled peer: any
// other host that can reach the port could otherwise inject video, audio, chat
// and NACKs into the call. The check has to match the address the transport
// actually reports, which on the dual-stack ":port" socket tvcp call listens on
// is an IPv4-mapped IPv6 address for an IPv4 peer.
func TestCallAcceptsOnlyTheDialledPeer(t *testing.T) {
	transport, err := network.NewTransport(":0")
	if err != nil {
		t.Fatalf("NewTransport: %v", err)
	}
	defer func() { _ = transport.Close() }()
	port := transport.LocalAddr().Port
	local := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port}

	listen := func() *net.UDPConn {
		c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		return c
	}
	peer, stranger := listen(), listen()
	defer peer.Close()
	defer stranger.Close()

	// The peer as tvcp call resolves it from the command line ("host:port").
	dialled, err := net.ResolveUDPAddr("udp", peer.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}

	send := func(c *net.UDPConn) *net.UDPAddr {
		data, err := (&network.Packet{Type: network.PacketTypeTextChat, Payload: []byte("x")}).Encode()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.WriteToUDP(data, local); err != nil {
			t.Fatal(err)
		}
		_, src, err := transport.ReceivePacket()
		if err != nil {
			t.Fatalf("ReceivePacket: %v", err)
		}
		return src
	}

	if src := send(peer); !sameUDPAddr(src, dialled) {
		t.Errorf("packet from the dialled peer %v (reported as %v) was rejected", dialled, src)
	}
	if src := send(stranger); sameUDPAddr(src, dialled) {
		t.Errorf("packet from %v was accepted as coming from the dialled peer %v", src, dialled)
	}

	other := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 2), Port: dialled.Port}
	for _, tc := range []struct {
		a, b *net.UDPAddr
		want bool
	}{
		{dialled, dialled, true},
		{other, dialled, false},
		{nil, dialled, false},
		{dialled, nil, false},
		{&net.UDPAddr{IP: net.ParseIP("200:abc::1"), Port: 5000}, &net.UDPAddr{IP: net.ParseIP("200:abc::1"), Port: 5000}, true},
		{&net.UDPAddr{IP: net.ParseIP("200:abc::1"), Port: 5000}, &net.UDPAddr{IP: net.ParseIP("200:abc::2"), Port: 5000}, false},
	} {
		if got := sameUDPAddr(tc.a, tc.b); got != tc.want {
			t.Errorf("sameUDPAddr(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

// Packets from anyone but the peer are dropped, but not silently: each unexpected
// sender is named once (a NAT'd or multi-homed peer shows up here), and the
// reports stop after a few so a flood cannot fill the screen.
func TestCallReportsIgnoredSendersOnce(t *testing.T) {
	peer := &net.UDPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 5000}
	var f foreignSources
	other := &net.UDPAddr{IP: net.IPv4(192, 0, 2, 9), Port: 5000}
	if msg := f.report(other, peer); !strings.Contains(msg, "192.0.2.9:5000") || !strings.Contains(msg, "192.0.2.1:5000") {
		t.Errorf("first packet from a stranger: %q, want both addresses named", msg)
	}
	if msg := f.report(other, peer); msg != "" {
		t.Errorf("second packet from the same stranger reported again: %q", msg)
	}

	reports := 1
	for port := 1; port < 100; port++ {
		if f.report(&net.UDPAddr{IP: net.IPv4(198, 51, 100, 1), Port: port}, peer) != "" {
			reports++
		}
	}
	if reports != maxForeignReports+1 {
		t.Errorf("%d reports for 100 strangers, want %d (one each up to the cap, then one notice)", reports, maxForeignReports+1)
	}
	if len(f.seen) > maxForeignReports+1 {
		t.Errorf("tracked %d senders, want at most %d", len(f.seen), maxForeignReports+1)
	}
}
