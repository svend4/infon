//go:build experimental

package cards

import (
	"testing"
)

// TestNewUnoGame tests game initialization
func TestNewUnoGame(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.AddPlayer("player3", "Player 3")

	if game.ID != "game1" {
		t.Errorf("Expected ID 'game1', got '%s'", game.ID)
	}

	if len(game.Players) != 3 {
		t.Errorf("Expected 3 players, got %d", len(game.Players))
	}

	if game.State != UnoStateSetup {
		t.Errorf("Expected state Setup, got %v", game.State)
	}

	// Verify deck is initialized but not dealt
	if game.Deck == nil {
		t.Fatal("Deck should be initialized")
	}

	if len(game.Deck.Cards) != 108 {
		t.Errorf("Expected 108 cards in deck, got %d", len(game.Deck.Cards))
	}
}

// TestUnoDeckCreation tests deck composition
func TestUnoDeckCreation(t *testing.T) {
	deck := NewUnoDeck()

	if len(deck.Cards) != 108 {
		t.Errorf("Expected 108 cards, got %d", len(deck.Cards))
	}

	// Count card types
	colorCounts := make(map[UnoColor]int)
	valueCounts := make(map[UnoValue]int)
	wildCount := 0

	for _, card := range deck.Cards {
		colorCounts[card.Color]++
		valueCounts[card.Value]++
		if card.Color == ColorWild {
			wildCount++
		}
	}

	// Verify 4 colors + wild
	if len(colorCounts) != 5 {
		t.Errorf("Expected 5 color types, got %d", len(colorCounts))
	}

	// Verify 8 wild cards total (4 Wild + 4 WildDraw4)
	if wildCount != 8 {
		t.Errorf("Expected 8 wild cards, got %d", wildCount)
	}

	// Verify zeros (1 per color = 4 total)
	if valueCounts[Value0] != 4 {
		t.Errorf("Expected 4 zeros, got %d", valueCounts[Value0])
	}

	// Verify number 1-9 (2 per color x 4 colors = 8 each)
	if valueCounts[Value5] != 8 {
		t.Errorf("Expected 8 fives, got %d", valueCounts[Value5])
	}
}

// TestUnoGameStart tests starting the game
func TestUnoGameStart(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")

	err := game.Start()
	if err != nil {
		t.Fatalf("Failed to start game: %v", err)
	}

	if game.State != UnoStatePlaying {
		t.Errorf("Expected state Playing, got %v", game.State)
	}

	// Each player should have 7 cards
	for _, player := range game.Players {
		if len(player.Hand) != 7 {
			t.Errorf("Player %s should have 7 cards, got %d", player.ID, len(player.Hand))
		}
	}

	// Discard pile should have 1 card
	if len(game.DiscardPile) != 1 {
		t.Errorf("Discard pile should have 1 card, got %d", len(game.DiscardPile))
	}
}

// TestPlayNumberCard tests playing a number card
func TestPlayNumberCard(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	// Set up a scenario where player can play
	player := game.Players[game.CurrentPlayer]
	topCard := game.DiscardPile[len(game.DiscardPile)-1]

	// Give player a matching card
	matchingCard := &UnoCard{
		Color: topCard.Color,
		Value: Value5,
	}
	player.Hand = append(player.Hand, matchingCard)
	cardIndex := len(player.Hand) - 1

	initialDiscardSize := len(game.DiscardPile)
	initialHandSize := len(player.Hand)

	err := game.PlayCard(player.ID, cardIndex, ColorRed)
	if err != nil {
		t.Fatalf("Failed to play card: %v", err)
	}

	// Verify card was played
	if len(game.DiscardPile) != initialDiscardSize+1 {
		t.Error("Card should be added to discard pile")
	}

	if len(player.Hand) != initialHandSize-1 {
		t.Error("Card should be removed from hand")
	}
}

// TestPlaySkipCard tests skip card functionality
func TestPlaySkipCard(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.AddPlayer("player3", "Player 3")
	game.Start()

	// Force current player to 0
	game.CurrentPlayer = 0
	player := game.Players[0]

	// Give player a skip card matching current color (plus another card so game doesn't end)
	skipCard := &UnoCard{
		Color: game.CurrentColor,
		Value: ValueSkip,
	}
	player.Hand = []*UnoCard{
		skipCard,
		{Color: game.CurrentColor, Value: Value5}, // Extra card
	}

	err := game.PlayCard(player.ID, 0, ColorRed)
	if err != nil {
		t.Fatalf("Failed to play skip card: %v", err)
	}

	// Next player should be skipped (should be player 2, not player 1)
	if game.CurrentPlayer != 2 {
		t.Errorf("Expected current player to be 2 (skipped 1), got %d", game.CurrentPlayer)
	}
}

// TestPlayReverseCard tests reverse card functionality
func TestPlayReverseCard(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.AddPlayer("player3", "Player 3")
	game.Start()

	initialDirection := game.Direction

	// Give player a reverse card
	player := game.Players[game.CurrentPlayer]
	reverseCard := &UnoCard{
		Color: game.CurrentColor,
		Value: ValueReverse,
	}
	player.Hand = []*UnoCard{reverseCard}

	err := game.PlayCard(player.ID, 0, ColorRed)
	if err != nil {
		t.Fatalf("Failed to play reverse card: %v", err)
	}

	// Direction should be reversed
	if game.Direction == initialDirection {
		t.Error("Direction should be reversed")
	}
}

// TestPlayDraw2Card tests Draw2 card functionality
func TestPlayDraw2Card(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	game.CurrentPlayer = 0
	player := game.Players[0]
	nextPlayer := game.Players[1]

	// Give player a Draw2 card
	draw2Card := &UnoCard{
		Color: game.CurrentColor,
		Value: ValueDraw2,
	}
	player.Hand = []*UnoCard{
		draw2Card,
		{Color: game.CurrentColor, Value: Value3}, // Extra card so game doesn't end
	}

	err := game.PlayCard(player.ID, 0, ColorRed)
	if err != nil {
		t.Fatalf("Failed to play Draw2 card: %v", err)
	}

	// Draw stack should be set to 2
	if game.DrawStack != 2 {
		t.Errorf("DrawStack should be 2, got %d", game.DrawStack)
	}

	// Next player should now draw from stack
	initialHandSize := len(nextPlayer.Hand)
	game.DrawCard(nextPlayer.ID)

	if len(nextPlayer.Hand) != initialHandSize+2 {
		t.Errorf("Next player should have %d cards, got %d", initialHandSize+2, len(nextPlayer.Hand))
	}
}

// TestPlayWildCard tests wild card functionality
func TestPlayWildCard(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	player := game.Players[game.CurrentPlayer]

	// Give player a wild card
	wildCard := &UnoCard{
		Color: ColorWild,
		Value: ValueWild,
	}
	player.Hand = []*UnoCard{wildCard}

	// Play wild card and choose blue
	err := game.PlayCard(player.ID, 0, ColorBlue)
	if err != nil {
		t.Fatalf("Failed to play wild card: %v", err)
	}

	// Current color should be blue
	if game.CurrentColor != ColorBlue {
		t.Errorf("Expected current color Blue, got %v", game.CurrentColor)
	}
}

// TestPlayWildDraw4Card tests WildDraw4 card functionality
func TestPlayWildDraw4Card(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	game.CurrentPlayer = 0
	player := game.Players[0]
	nextPlayer := game.Players[1]

	// Give player a WildDraw4 card
	wildDraw4Card := &UnoCard{
		Color: ColorWild,
		Value: ValueWildDraw4,
	}
	player.Hand = []*UnoCard{
		wildDraw4Card,
		{Color: ColorGreen, Value: Value7}, // Extra card so game doesn't end
	}

	// Play wild draw 4 and choose green
	err := game.PlayCard(player.ID, 0, ColorGreen)
	if err != nil {
		t.Fatalf("Failed to play WildDraw4 card: %v", err)
	}

	// Draw stack should be set to 4
	if game.DrawStack != 4 {
		t.Errorf("DrawStack should be 4, got %d", game.DrawStack)
	}

	// Current color should be green
	if game.CurrentColor != ColorGreen {
		t.Errorf("Expected current color Green, got %v", game.CurrentColor)
	}

	// Next player should now draw from stack
	initialHandSize := len(nextPlayer.Hand)
	game.DrawCard(nextPlayer.ID)

	if len(nextPlayer.Hand) != initialHandSize+4 {
		t.Errorf("Next player should have %d cards, got %d", initialHandSize+4, len(nextPlayer.Hand))
	}
}

// TestDrawCard tests drawing a card
func TestDrawCard(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	player := game.Players[game.CurrentPlayer]
	initialHandSize := len(player.Hand)

	err := game.DrawCard(player.ID)
	if err != nil {
		t.Fatalf("Failed to draw card: %v", err)
	}

	if len(player.Hand) != initialHandSize+1 {
		t.Errorf("Player should have %d cards, got %d", initialHandSize+1, len(player.Hand))
	}
}

// TestCanPlayCard tests card validation
func TestCanPlayCard(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	// Set known state
	game.CurrentColor = ColorRed
	game.DiscardPile = []*UnoCard{
		{Color: ColorRed, Value: Value5},
	}

	tests := []struct {
		name     string
		card     *UnoCard
		canPlay  bool
	}{
		{
			name:    "Matching color",
			card:    &UnoCard{Color: ColorRed, Value: Value3},
			canPlay: true,
		},
		{
			name:    "Matching value",
			card:    &UnoCard{Color: ColorBlue, Value: Value5},
			canPlay: true,
		},
		{
			name:    "Wild card",
			card:    &UnoCard{Color: ColorWild, Value: ValueWild},
			canPlay: true,
		},
		{
			name:    "No match",
			card:    &UnoCard{Color: ColorBlue, Value: Value3},
			canPlay: false,
		},
	}

	topCard := game.DiscardPile[len(game.DiscardPile)-1]

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := game.canPlayCard(tt.card, topCard)
			if result != tt.canPlay {
				t.Errorf("Expected canPlay=%v, got %v for card %+v", tt.canPlay, result, tt.card)
			}
		})
	}
}

// TestInvalidPlays tests error conditions
func TestInvalidPlays(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	player := game.Players[game.CurrentPlayer]
	otherPlayer := game.Players[1-game.CurrentPlayer]

	// Test playing when not your turn
	err := game.PlayCard(otherPlayer.ID, 0, ColorRed)
	if err == nil {
		t.Error("Should not allow playing when not your turn")
	}

	// Test invalid card index
	err = game.PlayCard(player.ID, 999, ColorRed)
	if err == nil {
		t.Error("Should not allow invalid card index")
	}

	// Test playing non-matching card
	game.CurrentColor = ColorRed
	game.DiscardPile = []*UnoCard{
		{Color: ColorRed, Value: Value5},
	}
	player.Hand = []*UnoCard{
		{Color: ColorBlue, Value: Value3}, // Doesn't match
	}

	err = game.PlayCard(player.ID, 0, ColorRed)
	if err == nil {
		t.Error("Should not allow playing non-matching card")
	}
}

// TestCallUno tests UNO calling mechanism
func TestCallUno(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	player := game.Players[0]

	// Give player 2 cards
	player.Hand = []*UnoCard{
		{Color: ColorRed, Value: Value5},
		{Color: ColorRed, Value: Value6},
	}

	// Play one card to get to 1 card
	game.CurrentPlayer = 0
	game.CurrentColor = ColorRed
	game.PlayCard(player.ID, 0, ColorRed)

	// Player should have 1 card now
	if len(player.Hand) != 1 {
		t.Errorf("Player should have 1 card, got %d", len(player.Hand))
	}

	// Call UNO
	err := game.SayUno(player.ID)
	if err != nil {
		t.Fatalf("Failed to say UNO: %v", err)
	}

	if !player.SaidUno {
		t.Error("Player should have UNO called")
	}
}

// TestChallengeUno tests UNO challenge mechanism
func TestChallengeUno(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	player := game.Players[0]
	challenger := game.Players[1]

	// Give player 1 card but don't call UNO
	player.Hand = []*UnoCard{
		{Color: ColorRed, Value: Value5},
	}
	player.SaidUno = false

	initialHandSize := len(player.Hand)

	// Challenge successful (player didn't call UNO)
	err := game.ChallengeUno(challenger.ID, player.ID)
	if err != nil {
		t.Fatalf("Failed to challenge UNO: %v", err)
	}

	// Player should have drawn penalty cards
	if len(player.Hand) <= initialHandSize {
		t.Error("Player should have penalty cards for not calling UNO")
	}
}

// TestWinCondition tests game ending
func TestWinCondition(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	player := game.Players[game.CurrentPlayer]

	// Give player exactly 1 matching card
	game.CurrentColor = ColorRed
	player.Hand = []*UnoCard{
		{Color: ColorRed, Value: Value5},
	}
	player.SaidUno = true

	// Play the last card
	err := game.PlayCard(player.ID, 0, ColorRed)
	if err != nil {
		t.Fatalf("Failed to play final card: %v", err)
	}

	// Game should be finished
	if game.State != UnoStateFinished {
		t.Errorf("Expected state Finished, got %v", game.State)
	}

	if game.Winner == nil || game.Winner.ID != player.ID {
		winnerID := ""
		if game.Winner != nil {
			winnerID = game.Winner.ID
		}
		t.Errorf("Expected winner %s, got %s", player.ID, winnerID)
	}
}

// TestDeckReshuffle tests deck reshuffling when empty
func TestDeckReshuffle(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	// Empty the deck
	game.Deck.Cards = []*UnoCard{}

	// Add cards to discard pile (except top card)
	for i := 0; i < 10; i++ {
		game.DiscardPile = append(game.DiscardPile, &UnoCard{
			Color: ColorRed,
			Value: UnoValue(i % 10),
		})
	}

	initialDiscardSize := len(game.DiscardPile)

	// Try to draw - should trigger reshuffle
	player := game.Players[game.CurrentPlayer]
	err := game.DrawCard(player.ID)
	if err != nil {
		t.Fatalf("Failed to draw card (reshuffle): %v", err)
	}

	// Deck should have cards again (from reshuffled discard pile)
	// Discard pile should be smaller
	if len(game.Deck.Cards) == 0 && len(game.DiscardPile) >= initialDiscardSize {
		t.Error("Deck should be reshuffled from discard pile")
	}
}

// TestGetCurrentPlayer tests getting current player
func TestGetCurrentPlayer(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.AddPlayer("player3", "Player 3")
	game.Start()

	player := game.GetCurrentPlayer()
	if player == nil {
		t.Fatal("Current player should not be nil")
	}

	if player.ID != game.Players[game.CurrentPlayer].ID {
		t.Error("GetCurrentPlayer returned wrong player")
	}
}

// TestFormatPlayerHand tests player hand formatting
func TestFormatPlayerHand(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	player := game.Players[0]
	formatted := game.FormatPlayerHand(player.ID)

	if formatted == "" {
		t.Error("Formatted hand should not be empty")
	}
}

// TestFormatGameState tests game state formatting
func TestFormatGameState(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	formatted := game.FormatGameState()

	if formatted == "" {
		t.Error("Formatted game state should not be empty")
	}

	// Should contain game information
	if len(formatted) < 20 {
		t.Error("Formatted game state seems too short")
	}
}

// TestGetCurrentPlayer tests getting current player
func TestGetPlayerByID(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.Start()

	player := game.Players[0]

	if player.ID != "player1" {
		t.Errorf("Expected player ID player1, got %s", player.ID)
	}

	if len(player.Hand) != 7 {
		t.Errorf("Player should have 7 cards after start, got %d", len(player.Hand))
	}
}

// TestDrawStackAccumulation tests Draw2/Draw4 stacking
func TestDrawStackAccumulation(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.AddPlayer("player3", "Player 3")
	game.Start()

	// Player 1 plays Draw2
	game.CurrentPlayer = 0
	game.DrawStack = 0
	player1 := game.Players[0]
	game.CurrentColor = ColorRed
	player1.Hand = []*UnoCard{
		{Color: ColorRed, Value: ValueDraw2},
		{Color: ColorRed, Value: Value5}, // Extra card so game doesn't end
	}
	game.PlayCard(player1.ID, 0, ColorRed)

	// Draw stack should be 2
	if game.DrawStack != 2 {
		t.Errorf("Expected draw stack 2, got %d", game.DrawStack)
	}

	// Player 2 should be current
	if game.CurrentPlayer != 1 {
		t.Error("Should have moved to next player")
	}
}

// TestMultipleReverses tests multiple reverse cards
func TestMultipleReverses(t *testing.T) {
	game := NewUnoGame("game1")
	game.AddPlayer("player1", "Player 1")
	game.AddPlayer("player2", "Player 2")
	game.AddPlayer("player3", "Player 3")
	game.Start()

	initialDirection := game.Direction

	// Play reverse twice
	for i := 0; i < 2; i++ {
		player := game.Players[game.CurrentPlayer]
		player.Hand = []*UnoCard{
			{Color: game.CurrentColor, Value: ValueReverse},
			{Color: game.CurrentColor, Value: Value3}, // Extra card
		}
		game.PlayCard(player.ID, 0, ColorRed)
	}

	// Direction should be back to initial
	if game.Direction != initialDirection {
		t.Error("Two reverses should return to original direction")
	}
}

// BenchmarkNewUnoGame benchmarks game creation
func BenchmarkNewUnoGame(b *testing.B) {
	for i := 0; i < b.N; i++ {
		game := NewUnoGame("game1")
		game.AddPlayer("p1", "Player 1")
		game.AddPlayer("p2", "Player 2")
		game.AddPlayer("p3", "Player 3")
		game.AddPlayer("p4", "Player 4")
	}
}

// BenchmarkPlayCard benchmarks card playing
func BenchmarkPlayCard(b *testing.B) {
	game := NewUnoGame("game1")
	game.AddPlayer("p1", "Player 1")
	game.AddPlayer("p2", "Player 2")
	game.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		player := game.Players[game.CurrentPlayer]
		if len(player.Hand) > 0 && len(game.DiscardPile) > 0 {
			topCard := game.DiscardPile[len(game.DiscardPile)-1]
			// Find a playable card
			for idx, card := range player.Hand {
				if game.canPlayCard(card, topCard) {
					game.PlayCard(player.ID, idx, card.Color)
					break
				}
			}
		} else {
			game.DrawCard(player.ID)
		}
	}
}

// BenchmarkDeckShuffle benchmarks deck shuffling
func BenchmarkDeckShuffle(b *testing.B) {
	for i := 0; i < b.N; i++ {
		deck := NewUnoDeck()
		deck.Shuffle()
	}
}
