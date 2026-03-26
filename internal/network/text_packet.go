package network

import (
	"encoding/binary"
	"fmt"
	"time"
)

// TextMessage represents a text chat message
type TextMessage struct {
	Timestamp uint64 // Unix timestamp in milliseconds
	Sender    string // Sender identifier (optional)
	Message   string // The actual text message
}

// EncodeTextMessage encodes a text message to bytes
func EncodeTextMessage(msg *TextMessage) ([]byte, error) {
	// Message format:
	// - Timestamp: 8 bytes (uint64)
	// - Sender length: 2 bytes (uint16)
	// - Sender: N bytes (UTF-8 string)
	// - Message length: 2 bytes (uint16)
	// - Message: M bytes (UTF-8 string)

	senderBytes := []byte(msg.Sender)
	messageBytes := []byte(msg.Message)

	if len(senderBytes) > 255 {
		return nil, fmt.Errorf("sender name too long: %d bytes (max 255)", len(senderBytes))
	}

	if len(messageBytes) > 1024 {
		return nil, fmt.Errorf("message too long: %d bytes (max 1024)", len(messageBytes))
	}

	totalSize := 8 + 2 + len(senderBytes) + 2 + len(messageBytes)
	if totalSize > MaxPacketSize-PacketHeaderSize {
		return nil, fmt.Errorf("encoded message too large: %d bytes", totalSize)
	}

	buf := make([]byte, totalSize)

	// Write timestamp
	binary.BigEndian.PutUint64(buf[0:8], msg.Timestamp)

	// Write sender
	binary.BigEndian.PutUint16(buf[8:10], uint16(len(senderBytes)))
	copy(buf[10:10+len(senderBytes)], senderBytes)

	// Write message
	offset := 10 + len(senderBytes)
	binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(len(messageBytes)))
	copy(buf[offset+2:], messageBytes)

	return buf, nil
}

// DecodeTextMessage decodes bytes into a text message
func DecodeTextMessage(data []byte) (*TextMessage, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("text message too short: %d bytes", len(data))
	}

	msg := &TextMessage{}

	// Read timestamp
	msg.Timestamp = binary.BigEndian.Uint64(data[0:8])

	// Read sender
	senderLen := binary.BigEndian.Uint16(data[8:10])
	if len(data) < 10+int(senderLen)+2 {
		return nil, fmt.Errorf("invalid sender length: %d", senderLen)
	}

	msg.Sender = string(data[10 : 10+senderLen])

	// Read message
	offset := 10 + int(senderLen)
	messageLen := binary.BigEndian.Uint16(data[offset : offset+2])
	if len(data) < offset+2+int(messageLen) {
		return nil, fmt.Errorf("invalid message length: %d", messageLen)
	}

	msg.Message = string(data[offset+2 : offset+2+int(messageLen)])

	return msg, nil
}

// NewTextMessage creates a new text message with current timestamp
func NewTextMessage(sender, message string) *TextMessage {
	return &TextMessage{
		Timestamp: uint64(time.Now().UnixMilli()),
		Sender:    sender,
		Message:   message,
	}
}

// GetTime returns the message timestamp as time.Time
func (m *TextMessage) GetTime() time.Time {
	return time.UnixMilli(int64(m.Timestamp))
}

// FormatTime returns a formatted timestamp string
func (m *TextMessage) FormatTime() string {
	return m.GetTime().Format("15:04:05")
}
