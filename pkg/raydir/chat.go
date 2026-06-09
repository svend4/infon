package raydir

// ChatLog is a small ring buffer of recent chat lines shown alongside the shared
// world, so walkers can talk while they explore together. (Voice uses the same
// relay path with audio packets; this is the text channel.)
type ChatLog struct {
	lines []string
	max   int
}

// NewChatLog returns a chat log that keeps the last max lines (default 6).
func NewChatLog(max int) *ChatLog {
	if max < 1 {
		max = 6
	}
	return &ChatLog{max: max}
}

// Add appends "sender: message", dropping the oldest line past the cap.
func (c *ChatLog) Add(sender, message string) {
	c.lines = append(c.lines, sender+": "+message)
	if len(c.lines) > c.max {
		c.lines = append([]string(nil), c.lines[len(c.lines)-c.max:]...)
	}
}

// Lines returns the buffered lines, oldest first.
func (c *ChatLog) Lines() []string { return append([]string(nil), c.lines...) }
