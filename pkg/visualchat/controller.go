package visualchat

import "sync"

// PromptSetter is anything whose generation prompt can be changed at runtime —
// notably *device.NeuralGenerator. Declared here (not imported) so this package
// stays transport- and engine-agnostic and avoids an import cycle.
type PromptSetter interface {
	SetPrompt(string)
}

// State is the current visual-chat generation state derived from applied steers.
type State struct {
	Prompt   string
	Strength float64
	Seed     int64
}

// Controller maintains visual-chat State and pushes prompt changes to a
// PromptSetter as steers arrive. It is safe for concurrent use: steers may come
// from a network goroutine while the render loop reads State.
type Controller struct {
	mu     sync.Mutex
	state  State
	target PromptSetter
}

// NewController creates a controller seeded with an initial prompt, bound to an
// optional target (may be nil to only track State).
func NewController(initialPrompt string, target PromptSetter) *Controller {
	c := &Controller{state: State{Prompt: initialPrompt}, target: target}
	if target != nil {
		target.SetPrompt(initialPrompt)
	}
	return c
}

// Apply folds a steer into the state and, if the prompt changed and a target is
// bound, pushes the new prompt. It returns the resulting State.
func (c *Controller) Apply(s Steer) State {
	c.mu.Lock()
	defer c.mu.Unlock()

	newPrompt := s.Apply(c.state.Prompt)
	changed := newPrompt != c.state.Prompt
	c.state.Prompt = newPrompt
	if s.Strength > 0 {
		if s.Strength > 1 {
			s.Strength = 1
		}
		c.state.Strength = s.Strength
	}
	if s.Seed != 0 {
		c.state.Seed = s.Seed
	}

	if changed && c.target != nil {
		c.target.SetPrompt(newPrompt)
	}
	return c.state
}

// ApplyJSON decodes a steer message and applies it.
func (c *Controller) ApplyJSON(data []byte) (State, error) {
	s, err := DecodeSteer(data)
	if err != nil {
		return c.Snapshot(), err
	}
	return c.Apply(s), nil
}

// Snapshot returns the current state.
func (c *Controller) Snapshot() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}
