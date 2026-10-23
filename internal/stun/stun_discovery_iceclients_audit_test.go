package stun

import (
	"net"
	"testing"
	"time"
)

// Every STUN client the ICE agent opens while gathering must be closed by
// ICEAgent.Close (or earlier), not just the one for the last server.
func TestICEAgent_CloseReleasesAllGatherSocketsAudit(t *testing.T) {
	ok := func(srv *net.UDPConn, from *net.UDPAddr, req *Message) {
		sendSTUN(t, srv, from, BindingResponse, req.TransactionID, AttrXorMappedAddress, from)
	}
	fail := func(srv *net.UDPConn, from *net.UDPAddr, req *Message) {
		sendSTUN(t, srv, from, BindingErrorResponse, req.TransactionID, 0, nil)
	}
	s1 := startFakeSTUNServer(t, ok)
	s2 := startFakeSTUNServer(t, fail)
	s3 := startFakeSTUNServer(t, ok)

	ia := NewICEAgent([]string{s1.addr(), s2.addr(), s3.addr()}, nil)
	done := make(chan error, 1)
	go func() { done <- ia.GatherCandidates() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("GatherCandidates: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("GatherCandidates did not finish")
	}

	srflx := 0
	for _, c := range ia.GetLocalCandidates() {
		if c.Type == CandidateTypeSrflx {
			srflx++
		}
	}
	if srflx != 2 {
		t.Fatalf("got %d srflx candidates, want 2", srflx)
	}

	if err := ia.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var ports []int
	for _, s := range []*fakeSTUNServer{s1, s2, s3} {
		ports = append(ports, s.clientPorts()...)
	}
	if len(ports) != 3 {
		t.Fatalf("servers saw %d requests, want 3", len(ports))
	}
	for _, p := range ports {
		c, err := net.ListenUDP("udp", &net.UDPAddr{Port: p})
		if err != nil {
			t.Errorf("gathering socket on port %d still open after ICEAgent.Close: %v", p, err)
			continue
		}
		_ = c.Close()
	}
}
