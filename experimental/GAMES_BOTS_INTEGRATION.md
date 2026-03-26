# 🎮🤖 Games & Bots Integration Guide

**Дополнение к GAMES_BOTS_AGENTS.md и ROADMAP.md**

**Дата:** 2026-02-07
**Статус:** Planning Document
**Связанные документы:**
- [GAMES_BOTS_AGENTS.md](./GAMES_BOTS_AGENTS.md) - Основная документация
- [ROADMAP.md](./ROADMAP.md) - Общий roadmap
- [../GROUP_CALLS.md](../GROUP_CALLS.md) - Текущая реализация групповых звонков

---

## 🎯 Цель документа

Этот документ объясняет:
1. Как игры и боты интегрируются с существующей системой групповых звонков
2. Технические детали реализации
3. API для разработчиков
4. Пошаговый план внедрения

---

## 📐 Архитектура интеграции

### Текущая система (GROUP_CALLS.md)

```
GroupCall
├── PeerConnection (человек)
├── PeerConnection (человек)
└── PeerConnection (человек)
```

### С играми и ботами

```
GroupCall (расширенный)
├── HumanParticipant (человек)
├── HumanParticipant (человек)
├── BotParticipant (AI бот)
├── GameSession
│   ├── GameState
│   ├── GameRules
│   └── PlayerMappings
└── AgentCoordinator (для multi-agent систем)
```

---

## 🏗️ Расширение GroupCall для игр

### 1. Унификация участников

**Проблема:** Сейчас `GroupCall` работает только с людьми

**Решение:** Создать общий интерфейс `Participant`

```go
// experimental/participants/participant.go

type ParticipantType int
const (
    TypeHuman ParticipantType = iota
    TypeBot
    TypeAgent  // AI agent
)

type Participant interface {
    // Идентификация
    GetID() string
    GetName() string
    GetType() ParticipantType

    // Медиа
    SendVideo(frame *terminal.Frame) error
    SendAudio(samples []int16) error
    ReceiveVideo() *terminal.Frame
    ReceiveAudio() []int16

    // Чат
    SendMessage(msg string) error
    OnMessage(callback func(msg string))

    // События
    OnJoin(callback func())
    OnLeave(callback func())

    // Статус
    IsActive() bool
    GetStats() *ParticipantStats
}
```

### 2. Реализации участников

#### Human Participant (обертка над существующим)

```go
// experimental/participants/human.go

type HumanParticipant struct {
    peerConn    *group.PeerConnection  // Существующая структура
    // ... дополнительные поля
}

func (h *HumanParticipant) GetID() string {
    return h.peerConn.ID
}

func (h *HumanParticipant) SendVideo(frame *terminal.Frame) error {
    // Использует существующий network transport
    return h.peerConn.SendVideoFrame(frame)
}

// ... остальные методы
```

#### Bot Participant (новый)

```go
// experimental/participants/bot.go

type BotParticipant struct {
    id          string
    name        string
    botType     BotType

    // AI
    llmClient   LLMClient      // GPT-4, Claude, etc.
    personality string
    context     []Message

    // Медиа генерация
    videoGen    VideoGenerator  // Генерирует аватар/видео
    tts         TextToSpeech    // Text-to-Speech
    stt         SpeechToText    // Speech-to-Text

    // Коммуникация с GroupCall
    videoOut    chan *terminal.Frame
    audioOut    chan []int16
    videoIn     chan *terminal.Frame
    audioIn     chan []int16
    chatIn      chan string
    chatOut     chan string
}

func NewBotParticipant(name string, llm LLMClient) *BotParticipant {
    bot := &BotParticipant{
        id:        generateID(),
        name:      name,
        llmClient: llm,
        videoOut:  make(chan *terminal.Frame, 10),
        audioOut:  make(chan []int16, 100),
        chatIn:    make(chan string, 50),
        chatOut:   make(chan string, 50),
    }

    // Запуск goroutines для обработки
    go bot.processChat()
    go bot.generateVideo()
    go bot.listenToAudio()

    return bot
}

// Бот слушает аудио и отвечает
func (b *BotParticipant) listenToAudio() {
    for audio := range b.audioIn {
        // Speech-to-Text
        text := b.stt.Recognize(audio)

        // Если обращение к боту
        if strings.Contains(strings.ToLower(text), b.name) {
            // Генерировать ответ через LLM
            response := b.llmClient.Generate(text, b.context)

            // Text-to-Speech
            speech := b.tts.Synthesize(response)

            // Отправить аудио
            b.audioOut <- speech

            // Отправить текст в чат (опционально)
            b.chatOut <- response
        }
    }
}

// Генерация видео аватара
func (b *BotParticipant) generateVideo() {
    ticker := time.NewTicker(66 * time.Millisecond) // 15 FPS
    defer ticker.Stop()

    for range ticker.C {
        // Генерировать кадр аватара
        frame := b.videoGen.GenerateFrame()

        // Отправить
        b.videoOut <- frame
    }
}

// Обработка чата
func (b *BotParticipant) processChat() {
    for msg := range b.chatIn {
        // Добавить в контекст
        b.context = append(b.context, Message{
            Role:    "user",
            Content: msg,
        })

        // Генерировать ответ
        response := b.llmClient.Generate(msg, b.context)

        // Добавить в контекст
        b.context = append(b.context, Message{
            Role:    "assistant",
            Content: response,
        })

        // Отправить ответ
        b.chatOut <- response
    }
}
```

### 3. Расширенный GroupCall

```go
// internal/group/group_call_v2.go

type GroupCallV2 struct {
    // Старые поля
    peers       map[string]*PeerConnection
    transport   *network.Transport

    // Новые поля
    participants map[string]Participant  // Унифицированные участники
    gameSession  *GameSession            // Активная игра (если есть)
    agentCoord   *AgentCoordinator       // Координатор агентов
    chatHistory  []ChatMessage           // История чата

    // Каналы
    videoChan   chan VideoUpdate
    audioChan   chan AudioUpdate
    chatChan    chan ChatMessage
}

type VideoUpdate struct {
    ParticipantID string
    Frame         *terminal.Frame
}

type AudioUpdate struct {
    ParticipantID string
    Samples       []int16
}

type ChatMessage struct {
    From      string
    To        string  // empty = broadcast
    Message   string
    Timestamp time.Time
}

func (g *GroupCallV2) AddParticipant(p Participant) error {
    g.mu.Lock()
    defer g.mu.Unlock()

    g.participants[p.GetID()] = p

    // Подключить каналы
    go g.handleParticipantVideo(p)
    go g.handleParticipantAudio(p)
    go g.handleParticipantChat(p)

    return nil
}

func (g *GroupCallV2) AddBot(name string, llm LLMClient) (*BotParticipant, error) {
    bot := NewBotParticipant(name, llm)
    err := g.AddParticipant(bot)
    return bot, err
}

// Запуск игры
func (g *GroupCallV2) StartGame(gameType GameType, config GameConfig) error {
    // Создать игровую сессию
    session := NewGameSession(gameType, config)

    // Добавить участников
    for _, p := range g.participants {
        if config.IncludeAllParticipants || p.GetType() == TypeHuman {
            session.AddPlayer(p)
        }
    }

    // Опционально добавить ботов
    if config.FillWithBots {
        neededBots := config.MinPlayers - len(session.Players)
        for i := 0; i < neededBots; i++ {
            bot, _ := g.AddBot(fmt.Sprintf("Bot_%d", i), config.BotLLM)
            session.AddPlayer(bot)
        }
    }

    g.gameSession = session

    // Запустить игру
    go session.Run()

    return nil
}
```

---

## 🎮 Игровая архитектура

### 1. GameSession

```go
// experimental/games/session.go

type GameType int
const (
    GameTrivia GameType = iota
    GamePoker
    GameChess
    GameUNO
    GamePictionary
    GameTicTacToe
)

type GameSession struct {
    ID          string
    Type        GameType
    State       GameState
    Rules       GameRules
    Players     []Participant
    CurrentTurn int

    // Каналы для событий
    actionChan  chan GameAction
    stateChan   chan GameState
    chatChan    chan string

    // Управление
    running     bool
    stopChan    chan bool
}

type GameState interface {
    // Каждая игра реализует свой State
    GetStateJSON() ([]byte, error)
    IsGameOver() bool
    GetWinner() string
}

type GameRules interface {
    // Правила игры
    ValidateAction(action GameAction) error
    ApplyAction(state GameState, action GameAction) (GameState, error)
}

type GameAction struct {
    PlayerID    string
    ActionType  string
    Data        interface{}
}

func (g *GameSession) Run() {
    for g.running {
        select {
        case action := <-g.actionChan:
            // Валидировать действие
            if err := g.Rules.ValidateAction(action); err != nil {
                g.sendError(action.PlayerID, err)
                continue
            }

            // Применить действие
            newState, err := g.Rules.ApplyAction(g.State, action)
            if err != nil {
                g.sendError(action.PlayerID, err)
                continue
            }

            // Обновить состояние
            g.State = newState

            // Разослать обновление всем игрокам
            g.broadcastState()

            // Проверить конец игры
            if g.State.IsGameOver() {
                g.endGame()
                return
            }

        case <-g.stopChan:
            return
        }
    }
}

func (g *GameSession) broadcastState() {
    stateJSON, _ := g.State.GetStateJSON()

    for _, player := range g.Players {
        player.SendMessage(string(stateJSON))
    }
}
```

### 2. Пример игры: Trivia

```go
// experimental/games/trivia/trivia.go

type TriviaState struct {
    Questions      []Question
    CurrentQ       int
    Scores         map[string]int
    Answered       map[string]bool
    CorrectAnswer  int
    TimeRemaining  time.Duration
}

type TriviaRules struct {
    TimeLimit      time.Duration
    PointsCorrect  int
    PointsWrong    int
}

type Question struct {
    Text     string
    Options  []string
    Correct  int
    Category string
}

func (t *TriviaRules) ValidateAction(action GameAction) error {
    // Проверить что игрок еще не ответил
    if action.ActionType != "answer" {
        return fmt.Errorf("invalid action type")
    }

    answer, ok := action.Data.(int)
    if !ok {
        return fmt.Errorf("invalid answer format")
    }

    if answer < 0 || answer > 3 {
        return fmt.Errorf("invalid answer range")
    }

    return nil
}

func (t *TriviaRules) ApplyAction(state GameState, action GameAction) (GameState, error) {
    triviaState := state.(*TriviaState)
    answer := action.Data.(int)

    // Отметить что игрок ответил
    triviaState.Answered[action.PlayerID] = true

    // Проверить правильность
    if answer == triviaState.CorrectAnswer {
        triviaState.Scores[action.PlayerID] += t.PointsCorrect
    } else {
        triviaState.Scores[action.PlayerID] += t.PointsWrong
    }

    // Если все ответили - переход к следующему вопросу
    allAnswered := true
    for _, answered := range triviaState.Answered {
        if !answered {
            allAnswered = false
            break
        }
    }

    if allAnswered {
        triviaState.CurrentQ++
        triviaState.Answered = make(map[string]bool)
    }

    return triviaState, nil
}

func (t *TriviaState) IsGameOver() bool {
    return t.CurrentQ >= len(t.Questions)
}

func (t *TriviaState) GetWinner() string {
    maxScore := 0
    winner := ""

    for player, score := range t.Scores {
        if score > maxScore {
            maxScore = score
            winner = player
        }
    }

    return winner
}
```

### 3. Визуализация игры в терминале

```go
// experimental/games/renderer.go

type GameRenderer struct {
    width  int
    height int
}

func (r *GameRenderer) RenderTrivia(state *TriviaState) *terminal.Frame {
    frame := terminal.NewFrame(r.width, r.height)

    // Отрисовать вопрос
    y := 2
    frame.DrawText(2, y, state.Questions[state.CurrentQ].Text, terminal.ColorWhite)
    y += 3

    // Отрисовать варианты ответов
    for i, option := range state.Questions[state.CurrentQ].Options {
        text := fmt.Sprintf("%d) %s", i+1, option)
        frame.DrawText(4, y, text, terminal.ColorCyan)
        y += 2
    }

    // Отрисовать таблицу результатов
    y += 2
    frame.DrawText(2, y, "Scores:", terminal.ColorYellow)
    y++

    for player, score := range state.Scores {
        text := fmt.Sprintf("  %s: %d", player, score)
        frame.DrawText(2, y, text, terminal.ColorGreen)
        y++
    }

    return frame
}

func (r *GameRenderer) RenderPoker(state *PokerState) *terminal.Frame {
    // Отрисовать покерный стол
    // Карты игроков, общие карты, банк и т.д.
}
```

---

## 🤖 Интеграция ботов в игры

### 1. Game Bot

```go
// experimental/bots/gamebot.go

type GameBot struct {
    BotParticipant

    gameType    GameType
    strategy    Strategy
    llm         LLMClient
}

func (g *GameBot) OnGameAction(state GameState) GameAction {
    // Получить состояние игры
    stateJSON, _ := state.GetStateJSON()

    // Спросить LLM что делать
    prompt := fmt.Sprintf(`
You are playing %s.
Current game state: %s
What action should you take?
Respond with JSON: {"action": "...", "data": ...}
`, g.gameType, stateJSON)

    response := g.llm.Generate(prompt, nil)

    // Парсить ответ
    var action GameAction
    json.Unmarshal([]byte(response), &action)
    action.PlayerID = g.id

    return action
}

// Пример для Trivia бота
type TriviaBotStrategy struct{}

func (t *TriviaBotStrategy) DecideAnswer(question Question) int {
    // Простая стратегия - спросить LLM
    prompt := fmt.Sprintf(`
Question: %s
Options:
1) %s
2) %s
3) %s
4) %s

What is the correct answer? Respond with just the number (1-4).
`, question.Text,
   question.Options[0],
   question.Options[1],
   question.Options[2],
   question.Options[3])

    response := llm.Generate(prompt, nil)

    // Парсить ответ
    answer, _ := strconv.Atoi(strings.TrimSpace(response))
    return answer - 1
}

// Пример для Poker бота
type PokerBotStrategy struct {
    aggressiveness float64  // 0.0 - 1.0
}

func (p *PokerBotStrategy) DecideAction(hand []Card, community []Card, pot int) PokerAction {
    // Оценить силу руки
    handStrength := evaluateHand(hand, community)

    // Решение на основе агрессивности
    if handStrength > 0.7 {
        // Сильная рука - рейз
        return PokerAction{Type: "raise", Amount: pot / 2}
    } else if handStrength > 0.4 {
        // Средняя рука - колл или чек
        return PokerAction{Type: "call"}
    } else if rand.Float64() < p.aggressiveness {
        // Слабая рука - блеф
        return PokerAction{Type: "raise", Amount: pot / 4}
    } else {
        // Фолд
        return PokerAction{Type: "fold"}
    }
}
```

### 2. Комментатор бот

```go
// experimental/bots/commentator.go

type CommentatorBot struct {
    BotParticipant
    language string  // "en", "ru", etc.
}

func (c *CommentatorBot) CommentOnGameState(state GameState, lastAction GameAction) {
    // Получить описание состояния
    stateDesc := describeState(state)
    actionDesc := describeAction(lastAction)

    // Генерировать комментарий через LLM
    prompt := fmt.Sprintf(`
You are a professional game commentator.
Current state: %s
Last action: Player %s did %s

Provide an exciting commentary (2-3 sentences) in %s.
`, stateDesc, lastAction.PlayerID, actionDesc, c.language)

    commentary := c.llmClient.Generate(prompt, nil)

    // Отправить в чат
    c.SendMessage(commentary)

    // Опционально: озвучить через TTS
    if c.tts != nil {
        speech := c.tts.Synthesize(commentary)
        c.SendAudio(speech)
    }
}

// Пример использования:
// Poker game with commentator
commentator := NewCommentatorBot("Magnus AI", "en")
commentator.personality = "Energetic and analytical"
groupCall.AddParticipant(commentator)

session.OnAction(func(action GameAction) {
    commentator.CommentOnGameState(session.State, action)
})

// Пример комментариев:
// "Incredible! Alice goes all-in with just a pair of twos!
//  Is this a brilliant bluff or a rookie mistake?
//  Bob is contemplating his next move..."
```

---

## 🔗 Multi-Agent Systems

### 1. Agent Coordinator

```go
// experimental/agents/coordinator.go

type AgentCoordinator struct {
    agents      []*Agent
    messages    chan AgentMessage
    running     bool
}

type Agent struct {
    ID          string
    Name        string
    Role        string
    Participant Participant
    Goals       []string
    LLM         LLMClient
    Memory      []Interaction
}

type AgentMessage struct {
    From        string
    To          string  // empty = broadcast
    Type        MessageType
    Content     string
    ReplyTo     string
}

type MessageType int
const (
    TypeInform MessageType = iota
    TypeRequest
    TypePropose
    TypeAccept
    TypeReject
    TypeQuery
)

func (ac *AgentCoordinator) RouteMessage(msg AgentMessage) {
    if msg.To == "" {
        // Broadcast всем агентам
        for _, agent := range ac.agents {
            if agent.ID != msg.From {
                agent.ReceiveMessage(msg)
            }
        }
    } else {
        // Unicast конкретному агенту
        for _, agent := range ac.agents {
            if agent.ID == msg.To {
                agent.ReceiveMessage(msg)
                break
            }
        }
    }
}

func (a *Agent) ReceiveMessage(msg AgentMessage) {
    // Добавить в память
    a.Memory = append(a.Memory, Interaction{
        Type:      "received_message",
        Content:   msg.Content,
        Timestamp: time.Now(),
    })

    // Решить нужно ли отвечать
    shouldReply := a.shouldReplyToMessage(msg)

    if shouldReply {
        // Генерировать ответ через LLM
        response := a.generateResponse(msg)

        // Отправить ответ
        a.SendMessage(AgentMessage{
            From:    a.ID,
            To:      msg.From,
            Type:    TypeInform,
            Content: response,
            ReplyTo: msg.Content,
        })
    }
}

func (a *Agent) shouldReplyToMessage(msg AgentMessage) bool {
    // Простая эвристика - отвечать на вопросы и запросы
    if msg.Type == TypeQuery || msg.Type == TypeRequest {
        return true
    }

    // Или если сообщение упоминает этого агента
    if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(a.Name)) {
        return true
    }

    // Или если это связано с целями агента
    for _, goal := range a.Goals {
        if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(goal)) {
            return true
        }
    }

    return false
}

func (a *Agent) generateResponse(msg AgentMessage) string {
    // Контекст из памяти
    context := a.buildContext()

    prompt := fmt.Sprintf(`
You are %s, a %s.
Your goals: %s

You received this message from %s:
"%s"

How do you respond? Be concise and in character.
`, a.Name, a.Role, strings.Join(a.Goals, ", "), msg.From, msg.Content)

    response := a.LLM.Generate(prompt, context)

    return response
}
```

### 2. Пример: Poker Tournament между агентами

```go
// experimental/examples/poker_tournament.go

func RunPokerTournament(groupCall *GroupCallV2) {
    // Создать агентов с разными стратегиями
    alice := &Agent{
        Name: "Aggressive Alice",
        Role: "Aggressive poker player",
        Goals: []string{"Win by aggressive betting", "Bluff often"},
        LLM: openai.NewClient("gpt-4"),
    }

    bob := &Agent{
        Name: "Conservative Carl",
        Role: "Conservative poker player",
        Goals: []string{"Minimize losses", "Only bet with good hands"},
        LLM: anthropic.NewClient("claude-3-5-sonnet"),
    }

    charlie := &Agent{
        Name: "Calculator Clara",
        Role: "Mathematical poker player",
        Goals: []string{"Calculate pot odds", "Play by the math"},
        LLM: nil,  // Чистая математика, без LLM
    }

    // Создать ботов для каждого агента
    aliceBot := NewGameBot("alice", alice.LLM)
    bobBot := NewGameBot("bob", bob.LLM)
    charlieBot := NewGameBot("charlie", nil)

    // Связать агентов с ботами
    alice.Participant = aliceBot
    bob.Participant = bobBot
    charlie.Participant = charlieBot

    // Добавить в звонок
    groupCall.AddParticipant(aliceBot)
    groupCall.AddParticipant(bobBot)
    groupCall.AddParticipant(charlieBot)

    // Создать координатора
    coordinator := &AgentCoordinator{
        agents: []*Agent{alice, bob, charlie},
    }

    // Запустить игру
    groupCall.StartGame(GamePoker, GameConfig{
        MinPlayers: 3,
        MaxPlayers: 6,
    })

    // Агенты будут общаться между собой во время игры
    // Люди могут наблюдать и читать их сообщения
}

// Пример диалога агентов:
/*
Alice: "I'm raising to 100. Who's brave enough to call?"
Bob: "Too risky for me. I fold."
Clara: "Based on pot odds (3:1) and my hand strength (0.45), I fold."
Alice: "Easy money! Sometimes you have to take risks."
Bob: "Or you just got lucky with a bluff, Alice."
Alice: "You'll never know! 😏"
*/
```

---

## 📅 Интеграция с ROADMAP

### Предлагаемый roadmap для игр и ботов:

**Q1 2025 (Март): MVP Games & Basic Bots**

```
✅ Фаза 1A: Простые игры (3 недели)
├── Trivia (текстовая викторина)         - 3 дня
├── Tic-Tac-Toe                          - 2 дня
├── UNO (карточная игра)                 - 1 неделя
└── Game Session framework               - 1 неделя

✅ Фаза 1B: Базовые боты (3 недели)
├── Participant interface                - 3 дня
├── Chat Bot (текстовый)                 - 1 неделя
├── Game Bot (для игр выше)              - 1 неделя
└── Integration с GroupCall              - 4 дня

Итого: 6 недель
Код: ~2500 строк
Тесты: ~40 тестов
```

**Q2 2025 (Апрель-Июнь): Advanced Games & Voice Bots**

```
✅ Фаза 2A: Сложные игры (6 недель)
├── Poker (Texas Hold'em)                - 2 недели
├── Chess                                - 2 недели
├── Pictionary (с whiteboard)            - 2 недели

✅ Фаза 2B: Voice Bots (4 недели)
├── Speech-to-Text integration           - 1 неделя
├── Text-to-Speech integration           - 1 неделя
├── Voice Bot implementation             - 2 недели

Итого: 10 недель
Код: ~4000 строк
Тесты: ~60 тестов
```

**Q3 2025 (Июль-Сентябрь): Multi-Agent Systems**

```
✅ Фаза 3A: Agent Framework (4 недели)
├── Agent Coordinator                    - 1 неделя
├── Message routing                      - 1 неделя
├── Memory & Context                     - 1 неделя
├── Multi-LLM support                    - 1 неделя

✅ Фаза 3B: Agent Applications (6 недель)
├── Bot Tournaments (poker, chess)       - 2 недели
├── AI Debates                           - 2 недели
├── Collaborative problem solving        - 2 недели

Итого: 10 недель
Код: ~3500 строк
Тесты: ~50 тестов
```

**Q4 2025 (Октябрь-Декабрь): Advanced Features**

```
✅ Фаза 4A: Avatar Bots (6 недель)
├── 2D Avatar rendering                  - 2 недели
├── Emotion engine                       - 2 недели
├── Lip sync                             - 2 недели

✅ Фаза 4B: Real-time Games (6 недель)
├── Battle Arena (простой 2D)            - 4 недели
├── Swarm behavior simulation            - 2 недели

Итого: 12 недель
Код: ~5000 строк
Тесты: ~60 тестов
```

---

## 🚀 Quick Start Guide

### Шаг 1: Простая игра Trivia

```bash
# Установить зависимости
go get github.com/sashabaranov/go-openai

# Запустить групповой звонок с игрой
tvcp group --game trivia alice:5000 bob:5000

# Или добавить ботов
tvcp group --game trivia --bots 2 alice:5000
```

### Шаг 2: Добавить AI бота

```go
// Создать группо вой звонок
gc := group.NewGroupCallV2(":5000")

// Добавить людей
gc.AddPeer("alice", addr1)
gc.AddPeer("bob", addr2)

// Добавить AI бота
llm := openai.NewClient(apiKey)
bot, _ := gc.AddBot("AI Assistant", llm)

// Бот будет отвечать на вопросы в чате
gc.Start()
```

### Шаг 3: Запустить покер турнир ботов

```go
// Создать 4 покерных бота
bots := []string{
    "Aggressive Alice",
    "Conservative Carl",
    "Bluffer Bob",
    "Calculator Clara",
}

// Запустить турнир
tournament := poker.NewTournament(bots)
tournament.Start()

// Люди могут наблюдать и читать чат ботов
tournament.EnableSpectators(true)
```

---

## 🔧 API для разработчиков

### Создание собственной игры

```go
// 1. Реализовать GameState
type MyGameState struct {
    // ваши поля
}

func (m *MyGameState) GetStateJSON() ([]byte, error) {
    return json.Marshal(m)
}

func (m *MyGameState) IsGameOver() bool {
    // ваша логика
}

func (m *MyGameState) GetWinner() string {
    // ваша логика
}

// 2. Реализовать GameRules
type MyGameRules struct {
    // ваши параметры
}

func (r *MyGameRules) ValidateAction(action GameAction) error {
    // валидация
}

func (r *MyGameRules) ApplyAction(state GameState, action GameAction) (GameState, error) {
    // применить действие
}

// 3. Зарегистрировать игру
games.Register("mygame", func(config GameConfig) (GameRules, GameState) {
    return &MyGameRules{}, &MyGameState{}
})

// 4. Использовать
gc.StartGame("mygame", GameConfig{})
```

### Создание собственного бота

```go
// 1. Реализовать Participant interface
type MyBot struct {
    // ваши поля
}

func (b *MyBot) GetID() string { ... }
func (b *MyBot) SendVideo(frame *terminal.Frame) error { ... }
// ... остальные методы

// 2. Добавить в звонок
bot := &MyBot{}
gc.AddParticipant(bot)
```

---

## 📊 Метрики успеха

### MVP (Q1 2025)

- ✅ 3 работающие игры (Trivia, TicTacToe, UNO)
- ✅ Chat Bot отвечает на вопросы
- ✅ Game Bot может играть в игры
- ✅ 5+ одновременных участников (люди + боты)

### Advanced (Q2 2025)

- ✅ Poker и Chess работают
- ✅ Voice Bot может говорить и слушать
- ✅ Spectator mode для турниров ботов

### Multi-Agent (Q3 2025)

- ✅ 4 бота играют в покер между собой
- ✅ AI дебаты работают
- ✅ Люди могут наблюдать за агентами

### Advanced AI (Q4 2025)

- ✅ Avatar Bot с анимацией
- ✅ Real-time multiplayer game
- ✅ Swarm behavior demo

---

## 🎯 Итог

Этот документ показывает:

1. ✅ **Как интегрировать** игры и ботов с существующим GroupCall
2. ✅ **Техническую архитектуру** с примерами кода
3. ✅ **Roadmap** для поэтапной реализации
4. ✅ **API** для разработчиков собственных игр и ботов
5. ✅ **Quick Start** для быстрого начала

Следующий шаг: Выбрать MVP функции (Trivia + Chat Bot) и начать реализацию!

---

**Автор:** Claude Code
**Дата:** 2026-02-07
**Версия:** 1.0
**Статус:** Planning Document
