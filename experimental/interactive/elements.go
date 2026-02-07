//go:build experimental

package interactive

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Poll represents a poll/survey
type Poll struct {
	mu sync.RWMutex

	ID          string
	Question    string
	Options     []string
	Votes       map[string]int    // option index -> vote count
	Voters      map[string]int    // voter ID -> option index
	CreatedBy   string
	CreatedAt   time.Time
	ClosedAt    time.Time
	Anonymous   bool
	Multiple    bool              // Allow multiple choice
	MaxChoices  int               // Max choices if multiple
	Active      bool
}

// NewPoll creates a new poll
func NewPoll(id, question string, options []string, createdBy string) *Poll {
	return &Poll{
		ID:        id,
		Question:  question,
		Options:   options,
		Votes:     make(map[string]int),
		Voters:    make(map[string]int),
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		Anonymous: false,
		Multiple:  false,
		MaxChoices: 1,
		Active:    true,
	}
}

// Vote registers a vote
func (p *Poll) Vote(voterID string, optionIndex int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Active {
		return errors.New("poll is closed")
	}

	if optionIndex < 0 || optionIndex >= len(p.Options) {
		return errors.New("invalid option index")
	}

	if !p.Multiple {
		// Single choice - remove previous vote if exists
		if prevIndex, voted := p.Voters[voterID]; voted {
			p.Votes[p.Options[prevIndex]]--
		}
		p.Voters[voterID] = optionIndex
		p.Votes[p.Options[optionIndex]]++
	} else {
		// Multiple choice - check max choices
		currentChoices := 0
		for _, idx := range p.Voters {
			if idx == optionIndex {
				currentChoices++
			}
		}

		if currentChoices >= p.MaxChoices {
			return errors.New("maximum choices reached")
		}

		p.Voters[voterID] = optionIndex
		p.Votes[p.Options[optionIndex]]++
	}

	return nil
}

// Close closes the poll
func (p *Poll) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Active = false
	p.ClosedAt = time.Now()
}

// GetResults returns poll results
func (p *Poll) GetResults() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	results := make(map[string]int)
	for option, count := range p.Votes {
		results[option] = count
	}

	return results
}

// GetTotalVotes returns total number of votes
func (p *Poll) GetTotalVotes() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total := 0
	for _, count := range p.Votes {
		total += count
	}

	return total
}

// FormatResults formats poll results as a string
func (p *Poll) FormatResults() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📊 Poll: %s\n", p.Question))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	totalVotes := 0
	for _, count := range p.Votes {
		totalVotes += count
	}

	for i, option := range p.Options {
		votes := p.Votes[option]
		percentage := 0.0
		if totalVotes > 0 {
			percentage = float64(votes) / float64(totalVotes) * 100
		}

		// Progress bar
		barWidth := 20
		filled := int(percentage / 5) // 5% per character
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, option))
		sb.WriteString(fmt.Sprintf("   [%s] %.1f%% (%d votes)\n", bar, percentage, votes))
	}

	sb.WriteString(fmt.Sprintf("\nTotal votes: %d\n", totalVotes))

	if !p.Active {
		sb.WriteString("Status: Closed\n")
	} else {
		sb.WriteString("Status: Active\n")
	}

	return sb.String()
}

// Reaction represents an emoji reaction
type Reaction struct {
	mu sync.RWMutex

	ID        string
	Type      string    // Emoji or text
	Sender    string
	Timestamp time.Time
	Position  *Position // Optional position on screen
	Duration  time.Duration
}

// Position represents a screen position for reactions
type Position struct {
	X float64 // 0-1 (percentage)
	Y float64 // 0-1 (percentage)
}

// ReactionManager manages reactions in a session
type ReactionManager struct {
	mu sync.RWMutex

	reactions   []*Reaction
	maxReactions int
	ttl         time.Duration // Time to live for reactions
}

// NewReactionManager creates a new reaction manager
func NewReactionManager(maxReactions int, ttl time.Duration) *ReactionManager {
	return &ReactionManager{
		reactions:   make([]*Reaction, 0),
		maxReactions: maxReactions,
		ttl:         ttl,
	}
}

// AddReaction adds a new reaction
func (rm *ReactionManager) AddReaction(reactionType, sender string, position *Position) *Reaction {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	reaction := &Reaction{
		ID:        fmt.Sprintf("reaction-%d", time.Now().UnixNano()),
		Type:      reactionType,
		Sender:    sender,
		Timestamp: time.Now(),
		Position:  position,
		Duration:  rm.ttl,
	}

	rm.reactions = append(rm.reactions, reaction)

	// Cleanup old reactions
	rm.cleanupLocked()

	return reaction
}

// GetActiveReactions returns all active reactions
func (rm *ReactionManager) GetActiveReactions() []*Reaction {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	now := time.Now()
	active := make([]*Reaction, 0)

	for _, r := range rm.reactions {
		if now.Sub(r.Timestamp) < r.Duration {
			active = append(active, r)
		}
	}

	return active
}

// cleanupLocked removes expired reactions (must hold lock)
func (rm *ReactionManager) cleanupLocked() {
	now := time.Now()
	validReactions := make([]*Reaction, 0)

	for _, r := range rm.reactions {
		if now.Sub(r.Timestamp) < r.Duration {
			validReactions = append(validReactions, r)
		}
	}

	// Keep only recent reactions if too many
	if len(validReactions) > rm.maxReactions {
		validReactions = validReactions[len(validReactions)-rm.maxReactions:]
	}

	rm.reactions = validReactions
}

// GetReactionCounts returns counts by type
func (rm *ReactionManager) GetReactionCounts() map[string]int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	counts := make(map[string]int)
	now := time.Now()

	for _, r := range rm.reactions {
		if now.Sub(r.Timestamp) < r.Duration {
			counts[r.Type]++
		}
	}

	return counts
}

// Question represents a Q&A question
type Question struct {
	mu sync.RWMutex

	ID          string
	Text        string
	Asker       string
	AskerName   string
	AskedAt     time.Time
	Upvotes     int
	Upvoters    map[string]bool
	Answered    bool
	AnswerText  string
	AnsweredBy  string
	AnsweredAt  time.Time
}

// NewQuestion creates a new question
func NewQuestion(id, text, asker, askerName string) *Question {
	return &Question{
		ID:        id,
		Text:      text,
		Asker:     asker,
		AskerName: askerName,
		AskedAt:   time.Now(),
		Upvotes:   0,
		Upvoters:  make(map[string]bool),
		Answered:  false,
	}
}

// Upvote upvotes the question
func (q *Question) Upvote(voterID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.Upvoters[voterID] {
		return errors.New("already upvoted")
	}

	q.Upvoters[voterID] = true
	q.Upvotes++

	return nil
}

// RemoveUpvote removes an upvote
func (q *Question) RemoveUpvote(voterID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.Upvoters[voterID] {
		return errors.New("not upvoted")
	}

	delete(q.Upvoters, voterID)
	q.Upvotes--

	return nil
}

// Answer answers the question
func (q *Question) Answer(answer, answeredBy string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.Answered = true
	q.AnswerText = answer
	q.AnsweredBy = answeredBy
	q.AnsweredAt = time.Now()
}

// QAManager manages questions and answers
type QAManager struct {
	mu sync.RWMutex

	questions map[string]*Question
	order     []string // Question IDs in order
}

// NewQAManager creates a new Q&A manager
func NewQAManager() *QAManager {
	return &QAManager{
		questions: make(map[string]*Question),
		order:     make([]string, 0),
	}
}

// SubmitQuestion submits a new question
func (qam *QAManager) SubmitQuestion(id, text, asker, askerName string) *Question {
	qam.mu.Lock()
	defer qam.mu.Unlock()

	question := NewQuestion(id, text, asker, askerName)
	qam.questions[id] = question
	qam.order = append(qam.order, id)

	return question
}

// GetQuestion gets a question by ID
func (qam *QAManager) GetQuestion(id string) (*Question, error) {
	qam.mu.RLock()
	defer qam.mu.RUnlock()

	question, exists := qam.questions[id]
	if !exists {
		return nil, errors.New("question not found")
	}

	return question, nil
}

// GetAllQuestions returns all questions
func (qam *QAManager) GetAllQuestions() []*Question {
	qam.mu.RLock()
	defer qam.mu.RUnlock()

	questions := make([]*Question, 0, len(qam.order))
	for _, id := range qam.order {
		questions = append(questions, qam.questions[id])
	}

	return questions
}

// GetTopQuestions returns top N questions by upvotes
func (qam *QAManager) GetTopQuestions(n int) []*Question {
	qam.mu.RLock()
	defer qam.mu.RUnlock()

	questions := make([]*Question, 0, len(qam.order))
	for _, id := range qam.order {
		questions = append(questions, qam.questions[id])
	}

	// Sort by upvotes (descending)
	for i := 0; i < len(questions); i++ {
		for j := i + 1; j < len(questions); j++ {
			if questions[j].Upvotes > questions[i].Upvotes {
				questions[i], questions[j] = questions[j], questions[i]
			}
		}
	}

	if n > len(questions) {
		n = len(questions)
	}

	return questions[:n]
}

// GetUnansweredQuestions returns all unanswered questions
func (qam *QAManager) GetUnansweredQuestions() []*Question {
	qam.mu.RLock()
	defer qam.mu.RUnlock()

	unanswered := make([]*Question, 0)
	for _, id := range qam.order {
		q := qam.questions[id]
		if !q.Answered {
			unanswered = append(unanswered, q)
		}
	}

	return unanswered
}

// FormatQuestions formats questions list
func (qam *QAManager) FormatQuestions() string {
	qam.mu.RLock()
	defer qam.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("❓ Questions & Answers\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(qam.order) == 0 {
		sb.WriteString("No questions yet.\n")
		return sb.String()
	}

	for i, id := range qam.order {
		q := qam.questions[id]

		sb.WriteString(fmt.Sprintf("%d. ", i+1))

		if q.Answered {
			sb.WriteString("✅ ")
		} else {
			sb.WriteString("❔ ")
		}

		sb.WriteString(fmt.Sprintf("%s\n", q.Text))
		sb.WriteString(fmt.Sprintf("   Asked by: %s | Upvotes: %d\n", q.AskerName, q.Upvotes))

		if q.Answered {
			sb.WriteString(fmt.Sprintf("   💬 Answer: %s\n", q.AnswerText))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// HandRaising represents hand raising functionality
type HandRaising struct {
	mu sync.RWMutex

	raised    map[string]time.Time // participant ID -> time raised
	order     []string             // Order of raised hands
}

// NewHandRaising creates a new hand raising manager
func NewHandRaising() *HandRaising {
	return &HandRaising{
		raised: make(map[string]time.Time),
		order:  make([]string, 0),
	}
}

// RaiseHand raises a hand
func (hr *HandRaising) RaiseHand(participantID string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	if _, raised := hr.raised[participantID]; raised {
		return // Already raised
	}

	hr.raised[participantID] = time.Now()
	hr.order = append(hr.order, participantID)
}

// LowerHand lowers a hand
func (hr *HandRaising) LowerHand(participantID string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	if _, raised := hr.raised[participantID]; !raised {
		return // Not raised
	}

	delete(hr.raised, participantID)

	// Remove from order
	newOrder := make([]string, 0)
	for _, id := range hr.order {
		if id != participantID {
			newOrder = append(newOrder, id)
		}
	}
	hr.order = newOrder
}

// GetRaisedHands returns all participants with raised hands in order
func (hr *HandRaising) GetRaisedHands() []string {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	hands := make([]string, len(hr.order))
	copy(hands, hr.order)

	return hands
}

// GetCount returns the number of raised hands
func (hr *HandRaising) GetCount() int {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	return len(hr.raised)
}

// Clear clears all raised hands
func (hr *HandRaising) Clear() {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	hr.raised = make(map[string]time.Time)
	hr.order = make([]string, 0)
}
