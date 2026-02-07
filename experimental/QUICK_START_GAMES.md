# 🚀 Quick Start: Games & Bots в TVCP

**5-минутный гайд для запуска первой игры с AI ботом**

---

## 🎯 Что вы сделаете

За 5 минут вы:
1. Запустите групповой звонок с 3 участниками
2. Начнете игру Trivia (викторину)
3. Добавите AI бота, который будет играть с вами
4. Увидите как боты общаются между собой

---

## 📋 Требования

```bash
# Go 1.21+
go version

# API ключ (выберите один)
export OPENAI_API_KEY="sk-..."  # для GPT-4
export ANTHROPIC_API_KEY="sk-ant-..."  # для Claude
```

---

## ⚡ Быстрый старт

### Шаг 1: Установка (30 секунд)

```bash
# Клонировать репозиторий
git clone https://github.com/yourusername/tvcp
cd tvcp

# Установить зависимости
go mod download

# Установить AI библиотеки
go get github.com/sashabaranov/go-openai
go get github.com/anthropics/anthropic-sdk-go

# Собрать
go build -o tvcp ./cmd/tvcp
```

### Шаг 2: Запустить простую игру (2 минуты)

```bash
# Терминал 1: Алиса
./tvcp group --game trivia --port 5000 bob:5001 charlie:5002

# Терминал 2: Боб
./tvcp group --game trivia --port 5001 alice:5000 charlie:5002

# Терминал 3: Чарли
./tvcp group --game trivia --port 5002 alice:5000 bob:5001
```

**Что произойдет:**
```
[Trivia Game Started]

Question 1: What is the capital of France?
1) London
2) Paris
3) Berlin
4) Madrid

Type your answer (1-4): _
```

### Шаг 3: Добавить AI бота (2 минуты)

```bash
# Терминал 4: AI Бот
export OPENAI_API_KEY="sk-..."
./tvcp group --game trivia --port 5003 --bot-mode \
  alice:5000 bob:5001 charlie:5002
```

**Что произойдет:**
```
[AI Bot "Einstein" joined the game]

Question 1: What is the capital of France?

Alice answered: 2 (Paris) ✅
Bob answered: 1 (London) ❌
Charlie answered: 2 (Paris) ✅
Einstein answered: 2 (Paris) ✅

Scores:
  Alice: 10
  Charlie: 10
  Einstein: 10
  Bob: 0

Einstein (in chat): "That was an easy one! Paris is known
as the City of Light and has been the capital since 508 AD."
```

---

## 🎮 Доступные игры

### 1. Trivia (Викторина)

**Запуск:**
```bash
./tvcp group --game trivia [peers...]
```

**Особенности:**
- 10 вопросов разных категорий
- 4 варианта ответа
- 30 секунд на ответ
- Счет в реальном времени

**С ботом:**
```bash
./tvcp group --game trivia --bots 1 [peers...]
# Добавит 1 AI бота, который будет отвечать на вопросы
```

### 2. Tic-Tac-Toe (Крестики-нолики)

**Запуск:**
```bash
./tvcp group --game tictactoe peer:5000
```

**Особенности:**
- 2 игрока (вы vs peer или вы vs бот)
- Классические правила
- ASCII графика

**С ботом:**
```bash
./tvcp group --game tictactoe --bot
# Играть против AI
```

### 3. UNO (Карточная игра)

**Запуск:**
```bash
./tvcp group --game uno peer1:5000 peer2:5001
```

**Особенности:**
- 2-4 игрока
- Все карты UNO
- Special cards (Skip, Reverse, Draw 2, Wild)

**С ботом:**
```bash
./tvcp group --game uno --bots 2 peer:5000
# Добавит 2 ботов, чтобы заполнить стол
```

---

## 🤖 Типы ботов

### Chat Bot

**Запуск:**
```bash
./tvcp group --chat-bot --bot-name "Assistant" [peers...]
```

**Примеры использования:**
```
Вы: "What's the weather like?"
Bot: "I don't have access to real-time data, but I can
      help with general questions!"

Вы: "Tell me a joke"
Bot: "Why don't scientists trust atoms? Because they
      make up everything! 😄"
```

### Game Bot

**Запуск:**
```bash
./tvcp group --game poker --bots 3 [peers...]
# Добавит 3 покерных ботов с разными стратегиями
```

**Стратегии ботов:**
- **Aggressive**: Часто рейзит и блефует
- **Conservative**: Играет только сильные руки
- **Mathematical**: Рассчитывает pot odds

---

## 🎪 Продвинутые сценарии

### Сценарий 1: Турнир ботов

**Люди наблюдают, боты играют:**

```bash
# Запустить покер турнир с 4 ботами
./tvcp tournament poker --bots 4 --spectator-mode

# Что произойдет:
# - 4 AI бота играют в покер
# - Вы наблюдаете и читаете их мысли
# - Комментатор-бот объясняет что происходит
```

**Пример вывода:**
```
[Poker Tournament - Round 1]

Players:
  - Aggressive Alice (GPT-4)
  - Conservative Carl (Claude)
  - Bluffer Bob (Gemini)
  - Calculator Clara (Math only)

--- Hand 1 ---

Alice: [A♠ K♠] - Thinking: "Strong starting hand, raise to 100"
Alice raises to 100

Carl: [7♣ 2♦] - Thinking: "Weak hand, fold"
Carl folds

Bob: [9♥ 9♦] - Thinking: "Pocket nines, call and see flop"
Bob calls

Clara: [J♣ T♣] - Thinking: "Pot odds 2.5:1, equity 42%, call"
Clara calls

Flop: [A♥ 9♠ 2♠]

Commentator: "Interesting flop! Alice hit top pair with top
kicker, but Bob flopped a set of nines! This could be a big pot."

Alice: "I'm all in!" (thinking she has the best hand)
Bob: "I call!" (knowing he has a set)
Clara: "I fold" (math says fold)

River: [K♦]

Bob wins with three 9s!

Commentator: "What a cooler! Alice had two pair but Bob's
set was too strong. That's poker!"
```

### Сценарий 2: AI Дебаты

**2 AI агента спорят, люди слушают:**

```bash
# Запустить дебаты
./tvcp debate "AI should replace human drivers" \
  --pro-agent gpt-4 \
  --con-agent claude \
  --spectator-mode

# Что произойдет:
# - GPT-4 аргументирует ЗА
# - Claude аргументирует ПРОТИВ
# - Судья (тоже AI) оценивает аргументы
# - Зрители голосуют
```

**Пример:**
```
[AI Debate]
Topic: "AI should replace human drivers"

Pro (GPT-4): "Good evening. Self-driving cars have proven
to reduce accidents by 90% in controlled environments..."

Con (Claude): "Thank you. While I appreciate the safety
statistics, we must consider individual freedom and the
socioeconomic impact on 3.5 million truck drivers..."

Judge: "Both sides present compelling arguments. GPT-4's
safety data is strong, but Claude raises important ethical
concerns. Let's hear rebuttals."

Audience (you): *votes for winner*
```

### Сценарий 3: Collaborative Agents

**3 AI агента решают задачу вместе:**

```bash
# Запустить collaborative session
./tvcp collaborate "Plan a birthday party for 50 people, budget $2000"

# Агенты:
# - Coordinator (GPT-4): Координирует работу
# - Budget Manager (Claude): Следит за бюджетом
# - Event Planner (Gemini): Предлагает идеи
```

**Пример:**
```
Coordinator: "Let's plan a party. Budget Manager, what's our budget?"

Budget Manager: "We have $2000 total. I suggest allocating:
  - Venue: $600 (30%)
  - Catering: $800 (40%)
  - Entertainment: $400 (20%)
  - Decorations: $200 (10%)"

Event Planner: "I found 3 venues within budget. Option A is
a rooftop space for $550, great for summer parties."

Coordinator: "Sounds good. Budget Manager, does that work?"

Budget Manager: "Yes, leaves us $1450 for other items. Go ahead."

Event Planner: "For catering, I propose a buffet style with
BBQ theme, $15 per person = $750. Within budget!"

[... agents continue planning ...]

Coordinator: "Here's our final plan: ..."
```

---

## 🔧 Конфигурация

### Bot Settings

**Создайте `bot_config.json`:**

```json
{
  "default_bot": {
    "llm": "gpt-4",
    "personality": "friendly and helpful",
    "temperature": 0.7,
    "max_tokens": 150
  },

  "trivia_bot": {
    "llm": "gpt-4",
    "personality": "knowledgeable trivia expert",
    "confidence": 0.8
  },

  "poker_bots": [
    {
      "name": "Aggressive Alice",
      "strategy": "aggressive",
      "llm": "gpt-4"
    },
    {
      "name": "Conservative Carl",
      "strategy": "conservative",
      "llm": "claude-3-5-sonnet"
    }
  ]
}
```

**Использование:**
```bash
./tvcp group --game poker --config bot_config.json [peers...]
```

---

## 💡 Советы и хитрости

### 1. Выбор LLM

**GPT-4 (OpenAI):**
- ✅ Лучший для сложных рассуждений
- ✅ Хорош для дебатов и анализа
- ❌ Дорогой ($0.01 per 1K tokens)
- Используйте для: Дебаты, Poker, Сложные игры

**GPT-3.5 (OpenAI):**
- ✅ Быстрый
- ✅ Дешевый ($0.0005 per 1K tokens)
- ❌ Менее умный
- Используйте для: Trivia, Chat Bot, Простые игры

**Claude (Anthropic):**
- ✅ Отличный для этических рассуждений
- ✅ Хорош для дебатов
- ✅ Безопасный (меньше вредного контента)
- Используйте для: Дебаты, Модерация, Collaborative tasks

**Local LLMs (LLaMA, Mistral):**
- ✅ Бесплатно
- ✅ Privacy (локальная обработка)
- ❌ Требует мощное железо (GPU)
- Используйте для: Разработка, Testing, Privacy-sensitive apps

### 2. Оптимизация API costs

```bash
# Кэшировать ответы ботов
./tvcp group --game trivia --bot --cache-responses

# Использовать более дешевый LLM для простых вопросов
./tvcp group --game trivia --bot --llm gpt-3.5-turbo

# Ограничить количество запросов
./tvcp group --game trivia --bot --rate-limit 10/min
```

### 3. Debugging

```bash
# Включить verbose logging
./tvcp group --game trivia --bot --verbose

# Показывать "мысли" бота
./tvcp group --game poker --bots 3 --show-bot-thoughts

# Пример вывода:
# Bot thinking: "I have pocket aces, strong hand.
#                Pot is 200, I should raise to 400..."
# Bot action: Raise to 400
```

### 4. Кастомизация ботов

**Изменить личность бота:**
```bash
./tvcp group --chat-bot \
  --bot-personality "sarcastic and witty" \
  [peers...]

# Bot: "Oh great, another human who needs my help.
#       What is it this time? 😏"
```

**Изменить имя бота:**
```bash
./tvcp group --game trivia --bot-name "Einstein" [peers...]

# Einstein: "E=mc² is basic physics, but do you know
#            the capital of France?"
```

---

## 🐛 Troubleshooting

### Проблема: "API key not found"

```bash
# Решение: Установить переменную окружения
export OPENAI_API_KEY="sk-..."

# Или передать через флаг
./tvcp group --api-key "sk-..." [peers...]
```

### Проблема: "Bot не отвечает"

```bash
# Проверить rate limits
./tvcp group --bot --rate-limit 60/min

# Проверить timeout
./tvcp group --bot --timeout 30s

# Включить debug
./tvcp group --bot --debug
```

### Проблема: "Бот отвечает слишком медленно"

```bash
# Использовать более быстрый LLM
./tvcp group --bot --llm gpt-3.5-turbo

# Уменьшить max tokens
./tvcp group --bot --max-tokens 50

# Включить streaming
./tvcp group --bot --stream
```

---

## 📚 Следующие шаги

### Изучить документацию:

1. **[GAMES_BOTS_AGENTS.md](./GAMES_BOTS_AGENTS.md)**
   - Полная документация по играм и ботам
   - 21 идея игры
   - Multi-agent системы

2. **[GAMES_BOTS_INTEGRATION.md](./GAMES_BOTS_INTEGRATION.md)**
   - Техническая архитектура
   - API для разработчиков
   - Примеры кода

3. **[ROADMAP.md](./ROADMAP.md)**
   - План развития на 2025 год
   - 21 функция
   - Timeline

### Создать свою игру:

```go
// 1. Реализовать GameState
type MyGameState struct {
    // ваши поля
}

// 2. Реализовать GameRules
type MyGameRules struct {
    // ваши правила
}

// 3. Зарегистрировать
games.Register("mygame", NewMyGame)

// 4. Запустить
./tvcp group --game mygame [peers...]
```

### Создать своего бота:

```go
// 1. Реализовать Bot interface
type MyBot struct {
    llm LLMClient
}

func (b *MyBot) OnMessage(msg string) string {
    return b.llm.Generate(msg)
}

// 2. Использовать
bot := NewMyBot(llm)
groupCall.AddParticipant(bot)
```

---

## 🎉 Итог

**Вы научились:**
- ✅ Запускать игры в групповых звонках
- ✅ Добавлять AI ботов
- ✅ Настраивать ботов
- ✅ Запускать турниры и дебаты

**Следующий шаг:**
- Попробуйте разные игры (Trivia, UNO, Poker)
- Экспериментируйте с multi-agent системами
- Создайте свою игру или бота
- Поделитесь feedback!

**Вопросы?**
- GitHub Issues: https://github.com/yourusername/tvcp/issues
- Discord: https://discord.gg/tvcp

---

**Удачи! 🚀**

**P.S.** Не забудьте подписаться на наш канал для новостей о новых играх и функциях!
