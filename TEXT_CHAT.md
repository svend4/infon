# Text Chat

TVCP supports text messaging during calls and as a standalone feature.

## Features

- **💬 Real-time messaging**: Send and receive text messages
- **📝 Simple interface**: Command-line text chat
- **🔒 P2P messaging**: Direct peer-to-peer communication
- **⏰ Timestamps**: All messages include timestamps
- **👤 User identification**: Shows sender name with each message

## Usage

### Standalone Chat Mode

Start a text-only chat session:

```bash
# Chat with someone
tvcp chat localhost:5000

# Chat over Yggdrasil
tvcp chat [200:abc:def::1]:5000
```

**Interface:**
- Type your message and press Enter to send
- Messages appear with timestamp and sender
- Press Ctrl+C to exit

**Example session:**
```
💬 TVCP Text Chat
Remote: localhost:5000
Local port: 5000

Type your messages and press Enter to send.
Press Ctrl+C to exit.

> Hello!
💬 [14:32:15] hostname: Hello!
> How are you?
💬 [14:32:18] Remote: I'm good, thanks!
```

### Chat During Video Calls

Text messages can be sent and received during video calls:

```bash
# Make a call (chat is automatically enabled)
tvcp call alice
```

**During a call:**
- Incoming text messages appear at the bottom of the screen
- Format: `💬 [HH:MM:SS] Sender: Message`
- Messages don't interrupt video/audio streaming

**Note:** Interactive chat input during video calls is currently not supported. Use the standalone `chat` command for two-way text communication.

## Message Format

Messages use a binary packet format:

```
┌────────────────┬──────────────┬────────────┬───────────────┬──────────┐
│ Timestamp (8)  │ Sender Len   │ Sender     │ Message Len   │ Message  │
│ milliseconds   │ (2)          │ (UTF-8)    │ (2)           │ (UTF-8)  │
└────────────────┴──────────────┴────────────┴───────────────┴──────────┘
```

**Fields:**
- **Timestamp**: 8 bytes (uint64) - Unix timestamp in milliseconds
- **Sender Length**: 2 bytes (uint16) - Length of sender name
- **Sender**: Variable (UTF-8 string) - Sender identifier (max 255 bytes)
- **Message Length**: 2 bytes (uint16) - Length of message
- **Message**: Variable (UTF-8 string) - The text message (max 1024 bytes)

## Limitations

**Message Size:**
- Maximum sender name: 255 bytes
- Maximum message length: 1024 bytes (UTF-8)
- Total packet size must fit in UDP MTU (~1400 bytes)

**Features Not Yet Implemented:**
- Interactive chat input during video calls
- Message history/logging
- Typing indicators
- Read receipts
- Group chat
- File attachments
- Emoji rendering (emoji characters work but display depends on terminal)

## Technical Details

### Packet Type

Text messages use packet type `0x06` (`PacketTypeTextChat`)

### Encoding

```go
// Create a text message
msg := network.NewTextMessage("Alice", "Hello world!")

// Encode to bytes
data, err := network.EncodeTextMessage(msg)

// Send as packet
packet := &network.Packet{
    Type:      network.PacketTypeTextChat,
    Sequence:  transport.NextSequence(),
    Timestamp: uint64(time.Now().UnixMilli()),
    Payload:   data,
}
```

### Decoding

```go
// Receive packet
if packet.Type == network.PacketTypeTextChat {
    msg, err := network.DecodeTextMessage(packet.Payload)
    if err != nil {
        // Handle error
    }

    fmt.Printf("%s: %s\n", msg.Sender, msg.Message)
}
```

## Use Cases

### 1. DevOps Communication

```bash
# Quick text message to teammate
tvcp chat ops-server:5000
> Deploy completed successfully
```

### 2. Low-Bandwidth Messaging

Text chat uses minimal bandwidth:
- Average message: ~50-200 bytes
- Bandwidth: < 1 KB/s (compared to 382 KB/s for video+audio)
- Perfect for satellite/cellular connections

### 3. Call Coordination

```bash
# Send message during video call
# Messages appear without interrupting video
💬 [14:35:42] Alice: Can you see my screen?
💬 [14:35:45] Bob: Yes, looks good!
```

## Examples

### Two-Person Chat

**Terminal 1:**
```bash
tvcp chat localhost:5001
> Hi from terminal 1!
💬 [14:40:12] localhost: Hi from terminal 2!
```

**Terminal 2:**
```bash
tvcp chat localhost:5000
💬 [14:40:09] localhost: Hi from terminal 1!
> Hi from terminal 2!
```

### Chat Over Yggdrasil P2P

```bash
# Add contact
tvcp contacts add alice [200:1234:5678::1]

# Start chat
tvcp chat [200:1234:5678::1]:5000
> Hey Alice, are you there?
💬 [14:45:22] alice-laptop: Yes, what's up?
```

## Future Enhancements

- [ ] **Interactive chat during calls**: Type messages while video is running
- [ ] **Message history**: Save and view previous messages
- [ ] **Chat logs**: Optional logging to file
- [ ] **Multi-party chat**: Group messaging
- [ ] **Delivery confirmation**: Know when message is received
- [ ] **Encryption indicators**: Show when messages are encrypted
- [ ] **Rich text**: Markdown formatting support
- [ ] **File sharing**: Send small files via chat

## Troubleshooting

### Messages Not Received

**Symptom:**
```
> Hello?
(No response)
```

**Solutions:**
- Check both parties are listening on same port (default 5000)
- Verify firewall allows UDP traffic
- Confirm correct IP address/Yggdrasil address
- Test with `ping` first to verify connectivity

### Garbled Messages

**Symptom:**
```
💬 [14:50:00] ???: �����
```

**Solution:**
- Ensure terminal supports UTF-8
- Check `echo $LANG` shows UTF-8 (e.g., `en_US.UTF-8`)
- Both sender and receiver must use compatible character encodings

### Port Already in Use

**Symptom:**
```
Error creating transport: listen udp :5000: bind: address already in use
```

**Solution:**
```bash
# Use different port
tvcp chat localhost:5001

# Or find what's using port 5000
lsof -i :5000
```

## Comparison with Other Protocols

| Feature | TVCP Chat | IRC | XMPP | WhatsApp |
|---------|-----------|-----|------|----------|
| P2P | ✓ | ✗ | ✗ | ✗ |
| No server | ✓ | ✗ | ✗ | ✗ |
| Terminal-based | ✓ | ✓ | ✓ | ✗ |
| Video integration | ✓ | ✗ | Separate | ✓ |
| Bandwidth | Minimal | Low | Medium | Medium |

## Integration with Other TVCP Features

Text chat works seamlessly with:
- **Video calls**: Messages displayed during calls
- **Yggdrasil P2P**: Chat over mesh network
- **Contacts**: Use contact names instead of addresses
- **Network transport**: Same UDP transport as video/audio
