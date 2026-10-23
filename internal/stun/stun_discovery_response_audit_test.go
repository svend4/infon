package stun

import (
	"net"
	"sync"
	"testing"
	"time"
)

// fakeSTUNServer is a loopback UDP server that hands every decoded request to
// respond, and records the source address of every request it receives.
type fakeSTUNServer struct {
	conn *net.UDPConn

	mu    sync.Mutex
	froms []*net.UDPAddr
}

func startFakeSTUNServer(t *testing.T, respond func(srv *net.UDPConn, from *net.UDPAddr, req *Message)) *fakeSTUNServer {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &fakeSTUNServer{conn: conn}
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1500)
		for {
			n, from, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			req, err := (&STUNClient{}).decodeMessage(buf[:n])
			if err != nil {
				continue
			}
			s.mu.Lock()
			s.froms = append(s.froms, from)
			s.mu.Unlock()
			respond(conn, from, req)
		}
	}()
	t.Cleanup(func() {
		_ = conn.Close()
		<-done
	})
	return s
}

func (s *fakeSTUNServer) addr() string { return s.conn.LocalAddr().String() }

func (s *fakeSTUNServer) clientPorts() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	ports := make([]int, 0, len(s.froms))
	for _, f := range s.froms {
		ports = append(ports, f.Port)
	}
	return ports
}

// sendSTUN encodes a message of the given type carrying one XOR address
// attribute and sends it from conn to dst.
func sendSTUN(t *testing.T, conn *net.UDPConn, dst *net.UDPAddr, typ uint16, txID [12]byte, attrType uint16, addr *net.UDPAddr) {
	t.Helper()
	msg := &Message{Type: typ, MagicCookie: MagicCookie, TransactionID: txID}
	if addr != nil {
		v := (&TURNClient{}).encodeXorAddress(addr, txID)
		msg.Attributes = append(msg.Attributes, Attribute{Type: attrType, Length: uint16(len(v)), Value: v})
	}
	data, err := (&STUNClient{}).encodeMessage(msg)
	if err != nil {
		t.Errorf("encode: %v", err)
		return
	}
	if _, err := conn.WriteToUDP(data, dst); err != nil {
		t.Errorf("send: %v", err)
	}
}

// A STUN binding must only be answered by a Binding success response that
// comes from the server it was sent to and carries the request's transaction
// ID. Stray datagrams from other sources, stale responses with another
// transaction ID and messages of the wrong type must be skipped.
func TestGetMappedAddress_MatchesResponseToRequestAudit(t *testing.T) {
	bogus := &net.UDPAddr{IP: net.IPv4(6, 6, 6, 6), Port: 6666}

	spoofer, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("listen spoofer: %v", err)
	}
	defer spoofer.Close()

	srv := startFakeSTUNServer(t, func(srv *net.UDPConn, from *net.UDPAddr, req *Message) {
		other := req.TransactionID
		other[0] ^= 0xff
		// Another host, even one that knows the transaction ID.
		sendSTUN(t, spoofer, from, BindingResponse, req.TransactionID, AttrXorMappedAddress, bogus)
		// The server itself, but a stale/foreign transaction.
		sendSTUN(t, srv, from, BindingResponse, other, AttrXorMappedAddress, bogus)
		// Right transaction, wrong message type.
		sendSTUN(t, srv, from, BindingRequest, req.TransactionID, AttrXorMappedAddress, bogus)
		// The genuine answer.
		sendSTUN(t, srv, from, BindingResponse, req.TransactionID, AttrXorMappedAddress, from)
	})

	sc := NewSTUNClient(srv.addr())
	sc.SetTimeout(2 * time.Second)
	defer sc.Close()

	got, err := sc.GetMappedAddress()
	if err != nil {
		t.Fatalf("GetMappedAddress: %v", err)
	}
	ports := srv.clientPorts()
	if len(ports) != 1 {
		t.Fatalf("server saw %d requests, want 1", len(ports))
	}
	if !got.IP.Equal(net.IPv4(127, 0, 0, 1)) || got.Port != ports[0] {
		t.Fatalf("GetMappedAddress = %v, want 127.0.0.1:%d (took an unrelated datagram)", got, ports[0])
	}
}

// A Binding error response must be reported as an error, not mined for an
// address.
func TestGetMappedAddress_ErrorResponseAudit(t *testing.T) {
	bogus := &net.UDPAddr{IP: net.IPv4(6, 6, 6, 6), Port: 6666}
	srv := startFakeSTUNServer(t, func(srv *net.UDPConn, from *net.UDPAddr, req *Message) {
		sendSTUN(t, srv, from, BindingErrorResponse, req.TransactionID, AttrXorMappedAddress, bogus)
	})

	sc := NewSTUNClient(srv.addr())
	sc.SetTimeout(2 * time.Second)
	defer sc.Close()

	if got, err := sc.GetMappedAddress(); err == nil {
		t.Fatalf("GetMappedAddress accepted a Binding error response: %v", got)
	}
}

// The TURN Allocate transaction has the same requirement.
func TestTURNAllocate_MatchesResponseToRequestAudit(t *testing.T) {
	bogus := &net.UDPAddr{IP: net.IPv4(6, 6, 6, 6), Port: 6666}
	relay := &net.UDPAddr{IP: net.IPv4(198, 51, 100, 7), Port: 49152}

	spoofer, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("listen spoofer: %v", err)
	}
	defer spoofer.Close()

	srv := startFakeSTUNServer(t, func(srv *net.UDPConn, from *net.UDPAddr, req *Message) {
		if req.Type != AllocateRequest {
			return
		}
		other := req.TransactionID
		other[0] ^= 0xff
		sendSTUN(t, spoofer, from, AllocateResponse, req.TransactionID, AttrXorRelayedAddress, bogus)
		sendSTUN(t, srv, from, AllocateResponse, other, AttrXorRelayedAddress, bogus)
		sendSTUN(t, srv, from, AllocateResponse, req.TransactionID, AttrXorRelayedAddress, relay)
	})

	tc := NewTURNClient(srv.addr(), "user", "pass")
	tc.timeout = 2 * time.Second
	defer tc.Close()

	got, err := tc.Allocate()
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if !got.IP.Equal(relay.IP) || got.Port != relay.Port {
		t.Fatalf("Allocate = %v, want %v (took an unrelated datagram)", got, relay)
	}
}
