package group

import (
	"net"
	"testing"
	"time"

	"github.com/svend4/infon/internal/network"
)

// A peer that has gone quiet must get a keep-alive (so a late or recovered peer
// can reconnect), an active peer must not, and neither is removed.
func TestPingInactiveKeepsQuietPeersReachable(t *testing.T) {
	gc, err := NewGroupCall("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewGroupCall: %v", err)
	}
	defer func() { _ = gc.Stop() }()
	_ = gc.Start()

	listen := func() *net.UDPConn {
		c, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		return c
	}
	quiet, active := listen(), listen()
	defer quiet.Close()
	defer active.Close()

	if err := gc.AddPeer("quiet", quiet.LocalAddr().(*net.UDPAddr)); err != nil {
		t.Fatal(err)
	}
	if err := gc.AddPeer("active", active.LocalAddr().(*net.UDPAddr)); err != nil {
		t.Fatal(err)
	}
	p, _ := gc.GetPeer("quiet")
	p.mutex.Lock()
	p.LastSeen = time.Now().Add(-time.Minute) // silent for longer than the activity window
	p.mutex.Unlock()

	if n := gc.PingInactive(); n != 1 {
		t.Fatalf("PingInactive pinged %d peers, want 1", n)
	}

	// nextKeepAlive reads until a keep-alive arrives (AddPeer sends a JOIN first)
	// or the deadline passes.
	nextKeepAlive := func(c *net.UDPConn, wait time.Duration) bool {
		buf := make([]byte, 1500)
		_ = c.SetReadDeadline(time.Now().Add(wait))
		for {
			n, _, err := c.ReadFromUDP(buf)
			if err != nil {
				return false
			}
			if pkt, err := network.Decode(buf[:n]); err == nil && pkt.Type == network.PacketTypeKeepAlive {
				return true
			}
		}
	}
	if !nextKeepAlive(quiet, 2*time.Second) {
		t.Fatal("quiet peer got no keep-alive")
	}
	if nextKeepAlive(active, 200*time.Millisecond) {
		t.Fatal("active peer was pinged; only quiet peers should be")
	}

	if gc.PeerCount() != 2 {
		t.Fatalf("PingInactive changed membership: %d peers, want 2", gc.PeerCount())
	}
}
