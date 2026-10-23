package stun

import (
	"bytes"
	"net"
	"testing"
	"time"
)

// ReceiveData accepts Data indications only from the TURN server. Before, any
// host that could reach the client's port could send one naming any peer, and
// the client delivered its payload as that peer's data.
func TestTURNReceiveDataOnlyFromServer(t *testing.T) {
	listen := func() *net.UDPConn {
		c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		t.Cleanup(func() { _ = c.Close() })
		return c
	}
	client, server, stranger := listen(), listen(), listen()

	tc := NewTURNClient(server.LocalAddr().String(), "u", "p")
	tc.conn = client
	tc.timeout = time.Second
	tc.server = server.LocalAddr().(*net.UDPAddr)

	peer := &net.UDPAddr{IP: net.IPv4(203, 0, 113, 5), Port: 4000}
	indication := func(payload string) []byte {
		msg := tc.createSendIndication(peer, []byte(payload))
		msg.Type = DataIndication
		data, err := tc.encodeMessage(msg)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	if _, err := stranger.WriteToUDP(indication("forged"), client.LocalAddr().(*net.UDPAddr)); err != nil {
		t.Fatal(err)
	}
	if data, from, err := tc.ReceiveData(); err == nil {
		t.Errorf("Data indication from a stranger delivered: %q from %v", data, from)
	}

	if _, err := server.WriteToUDP(indication("relayed"), client.LocalAddr().(*net.UDPAddr)); err != nil {
		t.Fatal(err)
	}
	data, from, err := tc.ReceiveData()
	if err != nil || !bytes.Equal(data, []byte("relayed")) || !from.IP.Equal(peer.IP) || from.Port != peer.Port {
		t.Errorf("relayed data = %q from %v, %v; want %q from %v", data, from, err, "relayed", peer)
	}
}
