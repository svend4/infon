package network

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// NetworkCondition represents simulated network conditions
type NetworkCondition struct {
	Latency       time.Duration // Base latency
	Jitter        time.Duration // Latency variation
	PacketLoss    float64       // Packet loss percentage (0.0-100.0)
	Bandwidth     int           // Bandwidth limit in KB/s (0 = unlimited)
	Corruption    float64       // Packet corruption percentage (0.0-100.0)
	Reordering    float64       // Packet reordering percentage (0.0-100.0)
	Duplication   float64       // Packet duplication percentage (0.0-100.0)
}

// Preset network conditions
var (
	// PerfectNetwork represents ideal network conditions
	PerfectNetwork = NetworkCondition{
		Latency:       0,
		Jitter:        0,
		PacketLoss:    0.0,
		Bandwidth:     0, // Unlimited
		Corruption:    0.0,
		Reordering:    0.0,
		Duplication:   0.0,
	}

	// Good4G represents good 4G mobile network
	Good4G = NetworkCondition{
		Latency:       30 * time.Millisecond,
		Jitter:        10 * time.Millisecond,
		PacketLoss:    0.1,
		Bandwidth:     5000, // 5 MB/s
		Corruption:    0.0,
		Reordering:    0.1,
		Duplication:   0.0,
	}

	// Regular4G represents average 4G network
	Regular4G = NetworkCondition{
		Latency:       50 * time.Millisecond,
		Jitter:        20 * time.Millisecond,
		PacketLoss:    0.5,
		Bandwidth:     2000, // 2 MB/s
		Corruption:    0.0,
		Reordering:    0.2,
		Duplication:   0.0,
	}

	// Poor4G represents poor 4G network
	Poor4G = NetworkCondition{
		Latency:       100 * time.Millisecond,
		Jitter:        50 * time.Millisecond,
		PacketLoss:    2.0,
		Bandwidth:     500, // 500 KB/s
		Corruption:    0.1,
		Reordering:    1.0,
		Duplication:   0.1,
	}

	// Good3G represents good 3G network
	Good3G = NetworkCondition{
		Latency:       100 * time.Millisecond,
		Jitter:        30 * time.Millisecond,
		PacketLoss:    1.0,
		Bandwidth:     400, // 400 KB/s
		Corruption:    0.1,
		Reordering:    0.5,
		Duplication:   0.0,
	}

	// EdgeNetwork represents 2G/EDGE network
	EdgeNetwork = NetworkCondition{
		Latency:       400 * time.Millisecond,
		Jitter:        150 * time.Millisecond,
		PacketLoss:    5.0,
		Bandwidth:     30, // 30 KB/s
		Corruption:    0.5,
		Reordering:    2.0,
		Duplication:   0.2,
	}

	// Satellite represents satellite connection
	Satellite = NetworkCondition{
		Latency:       600 * time.Millisecond,
		Jitter:        200 * time.Millisecond,
		PacketLoss:    3.0,
		Bandwidth:     1000, // 1 MB/s
		Corruption:    0.2,
		Reordering:    1.0,
		Duplication:   0.1,
	}
)

// NetworkSimulator simulates network conditions for testing
type NetworkSimulator struct {
	mu        sync.RWMutex
	enabled   bool
	condition NetworkCondition
	rng       *rand.Rand

	// Bandwidth limiting
	bandwidthBucket  int       // Bytes available
	lastBandwidthUpdate time.Time

	// Statistics
	totalPackets      uint64
	droppedPackets    uint64
	delayedPackets    uint64
	corruptedPackets  uint64
	reorderedPackets  uint64
	duplicatedPackets uint64
}

// NewNetworkSimulator creates a new network simulator
func NewNetworkSimulator() *NetworkSimulator {
	return &NetworkSimulator{
		enabled:   false,
		condition: PerfectNetwork,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
		lastBandwidthUpdate: time.Now(),
	}
}

// Enable enables network simulation with the given condition
func (ns *NetworkSimulator) Enable(condition NetworkCondition) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.enabled = true
	ns.condition = condition
	ns.lastBandwidthUpdate = time.Now()
}

// Disable disables network simulation
func (ns *NetworkSimulator) Disable() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.enabled = false
}

// IsEnabled returns whether simulation is enabled
func (ns *NetworkSimulator) IsEnabled() bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	return ns.enabled
}

// GetCondition returns the current network condition
func (ns *NetworkSimulator) GetCondition() NetworkCondition {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	return ns.condition
}

// ShouldSendPacket determines if a packet should be sent (or dropped)
func (ns *NetworkSimulator) ShouldSendPacket(packetSize int) (shouldSend bool, delay time.Duration) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.totalPackets++

	if !ns.enabled {
		return true, 0
	}

	// Check packet loss
	if ns.rng.Float64()*100.0 < ns.condition.PacketLoss {
		ns.droppedPackets++
		return false, 0
	}

	// Check bandwidth limit
	if ns.condition.Bandwidth > 0 {
		now := time.Now()
		timeDelta := now.Sub(ns.lastBandwidthUpdate)

		// Refill bandwidth bucket
		bytesAllowed := int(float64(ns.condition.Bandwidth) * timeDelta.Seconds() * 1024.0)
		ns.bandwidthBucket += bytesAllowed
		ns.lastBandwidthUpdate = now

		// Cap bucket size
		maxBucket := ns.condition.Bandwidth * 1024 // 1 second worth
		if ns.bandwidthBucket > maxBucket {
			ns.bandwidthBucket = maxBucket
		}

		// Check if we have enough bandwidth
		if ns.bandwidthBucket < packetSize {
			ns.droppedPackets++
			return false, 0
		}

		// Consume bandwidth
		ns.bandwidthBucket -= packetSize
	}

	// Calculate delay (latency + jitter)
	baseDelay := ns.condition.Latency
	var jitter time.Duration
	if ns.condition.Jitter > 0 {
		jitter = time.Duration(ns.rng.Int63n(int64(ns.condition.Jitter)))
	}
	delay = baseDelay + jitter

	if delay > 0 {
		ns.delayedPackets++
	}

	return true, delay
}

// ShouldCorruptPacket determines if a packet should be corrupted
func (ns *NetworkSimulator) ShouldCorruptPacket() bool {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if !ns.enabled {
		return false
	}

	if ns.rng.Float64()*100.0 < ns.condition.Corruption {
		ns.corruptedPackets++
		return true
	}

	return false
}

// ShouldReorderPacket determines if a packet should be reordered
func (ns *NetworkSimulator) ShouldReorderPacket() bool {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if !ns.enabled {
		return false
	}

	if ns.rng.Float64()*100.0 < ns.condition.Reordering {
		ns.reorderedPackets++
		return true
	}

	return false
}

// ShouldDuplicatePacket determines if a packet should be duplicated
func (ns *NetworkSimulator) ShouldDuplicatePacket() bool {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if !ns.enabled {
		return false
	}

	if ns.rng.Float64()*100.0 < ns.condition.Duplication {
		ns.duplicatedPackets++
		return true
	}

	return false
}

// GetStatistics returns simulation statistics
func (ns *NetworkSimulator) GetStatistics() SimulatorStatistics {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	return SimulatorStatistics{
		TotalPackets:      ns.totalPackets,
		DroppedPackets:    ns.droppedPackets,
		DelayedPackets:    ns.delayedPackets,
		CorruptedPackets:  ns.corruptedPackets,
		ReorderedPackets:  ns.reorderedPackets,
		DuplicatedPackets: ns.duplicatedPackets,
		DropRate:          float64(ns.droppedPackets) / float64(ns.totalPackets) * 100.0,
	}
}

// SimulatorStatistics represents statistics about the simulation
type SimulatorStatistics struct {
	TotalPackets      uint64
	DroppedPackets    uint64
	DelayedPackets    uint64
	CorruptedPackets  uint64
	ReorderedPackets  uint64
	DuplicatedPackets uint64
	DropRate          float64 // Percentage
}

// Reset resets simulation statistics
func (ns *NetworkSimulator) Reset() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.totalPackets = 0
	ns.droppedPackets = 0
	ns.delayedPackets = 0
	ns.corruptedPackets = 0
	ns.reorderedPackets = 0
	ns.duplicatedPackets = 0
	ns.bandwidthBucket = 0
	ns.lastBandwidthUpdate = time.Now()
}

// CorruptPacket corrupts a packet by flipping random bits
func (ns *NetworkSimulator) CorruptPacket(data []byte) {
	if len(data) == 0 {
		return
	}

	// Flip 1-5 random bits
	numBits := 1 + ns.rng.Intn(5)
	for i := 0; i < numBits; i++ {
		byteIndex := ns.rng.Intn(len(data))
		bitIndex := ns.rng.Intn(8)
		data[byteIndex] ^= 1 << uint(bitIndex)
	}
}

// GetPresetName returns the name of a preset condition
func GetPresetName(condition NetworkCondition) string {
	switch condition {
	case PerfectNetwork:
		return "Perfect"
	case Good4G:
		return "Good 4G"
	case Regular4G:
		return "Regular 4G"
	case Poor4G:
		return "Poor 4G"
	case Good3G:
		return "Good 3G"
	case EdgeNetwork:
		return "EDGE/2G"
	case Satellite:
		return "Satellite"
	default:
		return "Custom"
	}
}

// FormatCondition formats a network condition as a human-readable string
func FormatCondition(c NetworkCondition) string {
	bwStr := "unlimited"
	if c.Bandwidth > 0 {
		bwStr = fmt.Sprintf("%d KB/s", c.Bandwidth)
	}

	return fmt.Sprintf("Latency: %v, Jitter: %v, Loss: %.1f%%, Bandwidth: %s, Corruption: %.1f%%, Reorder: %.1f%%, Dup: %.1f%%",
		c.Latency, c.Jitter, c.PacketLoss, bwStr, c.Corruption, c.Reordering, c.Duplication)
}
