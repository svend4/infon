package sfu

import (
	"net"
	"testing"
	"time"
)

// A stopped server can be started again and then processes packets; stopping
// it a second time must not panic.
func TestServerRestartProcessesPackets(t *testing.T) {
	s, err := NewSFUServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewSFUServer: %v", err)
	}
	if err := s.Start(); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := s.Start(); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Stop after restart panicked: %v", r)
		}
	}()
	defer func() { _ = s.Stop() }()

	s.mu.RLock()
	serverAddr := s.conn.LocalAddr().(*net.UDPAddr)
	s.mu.RUnlock()

	client := listenLoopback(t)
	join := joinPacket("room0001", "part0001", "")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := client.WriteToUDP(join, serverAddr); err != nil {
			t.Fatalf("WriteToUDP: %v", err)
		}
		if s.GetStatistics().PacketsReceived > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("restarted server processed no packets")
}
