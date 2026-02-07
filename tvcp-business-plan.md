# TERMINAL VIDEO COMMUNICATION PLATFORM (TVCP)

## Бизнес-план, техническая документация и дорожная карта вывода на рынок

> На основе проектов [Say](https://github.com/svanichkin/say) и [.babe](https://github.com/svanichkin/babe)  
> Версия 1.0 — Февраль 2026

---

## Содержание

1. [Уникальная идея и ценностное предложение](#1-уникальная-идея-и-ценностное-предложение)
2. [Рыночная ниша и позиционирование](#2-рыночная-ниша-и-позиционирование)
3. [Анализ готовности к внедрению](#3-анализ-готовности-к-внедрению)
4. [Конкурентные преимущества и дополнение рынка](#4-конкурентные-преимущества-и-дополнение-рынка)
5. [Техническая архитектура прототипа](#5-техническая-архитектура-прототипа)
6. [Пошаговый план внедрения](#6-пошаговый-план-внедрения)
7. [Бизнес-модель и монетизация](#7-бизнес-модель-и-монетизация)
8. [Структура технической документации](#8-структура-технической-документации)
9. [Создание прототипа: пошаговое руководство](#9-создание-прототипа-пошаговое-руководство)
10. [Стратегия вывода на рынок](#10-стратегия-вывода-на-рынок)
11. [Итоговая дорожная карта](#11-итоговая-дорожная-карта)

---

## 1. Уникальная идея и ценностное предложение

### 1.1. Суть инновации

Terminal Video Communication Platform (TVCP) — это **первая в мире система видеосвязи, работающая полностью внутри текстового терминала**. Вместо передачи растровых пикселей (как Zoom, WebRTC, FaceTime), платформа кодирует видеокадры в Unicode-символы блочной графики (▀▄▌▐▖▗▘▝) с ANSI-раскраской, достигая визуально узнаваемого изображения при трафике в 10–20 раз меньше, чем у любого существующего решения.

> **💡 Ключевой инсайт:** Человеческий мозг распознаёт лицо, эмоции и жесты собеседника даже при разрешении 80×40 «пикселей». Это значит, что для базовой видеосвязи не нужны мегабиты трафика — достаточно 200 kbps, если использовать правильную кодировку.

### 1.2. Три столпа платформы

- **Блочный видеокодек (.babe):** Bi-Level Adaptive Block Encoding — собственный формат сжатия изображений, основанный на паттернах 2×2 субпикселя с 2 цветами на блок. Кодирует быстрее QOI и JPEG, при этом размер файла между ними. Идеально ложится на потоковое видео благодаря блочной структуре.

- **Yggdrasil P2P mesh-сеть:** Полностью децентрализованная, end-to-end зашифрованная IPv6 overlay-сеть. Не требует серверов, аккаунтов, DNS. Каждый узел идентифицируется криптографическим ключом. Самовосстанавливающаяся маршрутизация.

- **Терминальный рендеринг:** 16 глифов квадрантных блоков + 24-bit True Color ANSI escape-коды. Каждое знакоместо кодирует 4 субпикселя с 2 цветами. Работает в любом современном терминале (iTerm2, WezTerm, Windows Terminal, kitty, GNOME Terminal).

### 1.3. Формула уникальности

```
TVCP = блочный видеокодек + P2P mesh + терминальный рендеринг
```

Ни один из существующих продуктов не объединяет все три компонента. Zoom и WebRTC используют растровые кодеки + серверную инфраструктуру + GUI. Signal использует E2E шифрование, но требует серверы и мобильные клиенты. TVCP работает там, где не работает ничего другое.

---

## 2. Рыночная ниша и позиционирование

### 2.1. Где находится ниша

TVCP не конкурирует с Zoom, Teams или Google Meet на их поле. Платформа создаёт новую категорию — **«терминальная видеосвязь»** — и занимает пространство, которое невозможно закрыть существующими решениями.

| Параметр | Zoom / Teams | WebRTC (Jitsi) | Signal | **TVCP (Say)** |
|---|---|---|---|---|
| GUI / Браузер нужен | ✓ Обязательно | ✓ Браузер | ✓ Приложение | **✗ Терминал** |
| Минимальный канал | 1.8 Mbps | 500 kbps | 300 kbps (аудио) | **200 kbps (видео!)** |
| Центральный сервер | ✓ Zoom CDN | ✓ SFU сервер | ✓ Signal серверы | **✗ P2P** |
| Аккаунт / регистрация | ✓ Email | ✗ Нет | ✓ Телефон | **✗ Криптоключ** |
| Headless-сервер | ✗ | ✗ | ✗ | **✓** |
| IoT / Embedded | ✗ | Сложно | ✗ | **✓** |
| Работает без интернета | ✗ | Локально можно | ✗ | **✓ Mesh** |
| Стоимость инфраструктуры | $13+/мес | $5–50/мес серверы | Бесплатно | **Бесплатно** |

### 2.2. Целевые сегменты рынка

#### Сегмент A: DevOps / SRE / Cloud Infrastructure

- **Размер:** ~3.5 млн DevOps-инженеров в мире (Statista 2025). TAM: ~$500M/год при $12/мес.
- **Боль:** Управление headless-серверами, контейнерами, Kubernetes-кластерами. Все взаимодействия через терминал, но для обсуждения проблем приходится переключаться в Zoom/Slack.
- **Решение:** Видеозвонок прямо из терминальной сессии. Не нужно покидать рабочую среду. Интеграция с tmux, screen, kubectl.

#### Сегмент B: IoT / Embedded / Edge Computing

- **Размер:** ~15 млрд IoT-устройств к 2026 (IoT Analytics). Адресный рынок: устройства с камерой и Linux — ~200M. TAM: ~$2B при $10/устройство/год.
- **Боль:** Удалённый мониторинг устройств с камерой (промышленные, сельскохозяйственные, научные). Существующие IP-камеры требуют RTSP-серверы и широкий канал (2–5 Mbps).
- **Решение:** Видеопоток с IoT-камеры через Say при 200 kbps. Единый бинарник Go, кросс-компиляция под ARM, MIPS, RISC-V.

#### Сегмент C: Remote / Satellite / Extreme Conditions

- **Размер:** Спутниковая связь: $7.5B рынок к 2028 (Grand View Research). Адресный: организации с полевыми командами — нефтегаз, геология, морской транспорт, военные, научные экспедиции. TAM: ~$300M.
- **Боль:** Спутниковый трафик стоит $1–15/МБ. Час Zoom = $2700–8100. Даже голосовая связь через Iridium ограничена 2.4 kbps.
- **Решение:** Видеозвонок за ~90 МБ/час (~$90–450 на спутнике вместо тысяч). На каналах 200–500 kbps — единственное решение с видео.

#### Сегмент D: Privacy / Decentralized Communication

- **Размер:** Рынок приватной связи: $1.2B к 2027 (MarketsandMarkets). Пользователи Signal: ~40M, Tor: ~2.5M ежедневно.
- **Боль:** Даже Signal требует телефонный номер и центральные серверы. Факт связи фиксируется метаданными.
- **Решение:** Видеозвонок без аккаунта, без сервера, без метаданных. Интеграция с Tor через Yggdrasil.

#### Сегмент E: Military / Government / Critical Infrastructure

- **Размер:** Военные коммуникации: $35B рынок к 2028 (Allied Market Research).
- **Боль:** Тактическая связь в полевых условиях без зависимости от гражданской инфраструктуры. Устойчивость к глушению и перехвату.
- **Решение:** Автономная mesh-сеть + минимальный трафик + E2E шифрование + работа на любом устройстве с терминалом.

---

## 3. Анализ готовности к внедрению

### 3.1. Текущее состояние проекта Say

| Компонент | Статус | Готовность | Что нужно доработать |
|---|---|---|---|
| Аудиокодек G.722 | ✓ Работает | 🟢 90% | Шумоподавление, AEC |
| Видеокодек .babe | ✓ Работает | 🟢 80% | Межкадровое сжатие (P-frames) |
| Терминальный рендеринг | ✓ Работает | 🟢 85% | Адаптация под размер терминала |
| Yggdrasil интеграция | ✓ Работает | 🟢 90% | NAT traversal для корп. сетей |
| NAT Punchthrough | ✓ Через Ygg | 🟠 75% | Fallback через TURN-like relay |
| Контактная книга | ✓ Базовая | 🟠 60% | UI, поиск, группы |
| Групповые звонки | ✗ Нет | 🔴 10% | Mesh-топология, микширование |
| Мобильные клиенты | ✗ Нет | 🔴 0% | iOS/Android Yggdrasil + камера |
| Документация | Базовый README | 🔴 30% | API docs, user guide, man pages |
| Тесты / CI | GitHub Actions | 🟠 40% | Unit tests, integration, fuzzing |

### 3.2. Технический долг и риски

- **Зависимость от Yggdrasil:** Проект в alpha-стадии. Возможны breaking changes. *Митигация:* абстрактный сетевой слой, поддержка fallback через WireGuard/Tailscale.

- **Качество видео:** 80×40 «пикселей» — достаточно для распознавания лица, но не для демонстрации документов или экрана. *Митигация:* Sixel/Kitty protocol для терминалов с поддержкой.

- **Совместимость терминалов:** Не все терминалы поддерживают 24-bit True Color. Windows cmd.exe не поддерживает Unicode блоки корректно. *Митигация:* auto-detect capabilities, fallback to 256 colors / ASCII.

- **Аудиозадержка:** G.722 + UDP даёт ~50–150ms latency в зависимости от mesh-маршрута. Для разговора приемлемо, для музыки — нет.

---

## 4. Конкурентные преимущества и дополнение рынка

### 4.1. Чем TVCP дополняет существующие решения

TVCP не замещает Zoom/Teams, а **расширяет покрытие видеосвязи** в зоны, где эти решения физически не работают. Это комплементарный продукт.

| Зона покрытия | Zoom/Teams | TVCP | Модель использования |
|---|---|---|---|
| Офис, дом (WiFi/LAN) | ✓ Основное | — | Zoom для совещаний |
| Мобильные (4G/5G) | ✓ Приложение | — | Zoom/Teams мобильные |
| Серверная / headless | ✗ Невозможно | **✓ Основное** | TVCP для инцидентов |
| IoT / embedded | ✗ Невозможно | **✓ Основное** | TVCP для мониторинга |
| Спутник / узкий канал | ✗ Невозможно | **✓ Основное** | TVCP единственный вариант |
| Mesh без интернета | ✗ Невозможно | **✓ Основное** | TVCP для полевых условий |
| Максимальная приватность | Ограничено | **✓ Нативно** | TVCP для чувствительных задач |

### 4.2. Экономическое преимущество

| Метрика | Zoom Pro | Self-hosted Jitsi | TVCP |
|---|---|---|---|
| Стоимость ПО | $13.33/мес/юзер | Бесплатно | **Бесплатно (MIT)** |
| Серверная инфраструктура | Включена | $20–200/мес VPS | **$0 (P2P)** |
| TURN/STUN серверы | Включены | $5–50/мес | **$0 (Yggdrasil)** |
| Трафик/час (1:1 видео) | 540 MB – 1.6 GB | 360 MB – 900 MB | **90 MB** |
| Мин. требования к железу | i3 + 8GB RAM | i3 + 4GB RAM | **Любой ARM/x86 + 256MB RAM** |
| Установка | Скачать клиент | Docker + настройка | **`go build` (1 команда)** |
| Зависимости | Electron/Chromium | Node.js, Oasis | **Нет (единый бинарник)** |

---

## 5. Техническая архитектура прототипа

### 5.1. Общая архитектура системы

Система состоит из 5 основных модулей, каждый из которых может быть заменён или улучшен независимо.

```
┌──────────┐     ┌──────────────┐     ┌────────────┐     ┌───────────────┐
│  Camera  │────▶│ Video Codec  │────▶│ UDP Packet │────▶│  Yggdrasil   │
│          │     │   (.babe)    │     │  Builder   │     │  Mesh P2P    │
└──────────┘     └──────────────┘     └────────────┘     └───────┬───────┘
                                                                 │
┌──────────┐     ┌──────────────┐     ┌────────────┐             │
│   Mic    │────▶│ Audio Codec  │────▶│ UDP Packet │─────────────┘
│          │     │   (G.722)    │     │  Builder   │
└──────────┘     └──────────────┘     └────────────┘

                        ▼ Yggdrasil Mesh (Internet / LAN / WiFi Mesh) ▼

┌───────────────┐     ┌────────────┐     ┌──────────────┐     ┌──────────────┐
│  Yggdrasil   │────▶│ UDP Packet │────▶│   Video      │────▶│  Terminal    │
│  Mesh P2P    │     │  Parser    │     │   Decoder    │     │  Renderer   │
└───────┬───────┘     └────────────┘     └──────────────┘     │  (ANSI+▀▄)  │
        │             ┌────────────┐     ┌──────────────┐     └──────────────┘
        └────────────▶│ UDP Packet │────▶│   Audio      │────▶│  Speaker    │
                      │  Parser    │     │   Decoder    │     └──────────────┘
                      └────────────┘     └──────────────┘
```

### 5.2. Модуль захвата видео

- **Linux:** V4L2 (Video4Linux2) — прямой доступ к `/dev/video0`
- **macOS:** AVFoundation через cgo binding
- **Windows:** Media Foundation через cgo binding
- **Формат:** YUYV или MJPEG, конвертация в RGB24, ресайз до 160×120 или 80×60

### 5.3. Видеокодек .babe

Алгоритм кодирования кадра:

1. Разбить кадр на блоки 2×2 пикселя
2. Для каждого блока: кластеризация 4 пикселей в 2 группы (k-means, k=2)
3. Перебор 16 глифов-паттернов, выбор с минимальной ошибкой реконструкции
4. Кодирование: `[glyph_index: 4 bit] + [fg_color: RGB565 16 bit] + [bg_color: RGB565 16 bit]` = **36 bit/блок**
5. Delta-кодирование между кадрами: передавать только изменившиеся блоки
6. RLE-сжатие одинаковых блоков. Huffman для частых паттернов

**Расчёт битрейта (I-frame, 80×60):**

```
40×30 блоков = 1200 блоков × 36 бит = 43200 бит = 5.4 КБ

С delta-кодированием при 25 FPS:
  ~2–4 КБ/кадр × 25 = 50–100 КБ/с = 400–800 kbps

С RLE:
  ~20 КБ/с = 160 kbps ✓
```

### 5.4. Аудиокодек

| Параметр | G.722 (текущий) | Opus (альтернатива) | Codec2 (ультра-узкий) |
|---|---|---|---|
| Битрейт | 48–64 kbps | 6–510 kbps | 0.7–3.2 kbps |
| Частота дискретизации | 16 kHz | 8–48 kHz | 8 kHz |
| Задержка | ~1 ms (алгоритмическая) | 2.5–60 ms | ~40 ms |
| CPU нагрузка | Минимальная | Низкая–средняя | Средняя |
| Качество речи | Хорошее (wideband) | Отличное | Приемлемое |
| Лицензия | ITU-T (бесплатно) | BSD (бесплатно) | LGPL |
| Зависимости | Нет (чистый Go) | libopus (cgo) | libcodec2 (cgo) |

**Рекомендация:** G.722 для основного режима (баланс качества и простоты), Opus как опция для пользователей с достаточным каналом, Codec2 как режим «экстремально узкого канала» (спутник, 2G).

### 5.5. Терминальный рендерер

Рендеринг каждого декодированного кадра в ANSI escape-последовательности:

1. Определить размер терминала (rows × cols) через `ioctl TIOCGWINSZ`
2. Для каждого блока: сформировать ANSI-строку:
   ```
   \033[38;2;R;G;Bm\033[48;2;R;G;Bm▀
   ╰── foreground ──╯╰── background ──╯╰ glyph
   ```
3. Cursor positioning: `\033[H` (home) перед каждым кадром для перезаписи
4. Double buffering: формировать весь кадр в строку, затем `write()` атомарно
5. Differential update: перерисовывать только изменившиеся символы (cursor addressing)

### 5.6. Сетевой стек

- **Сигнализация:** TCP handshake через Yggdrasil IPv6. Обмен capabilities (видео/аудио, кодеки, размер терминала).
- **Медиа-транспорт:** UDP через Yggdrasil overlay. Формат пакета:
  ```
  [sequence: 16 bit][timestamp: 32 bit][type: 2 bit][payload: variable]
                                         │
                                         ├─ 00 = audio
                                         ├─ 01 = video (delta)
                                         ├─ 10 = video (keyframe)
                                         └─ 11 = control
  ```
- **Congestion control:** Adaptive bitrate — мониторинг RTT и packet loss, автоматическое снижение FPS/разрешения при деградации канала.
- **NAT traversal:** Yggdrasil решает эту проблему на уровне overlay-сети. Для корп. сетей с DPI: поддержка TLS/QUIC peering.

---

## 6. Пошаговый план внедрения

### Фаза 0: Подготовка (Недели 1–2)

1. **Форк и аудит кода Say.** Клонировать github.com/svanichkin/say, провести code review, выявить архитектурные решения и технический долг.
2. **Настройка CI/CD.** GitHub Actions: build для linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64. Автоматические тесты при PR.
3. **Среда разработки.** Docker-compose с 2–3 контейнерами Yggdrasil для тестирования P2P-звонков локально без внешней сети.
4. **Метрики качества.** Инструментировать: FPS, latency, packet loss, CPU%, memory, bitrate. Grafana dashboard.

### Фаза 1: MVP — Стабильный 1:1 звонок (Недели 3–8)

**Цель:** Надёжный аудио+видео звонок между двумя терминалами с автоматическим переподключением.

| Задача | Приоритет | Сложность | Срок |
|---|---|---|---|
| Стабилизация аудио (шумоподавление, AEC) | P0 | Средняя | 1 неделя |
| Adaptive bitrate controller | P0 | Высокая | 1.5 недели |
| Auto-reconnect при обрыве | P0 | Средняя | 3 дня |
| Terminal capability detection (color depth, size) | P1 | Низкая | 2 дня |
| Fallback: 256 colors, ASCII-only mode | P1 | Средняя | 3 дня |
| Man page и --help | P1 | Низкая | 1 день |
| Integration tests (2 контейнера, автозвонок) | P1 | Средняя | 3 дня |
| Бенчмарки: latency, bitrate, CPU по платформам | P2 | Средняя | 2 дня |

**Deliverable:** Бинарник v1.0-mvp, README с quick start, Docker-образ для тестирования.

### Фаза 2: Расширение возможностей (Недели 9–16)

**Цель:** Фичи, необходимые для первых пользователей из DevOps-сегмента.

1. **Текстовый чат:** Параллельный текстовый канал (как IRC) во время звонка. Отправка файлов через stdin/stdout pipe.
2. **Screen sharing:** Захват терминала собеседника (tmux-like) + видео. Режим «pair programming».
3. **Запись звонка:** Сохранение в .cast (asciinema) + .wav. Позднее — HTML-рендер для шеринга.
4. **Sixel/Kitty protocol:** Автоопределение поддержки. Если терминал поддерживает — рендеринг в полноценных пикселях.
5. **Homebrew / apt / snap пакеты:** `brew install tvcp`, `apt install tvcp`, `snap install tvcp`.
6. **Конфигурация через TOML:** Аудио/видео настройки, предпочтительные кодеки, цветовые фильтры, keybindings.

### Фаза 3: Продуктизация (Недели 17–28)

**Цель:** Превращение инструмента в продукт с бизнес-моделью.

1. **Групповые звонки (до 5 участников):** Mesh-топология: каждый участник отправляет свой поток всем. Split-screen в терминале.
2. **Веб-портал:** tvcp.io — landing page, документация, web-клиент (через xterm.js + WebSocket прокси).
3. **SaaS: Managed Yggdrasil peers:** Платные relay-ноды с гарантированным uptime для корпоративных клиентов. $9.99/мес/команда.
4. **IoT SDK:** Библиотека на Go для интеграции видеостриминга в свои приложения. API: `tvcp.NewCall(addr)`, `call.SendFrame(frame)`.
5. **Enterprise features:** SSO, audit log, compliance (SOC 2), private Yggdrasil network, dedicated relay nodes.

### Фаза 4: Масштабирование (Месяцы 7–12)

1. **Мобильные клиенты:** iOS (Termius-like) и Android (Termux-like) с встроенным Yggdrasil и камерой.
2. **Межкадровое сжатие (P-frames):** Полноценный видеокодек с keyframes и delta-frames. Ожидаемое снижение битрейта на 50–70%.
3. **Интеграции:** PagerDuty, OpsGenie, Slack — автоматический звонок при critical alert. Jenkins/GitHub Actions — видео-нотификация при failed deploy.
4. **Hardware партнёрства:** Предустановка на IoT-платформы: Raspberry Pi OS, Ubuntu Core, Balena.io.

---

## 7. Бизнес-модель и монетизация

### 7.1. Open Core модель

Базовый продукт остаётся open-source (MIT), как у Redis, GitLab, Mattermost. Монетизация через enterprise-функции и managed services.

| Уровень | Цена | Что включено |
|---|---|---|
| **Community (Open Source)** | Бесплатно | 1:1 звонки, P2P, CLI, все кодеки, MIT лицензия |
| **Pro (для команд)** | $9.99/мес/5 юзеров | Групповые звонки, запись, screen sharing, managed relay nodes |
| **Enterprise** | $49.99/мес/20 юзеров | SSO, audit log, private Yggdrasil network, SLA 99.9%, приоритетная поддержка |
| **IoT Platform** | $0.01/устройство/день | SDK, fleet management dashboard, bulk provisioning, API |
| **Satellite Tier** | Custom pricing | Dedicated relay nodes, Codec2 ультра-узкий канал, 24/7 support |

### 7.2. Финансовые проекции (консервативный сценарий)

| Метрика | Год 1 | Год 2 | Год 3 |
|---|---|---|---|
| Community пользователи | 5,000 | 25,000 | 100,000 |
| Pro подписчики (команды) | 50 | 500 | 2,500 |
| Enterprise клиенты | 5 | 30 | 100 |
| IoT устройств | 1,000 | 20,000 | 200,000 |
| MRR (Monthly Recurring Revenue) | $1,500 | $25,000 | $150,000 |
| ARR (Annual Recurring Revenue) | $18,000 | $300,000 | $1,800,000 |

---

## 8. Структура технической документации

### 8.1. Пользовательская документация

1. **Quick Start Guide (README.md):** Установка, первый звонок за 2 минуты, системные требования.
2. **User Guide:** Все команды и флаги, настройка конфига, контактная книга, цветовые фильтры, troubleshooting.
3. **Man Page (say.1):** Стандартная Unix man page для `man say`.
4. **FAQ:** Совместимость терминалов, известные проблемы, сравнение с альтернативами.

### 8.2. Техническая документация для разработчиков

1. **Architecture Decision Records (ADR):** Документирование ключевых архитектурных решений: почему G.722, почему Yggdrasil, почему квадрантные блоки.
2. **Protocol Specification:** Формат пакетов, handshake sequence, capability negotiation, error codes.
3. **.babe Format Specification:** Бинарный формат файла, алгоритм кодирования/декодирования, расширения для видео.
4. **API Reference (GoDoc):** Публичный API для Go-разработчиков: `tvcp.Dial()`, `tvcp.Listen()`, `tvcp.Frame`, `tvcp.AudioChunk`.
5. **Contributing Guide:** Code style, PR process, testing requirements, release process.

### 8.3. Операционная документация

1. **Deployment Guide:** Docker, Kubernetes (Helm chart), systemd service, launchd (macOS).
2. **Monitoring Guide:** Prometheus metrics endpoint, Grafana dashboards, alerting rules.
3. **Security Audit Report:** Анализ криптографии Yggdrasil, модель угроз, рекомендации по hardening.

---

## 9. Создание прототипа: пошаговое руководство

### 9.1. Минимальный прототип (Weekend Hack, 2 дня)

**Цель:** доказать концепцию — видеопоток с камеры, отрендеренный в терминале и переданный по сети другому терминалу.

#### День 1: Локальный видеорендеринг

```go
// main.go — терминальная камера
package main

import (
    "fmt"
    "time"
    "github.com/example/gocam"
    "github.com/nfnt/resize"
)

func main() {
    cam := gocam.Open("/dev/video0", 320, 240)
    defer cam.Close()

    for {
        frame := cam.Read()
        resized := resize(frame, 80, 60)   // 80 cols × 60 rows source

        for y := 0; y < 60; y += 2 {       // 2 rows per terminal line
            for x := 0; x < 80; x++ {
                top := resized[y][x]        // upper pixel → foreground
                bot := resized[y+1][x]      // lower pixel → background
                fmt.Printf("\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm▀",
                    top.R, top.G, top.B,
                    bot.R, bot.G, bot.B)
            }
            fmt.Print("\033[0m\n")          // reset colors, newline
        }
        fmt.Print("\033[H")                 // cursor home — overwrite
        time.Sleep(40 * time.Millisecond)   // 25 FPS
    }
}
```

#### День 2: Сетевая передача

```go
// ── Sender ──
conn, _ := net.DialUDP("udp", nil, remoteAddr)
for {
    frame := cam.Read()
    encoded := babe.Encode(frame, 70)       // quality 70%
    packet := append(makeHeader(seq, ts), encoded...)
    conn.Write(packet)
    seq++
}

// ── Receiver ──
conn, _ := net.ListenUDP("udp", localAddr)
buf := make([]byte, 65536)
for {
    n, _ := conn.Read(buf)
    header, payload := parsePacket(buf[:n])
    frame := babe.Decode(payload)
    renderToTerminal(frame)
}
```

### 9.2. Функциональный прототип (2 недели)

| День | Задача | Результат |
|---|---|---|
| 1–2 | Форк Say, build, тест локально | Работающий бинарник, первый звонок |
| 3–4 | Аудит кода, рефакторинг модулей | Чистая архитектура: `codec/`, `network/`, `ui/`, `device/` |
| 5 | Добавить adaptive bitrate | Автоматическое снижение FPS при packet loss |
| 6 | Terminal capability detection | Auto-detect 24bit/256/ASCII, размер терминала |
| 7–8 | Квадрантный режим (16 глифов) | Режим `--quadrant` с 4 субпикселями на символ |
| 9 | Шумоподавление (RNNoise) | Чистый аудиопоток без фонового шума |
| 10 | Auto-reconnect + heartbeat | Устойчивость к обрывам |
| 11–12 | Docker-compose тестовая среда | 3 контейнера, автоматический тест звонка |
| 13 | Бенчмарки + документация | README, benchmarks.md, demo GIF |
| 14 | Release v0.1.0-alpha | GitHub Release, бинарники для 5 платформ |

### 9.3. Структура проекта

```
tvcp/
├── cmd/
│   └── tvcp/
│       └── main.go              # Entry point, CLI flags
├── internal/
│   ├── codec/
│   │   ├── babe/                # .babe video encoder/decoder
│   │   │   ├── encode.go
│   │   │   ├── decode.go
│   │   │   └── quantize.go      # k-means 2-color clustering
│   │   ├── audio/
│   │   │   ├── g722.go          # G.722 codec
│   │   │   ├── opus.go          # Opus (optional)
│   │   │   └── codec2.go        # Codec2 ultra-low (optional)
│   │   └── glyphs.go            # Unicode block patterns (16 quadrants)
│   ├── device/
│   │   ├── camera_linux.go      # V4L2
│   │   ├── camera_darwin.go     # AVFoundation
│   │   ├── camera_windows.go    # Media Foundation
│   │   ├── mic.go               # Audio capture (malgo/miniaudio)
│   │   └── speaker.go           # Audio playback
│   ├── network/
│   │   ├── yggdrasil.go         # Embedded Yggdrasil node
│   │   ├── signaling.go         # TCP handshake, capability exchange
│   │   ├── transport.go         # UDP media transport
│   │   ├── congestion.go        # Adaptive bitrate controller
│   │   └── packet.go            # Packet format: header + payload
│   ├── ui/
│   │   ├── renderer.go          # ANSI terminal renderer
│   │   ├── renderer_sixel.go    # Sixel protocol (high-res)
│   │   ├── renderer_kitty.go    # Kitty graphics protocol
│   │   ├── detect.go            # Terminal capability detection
│   │   └── layout.go            # Split-screen for group calls
│   └── contacts/
│       └── contacts.go          # Folder-based address book
├── configs/
│   └── default.toml             # Default configuration
├── docs/
│   ├── architecture.md
│   ├── protocol-spec.md
│   ├── babe-format-spec.md
│   └── adr/                     # Architecture Decision Records
├── scripts/
│   ├── docker-compose.yml       # Test environment (3 Ygg nodes)
│   └── benchmark.sh
├── Makefile
├── go.mod
├── go.sum
├── README.md
└── LICENSE                      # MIT
```

---

## 10. Стратегия вывода на рынок

### 10.1. Этап 1: Community Launch (Месяц 1–3)

**Цель:** 1000 GitHub stars, 200 активных пользователей, обратная связь от early adopters.

**Каналы продвижения:**

1. **Hacker News:** `Show HN: Terminal-native video calls over P2P mesh — 200 kbps, no servers, no accounts`. Идеальная HN-тема: unusual tech + privacy + terminal culture.
2. **Reddit:** r/selfhosted, r/privacy, r/linux, r/devops, r/sysadmin, r/cyberpunk, r/commandline. Каждый сабреддит — отдельный пост с фокусом на их интересы.
3. **Habr:** Технический пост (как у svanichkin) с деталями алгоритма + демо GIF. Хабы: Демосцена, Go, Информационная безопасность, DevOps.
4. **Dev.to / Hashnode:** Tutorial-формат: «How I built video calls in 200 kbps using Unicode blocks».
5. **YouTube/Twitter:** 30-секундное демо: split screen — обычный Zoom слева, терминальный звонок справа. Wow-эффект.

### 10.2. Этап 2: DevOps Adoption (Месяц 4–6)

**Цель:** 50 Pro-подписок, интеграция с PagerDuty/OpsGenie.

1. **Homebrew / apt пакеты:** `brew install tvcp` — установка за 10 секунд.
2. **Docker Hub:** `docker run tvcp` — ready-to-use контейнер для тестирования.
3. **Конференции:** KubeCon, DevOpsDays, FOSDEM — lightning talk + live demo из терминала на сцене.
4. **Интеграция с alerting:** PagerDuty webhook → автоматический звонок дежурному. *«Your server calls you, literally.»*
5. **Blog posts:** «Why I replaced Zoom for incident response with a terminal tool» — целевой контент для DevOps-аудитории.

### 10.3. Этап 3: IoT + Satellite (Месяц 7–12)

**Цель:** 1000 IoT-устройств, 5 enterprise-клиентов, первые пилоты с нефтегазом / морским транспортом.

1. **Партнёрство с IoT-платформами:** Balena.io, Pantavisor, Mender.io — предустановка TVCP как опции для remote video monitoring.
2. **Пилот с нефтегазовой компанией:** Бесплатный 3-месячный пилот. Метрики: экономия трафика, time-to-connect, user satisfaction. Case study.
3. **Satellite ISP партнёрства:** Интеграция с Starlink, Iridium, Inmarsat BGAN. Совместный маркетинг: *«Video calls over satellite for $90/hour instead of $2700»*.
4. **Military/Gov outreach:** Контакт через SBIR/STTR программы (USA) или аналоги в EU. Демо для тактической связи.

### 10.4. Этап 4: Масштабирование (Год 2+)

1. **Series A fundraising:** $2–5M при ARR $300K+. Фокус: команда, IoT-платформа, enterprise sales.
2. **Мобильные клиенты:** iOS и Android приложения с встроенным Yggdrasil.
3. **Marketplace:** Плагины и интеграции от сообщества: OBS плагин, Slack bot, Telegram bot, VS Code extension.
4. **Конкурентный барьер:** Сетевой эффект Yggdrasil mesh + экосистема интеграций + brand recognition в нише terminal-first tools.

---

## 11. Итоговая дорожная карта

| Период | Фаза | Ключевой результат | Метрика успеха |
|---|---|---|---|
| Недели 1–2 | 🔧 Подготовка | CI/CD, тестовая среда, аудит кода | Build passing на 5 платформах |
| Недели 3–8 | 🚀 MVP | Стабильный 1:1 звонок | Latency < 200ms, 0 crashes за 1 час |
| Недели 9–16 | 📦 Расширение | Screen share, запись, пакетные менеджеры | Homebrew install работает |
| Недели 17–28 | 💼 Продуктизация | Групповые звонки, SaaS, IoT SDK | 50 Pro-подписок |
| Месяцы 7–12 | 📈 Масштабирование | Мобильные клиенты, enterprise | ARR $300K |
| Год 2 | 🌍 Рост | Series A, marketplace, партнёрства | 100K community users |

---

> **Заключение:** TVCP — это не «ещё один видеозвонок». Это новая категория продукта, которая открывает видеосвязь там, где она была физически невозможна: в терминалах, на IoT-устройствах, через спутниковые каналы, в mesh-сетях без интернета. Проект Say доказал жизнеспособность концепции. Следующий шаг — превратить эксперимент энтузиаста в продукт, который изменит представление о том, где и как может работать видеосвязь.

---

*Документ создан на основе анализа проектов [Say](https://github.com/svanichkin/say) и [.babe](https://github.com/svanichkin/babe) Сергея Ваничкина, а также [статьи на Habr](https://habr.com/ru/articles/973868/).*
