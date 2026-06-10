package raydir

// branch.go lets the walk fork. At a crossroads the world offers two ways — a high
// road or a low one, press on into the night or make camp — and which you take
// steers what the director builds next. The choice is the most direct kind there
// is: you simply walk left or right. Branching is a small, pure state machine (the
// set of forks, where you are in it, and the path you've taken), so it runs
// offline and is fully testable; the chosen branch's prompt feeds the same
// director.

// Choice is one arm of a fork: a short label and the director prompt it leads to.
type Choice struct {
	Label  string
	Prompt string
}

// Fork is a crossroads: a question and two ways to answer it.
type Fork struct {
	Question string
	Left     Choice
	Right    Choice
}

// Branching walks an ordered set of forks, remembering the path taken and the
// prompt of the branch currently in effect.
type Branching struct {
	Forks []Fork
	cur   int      // index of the next fork to present
	seed  string   // the active prompt (from the last branch taken)
	taken []string // labels chosen, in order
}

// NewBranching builds a branching from a list of forks.
func NewBranching(forks []Fork) *Branching { return &Branching{Forks: forks} }

// DefaultBranching is a built-in set of three thematic crossroads.
func DefaultBranching() *Branching {
	return NewBranching([]Fork{
		{Question: "The path splits.",
			Left:  Choice{Label: "High Road", Prompt: "a mountain pass, vast and open and grand"},
			Right: Choice{Label: "Low Road", Prompt: "a forest valley, quiet and green with trees"}},
		{Question: "The day wanes.",
			Left:  Choice{Label: "Press On", Prompt: "onward into the night, dark with fireflies"},
			Right: Choice{Label: "Make Camp", Prompt: "a camp by water, calm and still"}},
		{Question: "A door, and a stair.",
			Left:  Choice{Label: "Descend", Prompt: "down into a glowing crystal cave"},
			Right: Choice{Label: "Ascend", Prompt: "up to a golden summit, grand and bright"}},
	})
}

// PendingFork returns the fork awaiting a choice, or ok=false when the forks are
// exhausted.
func (b *Branching) PendingFork() (Fork, bool) {
	if b.cur >= len(b.Forks) {
		return Fork{}, false
	}
	return b.Forks[b.cur], true
}

// Choose takes the right arm (goRight) or the left at the current fork, makes its
// prompt the one in effect, records the path, and advances. ok=false if there is
// no fork left to choose.
func (b *Branching) Choose(goRight bool) (Choice, bool) {
	if b.cur >= len(b.Forks) {
		return Choice{}, false
	}
	ch := b.Forks[b.cur].Left
	if goRight {
		ch = b.Forks[b.cur].Right
	}
	b.seed = ch.Prompt
	b.taken = append(b.taken, ch.Label)
	b.cur++
	return ch, true
}

// Prompt is the director seed currently in effect (the last branch taken), or a
// neutral crossroads prompt before any choice.
func (b *Branching) Prompt() string {
	if b.seed == "" {
		return "a crossroads of paths"
	}
	return b.seed
}

// Path is the labels of the branches taken, in order.
func (b *Branching) Path() []string { return append([]string(nil), b.taken...) }

// Done reports whether every fork has been chosen.
func (b *Branching) Done() bool { return b.cur >= len(b.Forks) }
