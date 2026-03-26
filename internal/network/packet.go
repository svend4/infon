package network

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Packet types
const (
	PacketTypeFrame     uint8 = 0x01 // Video frame data
	PacketTypeAudio     uint8 = 0x02 // Audio data
	PacketTypeControl   uint8 = 0x03 // Control message
	PacketTypeHandshake uint8 = 0x04 // Connection handshake
	PacketTypeKeepAlive uint8 = 0x05 // Keep-alive ping
	PacketTypeTextChat  uint8 = 0x06 // Text chat message
	PacketTypeScreen    uint8 = 0x07 // Screen sharing (terminal output)
)

// Packet represents a network packet
type Packet struct {
	Type      uint8  // Packet type
	Sequence  uint32 // Sequence number
	Timestamp uint64 // Timestamp in milliseconds
	Payload   []byte // Packet payload
}

var (
	ErrInvalidPacket = errors.New("invalid packet format")
	ErrPacketTooLarge = errors.New("packet exceeds maximum size")
)

const (
	// MaxPacketSize is the maximum UDP packet size (MTU - headers)
	MaxPacketSize = 1400 // bytes, safe for most networks

	// PacketHeaderSize is the size of packet header
	PacketHeaderSize = 1 + 4 + 8 // type(1) + sequence(4) + timestamp(8) = 13 bytes
)

// Encode serializes the packet to bytes
func (p *Packet) Encode() ([]byte, error) {
	totalSize := PacketHeaderSize + len(p.Payload)

	if totalSize > MaxPacketSize {
		return nil, ErrPacketTooLarge
	}

	buf := make([]byte, totalSize)

	// Write header
	buf[0] = p.Type
	binary.BigEndian.PutUint32(buf[1:5], p.Sequence)
	binary.BigEndian.PutUint64(buf[5:13], p.Timestamp)

	// Write payload
	copy(buf[13:], p.Payload)

	return buf, nil
}

// Decode deserializes bytes into a packet
func Decode(data []byte) (*Packet, error) {
	if len(data) < PacketHeaderSize {
		return nil, ErrInvalidPacket
	}

	p := &Packet{
		Type:      data[0],
		Sequence:  binary.BigEndian.Uint32(data[1:5]),
		Timestamp: binary.BigEndian.Uint64(data[5:13]),
		Payload:   make([]byte, len(data)-PacketHeaderSize),
	}

	copy(p.Payload, data[13:])

	return p, nil
}

// String returns a human-readable representation
func (p *Packet) String() string {
	return fmt.Sprintf("Packet{Type:%d, Seq:%d, TS:%d, Size:%d}",
		p.Type, p.Sequence, p.Timestamp, len(p.Payload))
}
