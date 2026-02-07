# 🗺️ TVCP Experimental Features Roadmap

## 📋 Общий план развития (2024-2025)

### ✅ Уже реализовано (Q4 2024)

| Функция | Статус | Строки | Тесты | Стабильность |
|---------|--------|--------|-------|--------------|
| Windows WASAPI | ✅ Done | 1,210 | 21/21 | ✅ Stable |
| Mobile iOS | ✅ Done | 1,180 | - | ✅ Stable |
| Mobile Android | ✅ Done | 1,060 | - | ✅ Stable |
| SFU Server | ✅ Done | 1,440 | 22/22 | ✅ Stable |
| WebRTC Processing | ✅ Done | 1,550 | 31/31 | 🔶 Beta |
| FFT Filters | ✅ Done | 960 | 24/24 | 🔶 Beta |
| Advanced AEC | ✅ Done | 1,050 | 23/23 | ⚠️ Alpha |
| Hardware Accel | ✅ Done | 1,200 | 23/23 | ⚠️ Alpha |

**Итого:** ~10,850 строк, 144 теста

---

## 🎯 Фаза 1: Foundation Features (Q1 2025 - 3 месяца)

### Приоритет: Must-Have функции для базовой конкурентоспособности

#### 1.1 File Transfer (2 недели)

**Цель:** Передача файлов во время звонка

**Спецификация:**
```go
// experimental/fileshare/transfer.go

type FileTransfer struct {
    ID          string
    Filename    string
    Size        int64
    MimeType    string
    Sender      string
    Recipient   string
    Progress    float64
    Status      TransferStatus
    Chunks      []Chunk
}

// API
transfer := NewFileTransfer("document.pdf", recipient)
transfer.OnProgress(func(progress float64) {
    fmt.Printf("Progress: %.1f%%\n", progress*100)
})
transfer.Send()
```

**Технические детали:**
- UDP с подтверждениями (надёжная доставка)
- Chunking: 64KB блоки
- SHA256 verification
- Resume support (сохранение прогресса)
- Throttling (не забивать канал)

**Оценка:**
- Код: 400-500 строк
- Тесты: 15-20 тестов
- Сложность: ⭐⭐ (средняя)

---

#### 1.2 Screen Sharing (3 недели)

**Цель:** Демонстрация экрана в реальном времени

**Спецификация:**
```go
// experimental/screen/capture.go

type ScreenCapture struct {
    Source      CaptureSource    // Display/Window/Region
    FPS         int              // 15-30 fps
    Quality     int              // 1-10
    Encoder     VideoEncoder     // H.264
    Cursor      bool             // Show cursor
}

// Platform-specific
// capture_windows.go - DXGI Desktop Duplication API
// capture_linux.go   - X11 XShm / Wayland
// capture_darwin.go  - CGDisplayStream
```

**Технические детали:**
- Захват экрана платформозависимый
- Differential encoding (только изменения)
- Adaptive quality (по bandwidth)
- Cursor overlay
- Audio passthrough (системный звук опционально)

**Оценка:**
- Код: 800-1000 строк (с платформами)
- Тесты: 20-25 тестов
- Сложность: ⭐⭐⭐ (высокая)

**Зависимости:**
```
Windows: dwmapi.dll, dxgi.dll
Linux: libX11, libXext
macOS: CoreGraphics, CoreMedia
```

---

#### 1.3 Recording (2 недели)

**Цель:** Запись звонков локально

**Спецификация:**
```go
// experimental/recording/recorder.go

type Recorder struct {
    OutputPath   string
    Format       RecordFormat     // MP4, WebM, Audio-only
    VideoCodec   string           // H264, VP8
    AudioCodec   string           // AAC, Opus
    Bitrate      int              // Video bitrate
    Mixing       bool             // Mix multiple sources
}

// Usage
recorder := NewRecorder()
recorder.SetFormat(FormatMP4)
recorder.AddSource(localAudio)
recorder.AddSource(remoteAudio)
recorder.AddSource(localVideo)
recorder.Start("call_2024_01_15.mp4")
```

**Технические детали:**
- Container: MP4 (libmp4v2) или WebM (libwebm)
- Video: использовать существующие кодеки
- Audio: AAC (libfdk-aac) или Opus
- Muxing: синхронизация A/V
- File rotation (split по времени/размеру)

**Оценка:**
- Код: 500-600 строк
- Тесты: 15-20 тестов
- Сложность: ⭐⭐⭐ (средняя-высокая)

**Зависимости:**
```
- FFmpeg libraries (libavcodec, libavformat)
или
- Собственная реализация muxer
```

---

#### 1.4 Security: Waiting Room (1 неделя)

**Цель:** Контроль доступа к звонкам

**Спецификация:**
```go
// experimental/security/waitingroom.go

type WaitingRoom struct {
    Enabled      bool
    AutoAdmit    bool             // Auto-admit known users
    Capacity     int              // Max waiting
    Timeout      time.Duration    // Auto-reject after
}

// Events
OnJoinRequest(participant) {
    // Notify host
    // Host approves/rejects
}
```

**Технические детали:**
- Отдельная SFU комната для ожидания
- Signaling для approve/reject
- UI notifications
- Persistence (помнить approved users)

**Оценка:**
- Код: 300-400 строк
- Тесты: 10-15 тестов
- Сложность: ⭐⭐ (средняя)

---

### Фаза 1: Итого

| Метрика | Значение |
|---------|----------|
| **Время** | 8 недель (~2 месяца) |
| **Код** | ~2,000-2,500 строк |
| **Тесты** | ~70 тестов |
| **Новые файлы** | 12-15 файлов |

---

## 🎨 Фаза 2: Collaboration Features (Q2 2025 - 3 месяца)

### Приоритет: Функции для совместной работы

#### 2.1 Whiteboard (3 недели)

**Спецификация:**
```go
// experimental/whiteboard/canvas.go

type Whiteboard struct {
    Width       int
    Height      int
    Layers      []Layer
    Tools       []Tool
    History     []Action        // Undo/Redo
}

type Tool interface {
    Draw(ctx *DrawContext) error
}

// Инструменты
- Pen (ручка с сглаживанием)
- Marker (маркер, прозрачность)
- Eraser (ластик)
- Line, Rectangle, Circle, Arrow
- Text
- Image paste
```

**Синхронизация:**
```go
// CRDT (Conflict-free Replicated Data Type)
type WhiteboardSync struct {
    VectorClock  map[string]int
    Operations   []Operation
}

// Каждое действие = операция
// Операции коммутативны
// Eventual consistency
```

**Оценка:**
- Код: 600-800 строк
- Тесты: 20-25 тестов
- Сложность: ⭐⭐⭐

---

#### 2.2 Interactive Elements (2 недели)

**Polls:**
```go
// experimental/interactive/polls.go

type Poll struct {
    ID          string
    Question    string
    Options     []string
    Votes       map[string]int
    Anonymous   bool
    Multiple    bool          // Multiple choice
}

poll := NewPoll("What time works best?")
poll.AddOption("10:00 AM")
poll.AddOption("2:00 PM")
poll.Broadcast()
```

**Reactions:**
```go
// experimental/interactive/reactions.go

type Reaction struct {
    Type        ReactionType    // 👍 👎 ❤️ 😂 🎉
    Sender      string
    Timestamp   time.Time
    Position    *Position       // Optional position
}

// Ephemeral reactions (disappear after 3s)
```

**Q&A:**
```go
// experimental/interactive/qa.go

type Question struct {
    ID          string
    Text        string
    Asker       string
    Upvotes     int
    Answered    bool
    Answer      string
}
```

**Оценка:**
- Код: 400-500 строк
- Тесты: 15-20 тестов
- Сложность: ⭐⭐

---

#### 2.3 Breakout Rooms (3 недели)

**Спецификация:**
```go
// server/breakout/manager.go

type BreakoutRoomManager struct {
    MainRoom    *Room
    Breakouts   []*Room
    Duration    time.Duration
    AutoReturn  bool
}

// Создание комнат
manager.CreateRooms(5)                    // 5 комнат
manager.AssignRandom()                     // Случайное распределение
manager.Assign(participant, room)          // Ручное

// Управление
manager.BroadcastToAll("5 minutes left")
manager.StartTimer(15 * time.Minute)
manager.ReturnToMain()
```

**Технические детали:**
- Динамические SFU комнаты
- Миграция участников между комнатами
- State synchronization
- Broadcast messaging

**Оценка:**
- Код: 700-900 строк
- Тесты: 25-30 тестов
- Сложность: ⭐⭐⭐⭐

---

### Фаза 2: Итого

| Метрика | Значение |
|---------|----------|
| **Время** | 8 недель (~2 месяца) |
| **Код** | ~1,700-2,200 строк |
| **Тесты** | ~65 тестов |

---

## 🤖 Фаза 3: AI Enhancement (Q3 2025 - 3 месяца)

### Приоритет: AI-powered функции

#### 3.1 Virtual Background (4 недели)

**Спецификация:**
```go
// experimental/ai/background/segmentation.go

type BackgroundProcessor struct {
    Model       *SegmentationModel
    Backend     Backend           // CPU/GPU/CoreML
    Mode        BackgroundMode    // Blur/Replace/None
}

type BackgroundMode int
const (
    ModeNone BackgroundMode = iota
    ModeBlur                       // Gaussian blur
    ModeReplace                    // Image/Video
    ModeVirtual                    // 3D environment
)
```

**ML Models:**
- MediaPipe Selfie Segmentation
- TensorFlow Lite models
- Core ML (iOS)
- ONNX Runtime

**Оценка:**
- Код: 800-1000 строк
- Тесты: 15-20 тестов
- Сложность: ⭐⭐⭐⭐⭐

---

#### 3.2 Live Transcription (3 недели)

**Спецификация:**
```go
// experimental/ai/transcription/stt.go

type Transcription struct {
    Provider    STTProvider      // Whisper/Google/Azure
    Language    string
    RealTime    bool
    Speakers    []Speaker        // Speaker diarization
}

// Providers
- OpenAI Whisper (offline, бесплатный)
- Google Cloud Speech-to-Text
- Azure Speech Services
- AWS Transcribe
```

**Оценка:**
- Код: 500-700 строк
- Тесты: 15-20 тестов
- Сложность: ⭐⭐⭐⭐

---

#### 3.3 Translation (2 недели)

**Спецификация:**
```go
// experimental/ai/translation/translator.go

type RealTimeTranslator struct {
    STT         *Transcription    // Speech recognition
    Translator  *TextTranslator   // Text translation
    TTS         *TextToSpeech     // Speech synthesis
    SourceLang  string
    TargetLang  string
}

// Pipeline: Speech → Text → Translate → Speech
```

**Оценка:**
- Код: 400-500 строк
- Тесты: 10-15 тестов
- Сложность: ⭐⭐⭐⭐

---

### Фаза 3: Итого

| Метрика | Значение |
|---------|----------|
| **Время** | 9 недель (~2+ месяца) |
| **Код** | ~1,700-2,200 строк |
| **Тесты** | ~45 тестов |
| **ML Models** | 3-5 models |

---

## 🔐 Фаза 4: Enterprise Features (Q4 2025 - 4 месяца)

### Приоритет: Корпоративные функции

#### 4.1 End-to-End Encryption (4 недели)

**Спецификация:**
```go
// experimental/security/e2ee.go

// Signal Protocol implementation
type E2EESession struct {
    IdentityKey     *ecdh.PrivateKey
    SignedPreKey    *ecdh.PrivateKey
    OneTimePreKeys  []*ecdh.PrivateKey
    RatchetState    *DoubleRatchet
}

// Key exchange
session.GeneratePreKeys(100)
session.ExchangeKeys(peerPublicKey)

// Encryption
ciphertext := session.Encrypt(plaintext)
plaintext := session.Decrypt(ciphertext)
```

**Проблемы:**
- SFU не может видеть медиа
- Нужен специальный E2E SFU mode
- Or P2P для E2E звонков

**Оценка:**
- Код: 1000-1500 строк
- Тесты: 30-40 тестов
- Сложность: ⭐⭐⭐⭐⭐

---

#### 4.2 Live Streaming (3 недели)

**Спецификация:**
```go
// experimental/streaming/rtmp.go

type LiveStream struct {
    Destination  StreamDestination
    Bitrate      int
    Resolution   Resolution
    Encoder      VideoEncoder
}

// Destinations
- YouTube Live (RTMP)
- Twitch (RTMP)
- Facebook Live
- Custom RTMP server
```

**Оценка:**
- Код: 600-800 строк
- Тесты: 15-20 тестов
- Сложность: ⭐⭐⭐⭐

---

#### 4.3 Analytics & Monitoring (3 недели)

**Спецификация:**
```go
// experimental/analytics/metrics.go

type CallAnalytics struct {
    Duration        time.Duration
    Participants    int
    PeakParticipants int
    QualityMetrics  *QualityMetrics
    NetworkStats    *NetworkStats
}

type QualityMetrics struct {
    AudioPacketLoss float64
    VideoPacketLoss float64
    Jitter          time.Duration
    RTT             time.Duration
    MOS             float64       // Mean Opinion Score
}

// Export to
- Prometheus
- Grafana
- InfluxDB
- Custom webhooks
```

**Оценка:**
- Код: 500-700 строк
- Тесты: 20-25 тестов
- Сложность: ⭐⭐⭐

---

### Фаза 4: Итого

| Метрика | Значение |
|---------|----------|
| **Время** | 10 недель (~2.5 месяца) |
| **Код** | ~2,100-3,000 строк |
| **Тесты** | ~70 тестов |

---

## 📊 Общая статистика roadmap

### Суммарно по всем фазам:

| Фаза | Время | Код | Тесты | Функции |
|------|-------|-----|-------|---------|
| **Уже есть** | - | ~10,850 | 144 | 8 |
| **Фаза 1** | 2 мес | ~2,250 | 70 | 4 |
| **Фаза 2** | 2 мес | ~1,950 | 65 | 3 |
| **Фаза 3** | 2.5 мес | ~1,950 | 45 | 3 |
| **Фаза 4** | 2.5 мес | ~2,550 | 70 | 3 |
| **ИТОГО** | **9 мес** | **~19,550** | **394** | **21** |

### К концу 2025 года:

```
TVCP будет иметь:
✅ 21 основную функцию
✅ ~20,000 строк кода
✅ ~400 тестов
✅ Полная конкурентоспособность с Zoom/Teams
```

---

## 🎯 Рекомендации по реализации

### 1. Начните с Фазы 1
- Выберите 1-2 функции для начала
- File Transfer + Recording = хорошее начало
- Screen Sharing сложнее, но очень востребован

### 2. Используйте существующий код
- WebRTC Processing для аудио
- Hardware Accel для кодирования
- SFU для транспорта

### 3. Добавляйте постепенно
- Alpha → Beta → Candidate → Core
- Не спешите переводить в Core
- Собирайте feedback

### 4. Тестируйте тщательно
- Unit tests
- Integration tests
- Performance tests
- User acceptance tests

### 5. Документируйте
- API documentation
- Usage examples
- Known limitations
- Migration guides

---

## 📅 Timeline визуализация

```
2024 Q4:  ████████████████████ Базовая платформа готова
          |
2025 Q1:  ████████████████████ Фаза 1: Foundation
          |                    - File Transfer
          |                    - Screen Sharing
          |                    - Recording
          |                    - Waiting Room
2025 Q2:  ████████████████████ Фаза 2: Collaboration
          |                    - Whiteboard
          |                    - Interactive
          |                    - Breakout Rooms
          |
2025 Q3:  ████████████████████ Фаза 3: AI
          |                    - Virtual Background
          |                    - Transcription
          |                    - Translation
          |
2025 Q4:  ████████████████████ Фаза 4: Enterprise
          |                    - E2E Encryption
          |                    - Live Streaming
          |                    - Analytics
          |
2026 Q1:  ████████████████████ Production Ready!
```

---

## 🚀 Следующие шаги

1. **Выберите стартовую функцию** из Фазы 1
2. **Создайте branch** feature/file-transfer
3. **Создайте структуру** experimental/fileshare/
4. **Напишите API** (интерфейсы)
5. **Реализуйте MVP** (минимальный прототип)
6. **Добавьте тесты** (TDD approach)
7. **Зарегистрируйте** в features/registry.go
8. **Тестируйте** и итерируйте
9. **Документируйте** использование
10. **Merge в main**

Удачи! 🎉
