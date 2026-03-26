package stun

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// TURN message types (RFC 5766)
const (
	AllocateRequest         uint16 = 0x0003
	AllocateResponse        uint16 = 0x0103
	AllocateErrorResponse   uint16 = 0x0113
	RefreshRequest          uint16 = 0x0004
	RefreshResponse         uint16 = 0x0104
	RefreshErrorResponse    uint16 = 0x0114
	SendIndication          uint16 = 0x0016
	DataIndication          uint16 = 0x0017
	CreatePermissionRequest uint16 = 0x0008
	CreatePermissionResponse uint16 = 0x0108
	ChannelBindRequest      uint16 = 0x0009
	ChannelBindResponse     uint16 = 0x0109
)

// TURN-specific attribute types
const (
	AttrChannelNumber      uint16 = 0x000C
	AttrLifetime           uint16 = 0x000D
	AttrXorPeerAddress     uint16 = 0x0012
	AttrData               uint16 = 0x0013
	AttrXorRelayedAddress  uint16 = 0x0016
	AttrEvenPort           uint16 = 0x0018
	AttrRequestedTransport uint16 = 0x0019
	AttrDontFragment       uint16 = 0x001A
	AttrReservationToken   uint16 = 0x0022
)

// TURNClient handles TURN protocol for relayed communication
type TURNClient struct {
	mu sync.RWMutex

	serverAddr string
	username   string
	password   string

	conn       *net.UDPConn
	localAddr  *net.UDPAddr
	relayAddr  *net.UDPAddr

	timeout  time.Duration
	lifetime time.Duration

	// Permissions
	permissions map[string]time.Time // peer IP -> expiry time

	// Statistics
	allocations     uint64
	refreshes       uint64
	permissionsSet  uint64
	bytesSent       uint64
	bytesReceived   uint64
	lastRefresh     time.Time
}

// NewTURNClient creates a new TURN client
func NewTURNClient(serverAddr, username, password string) *TURNClient {
	return &TURNClient{
		serverAddr:  serverAddr,
		username:    username,
		password:    password,
		timeout:     5 * time.Second,
		lifetime:    600 * time.Second, // 10 minutes
		permissions: make(map[string]time.Time),
	}
}

// Connect establishes connection to TURN server
func (tc *TURNClient) Connect() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Create local UDP connection
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return fmt.Errorf("failed to create UDP socket: %w", err)
	}

	tc.conn = conn
	tc.localAddr = conn.LocalAddr().(*net.UDPAddr)

	return nil
}

// Allocate requests a relay address from TURN server
func (tc *TURNClient) Allocate() (*net.UDPAddr, error) {
	if tc.conn == nil {
		if err := tc.Connect(); err != nil {
			return nil, err
		}
	}

	// Create allocate request
	msg := tc.createAllocateRequest()

	// Add authentication
	if err := tc.addAuthentication(msg); err != nil {
		return nil, fmt.Errorf("failed to add authentication: %w", err)
	}

	// Send request
	data, err := tc.encodeMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", tc.serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server: %w", err)
	}

	tc.mu.Lock()
	_, err = tc.conn.WriteToUDP(data, serverAddr)
	tc.allocations++
	tc.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Receive response
	buffer := make([]byte, 1500)
	tc.conn.SetReadDeadline(time.Now().Add(tc.timeout))

	n, _, err := tc.conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	// Parse response
	response, err := tc.decodeMessage(buffer[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for error response
	if response.Type == AllocateErrorResponse {
		return nil, fmt.Errorf("allocation failed")
	}

	// Extract relayed address
	relayAddr, err := tc.extractRelayedAddress(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract relayed address: %w", err)
	}

	tc.mu.Lock()
	tc.relayAddr = relayAddr
	tc.lastRefresh = time.Now()
	tc.mu.Unlock()

	return relayAddr, nil
}

// Refresh extends the allocation lifetime
func (tc *TURNClient) Refresh() error {
	tc.mu.RLock()
	if tc.relayAddr == nil {
		tc.mu.RUnlock()
		return fmt.Errorf("no allocation to refresh")
	}
	tc.mu.RUnlock()

	// Create refresh request
	msg := tc.createRefreshRequest()

	// Add authentication
	if err := tc.addAuthentication(msg); err != nil {
		return fmt.Errorf("failed to add authentication: %w", err)
	}

	// Send request
	data, err := tc.encodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", tc.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve server: %w", err)
	}

	tc.mu.Lock()
	_, err = tc.conn.WriteToUDP(data, serverAddr)
	tc.refreshes++
	tc.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Receive response
	buffer := make([]byte, 1500)
	tc.conn.SetReadDeadline(time.Now().Add(tc.timeout))

	n, _, err := tc.conn.ReadFromUDP(buffer)
	if err != nil {
		return fmt.Errorf("failed to receive response: %w", err)
	}

	// Parse response
	response, err := tc.decodeMessage(buffer[:n])
	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Type == RefreshErrorResponse {
		return fmt.Errorf("refresh failed")
	}

	tc.mu.Lock()
	tc.lastRefresh = time.Now()
	tc.mu.Unlock()

	return nil
}

// CreatePermission creates permission for a peer address
func (tc *TURNClient) CreatePermission(peerAddr *net.UDPAddr) error {
	// Create permission request
	msg := tc.createPermissionRequest(peerAddr)

	// Add authentication
	if err := tc.addAuthentication(msg); err != nil {
		return fmt.Errorf("failed to add authentication: %w", err)
	}

	// Send request
	data, err := tc.encodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", tc.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve server: %w", err)
	}

	tc.mu.Lock()
	_, err = tc.conn.WriteToUDP(data, serverAddr)
	tc.permissionsSet++
	tc.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Receive response
	buffer := make([]byte, 1500)
	tc.conn.SetReadDeadline(time.Now().Add(tc.timeout))

	n, _, err := tc.conn.ReadFromUDP(buffer)
	if err != nil {
		return fmt.Errorf("failed to receive response: %w", err)
	}

	// Parse response
	_, err = tc.decodeMessage(buffer[:n])
	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Store permission
	tc.mu.Lock()
	tc.permissions[peerAddr.IP.String()] = time.Now().Add(5 * time.Minute)
	tc.mu.Unlock()

	return nil
}

// SendData sends data through the relay to a peer
func (tc *TURNClient) SendData(peerAddr *net.UDPAddr, data []byte) error {
	// Check if we have permission
	tc.mu.RLock()
	expiry, hasPermission := tc.permissions[peerAddr.IP.String()]
	tc.mu.RUnlock()

	if !hasPermission || time.Now().After(expiry) {
		// Need to create permission first
		if err := tc.CreatePermission(peerAddr); err != nil {
			return fmt.Errorf("failed to create permission: %w", err)
		}
	}

	// Create Send indication
	msg := tc.createSendIndication(peerAddr, data)

	// Encode and send
	encoded, err := tc.encodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", tc.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve server: %w", err)
	}

	tc.mu.Lock()
	n, err := tc.conn.WriteToUDP(encoded, serverAddr)
	tc.bytesSent += uint64(n)
	tc.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	return nil
}

// ReceiveData receives data from the relay
func (tc *TURNClient) ReceiveData() ([]byte, *net.UDPAddr, error) {
	buffer := make([]byte, 1500)

	tc.conn.SetReadDeadline(time.Now().Add(tc.timeout))

	n, _, err := tc.conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, nil, err
	}

	tc.mu.Lock()
	tc.bytesReceived += uint64(n)
	tc.mu.Unlock()

	// Parse message
	msg, err := tc.decodeMessage(buffer[:n])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode message: %w", err)
	}

	// Check if it's a Data indication
	if msg.Type != DataIndication {
		return nil, nil, fmt.Errorf("unexpected message type: 0x%04x", msg.Type)
	}

	// Extract peer address and data
	var peerAddr *net.UDPAddr
	var data []byte

	for _, attr := range msg.Attributes {
		if attr.Type == AttrXorPeerAddress {
			peerAddr, err = tc.parseXorAddress(attr.Value, msg.TransactionID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse peer address: %w", err)
			}
		} else if attr.Type == AttrData {
			data = attr.Value
		}
	}

	if peerAddr == nil || data == nil {
		return nil, nil, fmt.Errorf("missing peer address or data")
	}

	return data, peerAddr, nil
}

// Close closes the TURN client and deallocates relay
func (tc *TURNClient) Close() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Send refresh with lifetime 0 to deallocate
	if tc.conn != nil && tc.relayAddr != nil {
		msg := tc.createRefreshRequest()
		msg.Attributes = append(msg.Attributes, Attribute{
			Type:   AttrLifetime,
			Length: 4,
			Value:  []byte{0, 0, 0, 0},
		})

		data, _ := tc.encodeMessage(msg)
		serverAddr, _ := net.ResolveUDPAddr("udp", tc.serverAddr)
		tc.conn.WriteToUDP(data, serverAddr)
	}

	if tc.conn != nil {
		err := tc.conn.Close()
		tc.conn = nil
		return err
	}

	return nil
}

// GetRelayAddress returns the allocated relay address
func (tc *TURNClient) GetRelayAddress() *net.UDPAddr {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	return tc.relayAddr
}

// GetStatistics returns TURN client statistics
func (tc *TURNClient) GetStatistics() TURNStatistics {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	return TURNStatistics{
		Allocations:    tc.allocations,
		Refreshes:      tc.refreshes,
		PermissionsSet: tc.permissionsSet,
		BytesSent:      tc.bytesSent,
		BytesReceived:  tc.bytesReceived,
		RelayAddr:      tc.relayAddr,
		LastRefresh:    tc.lastRefresh,
	}
}

// TURNStatistics represents TURN client statistics
type TURNStatistics struct {
	Allocations    uint64
	Refreshes      uint64
	PermissionsSet uint64
	BytesSent      uint64
	BytesReceived  uint64
	RelayAddr      *net.UDPAddr
	LastRefresh    time.Time
}

// Private helper methods

func (tc *TURNClient) createAllocateRequest() *Message {
	msg := &Message{
		Type:        AllocateRequest,
		MagicCookie: MagicCookie,
	}

	// Generate transaction ID
	tc.generateTransactionID(&msg.TransactionID)

	// Add REQUESTED-TRANSPORT attribute (UDP = 17)
	msg.Attributes = append(msg.Attributes, Attribute{
		Type:   AttrRequestedTransport,
		Length: 4,
		Value:  []byte{17, 0, 0, 0}, // Protocol 17 (UDP), RFFU
	})

	// Add LIFETIME attribute
	lifetime := uint32(tc.lifetime.Seconds())
	lifetimeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lifetimeBytes, lifetime)

	msg.Attributes = append(msg.Attributes, Attribute{
		Type:   AttrLifetime,
		Length: 4,
		Value:  lifetimeBytes,
	})

	return msg
}

func (tc *TURNClient) createRefreshRequest() *Message {
	msg := &Message{
		Type:        RefreshRequest,
		MagicCookie: MagicCookie,
	}

	tc.generateTransactionID(&msg.TransactionID)

	// Add LIFETIME attribute
	lifetime := uint32(tc.lifetime.Seconds())
	lifetimeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lifetimeBytes, lifetime)

	msg.Attributes = append(msg.Attributes, Attribute{
		Type:   AttrLifetime,
		Length: 4,
		Value:  lifetimeBytes,
	})

	return msg
}

func (tc *TURNClient) createPermissionRequest(peerAddr *net.UDPAddr) *Message {
	msg := &Message{
		Type:        CreatePermissionRequest,
		MagicCookie: MagicCookie,
	}

	tc.generateTransactionID(&msg.TransactionID)

	// Add XOR-PEER-ADDRESS
	xorPeerAddr := tc.encodeXorAddress(peerAddr, msg.TransactionID)
	msg.Attributes = append(msg.Attributes, Attribute{
		Type:   AttrXorPeerAddress,
		Length: uint16(len(xorPeerAddr)),
		Value:  xorPeerAddr,
	})

	return msg
}

func (tc *TURNClient) createSendIndication(peerAddr *net.UDPAddr, data []byte) *Message {
	msg := &Message{
		Type:        SendIndication,
		MagicCookie: MagicCookie,
	}

	tc.generateTransactionID(&msg.TransactionID)

	// Add XOR-PEER-ADDRESS
	xorPeerAddr := tc.encodeXorAddress(peerAddr, msg.TransactionID)
	msg.Attributes = append(msg.Attributes, Attribute{
		Type:   AttrXorPeerAddress,
		Length: uint16(len(xorPeerAddr)),
		Value:  xorPeerAddr,
	})

	// Add DATA attribute
	msg.Attributes = append(msg.Attributes, Attribute{
		Type:   AttrData,
		Length: uint16(len(data)),
		Value:  data,
	})

	return msg
}

func (tc *TURNClient) addAuthentication(msg *Message) error {
	// Add USERNAME
	msg.Attributes = append(msg.Attributes, Attribute{
		Type:   AttrUsername,
		Length: uint16(len(tc.username)),
		Value:  []byte(tc.username),
	})

	// Calculate MESSAGE-INTEGRITY (simplified - would need proper HMAC-SHA1)
	// In a real implementation, this would:
	// 1. Add REALM and NONCE from server
	// 2. Calculate HMAC-SHA1 over the message
	key := md5.Sum([]byte(tc.username + ":" + tc.password))
	h := hmac.New(sha1.New, key[:])

	// Placeholder integrity value
	integrity := h.Sum(nil)

	msg.Attributes = append(msg.Attributes, Attribute{
		Type:   AttrMessageIntegrity,
		Length: 20,
		Value:  integrity,
	})

	return nil
}

func (tc *TURNClient) extractRelayedAddress(msg *Message) (*net.UDPAddr, error) {
	for _, attr := range msg.Attributes {
		if attr.Type == AttrXorRelayedAddress {
			return tc.parseXorAddress(attr.Value, msg.TransactionID)
		}
	}

	return nil, fmt.Errorf("no relayed address in response")
}

func (tc *TURNClient) encodeXorAddress(addr *net.UDPAddr, txID [12]byte) []byte {
	buf := make([]byte, 8)

	// Family (IPv4)
	buf[1] = 0x01

	// XOR port
	xorPort := uint16(addr.Port) ^ 0x2112
	binary.BigEndian.PutUint16(buf[2:4], xorPort)

	// XOR IP
	ip := addr.IP.To4()
	if ip != nil {
		ipInt := binary.BigEndian.Uint32(ip)
		xorIP := ipInt ^ MagicCookie
		binary.BigEndian.PutUint32(buf[4:8], xorIP)
	}

	return buf
}

func (tc *TURNClient) parseXorAddress(data []byte, txID [12]byte) (*net.UDPAddr, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("invalid XOR address length")
	}

	family := data[1]
	xorPort := binary.BigEndian.Uint16(data[2:4])
	port := xorPort ^ 0x2112

	var ip net.IP
	if family == 0x01 { // IPv4
		xorIP := binary.BigEndian.Uint32(data[4:8])
		actualIP := xorIP ^ MagicCookie

		ip = make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, actualIP)
	}

	return &net.UDPAddr{
		IP:   ip,
		Port: int(port),
	}, nil
}

func (tc *TURNClient) encodeMessage(msg *Message) ([]byte, error) {
	// Reuse STUN encoding logic
	sc := &STUNClient{}
	return sc.encodeMessage(msg)
}

func (tc *TURNClient) decodeMessage(data []byte) (*Message, error) {
	// Reuse STUN decoding logic
	sc := &STUNClient{}
	return sc.decodeMessage(data)
}

func (tc *TURNClient) generateTransactionID(txID *[12]byte) {
	// Use current time + random bytes
	now := time.Now().UnixNano()
	binary.BigEndian.PutUint64((*txID)[:8], uint64(now))

	// Random bytes for last 4
	for i := 8; i < 12; i++ {
		(*txID)[i] = byte(now >> (i * 8))
	}
}
