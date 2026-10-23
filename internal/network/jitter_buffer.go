package network

import (
	"sync"
	"time"
)

// JitterBuffer stores and orders packets to smooth out network jitter
type JitterBuffer struct {
	mu sync.Mutex

	// Buffer configuration
	maxSize      int           // Maximum buffer size
	minDelay     time.Duration // Minimum buffering delay
	maxDelay     time.Duration // Maximum buffering delay
	currentDelay time.Duration // Current adaptive delay

	// Packet storage
	buffer       map[uint32]*BufferedPacket
	nextExpected uint32    // Next sequence to output
	lastOutput   time.Time // Last packet output time

	// Statistics
	totalBuffered uint64
	totalOutput   uint64
	dropped       uint64
	underruns     uint64 // Times buffer was empty when output requested
}

// BufferedPacket stores a packet with arrival time
type BufferedPacket struct {
	Packet      *Packet
	ArrivalTime time.Time
}

// NewJitterBuffer creates a new jitter buffer
func NewJitterBuffer(maxSize int) *JitterBuffer {
	return &JitterBuffer{
		maxSize:      maxSize,
		minDelay:     50 * time.Millisecond,  // 50ms minimum
		maxDelay:     500 * time.Millisecond, // 500ms maximum
		currentDelay: 100 * time.Millisecond, // Start at 100ms
		buffer:       make(map[uint32]*BufferedPacket),
		lastOutput:   time.Now(),
	}
}

// Add adds a packet to the jitter buffer
func (jb *JitterBuffer) Add(packet *Packet) bool {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	// Check if buffer is full
	if len(jb.buffer) >= jb.maxSize {
		// Drop oldest packet
		jb.dropOldest()
		jb.dropped++
	}

	// Check if packet is too old (already output)
	if packet.Sequence < jb.nextExpected {
		jb.dropped++
		return false
	}

	// Check if duplicate
	if _, exists := jb.buffer[packet.Sequence]; exists {
		return false
	}

	// Add to buffer
	jb.buffer[packet.Sequence] = &BufferedPacket{
		Packet:      packet,
		ArrivalTime: time.Now(),
	}
	jb.totalBuffered++

	return true
}

// Get retrieves the next packet if available and buffering delay has passed
// Returns nil if no packet is ready.
//
// It outputs the lowest buffered sequence at or above nextExpected rather than
// requiring exactly nextExpected. Callers feed it packets whose sequence comes from
// a counter shared with video/text/NACK, so audio sequence numbers are monotonic but
// not contiguous; keying strictly on nextExpected (which started at 0, a sequence
// never produced for audio) meant Get never returned anything and remote audio was
// silent. Skipping to the lowest available sequence also means a genuinely lost
// packet no longer stalls playback forever — the buffer advances past it.
func (jb *JitterBuffer) Get() *Packet {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	// Find the lowest buffered sequence >= nextExpected.
	seq, ok := jb.lowestSeq()
	if !ok {
		jb.underruns++
		return nil
	}
	buffered := jb.buffer[seq]

	// Check if buffering delay has passed
	delay := time.Since(buffered.ArrivalTime)
	if delay < jb.currentDelay {
		// Not ready yet
		return nil
	}

	// Remove from buffer and advance past this sequence
	delete(jb.buffer, seq)
	jb.nextExpected = seq + 1
	jb.lastOutput = time.Now()
	jb.totalOutput++

	// Adapt delay based on buffer size
	jb.adaptDelay()

	return buffered.Packet
}

// Peek returns the next sequence number that Get would output, without removing it.
func (jb *JitterBuffer) Peek() (uint32, bool) {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	return jb.lowestSeq()
}

// lowestSeq returns the smallest buffered sequence at or above nextExpected. Caller
// must hold jb.mu.
func (jb *JitterBuffer) lowestSeq() (uint32, bool) {
	found := false
	var min uint32
	for seq := range jb.buffer {
		if seq < jb.nextExpected {
			continue
		}
		if !found || seq < min {
			min, found = seq, true
		}
	}
	return min, found
}

// Size returns current buffer size
func (jb *JitterBuffer) Size() int {
	jb.mu.Lock()
	defer jb.mu.Unlock()
	return len(jb.buffer)
}

// Reset clears the buffer
func (jb *JitterBuffer) Reset() {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	jb.buffer = make(map[uint32]*BufferedPacket)
	jb.nextExpected = 0
	jb.lastOutput = time.Now()
}

// GetStatistics returns jitter buffer statistics
func (jb *JitterBuffer) GetStatistics() JitterStatistics {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	return JitterStatistics{
		BufferSize:    uint32(len(jb.buffer)),
		TotalBuffered: jb.totalBuffered,
		TotalOutput:   jb.totalOutput,
		Dropped:       jb.dropped,
		Underruns:     jb.underruns,
		CurrentDelay:  jb.currentDelay,
	}
}

// adaptDelay adjusts buffering delay based on buffer utilization
func (jb *JitterBuffer) adaptDelay() {
	bufferRatio := float64(len(jb.buffer)) / float64(jb.maxSize)

	// Increase delay if buffer is filling up (> 70%)
	if bufferRatio > 0.7 {
		jb.currentDelay += 10 * time.Millisecond
		if jb.currentDelay > jb.maxDelay {
			jb.currentDelay = jb.maxDelay
		}
	}

	// Decrease delay if buffer is mostly empty (< 30%)
	if bufferRatio < 0.3 {
		jb.currentDelay -= 10 * time.Millisecond
		if jb.currentDelay < jb.minDelay {
			jb.currentDelay = jb.minDelay
		}
	}
}

// dropOldest removes the oldest packet from buffer
func (jb *JitterBuffer) dropOldest() {
	var oldestSeq uint32
	var oldestTime time.Time
	first := true

	for seq, buffered := range jb.buffer {
		if first || buffered.ArrivalTime.Before(oldestTime) {
			oldestSeq = seq
			oldestTime = buffered.ArrivalTime
			first = false
		}
	}

	if !first {
		delete(jb.buffer, oldestSeq)
	}
}

// JitterStatistics holds jitter buffer statistics
type JitterStatistics struct {
	BufferSize    uint32        // Current number of packets in buffer
	TotalBuffered uint64        // Total packets buffered
	TotalOutput   uint64        // Total packets output
	Dropped       uint64        // Packets dropped due to full buffer
	Underruns     uint64        // Times buffer was empty when read
	CurrentDelay  time.Duration // Current adaptive delay
}
