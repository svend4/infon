# Архитектура проекта TVCP (Monorepo)

## Структура репозитория

```
infon/
├── core/                          # Основное ядро (стабильное)
│   ├── audio/                     # Базовая аудио-подсистема
│   │   ├── capture.go            # Захват аудио
│   │   ├── playback.go           # Воспроизведение
│   │   ├── audio_linux.go        # ALSA (Linux)
│   │   ├── audio_darwin.go       # CoreAudio (macOS)
│   │   └── audio_windows.go      # WASAPI (Windows) ✅
│   ├── video/                     # Базовая видео-подсистема
│   │   ├── capture.go
│   │   └── display.go
│   ├── network/                   # Сетевая подсистема
│   │   ├── udp.go
│   │   └── protocol.go
│   └── codec/                     # Базовые кодеки
│       ├── codec.go
│       ├── h264.go
│       └── opus.go
│
├── experimental/                  # Экспериментальные функции
│   ├── audio/                     # Продвинутая аудио-обработка
│   │   ├── webrtc_processor.go   # WebRTC AEC/NS/AGC ✅
│   │   ├── fft_filter.go         # FFT фильтры ✅
│   │   └── advanced_aec.go       # NLMS/RLS алгоритмы ✅
│   ├── codec/                     # Аппаратное ускорение
│   │   └── hardware_accel.go     # GPU кодеки ✅
│   └── features.go                # Регистр экспериментальных функций
│
├── server/                        # Серверные компоненты
│   ├── sfu/                       # SFU сервер ✅
│   │   ├── sfu_server.go
│   │   └── sfu_server_test.go
│   ├── signaling/                 # Сигнальный сервер
│   └── turn/                      # TURN/STUN сервер
│
├── mobile/                        # Мобильные приложения
│   ├── ios/                       # iOS приложение ✅
│   │   ├── TVCPApp.swift
│   │   └── CallManager.swift
│   └── android/                   # Android приложение ✅
│       ├── MainActivity.kt
│       └── CallManager.kt
│
├── desktop/                       # Десктопные приложения
│   ├── main.go                    # Основной клиент
│   └── ui/                        # GUI интерфейс
│
├── pkg/                           # Общие пакеты (библиотеки)
│   ├── protocol/                  # Сетевой протокол
│   ├── utils/                     # Утилиты
│   └── logger/                    # Логирование
│
├── cmd/                           # Исполняемые файлы
│   ├── client/                    # Клиент
│   ├── server/                    # Сервер
│   └── tools/                     # Утилиты
│
├── internal/                      # Приватные пакеты (текущая структура)
│   ├── audio/                     # Миграция в core/experimental
│   ├── codec/                     # Миграция в core/experimental
│   └── sfu/                       # Миграция в server/
│
└── docs/                          # Документация
    ├── architecture.md
    ├── experimental-features.md
    └── api/
```

## Принципы организации

### 1. **Core (Основное ядро)**
- Стабильные, проверенные функции
- Минимальные зависимости
- Обязательные для работы программы
- Тщательное тестирование перед добавлением
- Semantic versioning

### 2. **Experimental (Экспериментальные)**
- Новые функции в разработке
- Могут иметь нестабильное API
- Включаются через build tags
- Не влияют на core если отключены
- Флаг `-tags experimental` для сборки

### 3. **Server (Серверные компоненты)**
- Независимые микросервисы
- Могут деплоиться отдельно
- Собственные циклы релизов

### 4. **Mobile (Мобильные приложения)**
- Полностью изолированные приложения
- Используют core через C API (cgo)
- Отдельные репозитории или submodules

## Build Tags для экспериментальных функций

### Пример использования:

```go
// experimental/audio/webrtc_processor.go
//go:build experimental
// +build experimental

package audio

// WebRTC процессинг доступен только с тегом experimental
```

### Сборка:

```bash
# Только core функции
go build ./cmd/client

# С экспериментальными функциями
go build -tags experimental ./cmd/client

# Все функции + отладка
go build -tags "experimental debug" ./cmd/client
```

## Go Workspace для монорепо

### go.work:
```
go 1.24

use (
    ./core
    ./experimental
    ./server
    ./desktop
    ./pkg
)
```

### Преимущества:
- Независимые версии модулей
- Локальная разработка без replace
- Изоляция зависимостей
- Простое управление версиями

## Регистр экспериментальных функций

### experimental/features.go:
```go
package experimental

// Feature представляет экспериментальную функцию
type Feature struct {
    Name        string
    Description string
    Enabled     bool
    Stability   Stability
}

type Stability int

const (
    Alpha      Stability = iota // Нестабильно, может сломаться
    Beta                        // Относительно стабильно
    Candidate                   // Готово к переходу в core
)

var Registry = map[string]Feature{
    "webrtc-processing": {
        Name:        "WebRTC Audio Processing",
        Description: "AEC3, noise suppression, AGC, VAD",
        Stability:   Beta,
    },
    "fft-filters": {
        Name:        "FFT Bandpass Filters",
        Description: "Frequency-domain filtering",
        Stability:   Beta,
    },
    "advanced-aec": {
        Name:        "Advanced AEC (NLMS/RLS)",
        Description: "Sophisticated echo cancellation",
        Stability:   Alpha,
    },
    "hardware-accel": {
        Name:        "Hardware Acceleration",
        Description: "GPU-accelerated video codecs",
        Stability:   Alpha,
    },
}
```

## Миграция текущего кода

### План миграции:

1. **Создать новую структуру:**
   ```bash
   mkdir -p core/audio core/codec
   mkdir -p experimental/audio experimental/codec
   mkdir -p server/sfu
   ```

2. **Переместить файлы:**
   ```bash
   # Core
   mv internal/audio/audio_*.go core/audio/

   # Experimental
   mv internal/audio/webrtc_processor.go experimental/audio/
   mv internal/audio/fft_filter.go experimental/audio/
   mv internal/audio/advanced_aec.go experimental/audio/
   mv internal/codec/hardware_accel.go experimental/codec/

   # Server
   mv internal/sfu/* server/sfu/
   ```

3. **Обновить импорты:**
   ```bash
   # Автоматическое обновление импортов
   gofmt -w -r 'github.com/svend4/infon/internal/audio -> github.com/svend4/infon/core/audio' .
   ```

4. **Добавить build tags:**
   ```bash
   # Добавить //go:build experimental в начало файлов
   ```

## Конфигурация функций

### config.yaml:
```yaml
# Основные настройки
core:
  audio:
    sample_rate: 16000
    channels: 1

# Экспериментальные функции (опционально)
experimental:
  # WebRTC обработка
  webrtc_processing:
    enabled: true
    aec: true
    noise_suppression: true
    agc: true
    vad: true

  # FFT фильтры
  fft_filters:
    enabled: false  # Отключено по умолчанию
    bandpass: [300, 3400]

  # Продвинутый AEC
  advanced_aec:
    enabled: false  # Экспериментально
    algorithm: "nlms"  # или "rls"

  # Аппаратное ускорение
  hardware_accel:
    enabled: true
    auto_detect: true
    fallback: true
```

## Документация

### experimental-features.md:
```markdown
# Экспериментальные функции

## Как включить

1. Сборка с тегом:
   ```bash
   go build -tags experimental
   ```

2. Включение в конфиге:
   ```yaml
   experimental:
     webrtc_processing:
       enabled: true
   ```

## Список функций

### WebRTC Processing (Beta)
- **Статус**: Относительно стабильно
- **Тесты**: 31/31 ✅
- **Производительность**: 60μs/фрейм
- **Риски**: Может увеличить задержку

### FFT Filters (Beta)
- **Статус**: Стабильно
- **Тесты**: 24/24 ✅
- **Производительность**: 1.8ms/100ms
- **Риски**: Высокая нагрузка на CPU

### Advanced AEC - NLMS/RLS (Alpha)
- **Статус**: Экспериментально
- **Тесты**: 23/23 ✅
- **Производительность**: NLMS: 609μs, RLS: 41ms
- **Риски**: RLS не для real-time

### Hardware Acceleration (Alpha)
- **Статус**: В разработке
- **Тесты**: 23/23 ✅
- **Риски**: Требует GPU, может не работать
```

## Преимущества такой структуры

✅ **Изоляция**: Экспериментальные функции не влияют на core
✅ **Гибкость**: Легко включать/выключать функции
✅ **Безопасность**: Core остаётся стабильным
✅ **Масштабируемость**: Легко добавлять новые компоненты
✅ **Тестирование**: Раздельное тестирование core и experimental
✅ **Документация**: Чёткое разделение что стабильно, что нет
✅ **CI/CD**: Разные пайплайны для разных частей

## Рекомендации

1. **Core функции**: Только после тщательного тестирования
2. **Experimental**: Помечать уровень стабильности (Alpha/Beta)
3. **Deprecation**: Удалять неиспользуемые experimental через 3 релиза
4. **Graduation**: Переводить Beta→Core после 3 месяцев стабильности
5. **Документация**: Каждая experimental функция должна иметь README
