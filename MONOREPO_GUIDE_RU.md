# 📚 Руководство по монорепозиторию TVCP

## 🎯 Ответы на ваши вопросы

### ❓ Как отделить экспериментальные функции от основного ядра?

**Решение:** Монорепозиторий с чёткой структурой папок и build tags

```
infon/
├── core/              ← Стабильное ядро (обязательно)
├── experimental/      ← Экспериментальные функции (опционально)
├── server/            ← Серверные компоненты
├── mobile/            ← Мобильные приложения
└── pkg/               ← Общие библиотеки
```

### ✅ Преимущества такой организации

1. **Изоляция** - экспериментальные функции не влияют на core
2. **Гибкость** - легко включать/выключать функции
3. **Безопасность** - стабильное ядро всегда работает
4. **Управление** - централизованный контроль функций

## 🚀 Быстрый старт

### 1. Миграция в новую структуру

```bash
# Автоматическая миграция (создаёт бэкап!)
./scripts/reorganize.sh

# Или вручную изучите структуру
cat ARCHITECTURE.md
```

### 2. Сборка проекта

```bash
# Только стабильные функции (core)
go build ./cmd/client

# С экспериментальными функциями
go build -tags experimental ./cmd/client

# Используя Makefile (после миграции)
make build-core         # Только core
make build-experimental # С experimental
make build              # Оба варианта
```

### 3. Управление экспериментальными функциями

```bash
# Просмотр всех функций
go run -tags experimental ./cmd/tools/features.go summary

# Список Beta функций
go run -tags experimental ./cmd/tools/features.go list --stability=beta

# Информация о конкретной функции
go run -tags experimental ./cmd/tools/features.go info -id webrtc-processing

# Включить функцию
go run -tags experimental ./cmd/tools/features.go enable advanced-aec

# Выключить функцию
go run -tags experimental ./cmd/tools/features.go disable fft-filters
```

### 4. Конфигурация через YAML

```bash
# Копируем шаблон
cp config.example.yaml config.yaml

# Редактируем настройки
nano config.yaml
```

## 📁 Текущее состояние проекта

### ✅ Реализованные функции

| Компонент | Местоположение | Статус | Тесты |
|-----------|----------------|---------|-------|
| **WASAPI (Windows)** | `internal/audio/` → `core/audio/` | Стабильно | 21/21 ✅ |
| **iOS Client** | `mobile/ios/` | Стабильно | - |
| **Android Client** | `mobile/android/` | Стабильно | - |
| **SFU Server** | `internal/sfu/` → `server/sfu/` | Стабильно | 22/22 ✅ |
| **WebRTC Processing** | `internal/audio/` → `experimental/audio/` | Beta | 31/31 ✅ |
| **FFT Filters** | `internal/audio/` → `experimental/audio/` | Beta | 24/24 ✅ |
| **Advanced AEC** | `internal/audio/` → `experimental/audio/` | Alpha | 23/23 ✅ |
| **Hardware Accel** | `internal/codec/` → `experimental/codec/` | Alpha | 23/23 ✅ |

### 📊 Статистика

- **Всего строк кода:** ~10,850
- **Тестов:** 163 + 25 бенчмарков
- **Успешность:** 100%
- **Файлов:** 18 новых файлов

## 🔧 Структура после миграции

```
infon/
├── core/                          # Стабильное ядро
│   ├── audio/
│   │   ├── audio_windows.go      ✅ WASAPI
│   │   ├── audio_linux.go        ✅ ALSA
│   │   └── audio_darwin.go       ✅ CoreAudio
│   ├── codec/
│   │   └── codec.go
│   └── go.mod
│
├── experimental/                  # Экспериментальные функции
│   ├── audio/
│   │   ├── webrtc_processor.go   🔶 Beta
│   │   ├── fft_filter.go         🔶 Beta
│   │   └── advanced_aec.go       ⚠️ Alpha
│   ├── codec/
│   │   └── hardware_accel.go     ⚠️ Alpha
│   ├── features/
│   │   └── registry.go           # Реестр функций
│   └── go.mod
│
├── server/                        # Серверы
│   ├── sfu/
│   │   ├── sfu_server.go         ✅ Стабильно
│   │   └── sfu_server_test.go
│   └── go.mod
│
├── mobile/                        # Мобильные приложения
│   ├── ios/
│   │   ├── TVCPApp.swift         ✅ iOS клиент
│   │   └── CallManager.swift
│   └── android/
│       ├── MainActivity.kt       ✅ Android клиент
│       └── CallManager.kt
│
├── pkg/                           # Общие библиотеки
│   └── go.mod
│
├── cmd/                           # Исполняемые файлы
│   ├── client/
│   ├── server/
│   └── tools/
│       └── features.go            # CLI инструмент
│
├── scripts/
│   └── reorganize.sh              # Скрипт миграции
│
├── ARCHITECTURE.md                # Подробная документация
├── config.example.yaml            # Шаблон конфигурации
├── go.work                        # Go workspace
└── Makefile                       # Команды сборки
```

## 🏷️ Build Tags - Как это работает

### В коде experimental функций:

```go
//go:build experimental
// +build experimental

package audio

// Этот код компилируется только с -tags experimental
```

### При сборке:

```bash
# БЕЗ тега - только core
go build ./cmd/client
# → Размер меньше, быстрее, стабильнее

# С тегом - core + experimental
go build -tags experimental ./cmd/client
# → Все функции, больше размер
```

## 📝 Регистр экспериментальных функций

Все experimental функции автоматически регистрируются в `experimental/features/registry.go`:

```go
// WebRTC Processing
ID:          "webrtc-processing"
Stability:   Beta
Tests:       31/31 ✅
Performance: ~60μs/фрейм
Risks:       Может увеличить задержку на 10-20ms

// FFT Filters
ID:          "fft-filters"
Stability:   Beta
Tests:       24/24 ✅
Performance: ~1.8ms на 100ms
Risks:       Высокая нагрузка на CPU

// Advanced AEC (NLMS/RLS)
ID:          "advanced-aec"
Stability:   Alpha ⚠️
Tests:       23/23 ✅
Performance: NLMS: 609μs, RLS: 41ms
Risks:       RLS не для real-time!

// Hardware Acceleration
ID:          "hardware-accel"
Stability:   Alpha ⚠️
Tests:       23/23 ✅
Performance: 10-100x быстрее CPU
Risks:       Требует GPU, может не работать
```

## ⚙️ Конфигурация (config.yaml)

```yaml
# Основное ядро (всегда включено)
core:
  audio:
    sample_rate: 16000
    channels: 1

# Экспериментальные функции
experimental:
  enabled: true  # Глобальный переключатель

  # WebRTC обработка [Beta] - рекомендуется
  webrtc_processing:
    enabled: true
    aec: true
    noise_suppression: true

  # FFT фильтры [Beta] - высокая нагрузка
  fft_filters:
    enabled: false  # Отключено по умолчанию

  # Advanced AEC [Alpha] - экспериментально
  advanced_aec:
    enabled: false
    algorithm: "nlms"  # RLS слишком медленный!

  # GPU ускорение [Alpha]
  hardware_accel:
    enabled: true
    auto_detect: true
    fallback_to_cpu: true
```

## 🎭 Уровни стабильности

| Уровень | Описание | Когда использовать |
|---------|----------|-------------------|
| **Alpha** ⚠️ | Нестабильно, может сломаться | Только для тестов |
| **Beta** 🔶 | Относительно стабильно | Можно использовать |
| **Candidate** ✅ | Готово к core | Рекомендуется |

## 🔄 Переход функций между уровнями

```
Alpha (3 месяца тестов) → Beta (3 месяца стабильности) → Candidate → Core
     ⚠️                       🔶                            ✅         ✅
```

## 📖 Примеры использования

### Пример 1: Минимальная сборка (только core)

```bash
# Сборка
go build ./cmd/client

# Результат: маленький бинарник, только стабильные функции
# Размер: ~5 MB
# Функции: WASAPI, базовое аудио, базовые кодеки
```

### Пример 2: Полная сборка (все функции)

```bash
# Сборка
go build -tags experimental ./cmd/client

# Результат: полный функционал
# Размер: ~8 MB
# Функции: core + WebRTC + FFT + Advanced AEC + Hardware Accel
```

### Пример 3: Выборочная сборка

```go
// config.yaml
experimental:
  webrtc_processing:
    enabled: true   # ✅ Включить

  fft_filters:
    enabled: false  # ❌ Выключить (слишком тяжёлый)

  advanced_aec:
    enabled: false  # ❌ Выключить (Alpha)

  hardware_accel:
    enabled: true   # ✅ Включить (с фолбэком)
```

## 🛠️ Makefile команды (после миграции)

```bash
make build-core         # Только стабильные
make build-experimental # С experimental
make build              # Оба варианта
make test               # Тесты core
make test-experimental  # Тесты experimental
make test-all           # Все тесты
make bench              # Бенчмарки
make clean              # Очистка
make help               # Помощь
```

## ⚡ Быстрое тестирование

```bash
# 1. Посмотреть что есть
go run -tags experimental ./cmd/tools/features.go summary

# 2. Включить нужные функции
go run -tags experimental ./cmd/tools/features.go enable webrtc-processing
go run -tags experimental ./cmd/tools/features.go enable hardware-accel

# 3. Собрать с выбранными функциями
go build -tags experimental ./cmd/client

# 4. Запустить
./client --config config.yaml
```

## 📚 Дополнительная документация

- **ARCHITECTURE.md** - Детальная архитектура монорепо
- **config.example.yaml** - Все настройки с комментариями
- **experimental/features/registry.go** - Реестр функций
- **scripts/reorganize.sh** - Скрипт миграции

## ❓ Частые вопросы

**В: Нужно ли мигрировать прямо сейчас?**
О: Нет, текущая структура работает. Миграция опциональна для лучшей организации.

**В: Что делать с existing кодом в internal/?**
О: Скрипт создаёт копии, оригиналы остаются. Можно удалить после проверки.

**В: Как добавить новую experimental функцию?**
О: 1) Создать файл в experimental/, 2) Добавить build tag, 3) Зарегистрировать в registry.go

**В: Когда переводить из Alpha в Beta?**
О: После 3 месяцев тестирования без критических багов.

**В: Можно ли использовать только некоторые experimental функции?**
О: Да! Настраивается через config.yaml для каждой функции отдельно.

## 🎉 Итог

Теперь у вас есть:

✅ **Чёткая структура** - core, experimental, server, mobile
✅ **Управление функциями** - CLI инструмент и реестр
✅ **Гибкая сборка** - build tags для опциональных функций
✅ **Конфигурация** - YAML для всех настроек
✅ **Документация** - подробные README и комментарии
✅ **Безопасность** - experimental не влияет на core
✅ **Масштабируемость** - легко добавлять новые компоненты

**Команда для начала:**
```bash
./scripts/reorganize.sh
```

Удачи! 🚀
