package network

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// Control message types
const (
	ControlTypeNACK uint8 = 0x01 // Request retransmission
	ControlTypeACK  uint8 = 0x02 // Acknowledge receipt
)

// RetransmissionManager handles packet retransmission
type RetransmissionManager struct {
	mu sync.Mutex

	// Sent packet buffer for retransmission
	sentPackets   map[uint32]*SentPacketInfo
	maxBufferSize int
	maxRetries    int
	retryTimeout  time.Duration

	// Statistics
	totalRetransmits uint64
	totalNACKs       uint64
}

// SentPacketInfo stores information about a sent packet
type SentPacketInfo struct {
	Packet      *Packet
	RemoteAddr  *net.UDPAddr
	SentTime    time.Time
	RetryCount  int
	LastRetry   time.Time
}

// NewRetransmissionManager creates a new retransmission manager
func NewRetransmissionManager() *RetransmissionManager {
	return &RetransmissionManager{
		sentPackets:   make(map[uint32]*SentPacketInfo),
		maxBufferSize: 200,              // Buffer last 200 packets
		maxRetries:    3,                // Retry up to 3 times
		retryTimeout:  200 * time.Millisecond, // 200ms timeout
	}
}

// OnPacketSent records a sent packet for potential retransmission
func (rm *RetransmissionManager) OnPacketSent(packet *Packet, remoteAddr *net.UDPAddr) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Store packet info
	rm.sentPackets[packet.Sequence] = &SentPacketInfo{
		Packet:     packet,
		RemoteAddr: remoteAddr,
		SentTime:   time.Now(),
		RetryCount: 0,
	}

	// Cleanup old packets if buffer is full
	if len(rm.sentPackets) > rm.maxBufferSize {
		rm.cleanupOldPackets()
	}
}

// OnACKReceived marks a packet as acknowledged
func (rm *RetransmissionManager) OnACKReceived(seq uint32) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.sentPackets, seq)
}

// ProcessNACK handles a NACK (retransmission request)
// Returns the packet to retransmit, or nil if not found/too many retries
func (rm *RetransmissionManager) ProcessNACK(seq uint32) (*Packet, *net.UDPAddr, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.totalNACKs++

	info, exists := rm.sentPackets[seq]
	if !exists {
		// Packet not in buffer (too old or never sent)
		return nil, nil, false
	}

	// Check retry limit
	if info.RetryCount >= rm.maxRetries {
		// Too many retries, give up
		delete(rm.sentPackets, seq)
		return nil, nil, false
	}

	// Update retry info
	info.RetryCount++
	info.LastRetry = time.Now()
	rm.totalRetransmits++

	return info.Packet, info.RemoteAddr, true
}

// GetRetransmitCandidates returns packets that should be retransmitted
// based on timeout (not used with NACK-based retransmission, but useful for timeout-based)
func (rm *RetransmissionManager) GetRetransmitCandidates() []*SentPacketInfo {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var candidates []*SentPacketInfo
	now := time.Now()

	for _, info := range rm.sentPackets {
		// Check if timeout expired
		timeSinceLastSend := now.Sub(info.SentTime)
		if info.RetryCount > 0 {
			timeSinceLastSend = now.Sub(info.LastRetry)
		}

		if timeSinceLastSend > rm.retryTimeout && info.RetryCount < rm.maxRetries {
			candidates = append(candidates, info)
		}
	}

	return candidates
}

// GetStatistics returns retransmission statistics
func (rm *RetransmissionManager) GetStatistics() RetransmissionStatistics {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	return RetransmissionStatistics{
		BufferedPackets:  uint32(len(rm.sentPackets)),
		TotalRetransmits: rm.totalRetransmits,
		TotalNACKs:       rm.totalNACKs,
	}
}

// cleanupOldPackets removes the oldest packets from buffer
func (rm *RetransmissionManager) cleanupOldPackets() {
	// Find oldest sequence number
	var oldestSeq uint32
	var oldestTime time.Time
	first := true

	for seq, info := range rm.sentPackets {
		if first || info.SentTime.Before(oldestTime) {
			oldestSeq = seq
			oldestTime = info.SentTime
			first = false
		}
	}

	// Remove oldest
	if !first {
		delete(rm.sentPackets, oldestSeq)
	}
}

// RetransmissionStatistics holds retransmission statistics
type RetransmissionStatistics struct {
	BufferedPackets  uint32 // Number of packets in retransmit buffer
	TotalRetransmits uint64 // Total retransmissions performed
	TotalNACKs       uint64 // Total NACKs received
}

// CreateNACKPacket creates a NACK control packet requesting retransmission
func CreateNACKPacket(lostSequences []uint32, currentSeq uint32) *Packet {
	// Payload format: [count:2][seq1:4][seq2:4]...
	payloadSize := 2 + len(lostSequences)*4
	payload := make([]byte, payloadSize)

	// Write count
	binary.BigEndian.PutUint16(payload[0:2], uint16(len(lostSequences)))

	// Write lost sequence numbers
	for i, seq := range lostSequences {
		offset := 2 + i*4
		binary.BigEndian.PutUint32(payload[offset:offset+4], seq)
	}

	return &Packet{
		Type:      PacketTypeControl,
		Sequence:  currentSeq,
		Timestamp: uint64(time.Now().UnixMilli()),
		Payload:   payload,
	}
}

// ParseNACKPayload extracts lost sequence numbers from a NACK packet
func ParseNACKPayload(payload []byte) ([]uint32, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("NACK payload too short")
	}

	count := binary.BigEndian.Uint16(payload[0:2])
	if len(payload) < int(2+count*4) {
		return nil, fmt.Errorf("invalid NACK payload size")
	}

	sequences := make([]uint32, count)
	for i := uint16(0); i < count; i++ {
		offset := 2 + i*4
		sequences[i] = binary.BigEndian.Uint32(payload[offset : offset+4])
	}

	return sequences, nil
}
