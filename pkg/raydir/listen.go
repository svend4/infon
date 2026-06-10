package raydir

import "strings"

// listen.go lets the shared world respond to what people say. The director's prompt
// for the next region is taken from the most recent thing said, so the world
// reshapes around the conversation — "make it a forest at night" and the regions
// ahead become exactly that. It works offline (the reference author reacts to
// keywords) and with a live model (which reads the prompt).

// DirectorPrompt picks the prompt for the next authored region: the most recent
// chat line if it is a usable wish, otherwise the fallback (the director's own
// rotation). Trivial/empty lines (greetings like "hi") fall through to the
// fallback so idle chatter doesn't reshape the world.
func DirectorPrompt(latest, fallback string) string {
	s := strings.TrimSpace(latest)
	if len([]rune(s)) < 4 {
		return fallback
	}
	return s
}
