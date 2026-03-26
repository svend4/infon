package network

import (
	"sync"
	"time"
)

// LossDetector tracks packet sequence numbers and detects losses
type LossDetector struct {
	mu sync.Mutex

	// Sequence tracking
	expectedSeq uint32        // Next expected sequence number
	received    map[uint32]bool // Received packets buffer
	maxWindow   uint32        // Maximum window size for out-of-order packets

	// Statistics
	totalReceived uint64
	totalLost     uint64
	outOfOrder    uint64
	duplicate     uint64

	// Loss detection
	lostPackets   []uint32      // List of detected lost packets
	lossThreshold time.Duration // Time to wait before declaring packet lost
	lastReceived  time.Time     // Last packet receive time
}

// NewLossDetector creates a new loss detector
func NewLossDetector() *LossDetector {
	return &LossDetector{
		expectedSeq:   0,
		received:      make(map[uint32]bool),
		maxWindow:     100,         // Allow up to 100 out-of-order packets
		lossThreshold: 100 * time.Millisecond, // 100ms window
		lastReceived:  time.Now(),
	}
}

// OnPacketReceived processes a received packet and updates statistics
// Returns true if this is a new packet, false if duplicate
func (ld *LossDetector) OnPacketReceived(seq uint32) bool {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	// Update last received time
	ld.lastReceived = time.Now()

	// Initialize expected sequence on first packet
	if ld.totalReceived == 0 {
		ld.expectedSeq = seq
	}

	// Check if duplicate
	if ld.received[seq] {
		ld.duplicate++
		return false
	}

	// Mark as received
	ld.received[seq] = true
	ld.totalReceived++

	// Check if in order
	if seq == ld.expectedSeq {
		// In order - advance expected sequence
		ld.expectedSeq++

		// Check if we have buffered packets that can now be processed
		for ld.received[ld.expectedSeq] {
			ld.expectedSeq++
		}

		// Clean up old entries from receive buffer
		ld.cleanupReceiveBuffer()

	} else if seq > ld.expectedSeq {
		// Out of order - future packet received
		ld.outOfOrder++

		// Detect missing packets (only within reasonable window)
		gap := seq - ld.expectedSeq
		if gap <= ld.maxWindow {
			for i := ld.expectedSeq; i < seq; i++ {
				if !ld.received[i] {
					ld.lostPackets = append(ld.lostPackets, i)
					ld.totalLost++
				}
			}
		}
	}
	// If seq < expectedSeq, it's either a late packet or retransmission

	return true
}

// GetLostPackets returns and clears the list of lost packet sequence numbers
func (ld *LossDetector) GetLostPackets() []uint32 {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	lost := ld.lostPackets
	ld.lostPackets = nil
	return lost
}

// GetStatistics returns current loss statistics
func (ld *LossDetector) GetStatistics() LossStatistics {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	lossRate := 0.0
	if ld.totalReceived+ld.totalLost > 0 {
		lossRate = float64(ld.totalLost) / float64(ld.totalReceived+ld.totalLost) * 100.0
	}

	return LossStatistics{
		TotalReceived: ld.totalReceived,
		TotalLost:     ld.totalLost,
		OutOfOrder:    ld.outOfOrder,
		Duplicate:     ld.duplicate,
		LossRate:      lossRate,
		BufferSize:    uint32(len(ld.received)),
	}
}

// Reset resets the loss detector state
func (ld *LossDetector) Reset() {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.expectedSeq = 0
	ld.received = make(map[uint32]bool)
	ld.totalReceived = 0
	ld.totalLost = 0
	ld.outOfOrder = 0
	ld.duplicate = 0
	ld.lostPackets = nil
	ld.lastReceived = time.Now()
}

// cleanupReceiveBuffer removes old entries from the receive buffer
func (ld *LossDetector) cleanupReceiveBuffer() {
	if uint32(len(ld.received)) <= ld.maxWindow {
		return
	}

	// Remove entries older than maxWindow
	minSeq := ld.expectedSeq - ld.maxWindow
	for seq := range ld.received {
		if seq < minSeq {
			delete(ld.received, seq)
		}
	}
}

// LossStatistics holds packet loss statistics
type LossStatistics struct {
	TotalReceived uint64  // Total packets received
	TotalLost     uint64  // Total packets lost
	OutOfOrder    uint64  // Packets received out of order
	Duplicate     uint64  // Duplicate packets
	LossRate      float64 // Loss rate as percentage
	BufferSize    uint32  // Current receive buffer size
}
