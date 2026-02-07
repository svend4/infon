//go:build experimental

package cards

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// UnoGame represents a UNO card game
type UnoGame struct {
	mu sync.RWMutex

	ID           string
	Players      []*UnoPlayer
	Deck         *UnoDeck
	DiscardPile  []*UnoCard
	CurrentPlayer int
	Direction    int // 1 = clockwise, -1 = counterclockwise
	State        UnoGameState
	CurrentColor UnoColor
	DrawStack    int // Stacking Draw2/Draw4 cards
	LastAction   string
	StartTime    time.Time
	Winner       *UnoPlayer

	OnCardPlayed func(player *UnoPlayer, card *UnoCard)
	OnPlayerDrew func(player *UnoPlayer, count int)
	OnGameEnd    func(winner *UnoPlayer)
}

// UnoGameState represents game state
type UnoGameState int

const (
	UnoStateSetup UnoGameState = iota
	UnoStatePlaying
	UnoStateFinished
)

// UnoColor represents card colors
type UnoColor int

const (
	ColorRed UnoColor = iota
	ColorBlue
	ColorGreen
	ColorYellow
	ColorWild // For wild cards
)

func (c UnoColor) String() string {
	switch c {
	case ColorRed:
		return "🔴 Red"
	case ColorBlue:
		return "🔵 Blue"
	case ColorGreen:
		return "🟢 Green"
	case ColorYellow:
		return "🟡 Yellow"
	case ColorWild:
		return "⚫ Wild"
	default:
		return "Unknown"
	}
}

// UnoValue represents card values
type UnoValue int

const (
	Value0 UnoValue = iota
	Value1
	Value2
	Value3
	Value4
	Value5
	Value6
	Value7
	Value8
	Value9
	ValueSkip
	ValueReverse
	ValueDraw2
	ValueWild
	ValueWildDraw4
)

func (v UnoValue) String() string {
	switch v {
	case Value0, Value1, Value2, Value3, Value4, Value5, Value6, Value7, Value8, Value9:
		return fmt.Sprintf("%d", int(v))
	case ValueSkip:
		return "Skip"
	case ValueReverse:
		return "Reverse"
	case ValueDraw2:
		return "Draw +2"
	case ValueWild:
		return "Wild"
	case ValueWildDraw4:
		return "Wild +4"
	default:
		return "Unknown"
	}
}

// UnoCard represents a UNO card
type UnoCard struct {
	Color UnoColor
	Value UnoValue
}

// String returns card representation
func (c *UnoCard) String() string {
	if c.Value == ValueWild || c.Value == ValueWildDraw4 {
		return fmt.Sprintf("%s %s", c.Color.String(), c.Value.String())
	}
	return fmt.Sprintf("%s %s", c.Color.String(), c.Value.String())
}

// UnoPlayer represents a player
type UnoPlayer struct {
	ID       string
	Name     string
	Hand     []*UnoCard
	Score    int
	SaidUno  bool
	Position int
}

// UnoDeck represents the card deck
type UnoDeck struct {
	Cards []*UnoCard
}

// NewUnoDeck creates a standard UNO deck
func NewUnoDeck() *UnoDeck {
	deck := &UnoDeck{
		Cards: make([]*UnoCard, 0, 108),
	}

	// Add number cards (0-9) for each color
	colors := []UnoColor{ColorRed, ColorBlue, ColorGreen, ColorYellow}
	for _, color := range colors {
		// One 0 per color
		deck.Cards = append(deck.Cards, &UnoCard{Color: color, Value: Value0})

		// Two of each 1-9 per color
		for value := Value1; value <= Value9; value++ {
			deck.Cards = append(deck.Cards, &UnoCard{Color: color, Value: value})
			deck.Cards = append(deck.Cards, &UnoCard{Color: color, Value: value})
		}

		// Two Skip per color
		deck.Cards = append(deck.Cards, &UnoCard{Color: color, Value: ValueSkip})
		deck.Cards = append(deck.Cards, &UnoCard{Color: color, Value: ValueSkip})

		// Two Reverse per color
		deck.Cards = append(deck.Cards, &UnoCard{Color: color, Value: ValueReverse})
		deck.Cards = append(deck.Cards, &UnoCard{Color: color, Value: ValueReverse})

		// Two Draw2 per color
		deck.Cards = append(deck.Cards, &UnoCard{Color: color, Value: ValueDraw2})
		deck.Cards = append(deck.Cards, &UnoCard{Color: color, Value: ValueDraw2})
	}

	// Add Wild cards (4 Wild, 4 Wild Draw4)
	for i := 0; i < 4; i++ {
		deck.Cards = append(deck.Cards, &UnoCard{Color: ColorWild, Value: ValueWild})
		deck.Cards = append(deck.Cards, &UnoCard{Color: ColorWild, Value: ValueWildDraw4})
	}

	return deck
}

// Shuffle shuffles the deck
func (d *UnoDeck) Shuffle() {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

// Draw draws n cards from the deck
func (d *UnoDeck) Draw(n int) []*UnoCard {
	if n > len(d.Cards) {
		n = len(d.Cards)
	}

	drawn := d.Cards[:n]
	d.Cards = d.Cards[n:]

	return drawn
}

// NewUnoGame creates a new UNO game
func NewUnoGame(id string) *UnoGame {
	return &UnoGame{
		ID:           id,
		Players:      make([]*UnoPlayer, 0),
		Deck:         NewUnoDeck(),
		DiscardPile:  make([]*UnoCard, 0),
		Direction:    1,
		State:        UnoStateSetup,
		DrawStack:    0,
	}
}

// AddPlayer adds a player to the game
func (g *UnoGame) AddPlayer(id, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != UnoStateSetup {
		return errors.New("game already started")
	}

	if len(g.Players) >= 10 {
		return errors.New("maximum 10 players")
	}

	player := &UnoPlayer{
		ID:       id,
		Name:     name,
		Hand:     make([]*UnoCard, 0),
		Position: len(g.Players),
		SaidUno:  false,
	}

	g.Players = append(g.Players, player)

	return nil
}

// Start starts the game
func (g *UnoGame) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != UnoStateSetup {
		return errors.New("game already started")
	}

	if len(g.Players) < 2 {
		return errors.New("need at least 2 players")
	}

	// Shuffle deck
	g.Deck.Shuffle()

	// Deal 7 cards to each player
	for _, player := range g.Players {
		player.Hand = g.Deck.Draw(7)
	}

	// Draw first card for discard pile
	firstCard := g.Deck.Draw(1)[0]
	g.DiscardPile = append(g.DiscardPile, firstCard)
	g.CurrentColor = firstCard.Color

	// Handle special first card
	if firstCard.Value == ValueWild || firstCard.Value == ValueWildDraw4 {
		// Pick random color
		colors := []UnoColor{ColorRed, ColorBlue, ColorGreen, ColorYellow}
		g.CurrentColor = colors[rand.Intn(4)]
	}

	g.State = UnoStatePlaying
	g.CurrentPlayer = 0
	g.StartTime = time.Now()

	return nil
}

// PlayCard plays a card from player's hand
func (g *UnoGame) PlayCard(playerID string, cardIndex int, chosenColor UnoColor) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != UnoStatePlaying {
		return errors.New("game not in playing state")
	}

	player := g.Players[g.CurrentPlayer]
	if player.ID != playerID {
		return errors.New("not your turn")
	}

	if cardIndex < 0 || cardIndex >= len(player.Hand) {
		return errors.New("invalid card index")
	}

	card := player.Hand[cardIndex]

	// Validate play
	topCard := g.DiscardPile[len(g.DiscardPile)-1]

	if !g.canPlayCard(card, topCard) {
		return errors.New("cannot play this card")
	}

	// Remove card from hand
	player.Hand = append(player.Hand[:cardIndex], player.Hand[cardIndex+1:]...)

	// Add to discard pile
	g.DiscardPile = append(g.DiscardPile, card)

	// Update current color
	if card.Value == ValueWild || card.Value == ValueWildDraw4 {
		g.CurrentColor = chosenColor
	} else {
		g.CurrentColor = card.Color
	}

	// Handle special cards
	g.handleSpecialCard(card)

	// Check if player won
	if len(player.Hand) == 0 {
		g.State = UnoStateFinished
		g.Winner = player
		g.calculateScores()

		if g.OnGameEnd != nil {
			go g.OnGameEnd(player)
		}

		return nil
	}

	// Check UNO call
	if len(player.Hand) == 1 {
		player.SaidUno = false // Reset for next turn
	}

	// Move to next player
	g.nextPlayer()

	if g.OnCardPlayed != nil {
		go g.OnCardPlayed(player, card)
	}

	return nil
}

// DrawCard draws a card for the current player
func (g *UnoGame) DrawCard(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.State != UnoStatePlaying {
		return errors.New("game not in playing state")
	}

	player := g.Players[g.CurrentPlayer]
	if player.ID != playerID {
		return errors.New("not your turn")
	}

	// Check if need to draw from stack
	drawCount := 1
	if g.DrawStack > 0 {
		drawCount = g.DrawStack
		g.DrawStack = 0
	}

	// Draw cards
	drawn := g.drawCardsFromDeck(drawCount)
	player.Hand = append(player.Hand, drawn...)

	if g.OnPlayerDrew != nil {
		go g.OnPlayerDrew(player, drawCount)
	}

	// Move to next player
	g.nextPlayer()

	return nil
}

// SayUno declares UNO (when player has one card left)
func (g *UnoGame) SayUno(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	player := g.findPlayer(playerID)
	if player == nil {
		return errors.New("player not found")
	}

	if len(player.Hand) != 1 {
		return errors.New("can only say UNO with 1 card")
	}

	player.SaidUno = true

	return nil
}

// ChallengeUno challenges a player who didn't say UNO
func (g *UnoGame) ChallengeUno(challengerID, targetID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	target := g.findPlayer(targetID)
	if target == nil {
		return errors.New("target player not found")
	}

	if len(target.Hand) != 1 {
		return errors.New("target doesn't have 1 card")
	}

	if target.SaidUno {
		return errors.New("target already said UNO")
	}

	// Penalty: draw 2 cards
	drawn := g.drawCardsFromDeck(2)
	target.Hand = append(target.Hand, drawn...)

	return nil
}

// canPlayCard checks if a card can be played
func (g *UnoGame) canPlayCard(card, topCard *UnoCard) bool {
	// Wild cards can always be played
	if card.Value == ValueWild || card.Value == ValueWildDraw4 {
		return true
	}

	// Match color or value
	if card.Color == g.CurrentColor || card.Value == topCard.Value {
		return true
	}

	return false
}

// handleSpecialCard handles special card effects
func (g *UnoGame) handleSpecialCard(card *UnoCard) {
	switch card.Value {
	case ValueSkip:
		g.nextPlayer() // Skip next player

	case ValueReverse:
		g.Direction *= -1
		if len(g.Players) == 2 {
			// In 2-player, reverse acts like skip
			g.nextPlayer()
		}

	case ValueDraw2:
		g.DrawStack += 2

	case ValueWildDraw4:
		g.DrawStack += 4
	}
}

// nextPlayer moves to the next player
func (g *UnoGame) nextPlayer() {
	g.CurrentPlayer = (g.CurrentPlayer + g.Direction + len(g.Players)) % len(g.Players)
}

// drawCardsFromDeck draws cards, reshuffling discard if needed
func (g *UnoGame) drawCardsFromDeck(count int) []*UnoCard {
	drawn := make([]*UnoCard, 0, count)

	for i := 0; i < count; i++ {
		if len(g.Deck.Cards) == 0 {
			// Reshuffle discard pile (keep top card)
			if len(g.DiscardPile) > 1 {
				g.Deck.Cards = g.DiscardPile[:len(g.DiscardPile)-1]
				g.DiscardPile = g.DiscardPile[len(g.DiscardPile)-1:]
				g.Deck.Shuffle()
			} else {
				break // No more cards
			}
		}

		card := g.Deck.Draw(1)
		if len(card) > 0 {
			drawn = append(drawn, card[0])
		}
	}

	return drawn
}

// findPlayer finds a player by ID
func (g *UnoGame) findPlayer(playerID string) *UnoPlayer {
	for _, p := range g.Players {
		if p.ID == playerID {
			return p
		}
	}
	return nil
}

// calculateScores calculates final scores
func (g *UnoGame) calculateScores() {
	winnerScore := 0

	// Winner gets points from all other players' hands
	for _, player := range g.Players {
		if player.ID == g.Winner.ID {
			continue
		}

		for _, card := range player.Hand {
			points := g.getCardPoints(card)
			winnerScore += points
		}
	}

	g.Winner.Score += winnerScore
}

// getCardPoints returns points value of a card
func (g *UnoGame) getCardPoints(card *UnoCard) int {
	switch card.Value {
	case Value0, Value1, Value2, Value3, Value4, Value5, Value6, Value7, Value8, Value9:
		return int(card.Value)
	case ValueSkip, ValueReverse, ValueDraw2:
		return 20
	case ValueWild, ValueWildDraw4:
		return 50
	default:
		return 0
	}
}

// GetCurrentPlayer returns the current player
func (g *UnoGame) GetCurrentPlayer() *UnoPlayer {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.CurrentPlayer < 0 || g.CurrentPlayer >= len(g.Players) {
		return nil
	}

	return g.Players[g.CurrentPlayer]
}

// GetTopCard returns the top card of discard pile
func (g *UnoGame) GetTopCard() *UnoCard {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.DiscardPile) == 0 {
		return nil
	}

	return g.DiscardPile[len(g.DiscardPile)-1]
}

// FormatGameState formats the game state for display
func (g *UnoGame) FormatGameState() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("🎴 UNO GAME 🎴\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(g.DiscardPile) > 0 {
		topCard := g.DiscardPile[len(g.DiscardPile)-1]
		sb.WriteString(fmt.Sprintf("Top Card: %s\n", topCard.String()))
		sb.WriteString(fmt.Sprintf("Current Color: %s\n", g.CurrentColor.String()))
	}

	if g.DrawStack > 0 {
		sb.WriteString(fmt.Sprintf("Draw Stack: +%d\n", g.DrawStack))
	}

	sb.WriteString(fmt.Sprintf("Direction: %s\n", g.getDirectionString()))
	sb.WriteString("\n")

	// Show players
	sb.WriteString("Players:\n")
	for i, player := range g.Players {
		marker := "  "
		if i == g.CurrentPlayer {
			marker = "▶ "
		}

		sb.WriteString(fmt.Sprintf("%s%s: %d cards", marker, player.Name, len(player.Hand)))

		if len(player.Hand) == 1 && player.SaidUno {
			sb.WriteString(" (UNO!)")
		}

		sb.WriteString("\n")
	}

	if g.State == UnoStateFinished && g.Winner != nil {
		sb.WriteString(fmt.Sprintf("\n🏆 Winner: %s! (+%d points)\n", g.Winner.Name, g.Winner.Score))
	}

	return sb.String()
}

// getDirectionString returns direction as string
func (g *UnoGame) getDirectionString() string {
	if g.Direction == 1 {
		return "Clockwise ↻"
	}
	return "Counter-clockwise ↺"
}

// FormatPlayerHand formats a player's hand
func (g *UnoGame) FormatPlayerHand(playerID string) string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	player := g.findPlayer(playerID)
	if player == nil {
		return "Player not found"
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s's Hand:\n", player.Name))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")

	for i, card := range player.Hand {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, card.String()))
	}

	sb.WriteString(fmt.Sprintf("\nTotal: %d cards\n", len(player.Hand)))

	return sb.String()
}
