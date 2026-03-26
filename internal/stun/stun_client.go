package stun

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// STUN message types (RFC 5389)
const (
	BindingRequest         uint16 = 0x0001
	BindingResponse        uint16 = 0x0101
	BindingErrorResponse   uint16 = 0x0111
	SharedSecretRequest    uint16 = 0x0002
	SharedSecretResponse   uint16 = 0x0102
	SharedSecretError      uint16 = 0x0112
)

// STUN attribute types
const (
	AttrMappedAddress     uint16 = 0x0001
	AttrResponseAddress   uint16 = 0x0002
	AttrChangeRequest     uint16 = 0x0003
	AttrSourceAddress     uint16 = 0x0004
	AttrChangedAddress    uint16 = 0x0005
	AttrUsername          uint16 = 0x0006
	AttrPassword          uint16 = 0x0007
	AttrMessageIntegrity  uint16 = 0x0008
	AttrErrorCode         uint16 = 0x0009
	AttrUnknownAttributes uint16 = 0x000A
	AttrReflectedFrom     uint16 = 0x000B
	AttrRealm             uint16 = 0x0014
	AttrNonce             uint16 = 0x0015
	AttrXorMappedAddress  uint16 = 0x0020
)

const (
	MagicCookie uint32 = 0x2112A442
	HeaderSize  int    = 20
)

// STUNClient handles STUN protocol communication
type STUNClient struct {
	mu sync.RWMutex

	serverAddr   string
	localAddr    *net.UDPAddr
	conn         *net.UDPConn
	timeout      time.Duration
	retries      int

	// Discovered addresses
	mappedAddr    *net.UDPAddr
	changedAddr   *net.UDPAddr
	natType       NATType

	// Statistics
	requestsSent    uint64
	responsesRecv   uint64
	timeouts        uint64
	lastRTT         time.Duration
}

// NATType represents the type of NAT detected
type NATType string

const (
	NATTypeUnknown             NATType = "unknown"
	NATTypeOpenInternet        NATType = "open_internet"
	NATTypeFullCone            NATType = "full_cone"
	NATTypeRestrictedCone      NATType = "restricted_cone"
	NATTypePortRestrictedCone  NATType = "port_restricted_cone"
	NATTypeSymmetric           NATType = "symmetric"
	NATTypeSymmetricFirewall   NATType = "symmetric_firewall"
)

// Message represents a STUN message
type Message struct {
	Type          uint16
	Length        uint16
	MagicCookie   uint32
	TransactionID [12]byte
	Attributes    []Attribute
}

// Attribute represents a STUN attribute
type Attribute struct {
	Type   uint16
	Length uint16
	Value  []byte
}

// NewSTUNClient creates a new STUN client
func NewSTUNClient(serverAddr string) *STUNClient {
	return &STUNClient{
		serverAddr: serverAddr,
		timeout:    3 * time.Second,
		retries:    3,
		natType:    NATTypeUnknown,
	}
}

// Connect establishes connection to STUN server
func (sc *STUNClient) Connect() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Create local UDP connection
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return fmt.Errorf("failed to create UDP socket: %w", err)
	}

	sc.conn = conn
	sc.localAddr = conn.LocalAddr().(*net.UDPAddr)

	// Set server as remote address
	err = conn.SetReadDeadline(time.Now().Add(sc.timeout))
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to set deadline: %w", err)
	}

	return nil
}

// GetMappedAddress performs STUN binding request to discover public address
func (sc *STUNClient) GetMappedAddress() (*net.UDPAddr, error) {
	if sc.conn == nil {
		if err := sc.Connect(); err != nil {
			return nil, err
		}
	}

	// Create binding request
	msg := sc.createBindingRequest()

	// Send request
	data, err := sc.encodeMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	serverAddr, err := net.ResolveUDPAddr("udp", sc.serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server: %w", err)
	}

	start := time.Now()

	sc.mu.Lock()
	_, err = sc.conn.WriteToUDP(data, serverAddr)
	sc.requestsSent++
	sc.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Receive response
	buffer := make([]byte, 1500)
	sc.conn.SetReadDeadline(time.Now().Add(sc.timeout))

	n, _, err := sc.conn.ReadFromUDP(buffer)
	if err != nil {
		sc.mu.Lock()
		sc.timeouts++
		sc.mu.Unlock()
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	sc.mu.Lock()
	sc.responsesRecv++
	sc.lastRTT = time.Since(start)
	sc.mu.Unlock()

	// Parse response
	response, err := sc.decodeMessage(buffer[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract mapped address
	mappedAddr, err := sc.extractMappedAddress(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract mapped address: %w", err)
	}

	sc.mu.Lock()
	sc.mappedAddr = mappedAddr
	sc.mu.Unlock()

	return mappedAddr, nil
}

// DetectNATType performs STUN tests to detect NAT type
func (sc *STUNClient) DetectNATType() (NATType, error) {
	// Test 1: Basic binding request
	mappedAddr, err := sc.GetMappedAddress()
	if err != nil {
		return NATTypeUnknown, err
	}

	// Check if we're on open internet (mapped == local)
	if mappedAddr.IP.Equal(sc.localAddr.IP) && mappedAddr.Port == sc.localAddr.Port {
		sc.mu.Lock()
		sc.natType = NATTypeOpenInternet
		sc.mu.Unlock()
		return NATTypeOpenInternet, nil
	}

	// For full NAT detection, we would need:
	// - Test 2: Binding request with change-request (change IP and port)
	// - Test 3: Binding request with change-request (change port only)
	// - Test 4: Multiple binding requests to detect symmetric behavior

	// Simplified detection: assume port-restricted cone NAT
	// (most common type for home routers)
	sc.mu.Lock()
	sc.natType = NATTypePortRestrictedCone
	sc.mu.Unlock()

	return NATTypePortRestrictedCone, nil
}

// Close closes the STUN client connection
func (sc *STUNClient) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.conn != nil {
		err := sc.conn.Close()
		sc.conn = nil
		return err
	}

	return nil
}

// GetStatistics returns STUN client statistics
func (sc *STUNClient) GetStatistics() Statistics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return Statistics{
		RequestsSent:  sc.requestsSent,
		ResponsesRecv: sc.responsesRecv,
		Timeouts:      sc.timeouts,
		LastRTT:       sc.lastRTT,
		MappedAddr:    sc.mappedAddr,
		NATType:       sc.natType,
	}
}

// Statistics represents STUN client statistics
type Statistics struct {
	RequestsSent  uint64
	ResponsesRecv uint64
	Timeouts      uint64
	LastRTT       time.Duration
	MappedAddr    *net.UDPAddr
	NATType       NATType
}

// Private helper methods

func (sc *STUNClient) createBindingRequest() *Message {
	msg := &Message{
		Type:        BindingRequest,
		MagicCookie: MagicCookie,
	}

	// Generate random transaction ID
	rand.Read(msg.TransactionID[:])

	return msg
}

func (sc *STUNClient) encodeMessage(msg *Message) ([]byte, error) {
	// Calculate message length (excluding header)
	length := 0
	for _, attr := range msg.Attributes {
		length += 4 + int(attr.Length) // 4 bytes for type+length
		// Padding to 4-byte boundary
		if attr.Length%4 != 0 {
			length += 4 - int(attr.Length)%4
		}
	}

	msg.Length = uint16(length)

	// Allocate buffer
	buf := make([]byte, HeaderSize+length)

	// Encode header
	binary.BigEndian.PutUint16(buf[0:2], msg.Type)
	binary.BigEndian.PutUint16(buf[2:4], msg.Length)
	binary.BigEndian.PutUint32(buf[4:8], msg.MagicCookie)
	copy(buf[8:20], msg.TransactionID[:])

	// Encode attributes
	offset := HeaderSize
	for _, attr := range msg.Attributes {
		binary.BigEndian.PutUint16(buf[offset:offset+2], attr.Type)
		binary.BigEndian.PutUint16(buf[offset+2:offset+4], attr.Length)
		copy(buf[offset+4:], attr.Value)

		attrLen := 4 + int(attr.Length)
		// Add padding
		if attr.Length%4 != 0 {
			attrLen += 4 - int(attr.Length)%4
		}
		offset += attrLen
	}

	return buf, nil
}

func (sc *STUNClient) decodeMessage(data []byte) (*Message, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("message too short")
	}

	msg := &Message{}

	// Decode header
	msg.Type = binary.BigEndian.Uint16(data[0:2])
	msg.Length = binary.BigEndian.Uint16(data[2:4])
	msg.MagicCookie = binary.BigEndian.Uint32(data[4:8])
	copy(msg.TransactionID[:], data[8:20])

	// Verify magic cookie
	if msg.MagicCookie != MagicCookie {
		return nil, fmt.Errorf("invalid magic cookie")
	}

	// Decode attributes
	offset := HeaderSize
	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}

		attr := Attribute{}
		attr.Type = binary.BigEndian.Uint16(data[offset : offset+2])
		attr.Length = binary.BigEndian.Uint16(data[offset+2 : offset+4])

		if offset+4+int(attr.Length) > len(data) {
			return nil, fmt.Errorf("invalid attribute length")
		}

		attr.Value = make([]byte, attr.Length)
		copy(attr.Value, data[offset+4:offset+4+int(attr.Length)])

		msg.Attributes = append(msg.Attributes, attr)

		// Move to next attribute (with padding)
		attrLen := 4 + int(attr.Length)
		if attr.Length%4 != 0 {
			attrLen += 4 - int(attr.Length)%4
		}
		offset += attrLen
	}

	return msg, nil
}

func (sc *STUNClient) extractMappedAddress(msg *Message) (*net.UDPAddr, error) {
	// Look for XOR-MAPPED-ADDRESS first (RFC 5389)
	for _, attr := range msg.Attributes {
		if attr.Type == AttrXorMappedAddress {
			return sc.parseXorMappedAddress(attr.Value, msg.TransactionID)
		}
	}

	// Fall back to MAPPED-ADDRESS (RFC 3489)
	for _, attr := range msg.Attributes {
		if attr.Type == AttrMappedAddress {
			return sc.parseMappedAddress(attr.Value)
		}
	}

	return nil, fmt.Errorf("no mapped address in response")
}

func (sc *STUNClient) parseMappedAddress(data []byte) (*net.UDPAddr, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("invalid mapped address length")
	}

	// Skip reserved byte
	family := data[1]
	port := binary.BigEndian.Uint16(data[2:4])

	var ip net.IP
	if family == 0x01 { // IPv4
		ip = net.IPv4(data[4], data[5], data[6], data[7])
	} else if family == 0x02 { // IPv6
		if len(data) < 20 {
			return nil, fmt.Errorf("invalid IPv6 address length")
		}
		ip = net.IP(data[4:20])
	} else {
		return nil, fmt.Errorf("unsupported address family: %d", family)
	}

	return &net.UDPAddr{
		IP:   ip,
		Port: int(port),
	}, nil
}

func (sc *STUNClient) parseXorMappedAddress(data []byte, txID [12]byte) (*net.UDPAddr, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("invalid XOR mapped address length")
	}

	family := data[1]
	xorPort := binary.BigEndian.Uint16(data[2:4])

	// XOR port with most significant 16 bits of magic cookie
	port := xorPort ^ 0x2112

	var ip net.IP
	if family == 0x01 { // IPv4
		// XOR with magic cookie
		xorIP := binary.BigEndian.Uint32(data[4:8])
		actualIP := xorIP ^ MagicCookie

		ip = make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, actualIP)
	} else if family == 0x02 { // IPv6
		if len(data) < 20 {
			return nil, fmt.Errorf("invalid IPv6 XOR address length")
		}

		// XOR with magic cookie + transaction ID
		xorKey := make([]byte, 16)
		binary.BigEndian.PutUint32(xorKey[0:4], MagicCookie)
		copy(xorKey[4:16], txID[:])

		ip = make(net.IP, 16)
		for i := 0; i < 16; i++ {
			ip[i] = data[4+i] ^ xorKey[i]
		}
	} else {
		return nil, fmt.Errorf("unsupported address family: %d", family)
	}

	return &net.UDPAddr{
		IP:   ip,
		Port: int(port),
	}, nil
}

// IsSymmetricNAT returns true if the detected NAT is symmetric
func (sc *STUNClient) IsSymmetricNAT() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.natType == NATTypeSymmetric
}

// GetLocalAddress returns the local UDP address
func (sc *STUNClient) GetLocalAddress() *net.UDPAddr {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.localAddr
}

// GetMappedAddressCached returns cached mapped address without new request
func (sc *STUNClient) GetMappedAddressCached() *net.UDPAddr {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.mappedAddr
}

// SetTimeout sets the request timeout
func (sc *STUNClient) SetTimeout(timeout time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.timeout = timeout
}

// SetRetries sets the number of retries for failed requests
func (sc *STUNClient) SetRetries(retries int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.retries = retries
}
