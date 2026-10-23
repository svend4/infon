package stun

import (
	"net"
	"testing"
	"time"
)

// TURNClient methods used before Allocate, or after Close, must return an
// error instead of dereferencing a nil connection. Each call gets its own
// client: a panic inside a locked section would leave tc.mu held and hang
// the next call on the same client.
func TestTURNClient_NilConnAudit(t *testing.T) {
	peer := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9}

	fresh := func(t *testing.T) *TURNClient {
		return NewTURNClient("127.0.0.1:9", "user", "pass")
	}
	closed := func(t *testing.T) *TURNClient {
		tc := NewTURNClient("127.0.0.1:9", "user", "pass")
		if err := tc.Connect(); err != nil {
			t.Fatalf("Connect: %v", err)
		}
		tc.mu.Lock()
		tc.relayAddr = &net.UDPAddr{IP: net.IPv4(198, 51, 100, 7), Port: 49152}
		tc.permissions[peer.IP.String()] = time.Now().Add(time.Hour)
		tc.mu.Unlock()
		_ = tc.Close()
		return tc
	}

	calls := map[string]func(tc *TURNClient) error{
		"Refresh": func(tc *TURNClient) error { return tc.Refresh() },
		"ReceiveData": func(tc *TURNClient) error {
			_, _, err := tc.ReceiveData()
			return err
		},
		"CreatePermission": func(tc *TURNClient) error { return tc.CreatePermission(peer) },
		"SendData":         func(tc *TURNClient) error { return tc.SendData(peer, []byte("x")) },
	}

	for _, state := range []struct {
		name string
		mk   func(t *testing.T) *TURNClient
	}{{"before Allocate", fresh}, {"after Close", closed}} {
		for name, call := range calls {
			t.Run(name+" "+state.name, func(t *testing.T) {
				tc := state.mk(t)
				done := make(chan struct{})
				go func() {
					defer close(done)
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("panicked: %v", r)
						}
					}()
					if err := call(tc); err == nil {
						t.Error("returned nil error without a connection")
					}
				}()
				select {
				case <-done:
				case <-time.After(10 * time.Second):
					t.Fatal("did not return")
				}
			})
		}
	}
}
