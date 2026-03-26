package network

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Transport handles UDP network communication
type Transport struct {
	conn     *net.UDPConn
	closed   bool
	mu       sync.Mutex
	sequence uint32
}

// NewTransport creates a new UDP transport
func NewTransport(listenAddr string) (*Transport, error) {
	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	return &Transport{
		conn:     conn,
		sequence: 0,
	}, nil
}

// SendPacket sends a packet to the remote address
func (t *Transport) SendPacket(packet *Packet, remoteAddr *net.UDPAddr) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("transport is closed")
	}

	// Encode packet
	data, err := packet.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode packet: %w", err)
	}

	// Send
	_, err = t.conn.WriteToUDP(data, remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	return nil
}

// ReceivePacket receives a packet from the network
func (t *Transport) ReceivePacket() (*Packet, *net.UDPAddr, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, nil, fmt.Errorf("transport is closed")
	}
	t.mu.Unlock()

	buf := make([]byte, MaxPacketSize)

	// Set read timeout
	t.conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	n, addr, err := t.conn.ReadFromUDP(buf)
	if err != nil {
		// Check if timeout
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, nil, netErr
		}
		return nil, nil, fmt.Errorf("failed to receive: %w", err)
	}

	// Decode packet
	packet, err := Decode(buf[:n])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode packet: %w", err)
	}

	return packet, addr, nil
}

// NextSequence returns the next sequence number
func (t *Transport) NextSequence() uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()

	seq := t.sequence
	t.sequence++
	return seq
}

// Close closes the transport
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	return t.conn.Close()
}

// LocalAddr returns the local address
func (t *Transport) LocalAddr() *net.UDPAddr {
	return t.conn.LocalAddr().(*net.UDPAddr)
}

// GetConn returns the underlying UDP connection
// This is used for direct packet receiving in group calls
func (t *Transport) GetConn() *net.UDPConn {
	return t.conn
}
