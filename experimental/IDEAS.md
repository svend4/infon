# История протоколов VoIP и видеотелефонии (1990-2024)

## 📅 Временная линия развития

### 1990-1995: Зарождение

**1991 - Speak Freely**
- Первая VoIP программа для Internet
- Только голос через IP
- Unix системы
```
Функции:
- Аудио кодеки: GSM, LPC
- Peer-to-peer соединения
- Без шифрования
```

**1993 - IETF RTP (RFC 1889)**
- Real-time Transport Protocol
- Основа для всех современных систем
```
Компоненты:
- RTP: передача медиа
- RTCP: контроль качества
- Синхронизация аудио/видео
```

**1995 - VocalTec Internet Phone**
- Первый коммерческий VoIP продукт
- Dial-up модемы (14.4 kbps)
```
Ограничения:
- Только PC → PC
- Плохое качество
- Высокая задержка (>1s)
```

### 1996-2000: Стандартизация

**1996 - H.323**
- ITU-T стандарт для видеоконференций
- Сложный протокол
```
H.323 Stack:
├── H.225: Сигнализация
├── H.245: Управление каналами
├── H.261/H.263: Видео кодеки
├── G.711/G.723: Аудио кодеки
└── RTP/RTCP: Транспорт
```

**1996 - Microsoft NetMeeting**
- Массовый продукт для Windows
```
Функции:
✅ Видеозвонки
✅ Чат
✅ Whiteboard (доска)
✅ File transfer
✅ Application sharing
✅ Desktop sharing
```

**1998 - CU-SeeMe**
- Первая популярная видеоконференция
```
Инновации:
- Reflector серверы (аналог SFU)
- Групповые видеозвонки
- Низкая скорость (28.8 kbps модемы)
```

**1999 - SIP (RFC 2543)**
- Session Initiation Protocol
- Конкурент H.323
```
Преимущества над H.323:
✅ Проще
✅ Текстовый протокол
✅ Расширяемый
✅ HTTP-подобный
```

### 2000-2005: Коммерциализация

**2003 - Skype**
- Революция в VoIP
```
Инновации:
✅ P2P архитектура
✅ NAT traversal (STUN/TURN)
✅ Проприетарные кодеки
✅ Отличное качество даже на dial-up
✅ Бесплатные звонки
```

**2004 - Google Talk (Jabber/XMPP)**
```
Функции:
- VoIP через XMPP
- Видеочат
- Интеграция с Gmail
- Федерация (межсерверная)
```

**2005 - WebRTC предшественники**
- Flash Media Server
- Adobe Flash P2P
```
Ограничения Flash:
❌ Плагин нужен
❌ Проблемы с безопасностью
❌ Медленный
```

### 2011-2015: WebRTC эра

**2011 - WebRTC**
- Google открывает исходники
```
Компоненты:
├── getUserMedia: Захват медиа
├── RTCPeerConnection: P2P соединения
├── RTCDataChannel: Передача данных
├── VP8/VP9: Видео кодеки
├── Opus: Аудио кодек
└── DTLS/SRTP: Шифрование
```

**2013 - Discord**
```
Функции:
- VoIP для геймеров
- Низкая задержка
- Групповые звонки
- Screen sharing
```

### 2016-2020: Современные системы

**2016 - Zoom**
```
Инновации:
- SFU архитектура
- Виртуальные фоны
- Breakout rooms
- Recording
- Live streaming
```

**2020 - Pandemic boom**
- Массовое использование видеоконференций
```
Новые функции:
- AI фоновый шум
- Виртуальные фоны AI
- Live transcription
- Translation
- Gesture recognition
```

## 🔧 Что было реализовано в этих системах

### 1. Базовые функции (есть везде)

```
✅ Аудио звонки (P2P)
✅ Видео звонки (P2P)
✅ Текстовый чат
✅ Групповые звонки
✅ Contact list
✅ Presence (онлайн/офлайн/занят)
```

### 2. Расширенные функции (1996-2000)

**NetMeeting (1996):**
```
├── Whiteboard - Общая доска для рисования
│   - Рисование в реальном времени
│   - Маркеры, фигуры, текст
│   - Совместное редактирование
│
├── Application Sharing - Показ приложений
│   - Управление приложением удалённо
│   - Передача только изменений экрана
│
├── File Transfer - Передача файлов
│   - Во время звонка
│   - Фоновая передача
│
└── Desktop Sharing - Удалённый рабочий стол
    - Полный контроль
    - Передача ввода клавиатуры/мыши
```

**CU-SeeMe (1998):**
```
├── Reflector Servers - MCU серверы
│   - Групповые конференции
│   - Mixing аудио
│   - Тайловое видео
│
├── Lurker Mode - Только просмотр
│   - Без камеры
│   - Экономия трафика
│
└── Conference Directory - Каталог конференций
    - Публичные комнаты
    - Поиск по темам
```

### 3. Телефонные функции (2000-2005)

**Asterisk PBX (2002):**
```
├── IVR (Interactive Voice Response)
│   - Голосовое меню
│   - DTMF навигация
│   - Text-to-Speech
│   - Speech Recognition
│
├── Voicemail
│   - Запись сообщений
│   - Email уведомления
│   - Voicemail-to-text
│
├── Call Features
│   - Call forwarding
│   - Call transfer (blind/attended)
│   - Call parking
│   - Call pickup
│   - Conference bridge
│   - Music on hold
│   - Call recording
│   - Call queues (ACD)
│
└── PSTN Integration
    - Звонки на обычные телефоны
    - SIP trunking
    - E.164 номера
```

### 4. Корпоративные функции (2005-2010)

**Microsoft Lync/OCS:**
```
├── Enterprise Voice
│   - PBX замена
│   - Корпоративная телефония
│   - Extension dialing
│
├── Collaboration
│   - Screen sharing
│   - PowerPoint presentation
│   - Polls (опросы)
│   - Q&A
│   - Hand raising
│
├── Federation
│   - Межкорпоративная связь
│   - XMPP gateway
│   - Skype connectivity
│
└── Integration
    - Outlook calendar
    - SharePoint
    - Exchange voicemail
    - Active Directory
```

### 5. Современные функции (2015-2024)

**Zoom/Teams/Meet:**
```
├── AI Features
│   - Virtual backgrounds (AI)
│   - Background blur
│   - Noise suppression (AI)
│   - Echo cancellation (ML)
│   - Auto-framing (следит за лицом)
│   - Beauty filters
│   - Gesture recognition
│
├── Content Sharing
│   - Screen sharing
│   - Specific window
│   - Portion of screen
│   - Dual screen share
│   - iPhone/iPad screen
│   - Remote control
│
├── Recording & Streaming
│   - Local recording
│   - Cloud recording
│   - Live streaming (YouTube/Facebook)
│   - Live transcription
│   - Auto-generated captions
│   - Translation (real-time)
│
├── Meetings Features
│   - Breakout rooms
│   - Waiting room
│   - Meeting templates
│   - Scheduling
│   - Recurring meetings
│   - Webinars
│   - Large meetings (100+ participants)
│
├── Accessibility
│   - Closed captions
│   - Sign language view
│   - Keyboard shortcuts
│   - Screen reader support
│   - High contrast mode
│
└── Security
    - E2E encryption
    - Waiting room
    - Password protection
    - Meeting lock
    - Participant permissions
    - Watermarks
```

## 🎯 Что МОЖНО добавить в TVCP (идеи)

### Категория A: Базовые (легко, используя существующий код)

#### 1. File Transfer (Передача файлов)
```go
// experimental/fileshare/transfer.go
Сложность: ⭐⭐
Время: 2-3 дня

Функции:
- Отправка файлов во время звонка
- Progress bar
- Resume support
- Throttling (ограничение скорости)

Технологии:
- UDP для передачи
- Chunking больших файлов
- Hash verification (SHA256)
```

#### 2. Screen Sharing (Демонстрация экрана)
```go
// experimental/screen/capture.go
Сложность: ⭐⭐⭐
Время: 5-7 дней

Функции:
- Захват всего экрана
- Выбор окна
- Выбор области
- Cursor tracking
- Annotations (рисование)

Технологии:
- Windows: DXGI Desktop Duplication
- Linux: X11 XShm или Wayland
- macOS: CGDisplayStream
- Differential encoding (только изменения)
```

#### 3. Recording (Запись звонков)
```go
// experimental/recording/recorder.go
Сложность: ⭐⭐
Время: 3-4 дня

Функции:
- Запись аудио
- Запись видео
- Запись экрана
- Mixing нескольких источников

Форматы:
- MP4 (H.264 + AAC)
- WebM (VP8/VP9 + Opus)
- Audio-only (MP3, Opus)
```

### Категория B: Средние (требуют интеграции)

#### 4. Virtual Background (Виртуальный фон)
```go
// experimental/video/background.go
Сложность: ⭐⭐⭐⭐
Время: 1-2 недели

Функции:
- Background blur
- Image replacement
- Video background
- Segmentation (человек vs фон)

Технологии:
- MediaPipe (Google)
- TensorFlow Lite
- Body segmentation model
- OpenGL/Metal для compositing
```

#### 5. Live Streaming (Трансляция)
```go
// experimental/streaming/rtmp.go
Сложность: ⭐⭐⭐
Время: 1 неделя

Функции:
- Stream to YouTube
- Stream to Twitch
- Stream to custom RTMP
- Scene composition

Технологии:
- RTMP protocol
- OBS integration
- FFmpeg для encoding
```

#### 6. Whiteboard (Совместная доска)
```go
// experimental/whiteboard/canvas.go
Сложность: ⭐⭐⭐
Время: 1 неделя

Функции:
- Рисование (ручка, маркер)
- Фигуры (прямоугольник, круг, стрелка)
- Текст
- Ластик
- Цвета
- Слои
- Undo/Redo
- Export to PNG/PDF

Технологии:
- Canvas синхронизация через WebSocket
- Vector graphics
- Conflict resolution (CRDT)
```

### Категория C: Сложные (новые системы)

#### 7. AI Features (AI функции)
```go
// experimental/ai/
Сложность: ⭐⭐⭐⭐⭐
Время: 2-4 недели

├── transcription/ - Транскрипция речи
│   - Speech-to-Text (Whisper)
│   - Real-time captions
│   - Multi-language
│
├── translation/ - Перевод
│   - Real-time translation
│   - Text-to-Speech (переведённый)
│
├── noise_ai/ - AI шумоподавление
│   - RNNoise
│   - Deep learning models
│   - Voice enhancement
│
└── background_ai/ - AI фон
    - Segmentation models
    - Style transfer
    - Face filters
```

#### 8. Spatial Audio (Пространственный звук)
```go
// experimental/audio/spatial.go
Сложность: ⭐⭐⭐⭐
Время: 2 недели

Функции:
- 3D позиционирование участников
- HRTF (Head-Related Transfer Function)
- Distance attenuation
- Reverb/room simulation

Применение:
- Виртуальные конференции
- VR meetings
- Gaming
```

#### 9. Breakout Rooms (Подгруппы)
```go
// server/breakout/
Сложность: ⭐⭐⭐⭐
Время: 1-2 недели

Функции:
- Создание подкомнат
- Автоматическое распределение
- Timer для комнат
- Broadcast to all rooms
- Return to main room

Архитектура:
- Динамические SFU комнаты
- Room migration
- State management
```

#### 10. Interactive Elements (Интерактив)
```go
// experimental/interactive/
Сложность: ⭐⭐⭐
Время: 1 неделя каждый

├── polls.go - Опросы
│   - Multiple choice
│   - Rating scale
│   - Live results
│
├── qa.go - Вопросы и ответы
│   - Submit questions
│   - Upvoting
│   - Moderation
│
├── reactions.go - Реакции
│   - Emoji reactions
│   - Hand raising
│   - Thumbs up/down
│
└── quiz.go - Викторины
    - Questions
    - Scoring
    - Leaderboard
```

### Категория D: Телефонные (интеграция с PSTN)

#### 11. VoIP Gateway
```go
// server/gateway/
Сложность: ⭐⭐⭐⭐⭐
Время: 1 месяц

Функции:
- SIP trunk интеграция
- Звонки на обычные телефоны
- Получение звонков с телефонов
- DTMF (тоновый набор)
- Caller ID

Технологии:
- FreeSWITCH/Asterisk интеграция
- SIP protocol
- RTP <-> ваш протокол bridge
```

#### 12. IVR System
```go
// experimental/ivr/
Сложность: ⭐⭐⭐⭐
Время: 2 недели

Функции:
- Voice menu (Нажмите 1 для...)
- Speech recognition
- Text-to-Speech
- Call routing
- Queue management

Компоненты:
- Menu builder (XML/JSON)
- DTMF decoder
- Audio prompts library
- Google/Amazon TTS API
```

#### 13. Call Center Features
```go
// experimental/callcenter/
Сложность: ⭐⭐⭐⭐⭐
Время: 1-2 месяца

├── queue.go - Очереди звонков
├── agent.go - Агенты
├── supervisor.go - Супервизоры
├── analytics.go - Аналитика
└── crm.go - CRM интеграция

Функции:
- ACD (Automatic Call Distribution)
- Skills-based routing
- Call monitoring
- Call whispering
- Call barging
- Analytics dashboard
```

### Категория E: Безопасность и конфиденциальность

#### 14. End-to-End Encryption
```go
// experimental/crypto/e2ee.go
Сложность: ⭐⭐⭐⭐⭐
Время: 2-3 недели

Функции:
- Signal Protocol
- Key exchange (ECDH)
- Perfect Forward Secrecy
- Authenticated encryption
- Key fingerprint verification

Проблемы:
- SFU несовместим с E2E (сервер не видит медиа)
- Нужен специальный режим
```

#### 15. Waiting Room / Security
```go
// experimental/security/
Сложность: ⭐⭐⭐
Время: 1 неделя

Функции:
- Waiting room (одобрение входа)
- Password protection
- Meeting lock
- Remove participants
- Mute all
- Disable camera for all
- Permissions system
```

### Категория F: Accessibility (Доступность)

#### 16. Closed Captions
```go
// experimental/captions/
Сложность: ⭐⭐⭐⭐
Время: 1-2 недели

Функции:
- Real-time transcription
- Multi-language
- Speaker identification
- Export transcript

Технологии:
- Google Cloud Speech-to-Text
- Azure Speech Services
- OpenAI Whisper (offline)
```

#### 17. Sign Language View
```go
// experimental/accessibility/
Сложность: ⭐⭐⭐
Время: 1 неделя

Функции:
- Pinned interpreter video
- Picture-in-picture
- High frame rate (60fps)
- High quality encode
```

### Категория G: VR/AR (Будущее)

#### 18. Virtual Reality Meetings
```go
// experimental/vr/
Сложность: ⭐⭐⭐⭐⭐
Время: 2+ месяца

Функции:
- 3D avatars
- Spatial audio
- Hand tracking
- Virtual environments
- WebXR integration

Платформы:
- Meta Quest
- HTC Vive
- Apple Vision Pro
```

## 📊 Приоритизация для TVCP

### Рекомендуемый порядок добавления:

**Фаза 1: Quick Wins (1-2 месяца)**
```
1. ✅ File Transfer (полезно, легко)
2. ✅ Recording (востребовано)
3. ✅ Screen Sharing (must-have)
4. ✅ Basic Security (waiting room, passwords)
```

**Фаза 2: Collaboration (2-3 месяца)**
```
5. ✅ Whiteboard
6. ✅ Polls and Reactions
7. ✅ Q&A
8. ✅ Breakout Rooms
```

**Фаза 3: AI Enhancement (3-4 месяца)**
```
9. ✅ Virtual Background (AI)
10. ✅ Live Transcription
11. ✅ Translation
12. ✅ Advanced Noise Cancellation
```

**Фаза 4: Enterprise (4-6 месяцев)**
```
13. ✅ E2E Encryption
14. ✅ Live Streaming
15. ✅ Analytics
16. ✅ Integrations (Calendar, etc.)
```

**Фаза 5: Advanced (6+ месяцев)**
```
17. ✅ VoIP Gateway
18. ✅ IVR System
19. ✅ VR/AR Support
20. ✅ Spatial Audio
```

## 🎯 Структура experimental/ для новых функций

```
experimental/
├── features/
│   └── registry.go          ← Существующий реестр
│
├── audio/                   ← Существующие
│   ├── webrtc_processor.go
│   ├── fft_filter.go
│   └── advanced_aec.go
│
├── codec/                   ← Существующие
│   └── hardware_accel.go
│
├── fileshare/              ← НОВОЕ
│   ├── transfer.go
│   ├── chunking.go
│   └── verification.go
│
├── screen/                 ← НОВОЕ
│   ├── capture.go
│   ├── capture_windows.go
│   ├── capture_linux.go
│   ├── capture_darwin.go
│   └── encoder.go
│
├── recording/              ← НОВОЕ
│   ├── recorder.go
│   ├── mixer.go
│   └── formats/
│       ├── mp4.go
│       └── webm.go
│
├── whiteboard/             ← НОВОЕ
│   ├── canvas.go
│   ├── shapes.go
│   ├── sync.go
│   └── export.go
│
├── streaming/              ← НОВОЕ
│   ├── rtmp.go
│   ├── youtube.go
│   └── twitch.go
│
├── ai/                     ← НОВОЕ
│   ├── transcription/
│   ├── translation/
│   ├── background/
│   └── noise/
│
├── interactive/            ← НОВОЕ
│   ├── polls.go
│   ├── qa.go
│   └── reactions.go
│
├── security/               ← НОВОЕ
│   ├── e2ee.go
│   ├── waitingroom.go
│   └── permissions.go
│
└── vr/                     ← НОВОЕ (далёкое будущее)
    ├── avatar.go
    ├── spatial.go
    └── webxr.go
```

## 📖 Исторические уроки

### Что работало хорошо:
✅ **Простота** (Skype был проще H.323)
✅ **Качество** (хороший звук важнее видео)
✅ **NAT traversal** (работало через файрволы)
✅ **Бесплатность** (vs платная телефония)

### Что не работало:
❌ **Сложность** (H.323 был слишком сложным)
❌ **Закрытость** (проприетарные протоколы умирают)
❌ **Централизация** (P2P лучше для privacy)
❌ **Плагины** (Flash умер, WebRTC победил)

### Выводы для TVCP:
1. Начните с простого (аудио/видео/чат)
2. Добавляйте функции постепенно
3. Используйте открытые стандарты
4. Фокус на качество, не количество функций
5. Experimental → Beta → Core pipeline

## 🚀 Следующие шаги

1. **Выберите 2-3 функции** из Фазы 1
2. **Создайте папки** в experimental/
3. **Спланируйте API** для каждой функции
4. **Реализуйте MVP** (минимально работающий прототип)
5. **Добавьте в Registry** с уровнем Alpha
6. **Тестируйте** и улучшайте
7. **Продвигайте** Alpha → Beta → Candidate → Core

Удачи! 🎉
